-- Query SQL untuk operasi CRUD tabel billing_settings.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel billing_settings dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant memiliki tepat satu baris billing_settings (UNIQUE pada tenant_id).

-- name: GetBillingSettingsByTenantID :one
-- Mengambil billing settings berdasarkan tenant ID.
SELECT *
FROM billing_settings
WHERE tenant_id = $1;

-- name: UpsertBillingSettings :one
-- Membuat atau memperbarui billing settings untuk tenant.
-- Menggunakan INSERT ON CONFLICT untuk upsert berdasarkan tenant_id.
INSERT INTO billing_settings (
    tenant_id, generate_days, grace_period_days, suspend_days,
    tax_enabled, tax_rate, penalty_enabled, penalty_type,
    penalty_amount, penalty_percentage, penalty_daily_amount, penalty_max_amount,
    invoice_prefix, new_customer_billing, timezone,
    auto_isolir, auto_open_isolir
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, $12,
    $13, $14, $15,
    $16, $17
)
ON CONFLICT (tenant_id) DO UPDATE SET
    generate_days = EXCLUDED.generate_days,
    grace_period_days = EXCLUDED.grace_period_days,
    suspend_days = EXCLUDED.suspend_days,
    tax_enabled = EXCLUDED.tax_enabled,
    tax_rate = EXCLUDED.tax_rate,
    penalty_enabled = EXCLUDED.penalty_enabled,
    penalty_type = EXCLUDED.penalty_type,
    penalty_amount = EXCLUDED.penalty_amount,
    penalty_percentage = EXCLUDED.penalty_percentage,
    penalty_daily_amount = EXCLUDED.penalty_daily_amount,
    penalty_max_amount = EXCLUDED.penalty_max_amount,
    invoice_prefix = EXCLUDED.invoice_prefix,
    new_customer_billing = EXCLUDED.new_customer_billing,
    timezone = EXCLUDED.timezone,
    auto_isolir = EXCLUDED.auto_isolir,
    auto_open_isolir = EXCLUDED.auto_open_isolir,
    updated_at = NOW()
RETURNING *;

-- name: ListAllBillingSettings :many
-- Mengambil semua billing settings (untuk cron job lintas tenant).
-- Query ini dijalankan tanpa RLS context (superuser/service role).
SELECT *
FROM billing_settings;
