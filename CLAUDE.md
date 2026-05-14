# CLAUDE.md

Instructions for AI coding agents working on this repository.
Goal: generated code is **consistent, safe, maintainable, and aligned with project architecture**.

---

# 1. Project Overview

A **SaaS URL shortener platform** built as a Go learning project.

Implemented features (as of now):

- Email/password authentication with JWT (access + refresh tokens)
- Email verification (soft-gate, 7-day grace period)
- Forgot-password / reset-password flow (opaque token, SHA-256 hashed)
- Short URL creation with plan-limit enforcement
- Public redirect endpoint (Redis → Postgres fallback)
- Subscription plans: Free / Pro / Business — feature-flag entitlements stored in `plans` table
- Plan feature-flag editing via admin role (prices/limits via migration only)
- Admin accounts via CLI-only creation (`cmd/admin`)
- Cache facade/fallback (`internal/cache` — Redis primary, in-memory fallback)

Planned / in progress:

- Stripe billing integration
- Background cleanup jobs (inactive URLs)
- Admin HTTP API (Batch 2)
- Rate-limit middleware
- Service + handler unit tests

**Tech stack:**

| Layer       | Technology                                             |
| ----------- | ------------------------------------------------------ |
| Backend     | Go 1.26, Gin v1.12                                     |
| Database    | PostgreSQL via `pgx/v5` + `pgxpool` + sqlc v1.31       |
| Cache       | Redis/Valkey via `go-redis/v9` + in-memory fallback    |
| Auth        | `golang-jwt/jwt/v5`, bcrypt (`golang.org/x/crypto`)    |
| Logging     | `go.uber.org/zap`                                      |
| Config      | `godotenv` + env vars                                  |
| Mailer      | `gopkg.in/mail.v2` (gomail) — SMTP/sendmail/console    |
| Frontend    | Vue 3.5 + Vite 8 + Pinia 3 + Vue Router 4 + TypeScript |
| HTTP client | Axios                                                  |
| Payment     | Stripe (planned)                                       |
| Infra       | Docker, Nginx (planned)                                |

---

# 2. Architecture Rules

Mandatory flow — never skip layers:

```text
Handler → Service → Repository
```

**Handler** (`internal/handler/`): parse request, validate payload, call service, format response. No business logic.

**Service** (`internal/service/`): all business rules, plan limits, short-code generation, token issuance, cache coordination. No HTTP or SQL.

**Repository** (`internal/repository/`): SQL via sqlc-generated queries + thin domain adapters. No HTTP, no business logic. All `Create*` methods return the persisted entity via `RETURNING *`.

**Cache** (`internal/cache/`): driver-agnostic `Cache` interface. Services depend on this, never on `*redis.Client`. Redis primary + in-memory `Chain` fallback. Do not use the fallback for rate limiting or distributed locks.

**Mailer** (`internal/mailer/`): `Mailer` interface with three transports: `smtp` (gomail + STARTTLS), `sendmail` (pipes to local MTA), `console` (stdout, dev default). Selected by `MAIL_TRANSPORT` env var. Each real transport implements `Probe()` — if the probe fails at startup, the factory falls back to console automatically. Services and event handlers call the interface, never a concrete transport. See `docs/29-mailer-transports.md`.

---

# 3. Backend Coding Rules

## Interfaces everywhere

Every service and repository must have an interface. Required for layer isolation and testing.

```go
type URLService interface {
    Create(ctx context.Context, userID uuid.UUID, planType string,
           req dto.CreateURLRequest, baseURL string) (*dto.URLResponse, error)
}
```

## Constructor injection, no globals

```go
func NewURLHandler(svc service.URLService, userRepo repository.UserRepository, baseURL string) *URLHandler
```

## Context propagation

```text
gin.Context → context.Context → service → repository
```

Never store context in a struct.

## DTO separation

Three separate types always:

- request DTO (binding tags, validation)
- response DTO (JSON tags)
- domain model (`internal/model/`)

Never return a DB model as an API response.

## Error handling

Repositories expose typed sentinels (`ErrURLNotFound`, `ErrShortCodeConflict`).
Services translate to service-level errors (`ErrPlanLimitReached`, `ErrTokenInvalid`).
Handlers map to HTTP status + error code per the contract below.

```json
{
  "success": false,
  "error": { "code": "PLAN_LIMIT_REACHED", "message": "..." }
}
```

See `docs/18-error-handling-contract.md` for the full code list.

## sqlc workflow

SQL lives in `internal/repository/queries/*.sql`. After editing:

