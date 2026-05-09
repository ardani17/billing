-- Kueri SQL untuk operasi CRUD tabel payment_gateway_configs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel payment_gateway_configs dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant bisa memiliki satu konfigurasi per provider (UNIQUE pada tenant_id, gateway_provider).

-- name: CreateGatewayConfig :one
-- Membuat konfigurasi gateway baru dan mengembalikan semua kolom.
INSERT INTO payment_gateway_configs (
    tenant_id, gateway_provider, api_key_encrypted,
    webhook_secret_encrypted, enabled_methods, payment_link_expiry_days
) VALUES (
    $1, $2, $3,
    $4, $5, $6
)
RETURNING *;

-- name: GetGatewayConfigByID :one
-- Mengambil konfigurasi gateway berdasarkan ID (tenant-scoped via RLS).
SELECT *
FROM payment_gateway_configs
WHERE id = $1;

-- name: UpdateGatewayConfig :one
-- Memperbarui konfigurasi gateway dan mengembalikan semua kolom.
-- Kolom yang diperbarui: api_key_encrypted, webhook_secret_encrypted,
-- enabled_methods, payment_link_expiry_days, updated_at.
UPDATE payment_gateway_configs SET
    api_key_encrypted = $2,
    webhook_secret_encrypted = $3,
    enabled_methods = $4,
    payment_link_expiry_days = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateGatewayConfig :exec
-- Menonaktifkan konfigurasi gateway (hapus lunak) dengan atur is_active=false.
UPDATE payment_gateway_configs SET
    is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: ListGatewayConfigsByTenant :many
-- Mengambil semua konfigurasi gateway untuk tenant tertentu.
SELECT *
FROM payment_gateway_configs
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: GetActiveGatewayConfigsByTenant :many
-- Mengambil konfigurasi gateway aktif untuk tenant tertentu.
SELECT *
FROM payment_gateway_configs
WHERE tenant_id = $1 AND is_active = true
ORDER BY created_at ASC;

-- name: GetActiveGatewayConfigByProvider :one
-- Mengambil konfigurasi gateway aktif berdasarkan provider untuk tenant.
SELECT *
FROM payment_gateway_configs
WHERE tenant_id = $1 AND gateway_provider = $2 AND is_active = true;

-- name: ExistsGatewayConfigByProvider :one
-- Mengecek apakah konfigurasi aktif sudah ada untuk provider di tenant.
SELECT EXISTS(
    SELECT 1 FROM payment_gateway_configs
    WHERE tenant_id = $1 AND gateway_provider = $2 AND is_active = true
) AS exists;
