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
│   └── admin/                         # (planned) CLI-only admin account management, see doc 25
│       └── main.go                    # subcommands: create-admin, list-admins, disable-admin, reset-admin-pw
│
├── internal/
│   ├── config/
│   │   └── config.go                  # env loading, typed Config struct
│   │
│   ├── database/
│   │   └── postgres.go                # pgxpool construction
│   │
│   ├── cache/                         # see §4 of 23-backend-architecture.md
│   │   ├── cache.go                   # Cache interface + ErrCacheMiss
│   │   ├── redis.go                   # Redis / Valkey driver
│   │   ├── in_memory.go               # process-local driver
│   │   ├── fallback.go                # Chain = facade + fallback
│   │   └── factory.go                 # New(ctx, Config) builder
│   │
│   ├── router/
│   │   └── router.go                  # gin routes + middleware registration
│   │
│   ├── middleware/
│   │   └── auth_middleware.go         # JWT
│   │   # rate_limit_middleware.go     (planned)
│   │
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── url_handler.go
│   │   └── redirect_handler.go
│   │   # billing_handler.go           (planned)
│   │
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── url_service.go
│   │   └── redirect_service.go
│   │   # billing_service.go           (planned)
│   │
│   ├── repository/
│   │   ├── queries/                   # sqlc source-of-truth SQL
│   │   │   ├── users.sql
│   │   │   ├── short_urls.sql
│   │   │   └── plans.sql
│   │   ├── sqlc/                      # GENERATED — do not edit
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   ├── querier.go
│   │   │   ├── users.sql.go
│   │   │   ├── short_urls.sql.go
│   │   │   └── plans.sql.go
│   │   ├── user_repository.go         # adapter over sqlc.Queries
│   │   ├── url_repository.go          # adapter over sqlc.Queries
│   │   └── plan_repository.go         # read-only: GetByCode, List
│   │   # subscription_repository.go   (planned)
│   │
│   ├── model/
│   │   ├── user.go
│   │   ├── short_url.go
│   │   └── plan.go
│   │   # subscription.go              (planned)
│   │
│   ├── dto/
│   │   ├── auth_dto.go
│   │   ├── url_dto.go
│   │   └── response_dto.go
│   │
│   └── worker/                        # background jobs (planned)
│
├── pkg/
│   ├── logger/
│   │   └── logger.go                  # zap wrapper
│   ├── validator/                     # planned
│   └── utils/
│       ├── jwt.go
│       └── shortcode.go
│
├── migrations/
│   ├── 0001_initial_schema.sql        # additive only, never edited after apply
│   └── 0002_add_plans_table.sql       # plans + seed + users.plan_code FK
│
├── sqlc.yaml                          # sqlc config (engine, pgx/v5, overrides)
├── tests/
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
They implement hand-written domain interfaces so services don't import
generated types. `Create` methods return the persisted entity via
`RETURNING *` so the caller gets a single round-trip answer.
Full rationale + workflow: [23-backend-architecture.md §3](23-backend-architecture.md).

---

### cache

Driver-agnostic `Cache` interface with Redis and in-memory
implementations, plus a `Chain` facade that provides transparent
fallback when the primary (Redis) is unreachable.

Details: [23-backend-architecture.md §4](23-backend-architecture.md).

---

### middleware

Cross-cutting concerns: JWT validation, rate limiting, logging, CORS,
panic recovery.

---

### worker

Background jobs (inactive-URL cleanup, analytics rollups, webhook
delivery). Planned. See [10-background-jobs.md](10-background-jobs.md).
