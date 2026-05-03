-- Query SQL untuk operasi CRUD tabel map_share_links.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel map_share_links dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- GetMapShareLinkByToken tidak menggunakan RLS karena diakses secara publik via token.

-- name: CreateMapShareLink :one
INSERT INTO map_share_links (
    tenant_id, token, visible_layers,
    expires_at, password_hash, created_by
) VALUES (
    $1, $2, $3,
    $4, $5, $6
)
RETURNING id, tenant_id, token, visible_layers,
    expires_at, password_hash, access_count,
    created_by, created_at;

-- name: GetMapShareLinkByToken :one
SELECT id, tenant_id, token, visible_layers,
    expires_at, password_hash, access_count,
    created_by, created_at
FROM map_share_links
WHERE token = $1;

-- name: DeleteMapShareLink :exec
DELETE FROM map_share_links
WHERE token = $1;

-- name: ListMapShareLinksByTenant :many
SELECT id, tenant_id, token, visible_layers,
    expires_at, password_hash, access_count,
    created_by, created_at
FROM map_share_links
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: IncrementShareLinkAccessCount :exec
UPDATE map_share_links SET access_count = access_count + 1
WHERE token = $1;
