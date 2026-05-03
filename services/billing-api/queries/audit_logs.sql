-- Query SQL untuk operasi CRUD tabel audit_logs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel audit_logs dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (tenant_id, entity_type, entity_id, action, actor_id, actor_name, changes, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAuditLogsByEntity :many
SELECT id, tenant_id, entity_type, entity_id, action, actor_id, actor_name, changes, metadata, created_at
FROM audit_logs
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC;
