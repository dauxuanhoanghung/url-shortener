# Testing Strategy

Tests are layered bottom-up, matching the dependency direction in
[23-backend-architecture.md](23-backend-architecture.md):

```text
pkg/          ← simplest, no deps, fastest feedback  (step 1)
  ↑
cache/        ← in-memory drop-in, no infra          (step 2)
  ↑
repository/   ← real Postgres (integration)          (step 3)
  ↑
service/      ← fake repo + fake mailer (unit)       (step 4)
  ↑
handler/      ← httptest + fake services             (step 5)
  ↑
end-to-end    ← full stack, happy paths only         (step 6)
```

Write tests in this order when adding a feature. Each layer only trusts
the layers below it. A service test does **not** need a real DB; a repo
test **does**.

---

## 1. What's in place today

| Layer                       | Coverage | Notes                                    |
| --------------------------- | -------- | ---------------------------------------- |
| `pkg/logger`                | 100%     | zap wrapper — release vs dev mode        |
| `pkg/utils` (shortcode+jwt) | 90.9%    | missing branch: `crypto/rand` error path |
| everything else             | 0%       | tracked in this doc as next steps        |

Run:

```bash
make test-pkg       # pkg/ only — fastest
make test           # full backend test suite
make test-cover     # with coverage per package
```

---

## 2. Layer 1 — `pkg/` (pure functions)

**What**: functions that take primitives, return primitives, touch no
external state. [pkg/logger/](../backend/pkg/logger/),
[pkg/utils/](../backend/pkg/utils/).

**How**:

- standard library `testing` only, no testify or mockery
- table-driven when inputs vary in shape
- subtests (`t.Run(name, ...)`) for readable failure names
- no `init()`, no package-level test setup — each test is self-contained

**Test shape** — see [pkg/utils/jwt_test.go](../backend/pkg/utils/jwt_test.go)
for a worked example covering: roundtrip, wrong secret, malformed input,
expired token, wrong signing key.

**Rule**: if a `pkg/` function grows an external dependency (reads env,
touches DB, makes HTTP call), it no longer belongs in `pkg/` — move it
to `internal/` and write a higher-layer test instead.

---

## 3. Layer 2 — `cache/` (in-memory driver)

**What**: [cache.InMemoryCache](../backend/internal/cache/in_memory.go)
and [cache.Chain](../backend/internal/cache/fallback.go).

**How**:

- still stdlib-only — `InMemoryCache` needs no infra
- **not** the Redis driver — that's integration (layer 3)
- test: TTL expiry, delete, fan-out writes in `Chain`, fallback on
  primary error (use a stub `Cache` that errors on demand)

**Rule**: test `Chain` with a fake "always-errors" cache to prove the
fallback actually kicks in. Don't mock Redis — that's a lie about the
real behavior.

---

## 4. Layer 3 — `repository/` (real Postgres)

**What**: all files in [internal/repository/](../backend/internal/repository/).
These are thin adapters over sqlc; the risk is in SQL correctness, not
Go logic.

**How**:

- **integration tests against a real Postgres container**. Do not mock
  `pgxpool.Pool`. A mocked DB catches zero migration drift.
- per-test transaction + rollback, or per-suite database reset via
  `TRUNCATE ... RESTART IDENTITY CASCADE`
- use build tag `//go:build integration` so `make test` can skip them
  on fast feedback loops

**Suggested helper** (not yet built):

```go
// tests/integration_helpers.go
func NewTestDB(t *testing.T) *pgxpool.Pool {
    // spin up testcontainers or connect to a DSN from env,
    // apply migrations, return pool + t.Cleanup for rollback
}
```

**What to cover per repo**:

- `Create` returns the row with DB-written fields populated
- `Get*` returns `ErrNotFound` on missing row (typed sentinel, not raw
  `pgx.ErrNoRows`)
- `Create` surfaces unique-violation as `ErrShortCodeConflict` (or
  equivalent domain sentinel)
- nullable column mapping: `pgtype.Timestamp` ↔ `*time.Time`

---

## 5. Layer 4 — `service/` (business logic)

**What**: everything in [internal/service/](../backend/internal/service/).

**How**:

- unit tests with **fakes, not mocks** — hand-written structs
  implementing the repo interface, no testify/mockery
