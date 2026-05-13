-- name: CreateAdminAuditEntry :exec
INSERT INTO admin_audit (id, actor_id, action, target_type, target_id, before, after, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
