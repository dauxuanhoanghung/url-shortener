# Backend Architecture

This document describes the **backend runtime architecture**: how packages
fit together, why pgx/pgxpool is used directly instead of an ORM, how sqlc
is layered on top of pgxpool, and the cache facade / fallback pattern in
`internal/cache`.

It complements [15-backend-folder-structure.md](15-backend-folder-structure.md)
(tree view) with rationale, wiring, and the "why" behind each choice.

---

## 1. Request Flow

```text
HTTP
 │
 ▼
router (gin)
 │
 ▼
middleware  ── JWT, CORS, rate-limit
 │
 ▼
handler     ── parse request, call service, format response
 │
 ▼
service     ── business rules, plan limits, validation
 │         \
 │          └──► cache (facade)  ── Redis + in-memory fallback
 ▼
repository  ── SQL via pgxpool
 │
 ▼
PostgreSQL
```

Rule: layers only call the layer directly below. Handlers never touch
`repository`; repositories never touch `http.Request`.

---

## 2. Package Layout

```text
backend/
├── cmd/api/              entrypoint (wires everything)
├── internal/
│   ├── config/           env loading, typed Config struct
│   ├── database/         pgxpool construction
│   ├── cache/            Cache interface + Redis/in-memory drivers + Chain
│   ├── router/           gin routes + middleware registration
│   ├── middleware/       JWT, email-verification, rate-limit
│   ├── handler/          HTTP layer
│   ├── service/          business logic
│   ├── repository/       SQL
│   ├── model/            domain structs
│   ├── dto/              API request/response contracts
│   ├── event/            in-memory event bus + domain event types + handlers
│   ├── sse/              SSE hub (user → channel map)
│   ├── mailer/           Mailer interface + SMTP/sendmail/console transports
│   └── worker/           background goroutine pools (metadata fetch, planned: cleanup)
├── pkg/
│   ├── logger/           zap wrapper
│   ├── ratelimit/        generic Limiter (no HTTP dep) — used by middleware + services
│   ├── validator/        shared validation helpers
│   └── utils/            short code, jwt helpers
├── migrations/           *.sql, applied in order, never modified
└── tests/                integration tests
```

Full tree in [15-backend-folder-structure.md](15-backend-folder-structure.md).

---

## 3. Database Access — pgx/pgxpool + sqlc

All repositories use sqlc-generated queries behind hand-written domain
interfaces. pgxpool is the underlying driver.

| Repository         | Backing SQL file          | Notes                                     |
| ------------------ | ------------------------- | ----------------------------------------- |
| `user_repository`  | `queries/users.sql`       |                                           |
| `url_repository`   | `queries/short_urls.sql`  |                                           |
| `plan_repository`  | `queries/plans.sql`       | read-only (plans are seeded, not written) |

All `Create*` repository methods **return the persisted entity** (not
just `error`). That lets callers chain on DB-assigned fields
(`created_at`, future defaults, sequence IDs) without a second
round-trip. Services pass the returned pointer directly to the next
step instead of re-reading their own input:

```go
user, err := s.userRepo.Create(ctx, &model.User{...})
if err != nil { return nil, err }
return s.generateAuthResponse(user)   // uses the row the DB actually wrote
```

### 3.1 Why pgxpool as the driver

- **Native Postgres protocol** — binary encoding, prepared-statement
  cache, COPY, LISTEN/NOTIFY. Fastest mainstream Go driver.
- **Transparent SQL** — no ORM translation layer between code and
  EXPLAIN output.
- **Full Postgres features** — JSONB, arrays, UUID are first-class.

### 3.2 Why not GORM / ent / bun

| Concern           | GORM / ORM                      | pgxpool (+ sqlc)      |
| ----------------- | ------------------------------- | --------------------- |
| SQL visibility    | hidden, generated at runtime    | literal in `.sql`     |
| N+1 risk          | high (implicit lazy loads)      | impossible (explicit) |
| Postgres features | partial (JSONB, arrays awkward) | full (native types)   |
| Type safety       | interface{} / reflection        | compile-time (sqlc)   |
| Learning value    | learns the ORM                  | learns SQL + pgx      |

### 3.3 How sqlc fits in

