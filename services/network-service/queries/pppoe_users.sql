-- Query SQL untuk operasi CRUD tabel pppoe_users.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel pppoe_users dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreatePPPoEUser :one
INSERT INTO pppoe_users (
    tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, sync_status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13
)
RETURNING id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at;

-- name: GetPPPoEUserByID :one
SELECT id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM pppoe_users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPPPoEUserByUsername :one
SELECT id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM pppoe_users
WHERE router_id = $1 AND username = $2 AND deleted_at IS NULL;

-- name: GetPPPoEUserByCustomerID :one
SELECT id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM pppoe_users
WHERE customer_id = $1 AND deleted_at IS NULL;

-- name: UpdatePPPoEUser :one
UPDATE pppoe_users SET
    username = $2,
    password_encrypted = $3,
    profile_name = $4,
    service = $5,
    remote_address = $6,
    comment = $7,
    disabled = $8,
    use_simple_queue = $9,
    status = $10,
    sync_status = $11,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at;

-- name: SoftDeletePPPoEUser :exec
UPDATE pppoe_users SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPPPoEUsers :many
SELECT id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM pppoe_users
WHERE router_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('sync_status')::varchar IS NULL OR sync_status = sqlc.narg('sync_status')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR username ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPPPoEUsers :one
SELECT COUNT(*) FROM pppoe_users
WHERE router_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('sync_status')::varchar IS NULL OR sync_status = sqlc.narg('sync_status')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR username ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: GetPPPoEUsersByRouterID :many
SELECT id, tenant_id, customer_id, router_id, username, password_encrypted,
    profile_name, service, remote_address, comment, disabled,
    use_simple_queue, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM pppoe_users
WHERE router_id = $1 AND deleted_at IS NULL AND status = 'active';

-- name: GetSyncStatusSummary :one
SELECT
    COUNT(CASE WHEN sync_status = 'synced' THEN 1 END) AS synced_count,
    COUNT(CASE WHEN sync_status = 'pending_create' THEN 1 END) AS pending_create_count,
    COUNT(CASE WHEN sync_status = 'pending_update' THEN 1 END) AS pending_update_count,
    COUNT(CASE WHEN sync_status = 'pending_delete' THEN 1 END) AS pending_delete_count,
    COUNT(CASE WHEN sync_status = 'out_of_sync' THEN 1 END) AS out_of_sync_count,
    COUNT(CASE WHEN sync_status = 'error' THEN 1 END) AS error_count,
    MAX(last_sync_at) AS last_sync_at
FROM pppoe_users
WHERE router_id = $1 AND deleted_at IS NULL;

-- name: UpdateSyncStatus :exec
UPDATE pppoe_users SET
    sync_status = $2,
    last_sync_at = $3,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: BulkUpdateSyncStatus :exec
UPDATE pppoe_users SET
    sync_status = $2,
    last_sync_at = $3,
    updated_at = NOW()
WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL;
