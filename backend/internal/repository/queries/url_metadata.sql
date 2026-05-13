-- name: CreateURLMetadata :one
INSERT INTO url_metadata (url_id)
VALUES ($1)
RETURNING id, url_id, title, description, og_image, favicon_url, http_status, fetch_status, fetched_at, created_at;

-- name: GetURLMetadataByURLID :one
SELECT id, url_id, title, description, og_image, favicon_url, http_status, fetch_status, fetched_at, created_at
FROM url_metadata
WHERE url_id = $1;

-- name: UpdateURLMetadataFetched :exec
UPDATE url_metadata
SET title        = $2,
    description  = $3,
    og_image     = $4,
    favicon_url  = $5,
    http_status  = $6,
    fetch_status = $7,
    fetched_at   = NOW()
WHERE id = $1;

-- name: ListPendingURLMetadata :many
SELECT id, url_id, title, description, og_image, favicon_url, http_status, fetch_status, fetched_at, created_at
FROM url_metadata
WHERE fetch_status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: ListShortURLsWithMetadataByUser :many
SELECT
    s.id, s.user_id, s.short_code, s.original_url, s.click_count,
    s.last_accessed_at, s.created_at, s.deleted_at,
    m.title, m.description, m.og_image, m.favicon_url, m.http_status, m.fetch_status
FROM short_urls s
LEFT JOIN url_metadata m ON m.url_id = s.id
WHERE s.user_id = $1 AND s.deleted_at IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;
