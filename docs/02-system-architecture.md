# System Architecture

## High-Level Request Flow

```text
Browser / API client
        │
        ▼
  Vue 3 frontend  (Vite, Pinia, Vue Router)
        │  axios
        ▼
  Gin HTTP router  (:8080)
        │
        ├─ middleware: gin.Logger, gin.Recovery
        ├─ middleware: AuthRequired (JWT)           ← protected routes
        └─ middleware: VerifiedEmailRequired        ← mutation routes (post-grace)
        │
        ▼
  Handler layer  (internal/handler/)
        │  calls
        ▼
  Service layer  (internal/service/)
        │  calls
        ├──► Cache  (internal/cache/ — Chain: Redis → in-memory fallback)
        │
        └──► Repository layer  (internal/repository/)
                    │  sqlc-generated queries
                    ▼
              PostgreSQL  (pgxpool)
```

Full layer rationale: [23-backend-architecture.md](23-backend-architecture.md).

---

## Services (implemented)

### AuthService

- Register: hash password → insert user → issue verify-email token → send email → return JWT
- Login: password verify → disabled check → return JWT
- VerifyEmail / ResendVerification: one-time token lookup, SHA-256 hash, single-live-token invariant
- ForgotPassword / ResetPassword: uniform 200 response (anti-enumeration), 30-min token TTL

### URLService

- Create: URL validation → plan-limit check → short-code generation (6 chars, 5-retry) → insert → return
- List: paginated by user, `deleted_at IS NULL`
- Delete: ownership check → soft-delete

### RedirectService

- Resolve: Redis GET → on miss: Postgres → Redis SET (24h TTL) → async click increment

### PlanRepository (read-only, no service yet)

- GetByCode, List — used by future plan-enforcement and admin services

---

## Services (planned)

### BillingService

- Stripe checkout session creation
- Webhook handling (subscription created / updated / deleted / payment failed)
- Plan cache invalidation on status change

### WorkerService

- Inactive URL cleanup (soft → hard delete)
- Token table cleanup (expired rows)
- Webhook delivery retries (future)

### AdminService (Batch 2)

- Plan feature-flag toggle (audited)
- User listing / read-only

---

## Databases

### PostgreSQL — tables

| Table           | Purpose                                                                          |
| --------------- | -------------------------------------------------------------------------------- |
| `users`         | Accounts, plan_code FK, role (user/admin), email_verified_at                     |
| `plans`         | Tier definitions + entitlements (seeded, not runtime-writable for prices/limits) |
| `short_urls`    | URLs with soft-delete, click count                                               |
| `subscriptions` | Stripe subscription tracking                                                     |
| `tokens`        | One-time tokens (verify_email, password_reset) — stored as SHA-256 hash          |
| `admin_audit`   | Audit log for every admin write                                                  |

Full schema: [03-database-design.md](03-database-design.md).

### Redis / Valkey — key patterns

```text
url:{short_code}     TTL 24h    redirect cache
plan:{user_id}       TTL 1h     entitlements cache
rate_limit:{ip}      —          rate limiting (Redis only, no in-memory fallback)
```

---

## Cache Architecture

`internal/cache/` exposes a single `Cache` interface backed by:

```text
Chain{ RedisCache, InMemoryCache }
```

Read path: Redis first → in-memory on miss or error (fallback).
Write path: fan-out to both layers; primary error returned.

Swap to `Driver: "memory"` in tests — no Redis required.
**Do not use the fallback for rate limiting or distributed locks.**

Details: [09-caching-strategy.md](09-caching-strategy.md), [23-backend-architecture.md §4](23-backend-architecture.md).

---

## Authentication Flow

```text
Register
  → insert user (email_verified_at=NULL, role='user')
  → issue verify_email token (SHA-256 stored, raw in email link)
  → ConsoleMailer (dev) / real provider (prod)
  → return JWT immediately (soft gate)

Email verification link clicked
  → POST /auth/verify-email {token}
  → hash lookup in tokens table
  → mark used → mark user verified

Forgot password
  → POST /auth/forgot-password {email}  ← always 200
  → issue password_reset token (30min TTL)
  → email sent

Reset password
  → POST /auth/reset-password {token, new_password}
  → hash lookup → update password_hash → mark token used
```

JWT access token: 15min. Refresh: 30d (admin: 5min / 8h).
Details: [05-authentication.md](05-authentication.md), [24-user-account-lifecycle.md](24-user-account-lifecycle.md).

---

## Admin Architecture

Admins have `role='admin'` in `users` table. No separate table.

Creation: **CLI only** via `cmd/admin create-admin`. No HTTP endpoint.
Every admin write accompanied by an `admin_audit` row in the same transaction.

Plan feature-flag edits: admin UI → service → DB + audit log.
Price / capacity fields: migration only (no runtime UPDATE).

Details: [25-admin-accounts.md](25-admin-accounts.md).

---

## Create URL Request Flow

```text
POST /urls
  → AuthRequired middleware (JWT)
  → VerifiedEmailRequired middleware (7-day grace window check)
  → URLHandler.Create
      → URLService.Create
          → validateURL (reject javascript:/data:/file:)
          → count(user_urls) < plan.max_urls
          → GenerateShortCode (6 chars, retry up to 5 on conflict)
          → URLRepository.Create (RETURNING *)
      → 201 {id, short_code, short_url}
```

---

## Redirect Request Flow

```text
GET /r/:short_code
  → RedirectHandler.Redirect
      → RedirectService.Resolve
          → cache.Get("url:{short_code}")
            hit  → go async: IncrementClick → 302
            miss → URLRepository.GetByShortCode
                 → cache.Set("url:{short_code}", 24h)
                 → go async: IncrementClick
                 → 302
          → 404 if not found
```

Details: [07-redirect-flow.md](07-redirect-flow.md).
