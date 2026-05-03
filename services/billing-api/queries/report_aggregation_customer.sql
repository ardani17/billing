-- Query SQL untuk aggregasi laporan pelanggan (customer reports).
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Mencakup: pertumbuhan pelanggan, distribusi, churn analysis, ARPU, CLV.
-- Tabel dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: GetCustomerGrowthData :one
-- Menghitung data pertumbuhan pelanggan untuk periode tertentu.
-- total_active: pelanggan aktif saat ini (status = 'aktif').
-- new_customers: pelanggan baru dalam periode (activation_date dalam range).
-- churned_customers: pelanggan berhenti dalam periode (status = 'berhenti', updated_at dalam range).
-- net_growth: new_customers - churned_customers.
SELECT
    COALESCE(COUNT(*) FILTER (WHERE c.status = 'aktif' AND c.deleted_at IS NULL), 0)::int AS total_active,
    COALESCE(COUNT(*) FILTER (
        WHERE c.activation_date >= @period_start::date
          AND c.activation_date < @period_end::date
          AND c.deleted_at IS NULL
    ), 0)::int AS new_customers,
    COALESCE(COUNT(*) FILTER (
        WHERE c.status = 'berhenti'
          AND c.updated_at >= @period_start::timestamptz
          AND c.updated_at < @period_end::timestamptz
    ), 0)::int AS churned_customers,
    (
        COALESCE(COUNT(*) FILTER (
            WHERE c.activation_date >= @period_start::date
              AND c.activation_date < @period_end::date
              AND c.deleted_at IS NULL
        ), 0) -
        COALESCE(COUNT(*) FILTER (
            WHERE c.status = 'berhenti'
              AND c.updated_at >= @period_start::timestamptz
              AND c.updated_at < @period_end::timestamptz
        ), 0)
    )::int AS net_growth
FROM customers c
WHERE c.tenant_id = $1;

-- name: GetMonthlyGrowthTrend :many
-- Menghitung trend pertumbuhan pelanggan per bulan untuk 12 bulan terakhir.
-- Mengembalikan total_active, new_customers, dan churned_customers per bulan.
SELECT
    TO_CHAR(month_series, 'YYYY-MM') AS month,
    COALESCE((
        SELECT COUNT(*)
        FROM customers c2
        WHERE c2.tenant_id = $1
          AND c2.status = 'aktif'
          AND c2.deleted_at IS NULL
          AND c2.activation_date <= (month_series + INTERVAL '1 month' - INTERVAL '1 day')::date
    ), 0)::int AS total_active,
    COALESCE((
        SELECT COUNT(*)
        FROM customers c3
        WHERE c3.tenant_id = $1
          AND c3.deleted_at IS NULL
          AND c3.activation_date >= month_series::date
          AND c3.activation_date < (month_series + INTERVAL '1 month')::date
    ), 0)::int AS new_customers,
    COALESCE((
        SELECT COUNT(*)
        FROM customers c4
        WHERE c4.tenant_id = $1
          AND c4.status = 'berhenti'
          AND c4.updated_at >= month_series
          AND c4.updated_at < (month_series + INTERVAL '1 month')
    ), 0)::int AS churned_customers
FROM generate_series(
    DATE_TRUNC('month', CURRENT_DATE - INTERVAL '11 months'),
    DATE_TRUNC('month', CURRENT_DATE),
    '1 month'::interval
) AS month_series
ORDER BY month_series ASC;

