-- Query SQL untuk operasi CRUD tabel routers.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel routers dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateRouter :one
INSERT INTO routers (
    tenant_id, name, host, port, username, password_encrypted,
    use_ssl, service_types, router_os_version, board_name,
    cpu_count, total_ram_mb, identity, status,
    health_check_interval_sec, notes
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10,
    $11, $12, $13, $14,
    $15, $16
)
RETURNING id, tenant_id, name, host, port, username, password_encrypted,
    use_ssl, service_types, router_os_version, board_name,
    cpu_count, total_ram_mb, identity, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    last_uptime_sec, failure_count, notes,
    deleted_at, created_at, updated_at;

-- name: GetRouterByID :one
SELECT id, tenant_id, name, host, port, username, password_encrypted,
    use_ssl, service_types, router_os_version, board_name,
    cpu_count, total_ram_mb, identity, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    last_uptime_sec, failure_count, notes,
    deleted_at, created_at, updated_at
FROM routers
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateRouter :one
UPDATE routers SET
    name = $2,
    host = $3,
    port = $4,
    username = $5,
    password_encrypted = $6,
    use_ssl = $7,
    service_types = $8,
    router_os_version = $9,
    board_name = $10,
    cpu_count = $11,
    total_ram_mb = $12,
    identity = $13,
    status = $14,
    health_check_interval_sec = $15,
    last_online_at = $16,
    notes = $17,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, name, host, port, username, password_encrypted,
    use_ssl, service_types, router_os_version, board_name,
    cpu_count, total_ram_mb, identity, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    last_uptime_sec, failure_count, notes,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteRouter :exec
UPDATE routers SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListRouters :many
SELECT id, tenant_id, name, host, port, username, password_encrypted,
    use_ssl, service_types, router_os_version, board_name,
    cpu_count, total_ram_mb, identity, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    last_uptime_sec, failure_count, notes,
    deleted_at, created_at, updated_at
FROM routers
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR name ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountRouters :one
SELECT COUNT(*) FROM routers
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR name ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: CountByStatus :many
SELECT status, COUNT(*) AS count
FROM routers
WHERE deleted_at IS NULL
GROUP BY status;

-- name: GetActiveRouters :many
SELECT id, tenant_id, name, host, port, username, password_encrypted,
    use_ssl, service_types, router_os_version, board_name,
    cpu_count, total_ram_mb, identity, status,
    health_check_interval_sec, last_online_at, last_checked_at,
    last_uptime_sec, failure_count, notes,
    deleted_at, created_at, updated_at
FROM routers
WHERE deleted_at IS NULL AND status != 'maintenance';

-- name: NameExists :one
SELECT EXISTS(
    SELECT 1 FROM routers
    WHERE tenant_id = $1 AND name = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: UpdateHealthCheck :exec
UPDATE routers SET
    last_checked_at = COALESCE($2, last_checked_at),
    last_online_at = COALESCE($3, last_online_at),
    last_uptime_sec = COALESCE($4, last_uptime_sec),
    failure_count = $5,
    status = COALESCE(NULLIF($6, ''), status),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
