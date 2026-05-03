-- Query SQL untuk operasi CRUD tabel olts.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel olts dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateOLT :one
INSERT INTO olts (
    tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, notes
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17, $18, $19,
    $20, $21, $22,
    $23, $24
)
RETURNING id, tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    failure_count, notes, deleted_at, created_at, updated_at;

-- name: GetOLTByID :one
SELECT id, tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    failure_count, notes, deleted_at, created_at, updated_at
FROM olts
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateOLT :one
UPDATE olts SET
    name = $2,
    host = $3,
    snmp_version = $4,
    snmp_port = $5,
    snmp_community_encrypted = $6,
    snmp_username = $7,
    snmp_auth_protocol = $8,
    snmp_auth_password_encrypted = $9,
    snmp_priv_protocol = $10,
    snmp_priv_password_encrypted = $11,
    cli_protocol = $12,
    cli_port = $13,
    cli_username = $14,
    cli_password_encrypted = $15,
    cli_enable_password_encrypted = $16,
    brand = $17,
    model = $18,
    firmware_version = $19,
    pon_port_count = $20,
    status = $21,
    health_check_interval_sec = $22,
    notes = $23,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    failure_count, notes, deleted_at, created_at, updated_at;

-- name: SoftDeleteOLT :exec
UPDATE olts SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListOLTs :many
SELECT id, tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    failure_count, notes, deleted_at, created_at, updated_at
FROM olts
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('brand')::varchar IS NULL OR brand = sqlc.narg('brand')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR name ILIKE '%' || sqlc.narg('search')::varchar || '%' OR host ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountOLTs :one
SELECT COUNT(*) FROM olts
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('brand')::varchar IS NULL OR brand = sqlc.narg('brand')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR name ILIKE '%' || sqlc.narg('search')::varchar || '%' OR host ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: CountOLTsByStatus :many
SELECT status, COUNT(*) AS count
FROM olts
WHERE deleted_at IS NULL
GROUP BY status;

-- name: GetActiveOLTs :many
SELECT id, tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    failure_count, notes, deleted_at, created_at, updated_at
FROM olts
WHERE deleted_at IS NULL AND status != 'maintenance';

-- name: GetOnlineOLTs :many
SELECT id, tenant_id, name, host, snmp_version, snmp_port,
    snmp_community_encrypted, snmp_username, snmp_auth_protocol,
    snmp_auth_password_encrypted, snmp_priv_protocol, snmp_priv_password_encrypted,
    cli_protocol, cli_port, cli_username, cli_password_encrypted,
    cli_enable_password_encrypted, brand, model, firmware_version,
    pon_port_count, total_ont_count, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    failure_count, notes, deleted_at, created_at, updated_at
FROM olts
WHERE deleted_at IS NULL AND status = 'online';

-- name: OLTNameExists :one
SELECT EXISTS(
    SELECT 1 FROM olts
    WHERE tenant_id = $1 AND name = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: UpdateOLTHealthCheck :exec
UPDATE olts SET
    last_checked_at = $2,
    last_online_at = $3,
    failure_count = $4,
    status = $5,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateOLTONTCounts :exec
UPDATE olts SET
    total_ont_count = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
