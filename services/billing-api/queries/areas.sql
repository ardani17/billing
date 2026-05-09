-- Kueri SQL untuk operasi CRUD tabel areas.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel areas dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateArea :one
INSERT INTO areas (tenant_id, name, description, odp_id, center_lat, center_lng)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, tenant_id, name, description, odp_id, center_lat, center_lng, created_at, updated_at;

-- name: GetAreaByID :one
SELECT id, tenant_id, name, description, odp_id, center_lat, center_lng, created_at, updated_at
FROM areas
WHERE id = $1;

-- name: UpdateArea :one
UPDATE areas SET
    name = $2,
    description = $3,
    odp_id = $4,
    center_lat = $5,
    center_lng = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, tenant_id, name, description, odp_id, center_lat, center_lng, created_at, updated_at;

-- name: DeleteArea :exec
DELETE FROM areas WHERE id = $1;

-- name: ListAreas :many
SELECT a.id, a.tenant_id, a.name, a.description, a.odp_id,
    a.center_lat, a.center_lng, a.created_at, a.updated_at,
    COUNT(c.id)::integer AS customer_count
FROM areas a
LEFT JOIN customers c ON c.area_id = a.id AND c.deleted_at IS NULL
WHERE a.tenant_id = $1
GROUP BY a.id, a.tenant_id, a.name, a.description, a.odp_id,
    a.center_lat, a.center_lng, a.created_at, a.updated_at
ORDER BY a.name ASC;

-- name: AreaNameExists :one
SELECT EXISTS(
    SELECT 1 FROM areas
    WHERE tenant_id = $1 AND name = $2 AND id != $3
) AS exists;

-- name: AreaCustomerCount :one
SELECT COUNT(*)::integer AS count
FROM customers
WHERE area_id = $1 AND deleted_at IS NULL;
