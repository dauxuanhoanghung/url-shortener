-- name: CreateToken :exec
INSERT INTO tokens (id, user_id, purpose, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTokenByHash :one
-- Only returns rows that are usable: not used, not expired.
SELECT id, user_id, purpose, token_hash, expires_at, used_at, created_at
FROM tokens
WHERE token_hash = $1
  AND purpose = $2
  AND used_at IS NULL
  AND expires_at > NOW();

-- name: MarkTokenUsed :exec
UPDATE tokens SET used_at = NOW() WHERE id = $1;

-- name: InvalidateUserTokensByPurpose :exec
-- Used when resending verification or issuing a fresh reset: any prior
-- unused token for the same purpose is retired so only one link is live.
UPDATE tokens SET used_at = NOW()
WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL;
