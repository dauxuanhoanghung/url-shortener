# Next Steps Roadmap

_Created: 2026-05-18_

## Current State Summary

Core MVP is complete (~70-90% per feature area):

- Auth lifecycle (register, verify, forgot/reset password) — done
- URL create/list/delete with plan limits — done
- Redirect with Redis cache fallback — done
- Event bus + async side-effects (email, metadata fetch) — done
- SSE dead-link notifications — done
- Plans seeded, limit enforcement working — done

---

## Phase 1 — Rate Limiting Middleware

**Status: [x] done — 2026-05-18**
**Priority: High — abuse protection before opening to real users**

### Backend tasks

1. Add `Increment(ctx, key, ttl) (int64, error)` to the `Cache` interface (Redis only)
2. Implement `Increment` in Redis driver via `INCR` + `EXPIRE`
3. Return `ErrNotSupported` from in-memory driver (rate limiting must use Redis only)
4. Delegate `Increment` to primary only in `Chain` fallback (not in-memory layer)
5. `RateLimitMiddleware(limit int, window time.Duration)` using Redis counter
6. Apply to: `POST /urls`, `POST /auth/login`, `POST /auth/register`, `POST /auth/forgot-password`
7. Return `429 Too Many Requests` with `RATE_LIMIT_EXCEEDED` error code

### Files to touch

- `backend/internal/cache/cache.go` — add `Increment` to interface
- `backend/internal/cache/redis.go` — implement `Increment`
- `backend/internal/cache/in_memory.go` — return `ErrNotSupported`
- `backend/internal/cache/fallback.go` — delegate to primary only
- `backend/internal/middleware/rate_limit_middleware.go` — new file
- `backend/internal/router/router.go` — wire middleware

### Key rule

Rate limiting uses Redis only — never the in-memory fallback (CLAUDE.md §9)

---

## Phase 2 — Admin API + UI

**Status: [x] done — 2026-05-19**
**Priority: High**

### Backend (done)

- [x] JWT now carries `role` claim; admin tokens use shorter TTL (5min/8h)
- [x] `RequireAdmin` middleware — returns 403 `ADMIN_REQUIRED` for non-admin JWT
- [x] `GET /admin/users?limit&offset` — paginated user list (joined with user_plans)
- [x] `GET /admin/users/:id` — single user detail
- [x] `POST /admin/users/:id/disable` / `enable` — soft disable, blocks self-disable
- [x] `PATCH /admin/plans/:code/features` — toggle JSONB feature flags (audited)
- [x] `GET /admin/audit?limit&offset` — audit log viewer
- [x] All admin write actions write to `admin_audit` with before/after JSONB

### Files added/modified (backend)

- `backend/internal/middleware/admin_middleware.go` (new)
- `backend/internal/handler/admin_handler.go` (new)
- `backend/internal/service/admin_service.go` (new)
- `backend/internal/dto/admin_dto.go` (new)
- `backend/internal/repository/queries/{users,plans,admin_audit}.sql` — added admin queries
- `backend/internal/repository/{user,plan,admin_audit}_repository.go` — added methods
- `backend/internal/model/user.go` — added `UserWithPlan`
- `backend/pkg/utils/jwt.go` — added `Role` claim + admin TTLs
- `backend/internal/middleware/auth_middleware.go` — sets `role` in gin context
- `backend/internal/router/router.go` — wired `/api/v1/admin/*` group
- `backend/cmd/api/main.go` — wired `auditRepo` + `adminService` + `adminHandler`

### Frontend (done)

- [x] `frontend/src/types/index.ts` — added `AdminUser`, `AdminAuditEntry` types
- [x] `frontend/src/services/adminService.ts` (new)
- [x] `frontend/src/stores/adminStore.ts` (new)
- [x] `frontend/src/stores/authStore.ts` — added `isAdmin` computed
- [x] `frontend/src/views/admin/AdminLayout.vue` — tabs + admin header (new)
- [x] `frontend/src/views/admin/AdminUsersView.vue` — user list + disable/enable (new)
- [x] `frontend/src/views/admin/AdminPlansView.vue` — feature-flag editor (new)
- [x] `frontend/src/views/admin/AdminAuditView.vue` — audit log viewer (new)
- [x] `frontend/src/router/index.ts` — nested admin routes + `requiresAdmin` guard
- [x] `frontend/src/components/Navbar.vue` — conditional "Admin" link (admin only)

---

## Phase 3 — Stripe Billing Integration

**Status: [ ] pending**
**Priority: Medium — unblocks plan upsells**

Pricing page exists with dead CTA buttons.

### Backend tasks

1. Migration: add `stripe_customer_id`, `stripe_subscription_id` to billing table
2. `POST /billing/checkout` — create Stripe checkout session, return URL
3. `POST /billing/portal` — create Stripe billing portal session
4. `POST /webhook/stripe` — handle `checkout.session.completed`, `customer.subscription.updated`, `customer.subscription.deleted`
5. On successful checkout: update `user_plans`, invalidate `plan:{user_id}` cache
6. Subscription enforcement on cancellation

### Files to create

- `backend/migrations/0006_add_stripe_billing.sql`
- `backend/internal/handler/billing_handler.go`
- `backend/internal/service/billing_service.go` + interface
- `backend/internal/repository/subscription_repository.go`

### Frontend tasks

- Wire "Upgrade" CTA in `frontend/src/views/PricingView.vue` to call `POST /billing/checkout` and redirect to Stripe-hosted page

---

## Phase 4 — Test Coverage Expansion

**Status: [ ] pending**
**Priority: Medium**

| Target        | Current Status          | Next                              |
| ------------- | ----------------------- | --------------------------------- |
| `pkg/`        | Done (100%/91%)         | —                                 |
| `cache/`      | Partial                 | InMemoryCache + Chain fakes       |
| `service/`    | Partial (url, metadata) | auth_service, redirect_service    |
| `handler/`    | Partial (url, sse)      | auth_handler, redirect_handler    |
| `repository/` | None                    | Integration tests (real Postgres) |
| `event/`      | None                    | Bus tests, handler tests          |

**Rule:** No testify, no mockery. Hand-written fakes only.

---

## Phase 5 — URL Cleanup Background Job

**Status: [ ] pending**
**Priority: Lower**

### Tasks

1. Migration: add `last_accessed_at` to `short_urls`
2. Update `redirect_service.go` to set `last_accessed_at` on each hit
3. `CleanupWorker` — runs on a daily ticker
4. Soft delete pass: mark unused 180+ days → `deleted_at`
5. Hard delete pass: remove records unused 365+ days
6. Invalidate Redis cache on soft delete
7. Emit `URLDeleted` event → SSE notification to user

---

## Implementation Order

| #   | Phase                                      | Est. Complexity       |
| --- | ------------------------------------------ | --------------------- |
| 1   | Rate limiting middleware                   | Small (1 session)     |
| 2   | Admin middleware + user list/detail        | Medium (1-2 sessions) |
| 3   | Admin plan feature-flag toggle + audit log | Small (1 session)     |
| 4   | Test: auth_service + redirect_service      | Medium (1 session)    |
| 5   | Stripe checkout + webhook                  | Large (2-3 sessions)  |
| 6   | URL cleanup worker                         | Medium (1 session)    |
