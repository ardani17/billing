-- Kueri SQL untuk operasi tabel vpn_subnets.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel vpn_subnets dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant mendapat 1 subnet /24: 10.99.{tenant_seq}.0/24.

-- name: GetVPNSubnetByTenantID :one
SELECT id, tenant_id, subnet_prefix, tenant_seq, server_ip, next_client_ip_seq, created_at
FROM vpn_subnets
WHERE tenant_id = $1;

-- name: CreateVPNSubnet :one
INSERT INTO vpn_subnets (
    tenant_id, subnet_prefix, tenant_seq, server_ip, next_client_ip_seq
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, tenant_id, subnet_prefix, tenant_seq, server_ip, next_client_ip_seq, created_at;

-- name: GetNextTenantSeq :one
-- Mengambil tenant_seq berikutnya yang tersedia untuk alokasi subnet baru.
SELECT COALESCE(MAX(tenant_seq), 0) + 1 AS next_seq
FROM vpn_subnets;

-- name: IncrementNextClientIPSeq :one
-- Menaikkan next_client_ip_seq dan mengembalikan nilai sebelumnya (untuk alokasi IP client).
UPDATE vpn_subnets
SET next_client_ip_seq = next_client_ip_seq + 1
WHERE tenant_id = $1
RETURNING next_client_ip_seq - 1 AS current_seq;
