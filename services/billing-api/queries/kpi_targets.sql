-- Kueri SQL untuk operasi CRUD tabel kpi_targets.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel kpi_targets dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant hanya memiliki satu baris kpi_targets (UNIQUE pada tenant_id).

-- name: GetKPITargetByTenant :one
-- Mengambil target KPI berdasarkan tenant_id.
SELECT * FROM kpi_targets
WHERE tenant_id = $1;

-- name: UpsertKPITarget :one
-- Membuat atau memperbarui target KPI untuk tenant.
-- Jika tenant sudah memiliki target, semua kolom diperbarui.
INSERT INTO kpi_targets (
    tenant_id,
    monthly_revenue_target,
    collection_rate_target,
    max_receivables,
    new_customers_monthly_target,
    max_churn_rate,
    total_customers_target,
    sla_uptime_target,
    max_active_alarms,
    min_signal_quality_percentage
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (tenant_id) DO UPDATE SET
    monthly_revenue_target = EXCLUDED.monthly_revenue_target,
    collection_rate_target = EXCLUDED.collection_rate_target,
    max_receivables = EXCLUDED.max_receivables,
    new_customers_monthly_target = EXCLUDED.new_customers_monthly_target,
    max_churn_rate = EXCLUDED.max_churn_rate,
    total_customers_target = EXCLUDED.total_customers_target,
    sla_uptime_target = EXCLUDED.sla_uptime_target,
    max_active_alarms = EXCLUDED.max_active_alarms,
    min_signal_quality_percentage = EXCLUDED.min_signal_quality_percentage,
    updated_at = NOW()
RETURNING *;
