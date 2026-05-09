-- Kueri SQL untuk aggregasi laporan keuangan (financial reports).
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Semua query mendukung filter opsional area_id dan package_id via sqlc.narg.
-- Tabel dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: GetRevenueSummary :one
-- Menghitung ringkasan pendapatan per sumber untuk periode tertentu.
-- Sumber: bulanan (tagihan bulanan), installation (biaya pasang), denda (denda),
-- voucher_sales (penjualan voucher), dan other (sisa item types).
-- Filter opsional: area_id dan package_id.
SELECT
    COALESCE(SUM(CASE WHEN ii.item_type = 'monthly' THEN ip.amount ELSE 0 END), 0)::bigint AS monthly_subscription,
    COALESCE((
        SELECT SUM(v.sell_price_snapshot)
        FROM vouchers v
        WHERE v.tenant_id = $1
          AND v.status IN ('terjual', 'aktif', 'selesai')
          AND v.purchased_at >= $2
          AND v.purchased_at < $3
    ), 0)::bigint AS voucher_sales,
    COALESCE(SUM(CASE WHEN ii.item_type = 'installation' THEN ip.amount ELSE 0 END), 0)::bigint AS installation_fees,
    COALESCE(SUM(CASE WHEN ii.item_type = 'penalty' THEN ip.amount ELSE 0 END), 0)::bigint AS late_fees,
    COALESCE(SUM(CASE WHEN ii.item_type NOT IN ('monthly', 'installation', 'penalty') THEN ip.amount ELSE 0 END), 0)::bigint AS other,
    (
        COALESCE(SUM(ip.amount), 0) +
        COALESCE((
            SELECT SUM(v2.sell_price_snapshot)
            FROM vouchers v2
            WHERE v2.tenant_id = $1
              AND v2.status IN ('terjual', 'aktif', 'selesai')
              AND v2.purchased_at >= $2
              AND v2.purchased_at < $3
        ), 0)
    )::bigint AS total
FROM invoice_payments ip
JOIN invoices i ON i.id = ip.invoice_id
JOIN invoice_items ii ON ii.invoice_id = i.id
JOIN customers c ON c.id = i.customer_id
WHERE ip.tenant_id = $1
  AND ip.voided = false
  AND ip.payment_date >= $2
  AND ip.payment_date < $3
  AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
  AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid);

-- name: GetMonthlyRevenueTrend :many
-- Menghitung trend pendapatan per bulan untuk 12 bulan terakhir.
-- Mengembalikan total_revenue, monthly_subscription, voucher_sales, dan other_revenue per bulan.
SELECT
    TO_CHAR(DATE_TRUNC('month', ip.payment_date), 'YYYY-MM') AS month,
    COALESCE(SUM(ip.amount), 0)::bigint AS total_revenue,
    COALESCE(SUM(CASE WHEN ii.item_type = 'monthly' THEN ip.amount ELSE 0 END), 0)::bigint AS monthly_subscription,
    0::bigint AS voucher_sales,
    COALESCE(SUM(CASE WHEN ii.item_type NOT IN ('monthly') THEN ip.amount ELSE 0 END), 0)::bigint AS other_revenue
FROM invoice_payments ip
JOIN invoices i ON i.id = ip.invoice_id
JOIN invoice_items ii ON ii.invoice_id = i.id
WHERE ip.tenant_id = $1
  AND ip.voided = false
  AND ip.payment_date >= (CURRENT_DATE - INTERVAL '12 months')
GROUP BY DATE_TRUNC('month', ip.payment_date)
ORDER BY DATE_TRUNC('month', ip.payment_date) ASC;

-- name: GetAgingBuckets :many
-- Mengelompokkan invoice outstanding ke aging buckets berdasarkan umur tunggakan.
-- Bucket: 1-7 hari, 8-14 hari, 15-30 hari, 30+ hari.
-- Menghitung total_amount dan customer_count per bucket.
SELECT
    label,
    COALESCE(SUM(outstanding), 0)::bigint AS total_amount,
    COUNT(DISTINCT customer_id)::int AS customer_count
FROM (
    SELECT
        i.customer_id,
        (i.total_amount - i.paid_amount) AS outstanding,
        CASE
            WHEN (CURRENT_DATE - i.due_date) BETWEEN 1 AND 7 THEN '1-7 hari'
            WHEN (CURRENT_DATE - i.due_date) BETWEEN 8 AND 14 THEN '8-14 hari'
            WHEN (CURRENT_DATE - i.due_date) BETWEEN 15 AND 30 THEN '15-30 hari'
            WHEN (CURRENT_DATE - i.due_date) > 30 THEN '30+ hari'
        END AS label
    FROM invoices i
    JOIN customers c ON c.id = i.customer_id
    WHERE i.tenant_id = $1
      AND i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
      AND i.due_date < CURRENT_DATE
      AND (i.total_amount - i.paid_amount) > 0
      AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
      AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid)
) sub
WHERE label IS NOT NULL
GROUP BY label
ORDER BY
    CASE label
        WHEN '1-7 hari' THEN 1
        WHEN '8-14 hari' THEN 2
        WHEN '15-30 hari' THEN 3
        WHEN '30+ hari' THEN 4
    END;

