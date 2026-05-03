-- Query SQL untuk operasi CRUD tabel vlans.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel vlans dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateVLAN :one
INSERT INTO vlans (
    tenant_id, olt_id, vlan_id, name, vlan_type, description
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, tenant_id, olt_id, vlan_id, name, vlan_type, description,
    deleted_at, created_at, updated_at;

-- name: GetVLANByID :one
SELECT id, tenant_id, olt_id, vlan_id, name, vlan_type, description,
    deleted_at, created_at, updated_at
FROM vlans
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateVLAN :one
UPDATE vlans SET
    name = $2,
    vlan_type = $3,
    description = $4,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, olt_id, vlan_id, name, vlan_type, description,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteVLAN :exec
UPDATE vlans SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListVLANs :many
SELECT id, tenant_id, olt_id, vlan_id, name, vlan_type, description,
    deleted_at, created_at, updated_at
FROM vlans
WHERE olt_id = $1 AND deleted_at IS NULL
ORDER BY vlan_id ASC
LIMIT $2 OFFSET $3;

-- name: CountVLANs :one
SELECT COUNT(*) FROM vlans
WHERE olt_id = $1 AND deleted_at IS NULL;

-- name: GetVLANByOLTAndVLANID :one
SELECT id, tenant_id, olt_id, vlan_id, name, vlan_type, description,
    deleted_at, created_at, updated_at
FROM vlans
WHERE olt_id = $1 AND vlan_id = $2 AND deleted_at IS NULL;

-- name: GetDefaultVLAN :one
SELECT id, tenant_id, olt_id, vlan_id, name, vlan_type, description,
    deleted_at, created_at, updated_at
FROM vlans
WHERE olt_id = $1 AND vlan_type = 'data' AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT 1;

-- name: VLANIDExists :one
SELECT EXISTS(
    SELECT 1 FROM vlans
    WHERE olt_id = $1 AND vlan_id = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: CountVLANActiveONTs :one
SELECT COUNT(*) FROM onts
WHERE vlan_id = $1 AND deleted_at IS NULL AND status != 'decommissioned';
