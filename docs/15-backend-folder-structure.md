# Backend Folder Structure (Go + Gin)

This is the **tree view** of the backend layout. For the **why** — rationale,
database access strategy (pgxpool / sqlc), cache facade — see
[23-backend-architecture.md](23-backend-architecture.md).

---

## Current Structure

```text
backend/
├── cmd/
│   ├── api/
│   │   └── main.go                    # wires config → pool → cache → repos → services → handlers
│   └── admin/                         # CLI-only admin account management, see doc 25
│       └── main.go                    # reads password from stdin, creates user with role='admin'
│
├── internal/
│   ├── config/
│   │   └── config.go                  # env loading, typed Config struct
│   │
│   ├── database/
│   │   └── postgres.go                # pgxpool construction
│   │
│   ├── cache/                         # see §4 of 23-backend-architecture.md
│   │   ├── cache.go                   # Cache interface + ErrCacheMiss + ErrNotSupported
│   │   ├── redis.go                   # Redis/Valkey driver (Increment via INCR+EXPIRE pipeline)
│   │   ├── in_memory.go               # process-local driver (Increment returns ErrNotSupported)
│   │   ├── fallback.go                # Chain facade: Increment delegates to primary only
│   │   └── factory.go                 # New(ctx, Config) builder
│   │
│   ├── router/
│   │   └── router.go                  # gin routes + middleware registration
│   │
│   ├── middleware/
│   │   ├── auth_middleware.go         # JWT validation, sets userID + role in context
│   │   ├── verified_email_middleware.go # email grace-period enforcement (7-day window)
│   │   ├── rate_limit_middleware.go   # per-IP / per-user HTTP rate limiting via pkg/ratelimit
│   │   └── admin_middleware.go        # RequireAdmin — 403 ADMIN_REQUIRED for non-admin JWT
│   │
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── url_handler.go
│   │   ├── redirect_handler.go
│   │   ├── plan_handler.go
│   │   ├── sse_handler.go
│   │   ├── admin_handler.go           # /admin/users, /admin/plans/:code/features, /admin/audit
│   │   └── # billing_handler.go       (planned — Phase 3)
│   │
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── url_service.go
│   │   ├── redirect_service.go
│   │   ├── plan_service.go
│   │   ├── admin_service.go           # write actions always record an admin_audit row
│   │   └── # billing_service.go       (planned — Phase 3)
│   │
│   ├── repository/
│   │   ├── queries/                   # sqlc source-of-truth SQL
│   │   │   ├── users.sql
│   │   │   ├── user_plans.sql
│   │   │   ├── short_urls.sql
│   │   │   ├── url_metadata.sql
│   │   │   ├── plans.sql
│   │   │   ├── tokens.sql
│   │   │   └── admin_audit.sql
│   │   ├── sqlc/                      # GENERATED — do not edit
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   ├── querier.go
│   │   │   └── *.sql.go
│   │   ├── user_repository.go
│   │   ├── user_plan_repository.go
│   │   ├── url_repository.go
│   │   ├── url_metadata_repository.go
│   │   ├── plan_repository.go
│   │   ├── token_repository.go
│   │   ├── admin_audit_repository.go
│   │   └── # subscription_repository.go (planned — Phase 3)
│   │
│   ├── model/
│   │   ├── user.go
│   │   ├── user_plan.go
│   │   ├── short_url.go
│   │   ├── url_metadata.go
│   │   └── plan.go
│   │
│   ├── dto/
│   │   ├── auth_dto.go
│   │   ├── url_dto.go
│   │   ├── plan_dto.go
│   │   └── response_dto.go
│   │
│   ├── event/
│   │   ├── bus.go                     # in-memory event bus (Publish/Subscribe, Sync/Async)
│   │   ├── events.go                  # event types: UserRegistered, URLCreated, etc.
│   │   └── handler/
│   │       ├── send_verification_email.go
│   │       ├── send_password_reset_email.go
│   │       ├── enqueue_metadata_fetch.go
│   │       └── token_helpers.go       # shared: random bytes → SHA-256 hash → base64
│   │
│   ├── sse/
│   │   └── hub.go                     # SSE hub: user → channel map, Subscribe/Notify
│   │
│   ├── mailer/
│   │   ├── mailer.go                  # Mailer interface + Message struct
│   │   ├── console.go                 # dev/fallback transport (logs to zap)
│   │   ├── smtp.go                    # SMTP + STARTTLS via gomail
│   │   ├── sendmail.go                # pipes to local MTA
│   │   └── factory.go                 # probes transport at startup, falls back to console
│   │
│   └── worker/
│       └── metadata_worker.go         # pool of goroutines: fetch og:title/desc, detect dead links
│
├── pkg/
│   ├── logger/
│   │   └── logger.go                  # zap wrapper
│   ├── ratelimit/
│   │   └── limiter.go                 # Limiter.Allow() + Key() helper — no HTTP dependency
│   ├── validator/
│   └── utils/
│       ├── jwt.go
│       └── shortcode.go
│
├── migrations/
│   ├── 0001_initial_schema.sql
│   ├── 0002_add_plans_table.sql
│   ├── 0003_add_user_lifecycle.sql
│   ├── 0004_add_admin_accounts.sql
│   └── 0005_extract_user_plans.sql
│
├── tests/
├── sqlc.yaml
├── go.mod
└── go.sum
```

