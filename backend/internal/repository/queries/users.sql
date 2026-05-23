-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, role, created_at, updated_at, email_verified_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, password_hash, role, email_verified_at, disabled_at, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, email_verified_at, disabled_at, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, email_verified_at, disabled_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: MarkUserEmailVerified :exec
UPDATE users SET email_verified_at = NOW(), updated_at = NOW()
WHERE id = $1 AND email_verified_at IS NULL;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: ListUsersForAdmin :many
SELECT u.id, u.email, u.role, u.email_verified_at, u.disabled_at,
       u.created_at, u.updated_at, up.plan_code
FROM users u
LEFT JOIN user_plans up ON up.user_id = u.id
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsersForAdmin :one
SELECT COUNT(*) FROM users;

-- name: SetUserDisabled :exec
UPDATE users SET disabled_at = $2, updated_at = NOW()
WHERE id = $1;