```bash
make sqlc-generate   # regenerates internal/repository/sqlc/ — do not edit generated files
```

Repository adapters (`*_repository.go`) bridge between sqlc-generated types (`pgtype.Timestamp`, etc.) and domain models (`time.Time`, etc.). The adapter is the only place that knows about `pgtype`.

---

# 4. Database Rules

## Naming

snake_case everywhere — tables, columns, indexes.

## Migrations

Files in `backend/migrations/`, numbered `0001_...`, `0002_...`.
**Never modify an applied migration.** Always add a new file.

Apply: `make migrate` — runs `psql` via `docker exec` into `urlshortener-postgres`.
Status: `make migrate-status`.
Full guide: [20-local-development.md §3](docs/20-local-development.md).

Current migrations:

| File                          | Adds                                                          |
| ----------------------------- | ------------------------------------------------------------- |
| `0001_initial_schema.sql`     | users (no plan cols), subscriptions, short_urls               |
| `0002_add_plans_table.sql`    | plans + seed (Free/Pro/Business), users.plan_code (removed later) |
| `0003_add_user_lifecycle.sql` | users.email_verified_at, tokens table                         |
| `0004_add_admin_accounts.sql` | users.role + CHECK, users.disabled_at, admin_audit            |
| `0005_extract_user_plans.sql` | user_plans table; drops users.plan_code + plan_type           |

## Transactions

Use for: billing updates, subscription changes, any multi-table write.
sqlc provides `WithTx(tx pgx.Tx)` on `*Queries`. Service layer owns the transaction boundary.

## user_plans table

One row per user. Created on register (free tier). Updated on plan upgrade/downgrade.
`URLService` reads `user_plans` → `plans` to enforce the URL limit.
`AuthService` reads `user_plans` to populate `plan_code` in the JWT response.

## plans table

Plan rows are **seeded** in migration 0002 — never inserted by application code.
`features` JSONB flags may be toggled by admin UI (audited).
`price_cents`, `max_*`, `analytics_*`, `api_rate_limit_*` are **migration-only** — no runtime UPDATE.

---

# 5. Business Rules

## Authentication

- Register → user created, `email_verified_at = NULL`, verification token issued, email sent
- Login → `ErrUserDisabled` check before password compare
- Unverified users have a **7-day grace period** on mutation endpoints
- After grace period, `POST /urls` returns `EMAIL_VERIFICATION_REQUIRED` (403) until verified
- Subscription checkout always requires verified email (no grace)

## Admin accounts

- Created only via `make create-admin EMAIL=x@y` (reads password from stdin)
- `role = 'admin'` in JWT claim — `RequireAdmin` middleware enforces this
- No HTTP endpoint for admin creation — ever

## URL creation

- Authenticated users only
- Plan limit enforced: `count(user_urls) < plan.max_urls`
- Short code: 6 chars `[a-zA-Z0-9]`, retry up to 5 times on collision

## Redirect

Public. `GET /r/:short_code` → Redis → Postgres → cache-fill → 302.

## Inactive URL cleanup (planned, not yet built)

- 180 days unused → soft delete (`deleted_at`)
- 365 days unused → hard delete

---

# 6. Cache Rules

Redis key patterns:

```text
url:{short_code}         TTL 24h   ← redirect cache
plan:{user_id}           TTL 1h    ← entitlements cache
rate_limit:{ip}          no TTL    ← rate limiting (Redis only, no fallback)
```

Cache facade: `internal/cache/Chain{redis, in-memory}`. On Redis error the in-memory layer serves reads. **Never use in-memory fallback for rate limits or distributed locks** — per-replica counters break the global limit.

Cache invalidated on: Stripe webhook (upgrade/downgrade), admin plan-feature toggle, password reset.

---

# 7. Token Rules (one-time tokens)

Table: `tokens` with `purpose` enum (`verify_email`, `password_reset`).

- Raw token: 32 random bytes, base64url-encoded
- Stored as: SHA-256 hex hash only — raw token never hits the DB
- Single-use: prior tokens for the same purpose are invalidated before issuing a new one
- TTLs: verify_email = 24h, password_reset = 30min

---

# 8. Frontend Rules (Vue 3 + TS)

- Composition API always (`<script setup>`)
- Pinia stores for cross-component state
- API calls go through `services/` only — never `axios` directly in components
- Types in `src/types/index.ts` — keep in sync with backend DTOs

Structure:

