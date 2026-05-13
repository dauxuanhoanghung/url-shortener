-- Admin accounts (role column) + audit log. Admins created via CLI only.
-- See docs/25-admin-accounts.md.

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
