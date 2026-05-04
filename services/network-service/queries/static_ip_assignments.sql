-- Query SQL untuk static IP assignments.

-- name: CreateStaticIPAssignment :one
INSERT INTO static_ip_assignments (
    tenant_id, router_id, customer_id, ip_address, address_list,
    queue_name, rate_limit, comment, status, sync_status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10
)
RETURNING id, tenant_id, router_id, customer_id, ip_address, address_list,
    queue_name, rate_limit, comment, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at;

-- name: GetStaticIPAssignmentByID :one
SELECT id, tenant_id, router_id, customer_id, ip_address, address_list,
    queue_name, rate_limit, comment, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM static_ip_assignments
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetStaticIPAssignmentByRouterAndIP :one
SELECT id, tenant_id, router_id, customer_id, ip_address, address_list,
    queue_name, rate_limit, comment, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM static_ip_assignments
WHERE router_id = $1 AND ip_address = $2 AND deleted_at IS NULL;

-- name: ListStaticIPAssignments :many
SELECT id, tenant_id, router_id, customer_id, ip_address, address_list,
    queue_name, rate_limit, comment, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at
FROM static_ip_assignments
WHERE router_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('search')::varchar IS NULL
       OR host(ip_address) ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR queue_name ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR comment ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStaticIPAssignments :one
SELECT COUNT(*) FROM static_ip_assignments
WHERE router_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('search')::varchar IS NULL
       OR host(ip_address) ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR queue_name ILIKE '%' || sqlc.narg('search')::varchar || '%'
       OR comment ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: UpdateStaticIPAssignment :one
UPDATE static_ip_assignments SET
    customer_id = $2,
    ip_address = $3,
    address_list = $4,
    queue_name = $5,
    rate_limit = $6,
    comment = $7,
    status = $8,
    sync_status = $9,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, router_id, customer_id, ip_address, address_list,
    queue_name, rate_limit, comment, status, last_sync_at, sync_status,
    created_at, updated_at, deleted_at;

-- name: SoftDeleteStaticIPAssignment :exec
UPDATE static_ip_assignments SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateStaticIPAssignmentSyncState :exec
UPDATE static_ip_assignments SET
    sync_status = $2,
    last_sync_at = $3,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