```text
src/
  services/      API client functions (authService, urlService)
  stores/        Pinia stores (authStore, urlStore)
  views/         One file per route (LoginView, DashboardView, ...)
  components/    Reusable UI pieces (Navbar, UrlForm, UrlList)
  types/         TypeScript interfaces matching backend DTOs
  router/        Vue Router (route guards for auth + guest)
```

Existing views: Landing, Login, Register, Dashboard, ForgotPassword, ResetPassword, VerifyEmail.

---

# 9. Security Rules

- URL validation: reject `javascript:`, `data:`, `file:`, non-http(s) schemes
- JWT: access 15min, refresh 30d (admin: 5min / 8h)
- Tokens: SHA-256 hash in DB, raw token only in the email link
- `POST /auth/forgot-password` always returns 200 — no email enumeration
- `POST /auth/login` returns same error for unknown email and wrong password
- SQL: always parameterized (`$1`, `$2` via pgx/sqlc) — never string interpolation
- Secrets: env vars only — no hardcoded values, no logging
- Rate limits: use Redis only (not in-memory fallback)

---

# 10. Testing Rules

Layers (see `docs/19-testing-strategy.md`):

```text
pkg/         → stdlib only, no deps           ← done (100%/91%)
cache/       → InMemoryCache + Chain fakes    ← next
service/     → hand-written fakes, no DB      ← next
handler/     → httptest + fake services       ← next
repository/  → real Postgres (integration)   ← later
```

**No testify. No mockery. No mocking pgxpool or redis.Client.**
Write hand-written fakes that implement repository/cache interfaces.
Coverage floor: `pkg/` 95%, `service/` 80%, `handler/` 70%.

Run:

```bash
make migrate                  # apply all pending migrations (idempotent)
make migrate FILE=000N_x.sql  # apply a single migration file
make migrate-status           # show applied vs pending
make test-pkg     # fastest — pkg/ only
make test         # full suite
make test-cover   # with per-package coverage
```

---

# 11. AI Agent Instructions

## Before writing code

Read these docs in order:

1. `docs/00-overview.md`
2. `docs/03-database-design.md` (current schema)
3. `docs/04-api-specification.md`
4. `docs/23-backend-architecture.md` (sqlc, cache, DI)
5. `docs/18-error-handling-contract.md`

For backend also read:

- `docs/15-backend-folder-structure.md`
- `docs/24-user-account-lifecycle.md` (if touching auth/tokens)
- `docs/25-admin-accounts.md` (if touching admin)
- `docs/22-subscription-plans.md` (if touching plans/entitlements)

For frontend also read:

- `docs/16-frontend-folder-structure.md`

## When making changes

1. Identify the layer to modify
2. Check affected business rules (§5 above) and error codes (doc 18)
3. If schema changes: create new migration → update `queries/*.sql` → `make sqlc-generate` → fix adapter
4. If new service: write hand-written fake first, service test second, implementation third
5. Prefer minimal safe changes — no speculative abstractions

## Never do

- SQL inside handlers
- HTTP logic in repositories
- Skip layers (handler → repository directly)
- Bypass auth or verified-email middleware
- Expose raw Go errors to the client
- Break existing DTO contracts or change API response format
- Modify applied migration files
- Create admin accounts via HTTP
- Use in-memory cache fallback for rate limiting or locks
- Store raw one-time tokens in DB (store SHA-256 hash only)

---

# 12. Libraries in use

| Package                          | Purpose                                    |
| -------------------------------- | ------------------------------------------ |
| `github.com/gin-gonic/gin v1.12` | HTTP router + middleware                   |
| `github.com/jackc/pgx/v5`        | PostgreSQL driver + pgxpool                |
| `github.com/sqlc-dev/sqlc v1.31` | SQL codegen (run via `make sqlc-generate`) |
| `github.com/redis/go-redis/v9`   | Redis/Valkey client                        |
| `github.com/golang-jwt/jwt/v5`   | JWT issue + validate                       |
| `golang.org/x/crypto`            | bcrypt                                     |
| `golang.org/x/term`              | stdin password prompt in CLI               |
| `github.com/google/uuid`         | UUID generation                            |
| `go.uber.org/zap`                | structured logging                         |
| `github.com/joho/godotenv`       | `.env` loading                             |
| `gopkg.in/mail.v2`               | SMTP mailer (gomail) — STARTTLS, multipart |
| stripe-go                        | Stripe billing (planned)                   |

**Not in use and not wanted:** GORM, ent, bun, testify, gomock, mockery.

---

# 13. Priority Order

1. correctness
2. security
3. maintainability
4. performance
5. developer ergonomics
