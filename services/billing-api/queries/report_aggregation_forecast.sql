-- Kueri SQL untuk data historis yang digunakan oleh forecasting engine.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Mengambil data 6 bulan terakhir untuk kalkulasi linear regression.
-- Tabel dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: GetMonthlyRevenueHistory :many
-- Mengambil data historis pendapatan bulanan untuk N bulan terakhir.
-- Digunakan sebagai input linear regression untuk proyeksi pendapatan.
-- Mengembalikan month_index (0-based) dan total_revenue per bulan.
SELECT
    ROW_NUMBER() OVER (ORDER BY month_series ASC)::float8 - 1 AS x,
    COALESCE(SUM(ip.amount), 0)::float8 AS y
FROM generate_series(
    DATE_TRUNC('month', CURRENT_DATE - (($1::int - 1) || ' months')::interval),
    DATE_TRUNC('month', CURRENT_DATE),
    '1 month'::interval
) AS month_series
LEFT JOIN invoice_payments ip ON ip.tenant_id = $2
    AND ip.voided = false
    AND ip.payment_date >= month_series::date
    AND ip.payment_date < (month_series + INTERVAL '1 month')::date
GROUP BY month_series
ORDER BY month_series ASC;

-- name: GetMonthlyCustomerHistory :many
-- Mengambil data historis jumlah pelanggan aktif bulanan untuk N bulan terakhir.
-- Digunakan sebagai input linear regression untuk proyeksi pertumbuhan pelanggan.
-- Mengembalikan month_index (0-based) dan total_active_customers per bulan.
SELECT
    ROW_NUMBER() OVER (ORDER BY month_series ASC)::float8 - 1 AS x,
    COALESCE((
        SELECT COUNT(*)
        FROM customers c
        WHERE c.tenant_id = $2
          AND c.status = 'aktif'
          AND c.deleted_at IS NULL
          AND c.activation_date <= (month_series + INTERVAL '1 month' - INTERVAL '1 day')::date
    ), 0)::float8 AS y
FROM generate_series(
    DATE_TRUNC('month', CURRENT_DATE - (($1::int - 1) || ' months')::interval),
    DATE_TRUNC('month', CURRENT_DATE),
    '1 month'::interval
) AS month_series
ORDER BY month_series ASC;

-- name: GetMonthlyReceivablesHistory :many
-- Mengambil data historis piutang bulanan untuk N bulan terakhir.
-- Digunakan sebagai input linear regression untuk proyeksi piutang.
-- Mengembalikan month_index (0-based) dan total_outstanding per bulan.
SELECT
    ROW_NUMBER() OVER (ORDER BY month_series ASC)::float8 - 1 AS x,
    COALESCE((
        SELECT SUM(i.total_amount - i.paid_amount)
        FROM invoices i
        WHERE i.tenant_id = $2
          AND i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
          AND i.due_date <= (month_series + INTERVAL '1 month' - INTERVAL '1 day')::date
          AND (i.total_amount - i.paid_amount) > 0
    ), 0)::float8 AS y
FROM generate_series(
    DATE_TRUNC('month', CURRENT_DATE - (($1::int - 1) || ' months')::interval),
    DATE_TRUNC('month', CURRENT_DATE),
    '1 month'::interval
) AS month_series
ORDER BY month_series ASC;
