-- name: CreateAdminAuditEntry :exec
INSERT INTO admin_audit (id, actor_id, action, target_type, target_id, before, after, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAdminAuditEntries :many
SELECT id, actor_id, action, target_type, target_id, before, after, created_at
FROM admin_audit
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAdminAuditEntries :one
SELECT COUNT(*) FROM admin_audit;
