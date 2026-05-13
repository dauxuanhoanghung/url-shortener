-- Move plan membership out of the users table into a dedicated user_plans table.
-- Motivation: users should not carry billing-domain columns.
-- See docs/03-database-design.md.

CREATE TABLE user_plans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_code   VARCHAR(50) NOT NULL REFERENCES plans(code),
    started_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_plans_user UNIQUE (user_id)   -- one active plan per user
);

CREATE INDEX idx_user_plans_user_id ON user_plans(user_id);
CREATE INDEX idx_user_plans_plan_code ON user_plans(plan_code);

-- Backfill from the existing plan_code column on users.
INSERT INTO user_plans (user_id, plan_code)
SELECT id, plan_code FROM users;

-- Drop the plan columns that are now owned by user_plans.
ALTER TABLE users
    DROP COLUMN plan_code,
    DROP COLUMN plan_type;
