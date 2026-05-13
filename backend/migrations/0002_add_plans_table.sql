-- Plan tiers and per-tier feature entitlements.
-- See docs/22-subscription-plans.md for the product-level spec.

CREATE TABLE plans (
    code                     VARCHAR(50) PRIMARY KEY,
    name                     VARCHAR(100) NOT NULL,
    price_cents              INTEGER NOT NULL,
    max_urls                 INTEGER NOT NULL,
    max_domains              INTEGER NOT NULL,
    max_team_members         INTEGER NOT NULL,
    analytics_retention_days INTEGER NOT NULL,
    api_rate_limit_per_min   INTEGER,
    features                 JSONB NOT NULL,
    created_at               TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Seed the three tiers. Feature flags mirror docs/22-subscription-plans.md §2.
-- Updating an existing tier should go through a new migration, not an UPDATE here.
INSERT INTO plans (code, name, price_cents, max_urls, max_domains, max_team_members,
                   analytics_retention_days, api_rate_limit_per_min, features)
VALUES
    ('free', 'Free', 0, 100, 0, 1, 7, NULL, '{
        "custom_alias": false,
        "custom_domain": false,
        "link_expiration": false,
        "password_protected": false,
        "geo_analytics": false,
        "bulk_import": false,
        "api_access": false,
        "webhooks": false,
        "remove_branding": false
    }'::jsonb),
    ('pro', 'Pro', 900, 10000, 1, 1, 90, 60, '{
        "custom_alias": true,
        "custom_domain": true,
        "link_expiration": true,
        "password_protected": true,
        "geo_analytics": true,
        "bulk_import": true,
        "api_access": true,
        "webhooks": false,
        "remove_branding": true
    }'::jsonb),
    ('business', 'Business', 2900, 100000, 5, 10, 365, 600, '{
        "custom_alias": true,
        "custom_domain": true,
        "link_expiration": true,
        "password_protected": true,
        "geo_analytics": true,
        "bulk_import": true,
        "api_access": true,
        "webhooks": true,
        "remove_branding": true
    }'::jsonb);

-- Link users to the plans table. Keep the free-text `plan_type` column during
-- transition: callers still read it today. A later migration will drop it once
-- all code paths read `plan_code` via PlanRepository.
ALTER TABLE users
    ADD COLUMN plan_code VARCHAR(50) NOT NULL DEFAULT 'free'
        REFERENCES plans(code);

-- Backfill existing rows so plan_code matches the legacy plan_type.
UPDATE users SET plan_code = plan_type WHERE plan_type IN ('free', 'pro', 'business');

CREATE INDEX idx_users_plan_code ON users(plan_code);
