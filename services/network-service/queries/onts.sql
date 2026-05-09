-- Kueri SQL untuk operasi CRUD tabel onts.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel onts dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateONT :one
INSERT INTO onts (
    tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12
)
RETURNING id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at;

-- name: GetONTByID :one
SELECT id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at
FROM onts
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetONTBySerialNumber :one
SELECT id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at
FROM onts
WHERE tenant_id = $1 AND serial_number = $2 AND deleted_at IS NULL;

-- name: UpdateONT :one
UPDATE onts SET
    pon_port_index = $2,
    ont_index = $3,
    serial_number = $4,
    customer_id = $5,
    odp_id = $6,
    vlan_id = $7,
    service_profile_id = $8,
    status = $9,
    provisioning_state = $10,
    description = $11,
    last_provisioned_at = $12,
    last_decommissioned_at = $13,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteONT :exec
UPDATE onts SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListONTs :many
SELECT id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at
FROM onts
WHERE deleted_at IS NULL
  AND (sqlc.narg('olt_id')::uuid IS NULL OR olt_id = sqlc.narg('olt_id')::uuid)
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('provisioning_state')::varchar IS NULL OR provisioning_state = sqlc.narg('provisioning_state')::varchar)
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('search')::varchar IS NULL OR serial_number ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountONTs :one
SELECT COUNT(*) FROM onts
WHERE deleted_at IS NULL
  AND (sqlc.narg('olt_id')::uuid IS NULL OR olt_id = sqlc.narg('olt_id')::uuid)
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('provisioning_state')::varchar IS NULL OR provisioning_state = sqlc.narg('provisioning_state')::varchar)
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('search')::varchar IS NULL OR serial_number ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: ListONTsByOLTAndStatus :many
SELECT id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at
FROM onts
WHERE olt_id = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY pon_port_index ASC, ont_index ASC;

-- name: GetONTByCustomerID :one
SELECT id, tenant_id, olt_id, pon_port_index, ont_index, serial_number,
    customer_id, odp_id, vlan_id, service_profile_id,
    status, provisioning_state, description,
    last_provisioned_at, last_decommissioned_at,
    deleted_at, created_at, updated_at
FROM onts
WHERE customer_id = $1 AND deleted_at IS NULL AND status != 'decommissioned';

-- name: ONTSerialNumberExists :one
SELECT EXISTS(
    SELECT 1 FROM onts
    WHERE tenant_id = $1 AND serial_number = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: ONTPositionExists :one
SELECT EXISTS(
    SELECT 1 FROM onts
    WHERE olt_id = $1 AND pon_port_index = $2 AND ont_index = $3 AND id != $4 AND deleted_at IS NULL
) AS exists;

-- name: UpdateONTStatus :exec
UPDATE onts SET
    status = $2,
    provisioning_state = $3,
    last_provisioned_at = CASE WHEN $2 = 'provisioned' THEN NOW() ELSE last_provisioned_at END,
    last_decommissioned_at = CASE WHEN $2 = 'decommissioned' THEN NOW() ELSE last_decommissioned_at END,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateONTPortMigration :exec
UPDATE onts SET
    pon_port_index = $2,
    ont_index = $3,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteUnregisteredONTsByOLT :execrows
DELETE FROM onts
WHERE olt_id = $1
  AND status = 'unregistered'
  AND serial_number != ALL($2::varchar[]);
