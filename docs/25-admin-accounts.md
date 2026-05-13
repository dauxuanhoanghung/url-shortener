# Admin Accounts

Admins are **site operators** — the people who run the platform — not
paying users with an elevated plan. This doc defines:

- how an admin account is different from a regular user
- how admin accounts are created (CLI only)
- what admins can do (narrow scope)
- how plan-editing works without breaking the "seeded via migration" rule

Related: [22-subscription-plans.md](22-subscription-plans.md) for plan
tiers, [24-user-account-lifecycle.md](24-user-account-lifecycle.md) for
regular user flows.

---

## 1. Admin vs User

| Concern            | User                      | Admin                          |
| ------------------ | ------------------------- | ------------------------------ |
| Created via        | `POST /auth/register`     | **CLI command only**           |
| Has subscription   | yes (free/pro/business)   | no (subscription rules N/A)    |
| Can create URLs    | yes (own account)         | no by default                  |
| Can edit plans     | no                        | yes (feature flags, §3)        |
| Login route        | `/auth/login`             | `/admin/login` (separate)      |
| Session token      | JWT with `role: "user"`   | JWT with `role: "admin"`       |
| MFA required       | no (v1)                   | yes (v1.1, TOTP)               |
| Email verification | soft-gated (see 24)       | asserted at account creation   |

The `role` claim on the JWT is the authoritative check. Middleware
`RequireAdmin` returns 403 `ADMIN_REQUIRED` for any non-admin JWT hitting
an admin-only route.

---

## 2. Account Creation — CLI Only

Admin accounts **cannot** be created through any HTTP endpoint, ever.
The only way to create one is a CLI command run with access to the
database:

```bash
go run ./cmd/admin create-admin \
    --email admin@short.ly \
    --password-stdin
# reads password from stdin, never from argv (argv shows in `ps`)
```

### 2.1 Why CLI-only

- **Attack surface**: no "create admin" route means no chance of
  a privilege-escalation bug exposing it.
- **Trust boundary**: anyone who can run the command already has DB
  credentials — promoting them to admin adds no new power they
  didn't effectively have.
- **Audit**: admin creation happens on a deploy host, recorded in
  shell history / deploy logs, not in a web access log that rotates.

### 2.2 CLI binary

New entrypoint: [backend/cmd/admin/main.go](../backend/cmd/admin/main.go).

Subcommands:

| Command           | Purpose                                                |
| ----------------- | ------------------------------------------------------ |
| `create-admin`    | Create a new admin account                             |
| `list-admins`     | List all admin accounts (email, created_at, last_login)|
| `disable-admin`   | Mark an admin account disabled (no delete, for audit) |
| `reset-admin-pw`  | Reset an admin password (password from stdin)          |

Every command reads DB connection details from the same config loader
as the API server. Same `.env`, same `POSTGRES_*` vars.

### 2.3 What the command does

```text
create-admin --email X --password-stdin
  ├─ read password from stdin (never argv)
  ├─ validate email format + password strength (12+ chars, mixed)
  ├─ check: no existing admin with this email
  ├─ bcrypt hash password
  ├─ INSERT INTO users (..., role='admin', email_verified_at=NOW())
  ├─ INSERT INTO admin_audit (actor_user_id=NULL, action='create_admin', ...)
  └─ print the new admin's ID to stdout
```

Admins are stored in the **same `users` table** — one identity table, a
`role` column discriminates. Keeps the codebase small. Trade-off: admin
rows have no plan/subscription meaning — billing code must skip users
with `role = 'admin'`.

---

## 3. What Admins Can Do

Narrow scope on purpose. Expand when a specific operational need
appears — do not pre-build UIs "in case we need it".

### 3.1 Plan feature-flag editing (v1)

Admins can toggle **feature flags** on plans (booleans in the `features`
JSONB column, per [22-subscription-plans.md §2](22-subscription-plans.md)).

Example: "enable `webhooks` on the Pro plan for a trial period".

They **cannot** change:

- `price_cents`
- `max_urls`, `max_domains`, `max_team_members`
- `analytics_retention_days`, `api_rate_limit_per_min`
- plan `code` or `name`

Those are billing-relevant — price changes must go through a migration
so the history is in git, reviewed, and auditable against Stripe. See §4.

### 3.2 User listing (read-only)

Admins can:

```text
GET /admin/users          paginated list (email, plan_code, created_at, last_login)
GET /admin/users/:id      full user record except password_hash
```

Useful for responding to "my account isn't working" tickets. No write
access from the UI — edits happen via future support tooling or CLI.

### 3.3 Everything else (v1.1+, not now)

Out of v1 admin scope, not built yet:

- impersonating a user for support
- soft-deleting abusive URLs
- refunds or manual subscription adjustments
- viewing another user's private URLs