---

## Layer Responsibilities

### handler

HTTP request/response only. Parses input, calls a service, formats
output using the contract from
[18-error-handling-contract.md](18-error-handling-contract.md).

Must NOT contain business logic or SQL.

---

### service

Business rules. Examples: plan-limit check, short-code generation with
collision retry, Stripe workflow, cache coordination.

Owns transaction boundaries when multi-statement operations are needed.

---

### repository

Database access only. One file per aggregate (user, url, subscription).
Each exposes a hand-written interface (`URLRepository`, `UserRepository`)
so services can be unit-tested against fakes.

All repositories are thin adapters over sqlc-generated code (`sqlc/`).
`Create` methods return the persisted entity via `RETURNING *` so the
caller gets a single round-trip answer.

Full rationale + workflow: [23-backend-architecture.md §3](23-backend-architecture.md).

---

### cache

Driver-agnostic `Cache` interface with Redis and in-memory
implementations, plus a `Chain` facade that provides transparent
fallback when the primary (Redis) is unreachable.

`Increment` is part of the interface for distributed atomic counters
(rate limiting). In-memory driver returns `ErrNotSupported` — callers
that need atomic counters must have Redis available.

Details: [23-backend-architecture.md §4](23-backend-architecture.md).

---

### middleware

Cross-cutting HTTP concerns. Currently:

- `auth_middleware.go` — validates JWT, injects `userID`, `email`, `role` into `gin.Context`
- `verified_email_middleware.go` — blocks mutation endpoints past the 7-day grace period
- `rate_limit_middleware.go` — thin adapter over `pkg/ratelimit.Limiter`; resolves identity (user ID if authed, IP otherwise), then calls `Limiter.Allow()`
- `admin_middleware.go` — `RequireAdmin()` reads the `role` context key (set by `AuthRequired`); 403 `ADMIN_REQUIRED` for non-admin JWTs

---

### pkg/ratelimit

Generic distributed rate limiter with no HTTP or Gin dependency.
`Limiter.Allow(ctx, key, limit, window)` returns `(bool, error)`.
`Key(namespace, identity)` builds canonical Redis key strings.

Can be used from middleware, services, or CLI tools — not tied to HTTP.

---

### event

In-memory event bus. Services publish domain events; async handlers
send emails, enqueue metadata fetches, etc. Decouples side-effects
from core business logic.

See [28-event-driven-service-layer.md](28-event-driven-service-layer.md).

---

### worker

Background goroutine pools. Currently: `MetadataWorker` fetches og:title,
description, favicon for new URLs and soft-deletes dead links (4xx).
Planned: cleanup worker, analytics aggregation.

See [10-background-jobs.md](10-background-jobs.md).
