# Development Roadmap

## Phase 1 — Core MVP
_Status: complete_

### Backend
- [x] Gin project setup
- [x] PostgreSQL connection (pgxpool)
- [x] Migrations (0001–0005)
- [x] JWT auth (access + refresh tokens)
- [x] Email verification (7-day grace period)
- [x] Forgot password / reset password flow
- [x] Create URL API with plan-limit enforcement
- [x] List URLs API with metadata joins
- [x] Delete URL API (soft delete)
- [x] Redirect API (Redis → Postgres → cache-fill → 302)
- [x] Subscription plans (seeded, feature flags via JSONB)
- [x] Admin account creation (CLI only)
- [x] Event bus (in-memory, Sync/Async)
- [x] Mailer (SMTP / sendmail / console transports with probe + fallback)
- [x] URL metadata worker (og:title, description, favicon, dead-link detection)
- [x] SSE hub (dead-link notifications to user)
- [x] Rate limiting middleware (`pkg/ratelimit` + Redis INCR)

### Frontend
- [x] Vue 3 + Vite + Pinia + TypeScript setup
- [x] Landing page
- [x] Login / Register views
- [x] Email verification view
- [x] Forgot password / Reset password views
- [x] Dashboard (URL list + create form)
- [x] Pricing view (plan list, current plan highlight)
- [x] SSE integration (live dead-link removal)
- [x] Auth store, URL store, Plan store

---

## Phase 2 — Admin API + UI
_Status: complete (2026-05-19)_

### Backend
- [x] `Role` claim added to JWT (admin tokens: 5min access / 8h refresh)
- [x] `RequireAdmin` middleware → 403 `ADMIN_REQUIRED`
- [x] `GET /admin/users` — paginated user list joined with user_plans
- [x] `GET /admin/users/:id` — user detail
- [x] `POST /admin/users/:id/disable` / `enable` — blocks self-disable
- [x] `PATCH /admin/plans/:code/features` — feature-flag toggle, audited
- [x] `GET /admin/audit` — audit log viewer, paginated

### Frontend (Vue 3)
- [x] `AdminLayout` with tabs (Users / Plans / Audit log)
- [x] `AdminUsersView` — table with role/plan/status badges, enable/disable actions
- [x] `AdminPlansView` — per-plan feature-flag checkboxes, save per plan
- [x] `AdminAuditView` — log table with before/after JSONB
- [x] Route guard `requiresAdmin` redirects non-admins to dashboard
- [x] Conditional "Admin" link in Navbar

See [plans/260518-next-steps-roadmap.md](../plans/260518-next-steps-roadmap.md) §Phase 2.

---

## Phase 3 — Stripe Billing
_Status: pending_

- [ ] Migration: stripe_customer_id / stripe_subscription_id
- [ ] `POST /billing/checkout` — Stripe checkout session
- [ ] `POST /billing/portal` — Stripe billing portal
- [ ] `POST /webhook/stripe` — checkout.session.completed, subscription updated/deleted
- [ ] Cache invalidation on plan change (`plan:{user_id}`)
- [ ] Frontend: wire "Upgrade" CTA in PricingView

---

## Phase 4 — Test Coverage
_Status: partial_

- [x] `pkg/` — 100% / 91%
- [ ] `cache/` — InMemoryCache + Chain + ratelimit fakes
- [ ] `service/` — auth_service, redirect_service (url_service done)
- [ ] `handler/` — auth_handler, redirect_handler (url_handler, sse_handler done)
- [ ] `event/` — bus tests, handler tests
- [ ] `repository/` — integration tests against real Postgres

Rule: no testify, no mockery. Hand-written fakes only.

---

## Phase 5 — URL Cleanup Background Job
_Status: pending_

- [ ] Migration: add `last_accessed_at` to `short_urls`
- [ ] Update redirect_service to set `last_accessed_at` on each hit
- [ ] `CleanupWorker` — daily ticker
- [ ] Soft delete: unused 180+ days → `deleted_at`
- [ ] Hard delete: unused 365+ days → `DELETE`
- [ ] Cache invalidation + `URLDeleted` SSE event

---

## Phase 6 — Performance & Observability
_Status: planned_

- [ ] Structured request logging (request ID, latency, status)
- [ ] Prometheus metrics (request rate, cache hit/miss, redirect latency)
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Click analytics (geo, referrer, device)