- `cache.Cache`: use the real `InMemoryCache` (it's free)
- `mailer.Mailer`: use a `captureMailer` that appends sent messages to a
  slice so tests can assert what was sent
- no DB, no Redis, no network — services must stay fast

**What to cover**:

- `auth_service.Register` → creates user, issues verify-email token,
  sends email with the right link
- `auth_service.ForgotPassword` → uniform return even for unknown email
  (no leak)
- `auth_service.ResetPassword` → rejects expired token
  (`ErrTokenInvalid`), marks token used, updates hash
- `url_service.Create` → enforces plan limit, retries on short-code
  conflict, gives up after 5 attempts
- `redirect_service.Resolve` → cache hit path, cache miss → DB → cache
  fill path

**Fake shape**:

```go
type fakeUserRepo struct {
    byEmail map[string]*model.User
    created []*model.User
}

func (f *fakeUserRepo) Create(ctx context.Context, u *model.User) (*model.User, error) {
    f.created = append(f.created, u)
    f.byEmail[u.Email] = u
    return u, nil
}
// ... implement the rest of UserRepository
```

Hand-written fakes beat mocks for clarity: the test reader sees what
the fake does, no matcher DSL to decode.

---

## 6. Layer 5 — `handler/` (HTTP boundary)

**What**: everything in [internal/handler/](../backend/internal/handler/).

**How**:

- `net/http/httptest` + `gin.TestMode`
- fake service (hand-written, as in layer 4)
- assert status code, error code in body (string-match on `error.code`),
  and shape of the success payload

**What to cover**:

- status code mapping (does `ErrPlanLimitReached` → 403 `PLAN_LIMIT_REACHED`?)
- request validation (missing fields → 400)
- middleware order (unauth → 401 before handler runs)

**Rule**: handler tests should not exercise business rules. If a test
needs to set up complex state, it belongs in layer 4.

---

## 7. Layer 6 — end-to-end (happy path only)

**What**: the full stack, one flow at a time. Run rarely — slow,
flaky-prone.

**How**: same Postgres container as layer 3 + the real API server +
a real HTTP client.

**What to cover**:

- register → verify email token from DB → login → create URL → redirect
- forgot password → reset → login with new password
- admin CLI `create-admin` → admin row exists, `admin_audit` row written

Skip UI E2E until the frontend has more than one flow that matters.

---

## 8. Coverage targets

| Layer       | Target | Rationale                                            |
| ----------- | ------ | ---------------------------------------------------- |
| `pkg/`      | 95%+   | pure functions, no excuse                            |
| `cache/`    | 90%+   | same                                                 |
| `service/`  | 80%+   | main risk surface, matching [CLAUDE.md](../CLAUDE.md) |
| `handler/`  | 70%+   | error-code mapping is the point, not handler loops   |
| `repository/` | n/a (integration) | tracked as "all queries tested once" |

Coverage targets are floors, not ceilings. A line hit once by a happy
path is not the same as a line with meaningful assertions.

---

## 9. What we don't do

- **No testify.** `if err != nil { t.Fatal(err) }` is fine.
- **No mockery / gomock.** Hand-written fakes — fewer moving parts.
- **No snapshot tests.** They rot and no one updates them intentionally.
- **No mocking of `*pgxpool.Pool` or `*redis.Client`.** Either use the
  real driver (integration) or don't touch it (move the call up to a
  layer that can be tested with a fake interface).

---

## 10. Frontend testing

Not yet — intentionally deferred. The frontend has one non-trivial
flow (login → create URL); unit tests on the Pinia store and a single
E2E (Playwright) on the register → create-URL flow are the natural
first step when it arrives. Tracked in the roadmap, not here.

---

## 11. Next steps

Ordered by value / cost:

1. **cache unit tests** — `InMemoryCache` TTL, `Chain` fallback. Free,
   high value (covers the resilience claim in doc 23).
2. **service unit tests with fakes** — start with `auth_service`
   (biggest behavior surface: register + verify + reset).
3. **integration harness** — `tests/integration/` package with a
   `NewTestDB(t)` helper; lets all repo tests land in one PR.
4. **handler tests** — status-code mapping, one test per handler method.
5. **E2E** — `tests/e2e/` with docker-compose bring-up. Last, not first.
