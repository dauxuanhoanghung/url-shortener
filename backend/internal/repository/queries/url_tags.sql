-- name: UpsertURLTag :exec
INSERT INTO url_tags (url_id, tag)
VALUES ($1, $2)
ON CONFLICT (url_id, tag) DO NOTHING;

-- name: DeleteURLTag :exec
DELETE FROM url_tags
WHERE url_id = $1 AND LOWER(tag) = $2;

-- name: DeleteAllTagsForURL :exec
DELETE FROM url_tags WHERE url_id = $1;

-- name: ListTagsForURL :many
SELECT tag FROM url_tags
WHERE url_id = $1
ORDER BY tag;

-- name: ListTagsForURLs :many
SELECT url_id, tag FROM url_tags
WHERE url_id = ANY($1::uuid[])
ORDER BY url_id, tag;

-- name: ListURLsByTag :many
SELECT su.id, su.user_id, su.short_code, su.original_url, su.click_count, su.last_accessed_at, su.created_at, su.deleted_at
FROM short_urls su
JOIN url_tags ut ON ut.url_id = su.id
WHERE su.user_id = $1
  AND LOWER(ut.tag) = $2
  AND su.deleted_at IS NULL
ORDER BY su.created_at DESC
LIMIT $3 OFFSET $4;
