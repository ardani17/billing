-- Kueri SQL untuk operasi CRUD tabel odps.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel odps dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateODP :one
INSERT INTO odps (
    tenant_id, olt_id, pon_port_index, name, splitter_type,
    capacity, used_ports, address, latitude, longitude, notes
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11
)
RETURNING id, tenant_id, olt_id, pon_port_index, name, splitter_type,
    capacity, used_ports, address, latitude, longitude, notes,
    deleted_at, created_at, updated_at;

-- name: GetODPByID :one
SELECT id, tenant_id, olt_id, pon_port_index, name, splitter_type,
    capacity, used_ports, address, latitude, longitude, notes,
    deleted_at, created_at, updated_at
FROM odps
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateODP :one
UPDATE odps SET
    name = $2,
    address = $3,
    latitude = $4,
    longitude = $5,
    notes = $6,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, olt_id, pon_port_index, name, splitter_type,
    capacity, used_ports, address, latitude, longitude, notes,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteODP :exec
UPDATE odps SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListODPs :many
SELECT id, tenant_id, olt_id, pon_port_index, name, splitter_type,
    capacity, used_ports, address, latitude, longitude, notes,
    deleted_at, created_at, updated_at
FROM odps
WHERE deleted_at IS NULL
  AND (sqlc.narg('olt_id')::uuid IS NULL OR olt_id = sqlc.narg('olt_id')::uuid)
  AND (sqlc.narg('pon_port_index')::integer IS NULL OR pon_port_index = sqlc.narg('pon_port_index')::integer)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountODPs :one
SELECT COUNT(*) FROM odps
WHERE deleted_at IS NULL
  AND (sqlc.narg('olt_id')::uuid IS NULL OR olt_id = sqlc.narg('olt_id')::uuid)
  AND (sqlc.narg('pon_port_index')::integer IS NULL OR pon_port_index = sqlc.narg('pon_port_index')::integer);

-- name: ODPNameExists :one
SELECT EXISTS(
    SELECT 1 FROM odps
    WHERE tenant_id = $1 AND name = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: GetODPsByOLTAndPort :many
SELECT id, tenant_id, olt_id, pon_port_index, name, splitter_type,
    capacity, used_ports, address, latitude, longitude, notes,
    deleted_at, created_at, updated_at
FROM odps
WHERE olt_id = $1 AND pon_port_index = $2 AND deleted_at IS NULL
ORDER BY name ASC;