[sqlc](https://sqlc.dev) generates Go code from `.sql` files. It does
**not** manage migrations, does **not** run at request time, and does
**not** hide SQL — the generated `*.sql.go` files contain the exact
query strings you wrote.

Layout:

```text
backend/
├── sqlc.yaml                              ← config, pinned to pgx/v5
├── migrations/
│   ├── 0001_initial_schema.sql            ← schema (sqlc reads for types)
│   └── 0002_add_plans_table.sql
└── internal/repository/
    ├── queries/                           ← source of truth (hand-written)
    │   ├── users.sql
    │   ├── short_urls.sql
    │   └── plans.sql
    ├── sqlc/                              ← GENERATED — do not edit
    │   ├── db.go                          ← DBTX + Queries + WithTx(tx)
    │   ├── models.go                      ← struct per table
    │   ├── querier.go                     ← Querier interface
    │   └── *.sql.go                       ← one file per queries/*.sql
    ├── user_repository.go                 ← adapter: domain ⇄ sqlc
    ├── url_repository.go
    └── plan_repository.go
```

How an adapter works — [user_repository.go](../backend/internal/repository/user_repository.go):

```go
type userRepository struct {
    q *sqlc.Queries
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
    return &userRepository{q: sqlc.New(db)}   // sqlc.Queries accepts DBTX
}

func (r *userRepository) Create(ctx context.Context, u *model.User) (*model.User, error) {
    row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{...})
    if err != nil {
        return nil, err
    }
    // RETURNING gives back the row the DB actually wrote; map → domain.
    return &model.User{ID: row.ID, ...}, nil
}
```

Two reasons the adapter exists (thin as it is):

1. **Type conversion.** sqlc maps nullable columns to `pgtype.Timestamp`
   / `pgtype.Int8` / `[]byte` (for JSONB). Domain models use `time.Time`,
   `int64`, `map[string]bool`, etc. The adapter is the only place that
   knows about `pgtype`, keeping the service layer clean.
2. **Stable interface.** Services depend on `repository.UserRepository`,
   not `sqlc.Queries`. Regenerating sqlc cannot break a service's
   compile even if the generated signatures change — only the adapter
   needs to be updated.

Workflow when schema changes:

```text
1. add migration file (0003_*.sql)
2. add / edit queries/*.sql
3. make sqlc-generate            ← regenerates internal/repository/sqlc/
4. fix compile errors in the adapter (Create* / Get* signatures may change)
5. services are untouched
```

Bad SQL fails at step 3, not at runtime in prod.

### 3.3.1 `Create` returns the entity

Repository `Create` methods return `(*model.X, error)`, not just
`error`. The SQL uses `RETURNING *` so the returned pointer reflects
what the DB actually wrote (including defaults like `created_at`
if omitted by the caller).

Service-side ID generation is kept (service owns the UUID, short-code
retry loop, etc.) — `RETURNING` is not used to auto-generate, it is
used so the caller gets a single round-trip answer it can pass
downstream without re-reading its own input.

### 3.4 Tooling

Config: [backend/sqlc.yaml](../backend/sqlc.yaml). Pinned to:

- `engine: postgresql`
- `sql_package: pgx/v5`
- `emit_interface: true` (emits `Querier` for mocking)
- UUID override → `github.com/google/uuid.UUID` (instead of `[16]byte`)

Install once per machine:

```bash
make sqlc-install        # wraps: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Regenerate after editing any `queries/*.sql` or migration:

```bash
make sqlc-generate
```

Both targets are in the repo-root [Makefile](../Makefile). The generate
target falls back to `$(go env GOPATH)/bin/sqlc` if `sqlc` is not on
`PATH` (common when installed via `go install`).

### 3.5 Transactions

sqlc generates `(*Queries).WithTx(tx pgx.Tx) *Queries`, which returns a
new `*Queries` bound to the transaction. Service layer owns the
transaction boundary:

```go
tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
defer tx.Rollback(ctx)

qtx := sqlc.New(tx)     // or: existingQueries.WithTx(tx)
qtx.UpdateUserPlan(ctx, ...)
qtx.CreateSubscription(ctx, ...)

return tx.Commit(ctx)
```

For the raw-pgxpool repositories, wrap the `*pgxpool.Pool` and `pgx.Tx`
behind the same `DBTX` interface sqlc uses (`Exec`, `Query`, `QueryRow`)
so the method signatures don't need a transaction variant.

---

## 4. Cache Layer

Located in [backend/internal/cache/](../backend/internal/cache/).

### 4.1 Goals

- services depend on an interface, not on `*redis.Client`
- swappable drivers for tests (in-memory) and prod (Redis / Valkey)
- **resilience**: if Redis is unreachable, redirects should still work

### 4.2 Files

| File           | Role                                                    |
| -------------- | ------------------------------------------------------- |
| `cache.go`     | `Cache` interface + `ErrCacheMiss` sentinel             |
| `redis.go`     | Redis/Valkey driver (implements `Cache`)                |
| `in_memory.go` | process-local driver (mutex + TTL, no deps)             |
| `fallback.go`  | `Chain` — composes multiple caches (facade + fallback)  |
| `factory.go`   | `New(ctx, Config)` — builds the right cache from config |

### 4.3 The Interface

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    // Increment atomically increments key by 1, sets ttl on first creation,
    // and returns the new value. Drivers that cannot safely implement
    // distributed counters return ErrNotSupported.
    Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
    Close() error
}

var ErrCacheMiss    = errors.New("cache: miss")
var ErrNotSupported = errors.New("cache: operation not supported by this driver")
```

Drivers translate their own "not found" into `ErrCacheMiss` so callers
can branch portably:

```go
v, err := c.Get(ctx, key)
switch {
case err == nil:
    // hit
case errors.Is(err, cache.ErrCacheMiss):
    // miss — go to DB
default:
    // driver error — log, fall through to DB
}
```

`Increment` uses Redis `INCR` + `EXPIRE` in a pipeline — atomic at the
command level. The in-memory driver returns `ErrNotSupported` because
per-process counters break global rate limits across replicas.
`Chain.Increment` delegates to `caches[0]` (the primary) only — it
never fans out to in-memory replicas.

Values are raw `[]byte`. Serialization (JSON, proto, plain string) is
the caller's job. This keeps the cache package free of domain types.

### 4.4 The Facade / Fallback Pattern — `Chain`

Your question: "is there any fallback or facade pattern for this?"
Yes — that is what [fallback.go](../backend/internal/cache/fallback.go)
adds.

`Chain` implements `Cache` by delegating to an ordered list of inner
caches:

```go
chain := cache.NewChain(redisCache, inMemoryCache)
```

**Read path** — first hit wins:

```text
Get(key)
 ├─ caches[0].Get → hit?     ── return
 │                 miss/err? ── continue
 ├─ caches[1].Get → hit?     ── return
 │                 miss/err? ── continue
 └─ return ErrCacheMiss (or last driver error)
```

Driver errors are not fatal: if Redis returns a network error, `Chain`
skips to the in-memory layer. That is the fallback.

**Write path** — fan-out:

```text
Set(key, value, ttl)
 ├─ caches[0].Set   ← authoritative, error is returned
 ├─ caches[1].Set   ← best-effort, error ignored
 └─ ...
```

Every `Set` mirrors the value into every layer, so after a Redis
read-through the in-memory layer also has a warm copy for the next read
if Redis goes down mid-request.

This is three patterns at once:

- **Facade**: one `Cache` value hides "which store served this?"
- **Chain of responsibility**: each inner cache decides hit vs. pass.
- **Cache-aside fallback**: degraded but working when the remote is down.

### 4.5 Wiring

In [cmd/api/main.go](../backend/cmd/api/main.go):

```go
appCache, err := cache.New(ctx, cache.Config{
    Driver:   "redis",
    Addr:     cfg.Redis.Addr(),
    Fallback: true,
})
if err != nil {
    log.Fatal("failed to init cache", zap.Error(err))
}
defer appCache.Close()

redirectService := service.NewRedirectService(urlRepo, appCache)
```

`Fallback: true` tells the factory to wrap the primary driver in a
`Chain` with a fresh `InMemoryCache`. For tests, pass `Driver: "memory"`
— no Redis required.

### 4.6 Service Usage

`service.RedirectService` depends on `cache.Cache`, never on
`*redis.Client`. See
[redirect_service.go](../backend/internal/service/redirect_service.go).

This is why the rewire was needed: the old constructor took
`*redis.Client`, which couldn't be swapped for in-memory or a chain.

### 4.7 When NOT to Use the Fallback

In-memory fallback is per-process and not shared across replicas. So:

- ✅ Redirect cache — stale reads during Redis outage are fine
- ✅ Public plan entitlements — read-mostly, staleness OK for 1h TTL
- ❌ Rate limiting — per-replica counters defeat the global limit
- ❌ Distributed locks — obviously not
- ❌ Stripe idempotency keys — correctness requires a shared store

For those cases, use `Driver: "redis"` with `Fallback: false` and let
the operation fail loudly if Redis is down.

### 4.8 Future Additions

- `SetNX(ctx, key, value, ttl)` for idempotency keys and distributed locks
- metrics: hit/miss per inner cache (wrap `Chain` with a metrics decorator)

Add only when a concrete caller needs them — avoid speculative interface bloat.

---

## 5. Dependency Injection

Constructor injection, no global state. Wiring happens once in
[cmd/api/main.go](../backend/cmd/api/main.go):

```text
pgxpool  ──► repositories ──► services ──► handlers ──► router
cache    ──►                   ▲
config   ───────────────────────┘
```

Every constructor returns an interface so tests can substitute fakes:

```go
func NewURLService(repo repository.URLRepository) URLService
func NewRedirectService(repo repository.URLRepository, c cache.Cache) RedirectService
```

No DI container. Graph is small; a 30-line `main.go` is clearer than
wire/fx config.

---

## 6. Error Handling

- Repositories expose typed sentinels (`ErrURLNotFound`,
  `ErrShortCodeConflict`).
- Services translate repository errors into service-level errors
  (`ErrPlanLimitReached`, `ErrURLForbidden`).
- Handlers translate service errors into the response contract from
  [18-error-handling-contract.md](18-error-handling-contract.md).

Raw `error.Error()` never reaches the client.

---

## 7. Configuration

[internal/config/config.go](../backend/internal/config/config.go) loads
from env vars (with `.env` fallback via `godotenv`). All config is
consumed at `main.go` boot time and passed explicitly to constructors.

Never call `os.Getenv` outside `config/`.

---

## 8. Testing Strategy (Target)

- Repositories: integration tests against a real Postgres (docker).
  Mocking pgxpool is not worth the effort; a real DB catches migration
  drift.
- Services: unit tests with fake `repository.URLRepository` and
  `cache.Cache` (the in-memory driver works as-is).
- Handlers: `httptest` with fake services.

See [19-testing-strategy.md](19-testing-strategy.md) for coverage targets.

---

## 9. Rate Limiting — `pkg/ratelimit`

Located in [backend/pkg/ratelimit/limiter.go](../backend/pkg/ratelimit/limiter.go).

`pkg/ratelimit` is a thin generic wrapper over `cache.Cache.Increment`.
It has no dependency on Gin or `net/http` so it can be used from
middleware, services, or CLI tools.

```go
limiter := ratelimit.New(appCache)

// anywhere — middleware, service, CLI
allowed, err := limiter.Allow(ctx, ratelimit.Key("login", ip), 10, time.Minute)
```

`Key(namespace, identity)` builds canonical Redis key strings:

```
rate_limit:login:192.168.1.1
rate_limit:/api/v1/urls:user:abc-123
```

The HTTP middleware in `internal/middleware/rate_limit_middleware.go` is
a thin adapter: it resolves the identity (user ID if `AuthRequired` has
already run, otherwise client IP), then calls `limiter.Allow()`.

Why a separate `pkg/ratelimit` instead of embedding logic in middleware:

- A future `LoginAttemptService` can take `*ratelimit.Limiter` as a
  constructor dep and track failed logins without touching HTTP.
- The `Key()` helper keeps key formats consistent across all callers.
- The package is unit-testable with the in-memory driver (even though
  in-memory returns `ErrNotSupported`, tests can swap in a fake cache).

---

## 10. Summary of Architectural Decisions

| Decision                        | Rationale                                                                  |
| ------------------------------- | -------------------------------------------------------------------------- |
| pgxpool instead of GORM         | transparent SQL, performance, learning goal                                |
| sqlc across all repositories    | type-safe codegen without hiding SQL                                       |
| Hand-written repo interfaces    | services decoupled from sqlc-generated code + adapter owns type conversion |
| `Create` returns `*model.X`     | single round-trip, caller reuses DB-written row downstream                 |
| Plans in a seeded table         | entitlement changes auditable via migration history                        |
| `cache.Cache` interface         | swap drivers, mockable in tests                                            |
| `Cache.Increment` on interface  | atomic distributed counter — Redis INCR, in-memory returns ErrNotSupported |
| `Chain` fallback                | redirects survive Redis outages                                            |
| `Chain.Increment` → primary only| per-replica counters break global rate limits                              |
| Redis primary for cache         | shared across replicas, required for rate limiting / locks                 |
| `pkg/ratelimit.Limiter`         | HTTP-agnostic counter usable from middleware, services, and CLI            |
| Constructor injection in main   | small graph, no runtime DI framework                                       |
