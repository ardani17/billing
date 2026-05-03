-- Query SQL untuk operasi CRUD tabel vpn_tunnels.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel vpn_tunnels dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateVPNTunnel :one
INSERT INTO vpn_tunnels (
    tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    bandwidth_cap_mbps, rate_limit_pps, active_endpoint, notes
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10,
    $11, $12,
    $13, $14, $15, $16,
    $17, $18, $19, $20
)
RETURNING id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at;

-- name: GetVPNTunnelByID :one
SELECT id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at
FROM vpn_tunnels
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateVPNTunnel :one
UPDATE vpn_tunnels SET
    tunnel_name = $2,
    router_id = $3,
    notes = $4,
    persistent_keepalive = $5,
    allowed_addresses = $6,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at;

-- name: SoftDeleteVPNTunnel :exec
UPDATE vpn_tunnels SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListVPNTunnels :many
-- Daftar tunnel dengan paginasi, filter opsional status/protocol, dan pencarian nama.
SELECT id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at
FROM vpn_tunnels
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('protocol')::varchar IS NULL OR protocol = sqlc.narg('protocol')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR tunnel_name ILIKE '%' || sqlc.narg('search')::varchar || '%')
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountVPNTunnels :one
-- Menghitung total tunnel untuk paginasi, dengan filter yang sama seperti ListVPNTunnels.
SELECT COUNT(*) FROM vpn_tunnels
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
  AND (sqlc.narg('protocol')::varchar IS NULL OR protocol = sqlc.narg('protocol')::varchar)
  AND (sqlc.narg('search')::varchar IS NULL OR tunnel_name ILIKE '%' || sqlc.narg('search')::varchar || '%');

-- name: GetVPNTunnelsByStatus :many
-- Mengambil semua tunnel dengan status tertentu (tenant-scoped via RLS).
SELECT id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at
FROM vpn_tunnels
WHERE status = $1 AND deleted_at IS NULL;

-- name: CountVPNTunnelsByStatus :many
-- Menghitung jumlah tunnel per status untuk dashboard summary.
SELECT status, COUNT(*) AS count
FROM vpn_tunnels
WHERE deleted_at IS NULL
GROUP BY status;

-- name: VPNTunnelNameExists :one
-- Mengecek apakah tunnel_name sudah ada di tenant (exclude ID tertentu untuk update).
SELECT EXISTS(
    SELECT 1 FROM vpn_tunnels
    WHERE tenant_id = $1 AND tunnel_name = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: VPNIPExists :one
-- Mengecek apakah vpn_ip sudah digunakan di tenant.
SELECT EXISTS(
    SELECT 1 FROM vpn_tunnels
    WHERE tenant_id = $1 AND vpn_ip = $2 AND deleted_at IS NULL
) AS exists;

-- name: UpdateVPNTunnelStatus :exec
-- Memperbarui status tunnel dan field terkait health check.
UPDATE vpn_tunnels SET
    status = $2,
    last_handshake_at = $3,
    latency_ms = $4,
    active_endpoint = $5,
    updated_at = now()
WHERE id = $1;

-- name: GetConnectedVPNTunnels :many
-- Mengambil semua tunnel dengan status 'connected' (cross-tenant untuk health monitor).
-- Query ini dijalankan tanpa RLS context oleh health monitor goroutine.
SELECT id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at
FROM vpn_tunnels
WHERE status = 'connected' AND deleted_at IS NULL;

-- name: GetDisconnectedVPNTunnels :many
-- Mengambil semua tunnel dengan status 'disconnected' (cross-tenant untuk recovery check).
-- Query ini dijalankan tanpa RLS context oleh health monitor goroutine.
SELECT id, tenant_id, router_id, tunnel_name, protocol, vpn_ip,
    server_endpoint, server_public_key, client_public_key,
    client_private_key_encrypted, pre_shared_key_encrypted,
    l2tp_username, l2tp_password_encrypted,
    status, listen_port, allowed_addresses, persistent_keepalive,
    last_handshake_at, latency_ms, bandwidth_cap_mbps, rate_limit_pps,
    active_endpoint, notes, created_at, updated_at, deleted_at
FROM vpn_tunnels
WHERE status = 'disconnected' AND deleted_at IS NULL;
