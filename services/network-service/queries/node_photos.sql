-- Query SQL untuk operasi CRUD tabel node_photos.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel node_photos dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateNodePhoto :one
INSERT INTO node_photos (
    tenant_id, map_node_id, file_path,
    file_size_bytes, caption, uploaded_by
) VALUES (
    $1, $2, $3,
    $4, $5, $6
)
RETURNING id, tenant_id, map_node_id, file_path,
    file_size_bytes, caption, uploaded_by,
    deleted_at, created_at;

-- name: ListNodePhotosByNode :many
SELECT id, tenant_id, map_node_id, file_path,
    file_size_bytes, caption, uploaded_by,
    deleted_at, created_at
FROM node_photos
WHERE map_node_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: SoftDeleteNodePhoto :exec
UPDATE node_photos SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountNodePhotosByNode :one
SELECT COUNT(*) FROM node_photos
WHERE map_node_id = $1 AND deleted_at IS NULL;