Each requires its own threat model and audit trail. Add one at a time
when a real support need appears.

---

## 4. Plan Editing — Reconciling Admin-Edit with "Seeded by Migration"

[22-subscription-plans.md §6.1](22-subscription-plans.md) says plan
changes go through migrations. Admin UI editing conflicts with that
*if* we allow it for everything — so we split:

| Field                                 | Editable how?    | Why                                    |
| ------------------------------------- | ---------------- | -------------------------------------- |
| `price_cents`                         | migration only   | Stripe reconciliation requires git history |
| `max_urls`, `max_domains`, `max_team_members` | migration only | billing-relevant capacity limits |
| `analytics_retention_days`, `api_rate_limit_per_min` | migration only | SLAs, must be reviewed |
| `features` (JSONB flags)              | **admin UI**     | toggling a flag should not need a deploy |
| `name`, `code`                        | migration only   | changing code breaks FK references     |

Admin UI writes to the `features` JSONB and records **who / when / what**
in an audit log. Git still has the history for billing-relevant fields;
ops can flip feature flags without shipping code.

### 4.1 Audit log

```sql
CREATE TABLE admin_audit (
    id           UUID PRIMARY KEY,
    actor_id     UUID REFERENCES users(id),   -- the admin who did it
    action       VARCHAR(100) NOT NULL,       -- 'plan_feature_toggled', etc.
    target_type  VARCHAR(50),                 -- 'plan', 'user', ...
    target_id    VARCHAR(100),                -- 'pro', UUID, ...
    before       JSONB,
    after        JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_actor  ON admin_audit(actor_id, created_at DESC);
CREATE INDEX idx_admin_audit_target ON admin_audit(target_type, target_id);
```

Every admin write goes through a service method that also writes an
`admin_audit` row in the same transaction. No write succeeds unless the
audit row is written.

---

## 5. Admin API

All routes under `/admin/*`, all require `role=admin` JWT.

```text
POST   /admin/login                    separate from /auth/login
GET    /admin/users                    paginated users
GET    /admin/users/:id                one user
GET    /admin/plans                    all plans
PATCH  /admin/plans/:code/features     toggle flags; body: { "webhooks": true }
GET    /admin/audit                    recent audit events
```

Response shapes follow [18-error-handling-contract.md](18-error-handling-contract.md).
Non-admin JWT on any of these → 403 `ADMIN_REQUIRED`.

Why a separate `/admin/login`: different rate-limit rules, different
cookie domain (can be locked to an admin subdomain), less surface for
credential-stuffing attacks against normal users. Same backend logic,
different route.

---

## 6. Security

- **No admin creation route**. Ever. If someone proposes `POST /auth/register-admin?secret=...`, the answer is no.
- **CLI requires DB access**. Someone with DB access could already promote themselves; the CLI is a convenience, not a privilege boundary.
- **Admin JWTs** have shorter access token TTL (5 min vs 15 min for users) and require re-login after 8 hours even with a valid refresh token.
- **Admin password policy**: min 12 chars, must include a digit and a symbol. Enforced in the CLI.
- **MFA**: deferred to v1.1, not v1. Out of scope here but tracked in
  [11-security.md](11-security.md) for when it lands.
- **`role` column** has a CHECK constraint so only `'user'` and `'admin'`
  are allowed. Adding new roles (e.g. `'support'`) requires a migration,
  not an INSERT from application code.

---

## 7. Database Additions (planned migration 0004)

Runs after migration 0003 (tokens, from [24-user-account-lifecycle.md](24-user-account-lifecycle.md)).

```sql
ALTER TABLE users
    ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user'
        CHECK (role IN ('user', 'admin')),
    ADD COLUMN disabled_at TIMESTAMP;

CREATE INDEX idx_users_role ON users(role) WHERE role = 'admin';

CREATE TABLE admin_audit (
    id           UUID PRIMARY KEY,
    actor_id     UUID REFERENCES users(id),
    action       VARCHAR(100) NOT NULL,
    target_type  VARCHAR(50),
    target_id    VARCHAR(100),
    before       JSONB,
    after        JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_actor  ON admin_audit(actor_id, created_at DESC);
CREATE INDEX idx_admin_audit_target ON admin_audit(target_type, target_id);
```

Note: no seed `INSERT`. The first admin is created via `go run ./cmd/admin create-admin`
on the deploy host after the migration runs.

---

## 8. Open Questions

- Where does admin UI live — same Vue app under a feature-gated route,
  or a separate admin SPA? My lean: **separate SPA** (different
  deployment, different auth flow, smaller bundle for regular users).
- IP allowlist for `/admin/*`? Useful if you have a known ops network;
  overkill if admins work from laptops. Punt.
- Audit log retention — how long? Suggest 1 year, then archive. Decide
  when it gets large.
