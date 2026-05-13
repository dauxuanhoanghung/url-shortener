# Database Design

## Tables

```sql
-- Core user identity. No billing columns — plan membership is in user_plans.
CREATE TABLE users (
    id               UUID PRIMARY KEY,
    email            VARCHAR(255) UNIQUE NOT NULL,
    password_hash    TEXT NOT NULL,
    email_verified_at TIMESTAMP,                         -- NULL until verified (doc 24)
    role             VARCHAR(20) NOT NULL DEFAULT 'user'
                         CHECK (role IN ('user', 'admin')), -- admins via CLI only (doc 25)
    disabled_at      TIMESTAMP,
    created_at       TIMESTAMP NOT NULL,
    updated_at       TIMESTAMP NOT NULL
);

-- Plan tiers and feature entitlements. Seeded in migration 0002. Never runtime-inserted.
-- Prices / limits change via migration only. features JSONB may be toggled via admin UI.
-- See docs/22-subscription-plans.md.
CREATE TABLE plans (
    code                     VARCHAR(50) PRIMARY KEY,  -- 'free' | 'pro' | 'business'
    name                     VARCHAR(100) NOT NULL,
    price_cents              INTEGER NOT NULL,
    max_urls                 INTEGER NOT NULL,
    max_domains              INTEGER NOT NULL,
    max_team_members         INTEGER NOT NULL,
    analytics_retention_days INTEGER NOT NULL,
    api_rate_limit_per_min   INTEGER,                  -- NULL = no API access
    features                 JSONB NOT NULL,           -- boolean flags per feature
    created_at               TIMESTAMP NOT NULL DEFAULT NOW()
);

-- One row per user, FK to plans. Single source of truth for a user's plan.
-- Replaces the old plan_type / plan_code columns that used to live on users.
CREATE TABLE user_plans (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_code  VARCHAR(50) NOT NULL REFERENCES plans(code),
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_plans_user UNIQUE (user_id)    -- one active plan per user
);

CREATE TABLE subscriptions (
    id                    UUID PRIMARY KEY,
    user_id               UUID REFERENCES users(id),
    stripe_customer_id    TEXT,
    stripe_subscription_id TEXT,
    status                VARCHAR(50),
    expires_at            TIMESTAMP,
    created_at            TIMESTAMP
);

CREATE TABLE short_urls (
    id               UUID PRIMARY KEY,
    user_id          UUID REFERENCES users(id),
    short_code       VARCHAR(20) UNIQUE NOT NULL,
    original_url     TEXT NOT NULL,
    click_count      BIGINT DEFAULT 0,
    last_accessed_at TIMESTAMP,
    created_at       TIMESTAMP,
    deleted_at       TIMESTAMP
);

-- One-time tokens for email verification and password reset.
-- Raw token never stored — only SHA-256 hash. See docs/24-user-account-lifecycle.md §4.
CREATE TYPE token_purpose AS ENUM ('verify_email', 'password_reset');

CREATE TABLE tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    token_purpose NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at    TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Audit log for every admin write. Every admin mutation writes here in the
-- same transaction. See docs/25-admin-accounts.md §4.1.
CREATE TABLE admin_audit (
    id          UUID PRIMARY KEY,
    actor_id    UUID REFERENCES users(id),   -- NULL for CLI-originated actions
    action      VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id   VARCHAR(100),
    before      JSONB,
    after       JSONB,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Indexes

```sql
CREATE INDEX idx_short_code          ON short_urls(short_code);
CREATE INDEX idx_last_accessed       ON short_urls(last_accessed_at);
CREATE INDEX idx_short_urls_user_id  ON short_urls(user_id);
CREATE INDEX idx_user_plans_user_id  ON user_plans(user_id);
CREATE INDEX idx_user_plans_plan_code ON user_plans(plan_code);
CREATE INDEX idx_tokens_hash         ON tokens(token_hash);
CREATE INDEX idx_tokens_user         ON tokens(user_id, purpose);
CREATE INDEX idx_tokens_expires      ON tokens(expires_at);
CREATE INDEX idx_users_role          ON users(role) WHERE role = 'admin';
CREATE INDEX idx_admin_audit_actor   ON admin_audit(actor_id, created_at DESC);
CREATE INDEX idx_admin_audit_target  ON admin_audit(target_type, target_id);
```

---

## Migration history

| Migration                     | Adds                                                                       |
| ----------------------------- | -------------------------------------------------------------------------- |
| `0001_initial_schema.sql`     | users (no plan cols), subscriptions, short_urls                            |
| `0002_add_plans_table.sql`    | plans (seeded Free/Pro/Business), users.plan_code FK (now removed by 0005) |
| `0003_add_user_lifecycle.sql` | users.email_verified_at, tokens                                            |
| `0004_add_admin_accounts.sql` | users.role + CHECK, users.disabled_at, admin_audit                         |
| `0005_extract_user_plans.sql` | user_plans table; drops users.plan_code + users.plan_type                  |

All migrations applied in order. **Never modify an applied migration.**

To apply: `make migrate` (see [20-local-development.md §3](20-local-development.md) for full reference).
To check status: `make migrate-status`.

---

## Design notes

- `users` holds only identity — email, password hash, role, verification timestamp.
- `user_plans` is the single source of truth for a user's active plan. One row per user, enforced by `UNIQUE (user_id)`.
- `plans` rows are seeded, not inserted by application code. Price / limit changes go through a new migration.
- `plan_code` in `user_plans` is a FK to `plans(code)`. Upgrading a user means `UPDATE user_plans SET plan_code = 'pro' WHERE user_id = $1`.
- `tokens` uses a SHA-256 hash — raw token exists only in the email link and never in the DB.
- `admin_audit` is append-only from the application. Deletes only via DBA tooling.