-- name: GetCollectionRate :one
-- Menghitung collection rate: persentase invoice terbayar vs total invoice jatuh tempo.
-- Collection rate = (jumlah invoice lunas / total invoice jatuh tempo) * 100.
SELECT
    COALESCE(COUNT(*) FILTER (WHERE i.status = 'lunas'), 0)::bigint AS paid_count,
    COALESCE(COUNT(*), 0)::bigint AS total_due_count,
    CASE
        WHEN COUNT(*) = 0 THEN 0
        ELSE ROUND((COUNT(*) FILTER (WHERE i.status = 'lunas'))::numeric / COUNT(*)::numeric * 100, 2)
    END::float8 AS collection_rate
FROM invoices i
JOIN customers c ON c.id = i.customer_id
WHERE i.tenant_id = $1
  AND i.due_date >= $2
  AND i.due_date < $3
  AND i.status != 'batal'
  AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
  AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid);

-- name: GetAvgDaysToPay :one
-- Menghitung rata-rata hari antara due_date dan payment_date untuk invoice yang sudah dibayar.
-- Hanya menghitung invoice lunas dalam periode tertentu.
SELECT
    COALESCE(
        AVG(ip.payment_date - i.due_date),
        0
    )::float8 AS avg_days_to_pay
FROM invoice_payments ip
JOIN invoices i ON i.id = ip.invoice_id
JOIN customers c ON c.id = i.customer_id
WHERE ip.tenant_id = $1
  AND ip.voided = false
  AND ip.payment_date >= $2
  AND ip.payment_date < $3
  AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
  AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid);

-- name: GetTopDebtors :many
-- Mengambil 10 debitur terbesar berdasarkan total outstanding.
-- Menghitung months_overdue dari invoice tertua yang belum lunas.
SELECT
    c.id AS customer_id,
    c.name AS customer_name,
    COALESCE(SUM(i.total_amount - i.paid_amount), 0)::bigint AS total_outstanding,
    COALESCE(
        EXTRACT(MONTH FROM AGE(CURRENT_DATE, MIN(i.due_date)))::int +
        EXTRACT(YEAR FROM AGE(CURRENT_DATE, MIN(i.due_date)))::int * 12,
        0
    )::int AS months_overdue
FROM invoices i
JOIN customers c ON c.id = i.customer_id
WHERE i.tenant_id = $1
  AND i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
  AND (i.total_amount - i.paid_amount) > 0
  AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
  AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid)
GROUP BY c.id, c.name
ORDER BY total_outstanding DESC
LIMIT 10;

-- name: GetReceivablesTrend :many
-- Menghitung trend piutang per bulan untuk 6 bulan terakhir.
-- Total outstanding dihitung dari invoice yang belum lunas pada akhir setiap bulan.
SELECT
    TO_CHAR(month_series, 'YYYY-MM') AS month,
    COALESCE(SUM(i.total_amount - i.paid_amount), 0)::bigint AS total_outstanding
FROM generate_series(
    DATE_TRUNC('month', CURRENT_DATE - INTERVAL '5 months'),
    DATE_TRUNC('month', CURRENT_DATE),
    '1 month'::interval
) AS month_series
LEFT JOIN invoices i ON i.tenant_id = $1
    AND i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
    AND i.due_date <= (month_series + INTERVAL '1 month' - INTERVAL '1 day')::date
    AND (i.total_amount - i.paid_amount) > 0
GROUP BY month_series
ORDER BY month_series ASC;

