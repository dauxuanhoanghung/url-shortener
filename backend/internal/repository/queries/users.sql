-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, plan_type, plan_code, role, created_at, updated_at, email_verified_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, email, password_hash, plan_type, plan_code, role, email_verified_at, disabled_at, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, plan_type, plan_code, role, email_verified_at, disabled_at, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, plan_type, plan_code, role, email_verified_at, disabled_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: MarkUserEmailVerified :exec
UPDATE users SET email_verified_at = NOW(), updated_at = NOW()
WHERE id = $1 AND email_verified_at IS NULL;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW()
WHERE id = $1;
