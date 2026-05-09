-- Kueri SQL untuk operasi CRUD tabel map_nodes.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel map_nodes dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- kecuali kueri yang secara eksplisit menangani trashed nodes.

-- name: CreateMapNode :one
INSERT INTO map_nodes (
    tenant_id, node_type, reference_id,
    latitude, longitude, custom_fields
) VALUES (
    $1, $2, $3,
    $4, $5, $6
)
RETURNING id, tenant_id, node_type, reference_id,
    latitude, longitude, custom_fields,
    deleted_at, created_at, updated_at;

-- name: GetMapNodeByID :one
SELECT id, tenant_id, node_type, reference_id,
    latitude, longitude, custom_fields,
    deleted_at, created_at, updated_at
FROM map_nodes
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMapNode :one
UPDATE map_nodes SET
    latitude = $2,
    longitude = $3,
    custom_fields = $4,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, node_type, reference_id,
    latitude, longitude, custom_fields,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteMapNode :exec
UPDATE map_nodes SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreMapNode :exec
UPDATE map_nodes SET deleted_at = NULL, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: ListMapNodesByBounds :many
SELECT mn.id, mn.tenant_id, mn.node_type, mn.reference_id,
    mn.latitude, mn.longitude, mn.custom_fields,
    mn.created_at, mn.updated_at,
    COALESCE(o.name, odp.name, '') AS name,
    COALESCE(o.status, ont.status, '') AS status,
    ont.serial_number,
    odp.splitter_type,
    odp.capacity,
    odp.used_ports,
    odp.address
FROM map_nodes mn
LEFT JOIN olts o ON mn.node_type = 'olt' AND mn.reference_id = o.id
LEFT JOIN odps odp ON mn.node_type = 'odp' AND mn.reference_id = odp.id
LEFT JOIN onts ont ON mn.node_type = 'ont' AND mn.reference_id = ont.id
WHERE mn.deleted_at IS NULL
  AND mn.latitude BETWEEN $1 AND $2
  AND mn.longitude BETWEEN $3 AND $4
  AND (sqlc.narg('node_type')::varchar IS NULL OR mn.node_type = sqlc.narg('node_type')::varchar)
ORDER BY mn.created_at DESC;

-- name: GetMapNodeByReference :one
SELECT id, tenant_id, node_type, reference_id,
    latitude, longitude, custom_fields,
    deleted_at, created_at, updated_at
FROM map_nodes
WHERE tenant_id = $1 AND node_type = $2 AND reference_id = $3 AND deleted_at IS NULL;

-- name: PencarianMapNodes :many
SELECT mn.id, mn.tenant_id, mn.node_type, mn.reference_id,
    mn.latitude, mn.longitude, mn.custom_fields,
    mn.created_at, mn.updated_at,
    COALESCE(o.name, odp.name, '') AS name,
    COALESCE(o.status, ont.status, '') AS status,
    ont.serial_number,
    odp.splitter_type,
    odp.capacity,
    odp.used_ports,
    odp.address
FROM map_nodes mn
LEFT JOIN olts o ON mn.node_type = 'olt' AND mn.reference_id = o.id
LEFT JOIN odps odp ON mn.node_type = 'odp' AND mn.reference_id = odp.id
LEFT JOIN onts ont ON mn.node_type = 'ont' AND mn.reference_id = ont.id
WHERE mn.deleted_at IS NULL
  AND mn.tenant_id = $1
  AND (
    o.name ILIKE '%' || $2 || '%'
    OR odp.name ILIKE '%' || $2 || '%'
    OR odp.address ILIKE '%' || $2 || '%'
    OR ont.serial_number ILIKE '%' || $2 || '%'
  )
ORDER BY mn.created_at DESC
LIMIT $3;

-- name: ListTrashedMapNodes :many
SELECT id, tenant_id, node_type, reference_id,
    latitude, longitude, custom_fields,
    deleted_at, created_at, updated_at
FROM map_nodes
WHERE deleted_at IS NOT NULL AND tenant_id = $1
ORDER BY deleted_at DESC;

-- name: PermanentDeleteExpiredMapNodes :execrows
DELETE FROM map_nodes
WHERE deleted_at IS NOT NULL AND deleted_at < $1;

-- name: CountPhotosByMapNode :one
SELECT COUNT(*) FROM node_photos
WHERE map_node_id = $1 AND deleted_at IS NULL;