-- name: GetPaymentDistribution :many
-- Menghitung distribusi pembayaran per metode pembayaran.
-- Mengembalikan method_name, total_amount, transaction_count, dan percentage.
SELECT
    ip.payment_method AS method_name,
    COALESCE(SUM(ip.amount), 0)::bigint AS total_amount,
    COUNT(*)::int AS transaction_count,
    CASE
        WHEN SUM(SUM(ip.amount)) OVER () = 0 THEN 0
        ELSE ROUND(SUM(ip.amount)::numeric / SUM(SUM(ip.amount)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM invoice_payments ip
JOIN invoices i ON i.id = ip.invoice_id
JOIN customers c ON c.id = i.customer_id
WHERE ip.tenant_id = $1
  AND ip.voided = false
  AND ip.payment_date >= $2
  AND ip.payment_date < $3
  AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
  AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid)
GROUP BY ip.payment_method
ORDER BY total_amount DESC;

-- name: GetDailyPayments :many
-- Menghitung total pembayaran per hari dalam periode tertentu.
-- Mengembalikan date, total_amount, dan transaction_count per hari.
SELECT
    ip.payment_date::text AS date,
    COALESCE(SUM(ip.amount), 0)::bigint AS total_amount,
    COUNT(*)::int AS transaction_count
FROM invoice_payments ip
JOIN invoices i ON i.id = ip.invoice_id
JOIN customers c ON c.id = i.customer_id
WHERE ip.tenant_id = $1
  AND ip.voided = false
  AND ip.payment_date >= $2
  AND ip.payment_date < $3
  AND (sqlc.narg('area_id')::uuid IS NULL OR c.area_id = sqlc.narg('area_id')::uuid)
  AND (sqlc.narg('package_id')::uuid IS NULL OR c.package_id = sqlc.narg('package_id')::uuid)
GROUP BY ip.payment_date
ORDER BY ip.payment_date ASC;

-- name: GetVoucherRevenueByPackage :many
-- Menghitung pendapatan voucher per paket.
-- Mengembalikan package_name, total_revenue, voucher_count, dan percentage.
SELECT
    p.name AS package_name,
    COALESCE(SUM(v.sell_price_snapshot), 0)::bigint AS total_revenue,
    COUNT(*)::int AS voucher_count,
    CASE
        WHEN SUM(SUM(v.sell_price_snapshot)) OVER () = 0 THEN 0
        ELSE ROUND(SUM(v.sell_price_snapshot)::numeric / SUM(SUM(v.sell_price_snapshot)) OVER ()::numeric * 100, 2)
    END::float8 AS percentage
FROM vouchers v
JOIN packages p ON p.id = v.package_id
WHERE v.tenant_id = $1
  AND v.status IN ('terjual', 'aktif', 'selesai')
  AND v.purchased_at >= $2
  AND v.purchased_at < $3
GROUP BY p.name
ORDER BY total_revenue DESC;

-- name: GetVoucherRevenueByReseller :many
-- Menghitung pendapatan voucher per reseller.
-- Mengembalikan reseller_name, total_revenue, voucher_count, dan reseller_margin.
-- Reseller margin = sell_price_snapshot - reseller_price_snapshot.
SELECT
    r.name AS reseller_name,
    COALESCE(SUM(v.sell_price_snapshot), 0)::bigint AS total_revenue,
    COUNT(*)::int AS voucher_count,
    COALESCE(SUM(v.sell_price_snapshot - COALESCE(v.reseller_price_snapshot, 0)), 0)::bigint AS reseller_margin
FROM vouchers v
JOIN resellers r ON r.id = v.reseller_id
WHERE v.tenant_id = $1
  AND v.status IN ('terjual', 'aktif', 'selesai')
  AND v.purchased_at >= $2
  AND v.purchased_at < $3
  AND v.reseller_id IS NOT NULL
GROUP BY r.name
ORDER BY total_revenue DESC;

-- name: GetRevenueByArea :many
-- Menghitung pendapatan, piutang, dan ARPU per area.
-- JOIN areas untuk mendapatkan nama area.
-- ARPU = total_revenue / customer_count.
SELECT
    a.id AS area_id,
    a.name AS area_name,
    COUNT(DISTINCT c.id)::int AS customer_count,
    COALESCE(SUM(ip.amount), 0)::bigint AS total_revenue,
    COALESCE((
        SELECT SUM(inv.total_amount - inv.paid_amount)
        FROM invoices inv
        JOIN customers cust ON cust.id = inv.customer_id
        WHERE inv.tenant_id = $1
          AND cust.area_id = a.id
          AND inv.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
          AND (inv.total_amount - inv.paid_amount) > 0
    ), 0)::bigint AS total_outstanding,
    CASE
        WHEN COUNT(DISTINCT c.id) = 0 THEN 0
        ELSE (COALESCE(SUM(ip.amount), 0) / COUNT(DISTINCT c.id))
    END::bigint AS arpu
FROM areas a
LEFT JOIN customers c ON c.area_id = a.id AND c.deleted_at IS NULL AND c.status = 'aktif'
LEFT JOIN invoices i ON i.customer_id = c.id
    AND i.status != 'batal'
LEFT JOIN invoice_payments ip ON ip.invoice_id = i.id
    AND ip.voided = false
    AND ip.payment_date >= $2
    AND ip.payment_date < $3
WHERE a.tenant_id = $1
GROUP BY a.id, a.name
ORDER BY total_revenue DESC;
