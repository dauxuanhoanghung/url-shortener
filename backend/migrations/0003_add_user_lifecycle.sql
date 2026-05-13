-- Email verification + one-time tokens (verify_email, password_reset).
-- See docs/24-user-account-lifecycle.md.

ALTER TABLE users
    ADD COLUMN email_verified_at TIMESTAMP;

-- Grandfather existing accounts: they were using the product before this
-- feature existed, so treat them as verified.
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

CREATE TYPE token_purpose AS ENUM ('verify_email', 'password_reset');

CREATE TABLE tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     token_purpose NOT NULL,
    token_hash  TEXT NOT NULL,
    expires_at  TIMESTAMP NOT NULL,
    used_at     TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tokens_hash    ON tokens(token_hash);
CREATE INDEX idx_tokens_user    ON tokens(user_id, purpose);
CREATE INDEX idx_tokens_expires ON tokens(expires_at);
