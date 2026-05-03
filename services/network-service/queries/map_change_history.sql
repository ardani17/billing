-- Query SQL untuk operasi pada tabel map_change_history.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel map_change_history dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Tabel ini bersifat append-only: hanya INSERT dan SELECT, tidak ada UPDATE atau DELETE.

-- name: CreateMapChangeHistory :one
INSERT INTO map_change_history (
    tenant_id, map_node_id, action,
    old_value, new_value, performed_by
) VALUES (
    $1, $2, $3,
    $4, $5, $6
)
RETURNING id, tenant_id, map_node_id, action,
    old_value, new_value, performed_by, created_at;

-- name: ListMapChangeHistoryByNode :many
SELECT id, tenant_id, map_node_id, action,
    old_value, new_value, performed_by, created_at
FROM map_change_history
WHERE map_node_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
