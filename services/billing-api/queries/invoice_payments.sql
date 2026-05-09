-- Kueri SQL untuk operasi CRUD tabel invoice_payments.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel invoice_payments dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateInvoicePayment :one
-- Membuat catatan pembayaran baru dan mengembalikan semua kolom.
INSERT INTO invoice_payments (
    tenant_id, invoice_id, amount, payment_method, payment_date,
    reference_number, notes, recorded_by_id, recorded_by_name
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
)
RETURNING *;

-- name: ListPaymentsByInvoice :many
-- Mengambil semua pembayaran non-void untuk invoice tertentu, diurutkan berdasarkan created_at.
SELECT *
FROM invoice_payments
WHERE invoice_id = $1 AND voided = false
ORDER BY created_at ASC;

-- name: VoidPayment :exec
-- Menandai pembayaran sebagai void dengan alasan dan waktu void.
UPDATE invoice_payments SET
    voided = true,
    voided_at = NOW(),
    voided_by = $2,
    void_reason = $3
WHERE id = $1;

-- name: GetPaymentByID :one
-- Mengambil satu pembayaran berdasarkan ID beserta nomor invoice melalui JOIN.
SELECT ip.*,
    i.invoice_number AS invoice_number
FROM invoice_payments ip
JOIN invoices i ON i.id = ip.invoice_id
WHERE ip.id = $1;

-- name: FindDuplicatePayment :one
-- Mengecek apakah ada pembayaran duplikat dalam 24 jam terakhir.
-- Duplikat didefinisikan sebagai pembayaran dengan customer_id, nominal, payment_method,
-- dan payment_date yang sama, belum di-void.
SELECT EXISTS(
    SELECT 1 FROM invoice_payments ip
    JOIN invoices i ON i.id = ip.invoice_id
    WHERE i.customer_id = $1
      AND ip.amount = $2
      AND ip.payment_method = $3
      AND ip.payment_date = $4
      AND ip.voided = false
      AND ip.created_at >= NOW() - INTERVAL '24 hours'
) AS exists;

-- name: GetPaymentSummaryToday :one
-- Mengambil ringkasan pembayaran hari ini (jumlah dan total nominal).
-- Menggunakan parameter timezone untuk konversi ke tanggal lokal tenant.
SELECT
    COUNT(*)::bigint AS count,
    COALESCE(SUM(amount), 0)::bigint AS total_amount
FROM invoice_payments
WHERE voided = false
  AND (created_at AT TIME ZONE $1)::date = (NOW() AT TIME ZONE $1)::date;

-- name: GetPaymentSummaryMonth :one
-- Mengambil ringkasan pembayaran untuk bulan dan tahun tertentu.
SELECT
    COUNT(*)::bigint AS count,
    COALESCE(SUM(amount), 0)::bigint AS total_amount
FROM invoice_payments
WHERE voided = false
  AND EXTRACT(MONTH FROM payment_date) = $1
  AND EXTRACT(YEAR FROM payment_date) = $2;

-- name: GetPaymentSummaryByMethod :many
-- Mengambil ringkasan pembayaran per metode pembayaran untuk bulan dan tahun tertentu.
SELECT
    payment_method,
    COUNT(*)::bigint AS count,
    COALESCE(SUM(amount), 0)::bigint AS total_amount
FROM invoice_payments
WHERE voided = false
  AND EXTRACT(MONTH FROM payment_date) = $1
  AND EXTRACT(YEAR FROM payment_date) = $2
GROUP BY payment_method;
