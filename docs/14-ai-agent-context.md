# AI Agent Context

Quick-reference for AI coding agents. Full details in the docs linked at each section.

**Read order before writing code:**

1. `docs/00-overview.md` — what's built, what's planned
2. `docs/03-database-design.md` — current schema + migration table
3. `docs/04-api-specification.md` — endpoints + error codes
4. `docs/23-backend-architecture.md` — sqlc workflow, cache, DI
5. `docs/18-error-handling-contract.md` — mandatory error shape

For auth/token work also read: `docs/24-user-account-lifecycle.md`
For admin work also read: `docs/25-admin-accounts.md`
For plan/entitlement work also read: `docs/22-subscription-plans.md`
For frontend work also read: `docs/16-frontend-folder-structure.md`

---

## Architecture

```text
Handler → Service → Repository   (never skip layers)
```

- Handler: HTTP only — parse, validate, call service, format response
- Service: all business rules, token issuance, cache coordination, mailer calls
- Repository: sqlc-backed adapters — SQL only, typed domain sentinels returned
- Cache: `internal/cache.Cache` interface — Redis+in-memory Chain; services never import `*redis.Client`
- Mailer: `internal/mailer.Mailer` interface — services never call providers directly

---

## Backend Conventions

**Go style:**
- Constructor injection everywhere, no package-level globals
- Interfaces for every service and repository
- `context.Context` passed through all layers
- Request DTO / Response DTO / Domain model are three separate types
- `Create*` repository methods return `(*model.X, error)` via `RETURNING *`

**sqlc:**
- SQL source: `internal/repository/queries/*.sql`
- Generated: `internal/repository/sqlc/` — **do not edit**
- After any `.sql` or migration change: `make sqlc-generate`
- Adapters (`*_repository.go`) bridge `pgtype.Timestamp` ↔ `time.Time`, etc.

**Error flow:**
- Repo → typed sentinel (`ErrURLNotFound`, `ErrShortCodeConflict`)
- Service → domain error (`ErrPlanLimitReached`, `ErrTokenInvalid`, `ErrUserDisabled`)
- Handler → HTTP status + `{"success":false,"error":{"code":"...","message":"..."}}` (doc 18)

**Migrations:**
- `backend/migrations/000N_description.sql` — additive only, never modify applied files
- Applied: 0001–0004. Next is 0005.

---

## Frontend Conventions

- `<script setup>` Composition API — no Options API
- Pinia for cross-component state
- `services/` for all API calls — never `axios` directly in components
- Types in `src/types/index.ts` — kept in sync with backend DTOs

---

## Naming Rules

| Context | Convention |
| ------- | ---------- |
| DB columns / tables | snake_case |
| JSON fields | snake_case (backend DTOs use `json:"field_name"`) |
| Go exported types | PascalCase |
| Go unexported | camelCase |
| SQL query names (sqlc) | `-- name: VerbNoun :one\|:many\|:exec\|:execrows` |

---

## Business Rules (quick-ref)

| Rule | Detail |
| ---- | ------ |
| URL creation requires auth | AuthRequired middleware |
| URL creation requires verified email | VerifiedEmailRequired middleware (7-day grace) |
| Plan limit before insert | `count(user_urls) < plan.max_urls` |
| Short code | 6 chars `[a-zA-Z0-9]`, retry 5× on conflict |
| Redirect is public | No auth, Redis → Postgres fallback |
| Admin via CLI only | `make create-admin EMAIL=x@y`, no HTTP route |
| Plan prices/limits via migration only | `features` JSONB toggles allowed via admin UI |
| One-time tokens | SHA-256 hash stored, raw token in email link only |
| Forgot-password uniform 200 | Anti-enumeration — same response whether email exists or not |
| Inactive URL cleanup | 180d soft-delete, 365d hard-delete (planned) |

---

## Key Invariants (never break these)

- No SQL in handlers
- No HTTP logic in repositories
- No raw Go errors to clients
- No admin creation via HTTP
- No raw one-time token stored in DB (SHA-256 hash only)
- No `redis.Client` or `pgxpool.Pool` imported in service layer
- No modification of applied migration files
- No in-memory cache fallback for rate limiting or locks
