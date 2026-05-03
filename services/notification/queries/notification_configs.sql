-- Query SQL untuk operasi CRUD tabel notification_configs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel notification_configs dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant bisa memiliki satu konfigurasi per channel (UNIQUE pada tenant_id, channel).

-- name: GetConfigsByTenant :many
-- Mengambil semua konfigurasi notifikasi untuk tenant tertentu.
SELECT *
FROM notification_configs
WHERE tenant_id = $1
ORDER BY priority ASC;

-- name: GetConfigByTenantAndChannel :one
-- Mengambil konfigurasi notifikasi berdasarkan tenant dan channel.
SELECT *
FROM notification_configs
WHERE tenant_id = $1 AND channel = $2;

-- name: UpsertConfig :one
-- Membuat atau memperbarui konfigurasi notifikasi per tenant per channel.
-- Menggunakan INSERT ON CONFLICT untuk upsert berdasarkan (tenant_id, channel).
INSERT INTO notification_configs (
    tenant_id, channel, provider, credentials, is_enabled, priority, settings
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (tenant_id, channel) DO UPDATE SET
    provider = EXCLUDED.provider,
    credentials = EXCLUDED.credentials,
    is_enabled = EXCLUDED.is_enabled,
    priority = EXCLUDED.priority,
    updated_at = NOW()
RETURNING *;

-- name: GetSettingsByTenant :one
-- Mengambil pengaturan umum notifikasi untuk tenant tertentu.
-- Mengambil settings dari baris pertama config milik tenant.
SELECT settings
FROM notification_configs
WHERE tenant_id = $1
LIMIT 1;

-- name: UpdateSettings :exec
-- Memperbarui pengaturan umum notifikasi untuk semua config milik tenant.
UPDATE notification_configs
SET settings = $2, updated_at = NOW()
WHERE tenant_id = $1;
