# URL Shortener — System Overview

## Project Summary

A **SaaS URL shortener platform** built as a Go learning project.
Authenticated users create short URLs; the public redirect endpoint resolves them.
Subscription plans gate feature entitlements and URL quotas.

---

## Implemented

| Domain         | What's built                                                                                   |
| -------------- | ---------------------------------------------------------------------------------------------- |
| Auth           | Register, login, JWT (access + refresh), email verification (soft-gate), forgot/reset password |
| URL management | Create (plan-limit enforced, 5-retry short-code collision), list, soft-delete                  |
| Redirect       | Public, Redis → Postgres fallback, click counter                                               |
| Plans          | Free / Pro / Business rows in `plans` table; `features` JSONB flags; plan_code FK on users     |
| Admin          | CLI-only account creation (`cmd/admin`), `admin_audit` log                                     |
| Cache          | `Cache` interface, Redis primary + in-memory `Chain` fallback                                  |
| Mailer         | `Mailer` interface, console (dev) implementation                                               |

---

## Planned / not yet built

- Stripe billing (checkout, webhook, subscription sync)
- Background cleanup jobs (soft-delete at 180d, hard-delete at 365d)
- Admin HTTP API (`/admin/*` — Batch 2)
- Rate-limit middleware
- Service + handler tests (layer 4 & 5 per `docs/19-testing-strategy.md`)
- URL metadata fetch + dead-link detection (background worker + SSE notify) — see [27-url-metadata-fetch.md](27-url-metadata-fetch.md)
- Custom domains, QR codes, team workspaces, API keys, webhooks (future tiers)

---

## Tech Stack

### Backend

| Concern          | Technology                            |
| ---------------- | ------------------------------------- |
| Language         | Go 1.26                               |
| HTTP             | Gin v1.12                             |
| Database driver  | pgx/v5 + pgxpool                      |
| Query generation | sqlc v1.31                            |
| Cache            | go-redis/v9 + in-memory fallback      |
| Auth tokens      | golang-jwt/jwt v5, bcrypt             |
| Logging          | zap                                   |
| Config           | godotenv + env vars                   |
| CLI              | `cmd/admin` (create-admin subcommand) |

### Frontend

| Concern   | Technology           |
| --------- | -------------------- |
| Framework | Vue 3.5 + TypeScript |
| Build     | Vite 8               |
| State     | Pinia 3              |
| Routing   | Vue Router 4         |
| HTTP      | Axios                |

### Infrastructure (planned / partial)

- PostgreSQL (running in Docker locally)
- Redis / Valkey (running in Docker locally)
- Docker + docker-compose
- Nginx (planned)
- Stripe (planned)

---

## Core Domains

| Domain                     | Doc                                                                                                            |
| -------------------------- | -------------------------------------------------------------------------------------------------------------- |
| Authentication + lifecycle | [05-authentication.md](05-authentication.md), [24-user-account-lifecycle.md](24-user-account-lifecycle.md)     |
| URL management             | [04-api-specification.md](04-api-specification.md)                                                             |
| Redirect flow              | [07-redirect-flow.md](07-redirect-flow.md)                                                                     |
| Subscription plans         | [22-subscription-plans.md](22-subscription-plans.md), [06-subscription-billing.md](06-subscription-billing.md) |
| Admin accounts             | [25-admin-accounts.md](25-admin-accounts.md)                                                                   |
| Backend architecture       | [23-backend-architecture.md](23-backend-architecture.md)                                                       |
| Database schema            | [03-database-design.md](03-database-design.md)                                                                 |
| Caching                    | [09-caching-strategy.md](09-caching-strategy.md)                                                               |
| Testing                    | [19-testing-strategy.md](19-testing-strategy.md)                                                               |
| Cleanup jobs               | [08-cleanup-retention.md](08-cleanup-retention.md), [10-background-jobs.md](10-background-jobs.md)             |
