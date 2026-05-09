-- Kueri SQL untuk aggregasi laporan operasional (operational reports).
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Mencakup: aktivitas admin, top actions, dan dashboard data.
-- Tabel dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: GetAdminActivity :many
-- Menghitung aktivitas per user dari audit_logs untuk periode tertentu.
-- Mengembalikan user_id, user_name, role, login_days, action_count, dan last_active_at.
-- login_days dihitung dari jumlah hari unik user melakukan aksi.
SELECT
    al.actor_id AS user_id,
    al.actor_name AS user_name,
    COALESCE(u.role, 'unknown') AS role,
    COUNT(DISTINCT DATE(al.created_at))::int AS login_days,
    COUNT(*)::int AS action_count,
    MAX(al.created_at) AS last_active_at
FROM audit_logs al
LEFT JOIN users u ON u.id = al.actor_id
WHERE al.tenant_id = $1
  AND al.created_at >= @period_start::timestamptz
  AND al.created_at < @period_end::timestamptz
GROUP BY al.actor_id, al.actor_name, u.role
ORDER BY action_count DESC;

-- name: GetTopActions :many
-- Menghitung aksi terbanyak dari audit_logs untuk periode tertentu.
-- Mengembalikan action_type, count, dan percentage dari total.
SELECT
    al.action AS action_type,
    COUNT(*)::int AS count,
    CASE
        WHEN SUM(COUNT(*)) OVER () = 0 THEN 0
        ELSE ROUND(COUNT(*)::numeric / SUM(COUNT(*)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM audit_logs al
WHERE al.tenant_id = $1
  AND al.created_at >= @period_start::timestamptz
  AND al.created_at < @period_end::timestamptz
GROUP BY al.action
ORDER BY count DESC
LIMIT 20;

-- name: GetDashboardData :one
-- Mengambil data ringkasan untuk dashboard widget.
-- Aggregasi metrik kunci: pelanggan aktif, trend, pendapatan bulan ini,
-- piutang, collection rate, churn rate, dan ARPU.
-- Dioptimasi untuk fast loading (target < 500ms).
SELECT
    -- Total pelanggan aktif
    COALESCE((
        SELECT COUNT(*)
        FROM customers c
        WHERE c.tenant_id = $1
          AND c.status = 'aktif'
          AND c.deleted_at IS NULL
    ), 0)::int AS total_active_customers,

    -- Trend pelanggan vs bulan lalu (persentase perubahan)
    CASE
        WHEN COALESCE((
            SELECT COUNT(*)
            FROM customers c2
            WHERE c2.tenant_id = $1
              AND c2.status = 'aktif'
              AND c2.deleted_at IS NULL
              AND c2.activation_date <= (DATE_TRUNC('month', CURRENT_DATE) - INTERVAL '1 day')::date
        ), 0) = 0 THEN 0
        ELSE ROUND(
            (
                (SELECT COUNT(*) FROM customers c3
                 WHERE c3.tenant_id = $1 AND c3.status = 'aktif' AND c3.deleted_at IS NULL)::numeric -
                (SELECT COUNT(*) FROM customers c4
                 WHERE c4.tenant_id = $1 AND c4.status = 'aktif' AND c4.deleted_at IS NULL
                   AND c4.activation_date <= (DATE_TRUNC('month', CURRENT_DATE) - INTERVAL '1 day')::date)::numeric
            ) /
            GREATEST((SELECT COUNT(*) FROM customers c5
                      WHERE c5.tenant_id = $1 AND c5.status = 'aktif' AND c5.deleted_at IS NULL
                        AND c5.activation_date <= (DATE_TRUNC('month', CURRENT_DATE) - INTERVAL '1 day')::date), 1)::numeric * 100,
            2
        )
    END::float8 AS customers_trend,

    -- Pendapatan bulan ini
    COALESCE((
        SELECT SUM(ip.amount)
        FROM invoice_payments ip
        WHERE ip.tenant_id = $1
          AND ip.voided = false
          AND ip.payment_date >= DATE_TRUNC('month', CURRENT_DATE)::date
          AND ip.payment_date < (DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month')::date
    ), 0)::bigint AS monthly_revenue,

    -- Total piutang (outstanding)
    COALESCE((
        SELECT SUM(i.total_amount - i.paid_amount)
        FROM invoices i
        WHERE i.tenant_id = $1
          AND i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
          AND (i.total_amount - i.paid_amount) > 0
    ), 0)::bigint AS total_receivables,

    -- Jumlah pelanggan dengan piutang
    COALESCE((
        SELECT COUNT(DISTINCT i.customer_id)
        FROM invoices i
        WHERE i.tenant_id = $1
          AND i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
          AND (i.total_amount - i.paid_amount) > 0
    ), 0)::int AS receivables_count,

    -- Collection rate bulan ini
    CASE
        WHEN COALESCE((
            SELECT COUNT(*)
            FROM invoices i2
            WHERE i2.tenant_id = $1
              AND i2.due_date >= DATE_TRUNC('month', CURRENT_DATE)::date
              AND i2.due_date < (DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month')::date
              AND i2.status != 'batal'
        ), 0) = 0 THEN 0
        ELSE ROUND(
            (SELECT COUNT(*) FROM invoices i3
             WHERE i3.tenant_id = $1
               AND i3.due_date >= DATE_TRUNC('month', CURRENT_DATE)::date
               AND i3.due_date < (DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month')::date
               AND i3.status = 'lunas')::numeric /
            GREATEST((SELECT COUNT(*) FROM invoices i4
                      WHERE i4.tenant_id = $1
                        AND i4.due_date >= DATE_TRUNC('month', CURRENT_DATE)::date
                        AND i4.due_date < (DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month')::date
                        AND i4.status != 'batal'), 1)::numeric * 100,
            2
        )
    END::float8 AS collection_rate,

    -- Churn rate bulan ini
    CASE
        WHEN COALESCE((
            SELECT COUNT(*)
            FROM customers c6
            WHERE c6.tenant_id = $1
              AND c6.activation_date < DATE_TRUNC('month', CURRENT_DATE)::date
              AND (c6.status != 'berhenti' OR c6.updated_at >= DATE_TRUNC('month', CURRENT_DATE))
              AND (c6.deleted_at IS NULL OR c6.deleted_at >= DATE_TRUNC('month', CURRENT_DATE))
        ), 0) = 0 THEN 0
        ELSE ROUND(
            (SELECT COUNT(*) FROM customers c7
             WHERE c7.tenant_id = $1
               AND c7.status = 'berhenti'
               AND c7.updated_at >= DATE_TRUNC('month', CURRENT_DATE)
               AND c7.updated_at < (DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month'))::numeric /
            GREATEST((SELECT COUNT(*) FROM customers c8
                      WHERE c8.tenant_id = $1
                        AND c8.activation_date < DATE_TRUNC('month', CURRENT_DATE)::date
                        AND (c8.status != 'berhenti' OR c8.updated_at >= DATE_TRUNC('month', CURRENT_DATE))
                        AND (c8.deleted_at IS NULL OR c8.deleted_at >= DATE_TRUNC('month', CURRENT_DATE))), 1)::numeric * 100,
            2
        )
    END::float8 AS churn_rate,

    -- ARPU bulan ini
    CASE
        WHEN COALESCE((
            SELECT COUNT(*)
            FROM customers c9
            WHERE c9.tenant_id = $1
              AND c9.status = 'aktif'
              AND c9.deleted_at IS NULL
        ), 0) = 0 THEN 0
        ELSE (
            COALESCE((
                SELECT SUM(ip2.amount)
                FROM invoice_payments ip2
                WHERE ip2.tenant_id = $1
                  AND ip2.voided = false
                  AND ip2.payment_date >= DATE_TRUNC('month', CURRENT_DATE)::date
                  AND ip2.payment_date < (DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month')::date
            ), 0) /
            GREATEST((
                SELECT COUNT(*)
                FROM customers c10
                WHERE c10.tenant_id = $1
                  AND c10.status = 'aktif'
                  AND c10.deleted_at IS NULL
            ), 1)
        )
    END::bigint AS arpu;