-- name: GetCustomerDistributionByPackage :many
-- Menghitung distribusi pelanggan aktif per paket.
-- Mengembalikan package_id, package_name, customer_count, dan percentage.
SELECT
    p.id::text AS id,
    p.name AS name,
    COUNT(c.id)::int AS count,
    CASE
        WHEN SUM(COUNT(c.id)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(c.id)::numeric / SUM(COUNT(c.id)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM packages p
LEFT JOIN customers c ON c.package_id = p.id
    AND c.status = 'aktif'
    AND c.deleted_at IS NULL
WHERE p.tenant_id = $1
GROUP BY p.id, p.name
ORDER BY count DESC;

-- name: GetCustomerDistributionByArea :many
-- Menghitung distribusi pelanggan aktif per area.
-- Mengembalikan area_id, area_name, customer_count, dan percentage.
SELECT
    a.id::text AS id,
    a.name AS name,
    COUNT(c.id)::int AS count,
    CASE
        WHEN SUM(COUNT(c.id)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(c.id)::numeric / SUM(COUNT(c.id)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM areas a
LEFT JOIN customers c ON c.area_id = a.id
    AND c.status = 'aktif'
    AND c.deleted_at IS NULL
WHERE a.tenant_id = $1
GROUP BY a.id, a.name
ORDER BY count DESC;

-- name: GetCustomerDistributionByStatus :many
-- Menghitung distribusi pelanggan per status.
-- Mengembalikan status name dan customer_count.
SELECT
    c.status AS name,
    COUNT(*)::int AS count,
    CASE
        WHEN SUM(COUNT(*)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(*)::numeric / SUM(COUNT(*)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM customers c
WHERE c.tenant_id = $1
  AND c.deleted_at IS NULL
GROUP BY c.status
ORDER BY count DESC;

-- name: GetCustomerDistributionByConnectionMethod :many
-- Menghitung distribusi pelanggan aktif per metode koneksi.
-- Metode: pppoe, hotspot, dhcp_binding, static.
SELECT
    c.connection_method AS name,
    COUNT(*)::int AS count,
    CASE
        WHEN SUM(COUNT(*)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(*)::numeric / SUM(COUNT(*)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM customers c
WHERE c.tenant_id = $1
  AND c.status = 'aktif'
  AND c.deleted_at IS NULL
GROUP BY c.connection_method
ORDER BY count DESC;

-- name: GetChurnAnalysis :one
-- Menghitung jumlah pelanggan churn dan churn rate untuk periode tertentu.
-- Churn rate = churned / total_active_at_start * 100.
SELECT
    COALESCE(COUNT(*) FILTER (
        WHERE c.status = 'berhenti'
          AND c.updated_at >= @period_start::timestamptz
          AND c.updated_at < @period_end::timestamptz
    ), 0)::int AS churned_count,
    CASE
        WHEN COUNT(*) FILTER (
            WHERE c.status != 'berhenti'
              AND c.deleted_at IS NULL
              AND c.activation_date < @period_start::date
        ) = 0 THEN 0
        ELSE ROUND(
            COUNT(*) FILTER (
                WHERE c.status = 'berhenti'
                  AND c.updated_at >= @period_start::timestamptz
                  AND c.updated_at < @period_end::timestamptz
            )::numeric /
            GREATEST(COUNT(*) FILTER (
                WHERE c.activation_date < @period_start::date
                  AND (c.status != 'berhenti' OR c.updated_at >= @period_start::timestamptz)
                  AND (c.deleted_at IS NULL OR c.deleted_at >= @period_start::timestamptz)
            ), 1)::numeric * 100,
            2
        )
    END::float8 AS churn_rate
FROM customers c
WHERE c.tenant_id = $1;

-- name: GetChurnByReason :many
-- Menghitung churn per alasan berhenti.
-- Alasan diambil dari field notes pelanggan yang berhenti.
-- Jika notes kosong, alasan = 'Tidak diketahui'.
SELECT
    COALESCE(NULLIF(TRIM(c.notes), ''), 'Tidak diketahui') AS reason,
    COUNT(*)::int AS count,
    CASE
        WHEN SUM(COUNT(*)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(*)::numeric / SUM(COUNT(*)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM customers c
WHERE c.tenant_id = $1
  AND c.status = 'berhenti'
  AND c.updated_at >= @period_start::timestamptz
  AND c.updated_at < @period_end::timestamptz
GROUP BY COALESCE(NULLIF(TRIM(c.notes), ''), 'Tidak diketahui')
ORDER BY count DESC;

-- name: GetChurnByPackage :many
-- Menghitung churn per paket.
-- Mengembalikan package_name, churned_count, dan percentage.
SELECT
    p.name AS name,
    COUNT(c.id)::int AS count,
    CASE
        WHEN SUM(COUNT(c.id)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(c.id)::numeric / SUM(COUNT(c.id)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM customers c
JOIN packages p ON p.id = c.package_id
WHERE c.tenant_id = $1
  AND c.status = 'berhenti'
  AND c.updated_at >= @period_start::timestamptz
  AND c.updated_at < @period_end::timestamptz
GROUP BY p.name
ORDER BY count DESC;

-- name: GetChurnByArea :many
-- Menghitung churn per area.
-- Mengembalikan area_name, churned_count, dan percentage.
SELECT
    COALESCE(a.name, 'Tanpa Area') AS name,
    COUNT(c.id)::int AS count,
    CASE
        WHEN SUM(COUNT(c.id)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(c.id)::numeric / SUM(COUNT(c.id)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM customers c
LEFT JOIN areas a ON a.id = c.area_id
WHERE c.tenant_id = $1
  AND c.status = 'berhenti'
  AND c.updated_at >= @period_start::timestamptz
  AND c.updated_at < @period_end::timestamptz
GROUP BY a.name
ORDER BY count DESC;

-- name: GetAvgCustomerLifetime :one
-- Menghitung rata-rata masa berlangganan pelanggan yang berhenti (dalam bulan).
-- Lifetime = selisih antara activation_date dan updated_at (saat berhenti).
SELECT
    COALESCE(
        AVG(
            EXTRACT(EPOCH FROM (c.updated_at - c.activation_date::timestamptz)) / (30.44 * 86400)
        ),
        0
    )::float8 AS avg_lifetime_months
FROM customers c
WHERE c.tenant_id = $1
  AND c.status = 'berhenti'
  AND c.updated_at >= @period_start::timestamptz
  AND c.updated_at < @period_end::timestamptz;

-- name: GetARPU :one
-- Menghitung Average Revenue Per User (ARPU) untuk periode tertentu.
-- ARPU = total revenue / jumlah pelanggan aktif rata-rata.
SELECT
    CASE
        WHEN COALESCE((
            SELECT COUNT(*)
            FROM customers c2
            WHERE c2.tenant_id = $1
              AND c2.status = 'aktif'
              AND c2.deleted_at IS NULL
        ), 0) = 0 THEN 0
        ELSE (
            COALESCE((
                SELECT SUM(ip.amount)
                FROM invoice_payments ip
                WHERE ip.tenant_id = $1
                  AND ip.voided = false
                  AND ip.payment_date >= @period_start::date
                  AND ip.payment_date < @period_end::date
            ), 0) /
            GREATEST((
                SELECT COUNT(*)
                FROM customers c3
                WHERE c3.tenant_id = $1
                  AND c3.status = 'aktif'
                  AND c3.deleted_at IS NULL
            ), 1)
        )
    END::bigint AS arpu;

-- name: GetCLV :one
-- Menghitung Customer Lifetime Value (CLV) untuk periode tertentu.
-- CLV = ARPU * rata-rata lifetime pelanggan (dalam bulan).
-- Menggunakan semua pelanggan yang pernah berhenti untuk menghitung avg lifetime.
SELECT
    CASE
        WHEN COALESCE((
            SELECT COUNT(*)
            FROM customers c2
            WHERE c2.tenant_id = $1
              AND c2.status = 'aktif'
              AND c2.deleted_at IS NULL
        ), 0) = 0 THEN 0
        ELSE (
            -- ARPU
            COALESCE((
                SELECT SUM(ip.amount)
                FROM invoice_payments ip
                WHERE ip.tenant_id = $1
                  AND ip.voided = false
                  AND ip.payment_date >= @period_start::date
                  AND ip.payment_date < @period_end::date
            ), 0) /
            GREATEST((
                SELECT COUNT(*)
                FROM customers c3
                WHERE c3.tenant_id = $1
                  AND c3.status = 'aktif'
                  AND c3.deleted_at IS NULL
            ), 1)
        ) *
        -- Rata-rata lifetime (bulan), minimal 1 bulan
        GREATEST(
            COALESCE((
                SELECT AVG(
                    EXTRACT(EPOCH FROM (c4.updated_at - c4.activation_date::timestamptz)) / (30.44 * 86400)
                )
                FROM customers c4
                WHERE c4.tenant_id = $1
                  AND c4.status = 'berhenti'
            ), 12)::bigint,
            1
        )
    END::bigint AS clv;
