-- Kueri SQL untuk operasi tabel dhcp_bindings.

-- name: CreateDHCPBinding :one
INSERT INTO dhcp_bindings (
    tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name,
    comment, disabled, status, sync_status
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, $12
)
RETURNING id, tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name, comment, disabled,
    status, last_sync_at, sync_status, created_at, updated_at, deleted_at;

-- name: GetDHCPBindingByID :one
SELECT id, tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name, comment, disabled,
    status, last_sync_at, sync_status, created_at, updated_at, deleted_at
FROM dhcp_bindings
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDHCPBindingByRouterAndMAC :one
SELECT id, tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name, comment, disabled,
    status, last_sync_at, sync_status, created_at, updated_at, deleted_at
FROM dhcp_bindings
WHERE router_id = $1 AND lower(mac_address) = lower($2) AND deleted_at IS NULL;

-- name: GetDHCPBindingByRouterAndIP :one
SELECT id, tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name, comment, disabled,
    status, last_sync_at, sync_status, created_at, updated_at, deleted_at
FROM dhcp_bindings
WHERE router_id = $1 AND ip_address = $2 AND deleted_at IS NULL;

-- name: ListDHCPBindings :many
SELECT id, tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name, comment, disabled,
    status, last_sync_at, sync_status, created_at, updated_at, deleted_at
FROM dhcp_bindings
WHERE router_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('search')::varchar IS NULL
       OR mac_address ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR host_name ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR host(ip_address) ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountDHCPBindings :one
SELECT COUNT(*) FROM dhcp_bindings
WHERE router_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('search')::varchar IS NULL
       OR mac_address ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR host_name ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR host(ip_address) ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: UpdateDHCPBinding :one
UPDATE dhcp_bindings SET
    customer_id = $2,
    router_lease_id = $3,
    server = $4,
    mac_address = $5,
    ip_address = $6,
    host_name = $7,
    comment = $8,
    disabled = $9,
    status = $10,
    sync_status = $11,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, router_id, customer_id, router_lease_id,
    server, mac_address, ip_address, host_name, comment, disabled,
    status, last_sync_at, sync_status, created_at, updated_at, deleted_at;

-- name: SoftDeleteDHCPBinding :exec
UPDATE dhcp_bindings SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateDHCPBindingSyncState :exec
UPDATE dhcp_bindings SET
    router_lease_id = $2,
    sync_status = $3,
    last_sync_at = $4,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
