-- Kueri SQL untuk operasi invoice terkait modul isolir.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel invoices dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Kueri ini digunakan oleh IsolirUsecase untuk mendeteksi invoice terlambat dan outstanding.

-- name: FindOverdueForIsolir :many
-- Mengambil invoice yang sudah melewati grace period untuk proses auto-isolir.
-- Hanya untuk pelanggan dengan status aktif (eligible untuk isolir).
-- Mengembalikan field invoice beserta customer_id.
SELECT i.id, i.tenant_id, i.customer_id, i.invoice_number,
    i.period_month, i.period_year, i.due_date,
    i.subtotal, i.tax_amount, i.penalty_amount, i.discount_amount,
    i.credit_applied, i.total_amount, i.paid_amount,
    i.status, i.notes, i.is_prepaid, i.prepaid_months,
    i.version, i.created_at, i.updated_at
FROM invoices i
JOIN customers c ON c.id = i.customer_id
WHERE i.tenant_id = $1
  AND i.status IN ('belum_bayar', 'terlambat')
  AND sqlc.arg('current_date')::date > i.due_date + sqlc.arg('grace_period_days')::integer
  AND c.status = 'aktif'
  AND c.deleted_at IS NULL;

-- name: FindOverdueForSuspend :many
-- Mengambil invoice yang sudah melewati suspend_days untuk proses suspend.
-- Hanya untuk pelanggan dengan status isolir (eligible untuk suspend).
SELECT i.id, i.tenant_id, i.customer_id, i.invoice_number,
    i.period_month, i.period_year, i.due_date,
    i.subtotal, i.tax_amount, i.penalty_amount, i.discount_amount,
    i.credit_applied, i.total_amount, i.paid_amount,
    i.status, i.notes, i.is_prepaid, i.prepaid_months,
    i.version, i.created_at, i.updated_at
FROM invoices i
JOIN customers c ON c.id = i.customer_id
WHERE i.tenant_id = $1
  AND i.status IN ('belum_bayar', 'terlambat')
  AND sqlc.arg('current_date')::date > i.due_date + sqlc.arg('suspend_days')::integer
  AND c.status = 'isolir'
  AND c.deleted_at IS NULL;

-- name: HasOutstandingInvoices :one
-- Mengecek apakah customer masih punya invoice yang belum lunas.
-- Invoice outstanding = status bukan lunas dan bukan batal.
SELECT EXISTS(
    SELECT 1 FROM invoices
    WHERE customer_id = sqlc.arg('customer_id')
      AND status NOT IN ('lunas', 'batal')
) AS exists;

-- name: SumOutstandingAmount :one
-- Menghitung total tagihan outstanding untuk customer tertentu.
-- Mengembalikan 0 jika tidak ada invoice outstanding.
SELECT COALESCE(SUM(total_amount), 0)::bigint AS total
FROM invoices
WHERE customer_id = sqlc.arg('customer_id')
  AND status NOT IN ('lunas', 'batal');

-- name: CountOutstandingInvoices :one
-- Menghitung jumlah invoice outstanding untuk customer tertentu.
SELECT COUNT(*)
FROM invoices
WHERE customer_id = sqlc.arg('customer_id')
  AND status NOT IN ('lunas', 'batal');
