-- name: CreateShortURL :one
INSERT INTO short_urls (id, user_id, short_code, original_url, click_count, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, short_code, original_url, click_count, last_accessed_at, created_at, deleted_at;

-- name: GetShortURLByID :one
SELECT id, user_id, short_code, original_url, click_count, last_accessed_at, created_at, deleted_at
FROM short_urls
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetShortURLByShortCode :one
SELECT id, user_id, short_code, original_url, click_count, last_accessed_at, created_at, deleted_at
FROM short_urls
WHERE short_code = $1 AND deleted_at IS NULL;

-- name: ListShortURLsByUser :many
SELECT id, user_id, short_code, original_url, click_count, last_accessed_at, created_at, deleted_at
FROM short_urls
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountShortURLsByUser :one
SELECT COUNT(*) FROM short_urls WHERE user_id = $1 AND deleted_at IS NULL;

-- name: IncrementShortURLClick :exec
UPDATE short_urls
SET click_count = click_count + 1, last_accessed_at = NOW()
WHERE short_code = $1 AND deleted_at IS NULL;

-- name: SoftDeleteShortURL :execrows
UPDATE short_urls
SET deleted_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;
