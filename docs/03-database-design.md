# Database Design

## users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    plan_type VARCHAR(50) NOT NULL DEFAULT 'free',                -- legacy, kept during transition
    plan_code VARCHAR(50) NOT NULL DEFAULT 'free' REFERENCES plans(code),
    email_verified_at TIMESTAMP,                                   -- NULL until verified (doc 24)
    role VARCHAR(20) NOT NULL DEFAULT 'user'
        CHECK (role IN ('user', 'admin')),                         -- admins created via CLI only (doc 25)
    disabled_at TIMESTAMP,                                         -- soft-disable for admin accounts
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Plan tiers and feature entitlements. Seeded in migration 0002.
-- See docs/22-subscription-plans.md for the product-level spec.
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

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    status VARCHAR(50),
    expires_at TIMESTAMP,
    created_at TIMESTAMP
);

CREATE TABLE short_urls (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    short_code VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    click_count BIGINT DEFAULT 0,
    last_accessed_at TIMESTAMP,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_short_code ON short_urls(short_code);
CREATE INDEX idx_last_accessed ON short_urls(last_accessed_at);
CREATE INDEX idx_user_id ON short_urls(user_id);

-- One-time tokens for email verification and password reset.
-- Single table discriminated by `purpose`. See docs/24-user-account-lifecycle.md §4.
CREATE TYPE token_purpose AS ENUM ('verify_email', 'password_reset');

CREATE TABLE tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     token_purpose NOT NULL,
    token_hash  TEXT NOT NULL,                  -- SHA-256 of the opaque token; raw token never stored
    expires_at  TIMESTAMP NOT NULL,
    used_at     TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tokens_hash    ON tokens(token_hash);
CREATE INDEX idx_tokens_user    ON tokens(user_id, purpose);
CREATE INDEX idx_tokens_expires ON tokens(expires_at);

-- Audit log for admin actions. Every admin write is paired with a row here
-- in the same transaction. See docs/25-admin-accounts.md §4.1.
CREATE TABLE admin_audit (
    id           UUID PRIMARY KEY,
    actor_id     UUID REFERENCES users(id),   -- NULL for CLI-originated actions
    action       VARCHAR(100) NOT NULL,        -- 'plan_feature_toggled', 'create_admin', ...
    target_type  VARCHAR(50),                  -- 'plan', 'user', ...
    target_id    VARCHAR(100),                 -- 'pro', UUID, ...
    before       JSONB,
    after        JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_actor  ON admin_audit(actor_id, created_at DESC);
CREATE INDEX idx_admin_audit_target ON admin_audit(target_type, target_id);

CREATE INDEX idx_users_role ON users(role) WHERE role = 'admin';
```

---

## Migration plan

| Migration                     | Adds                                       |
| ----------------------------- | ------------------------------------------ |
| `0001_initial_schema.sql`     | users, subscriptions, short_urls           |
| `0002_add_plans_table.sql`    | plans + seed, users.plan_code              |
| `0003_add_user_lifecycle.sql` | users.email_verified_at, tokens            |
| `0004_add_admin_accounts.sql` | users.role, users.disabled_at, admin_audit |

0003 and 0004 are planned (design in docs 24 and 25). 0001 and 0002 are
applied.
