-- Query SQL untuk operasi CRUD tabel invoices.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel invoices dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Query List dibangun secara dinamis di repository (sama seperti customer/package) — tidak di sqlc.

-- name: CreateInvoice :one
-- Membuat invoice baru dan mengembalikan semua kolom.
INSERT INTO invoices (
    tenant_id, customer_id, invoice_number, period_month, period_year,
    due_date, subtotal, tax_amount, penalty_amount, discount_amount,
    credit_applied, total_amount, paid_amount, status, notes,
    is_prepaid, prepaid_months, version
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15,
    $16, $17, $18
)
RETURNING *;

-- name: GetInvoiceByID :one
-- Mengambil invoice berdasarkan ID beserta data customer (name, customer_id_seq, phone, address)
-- dan nama paket (package_name) melalui JOIN.
SELECT i.*,
    c.name AS customer_name,
    c.customer_id_seq AS customer_id_seq,
    c.phone AS customer_phone,
    c.address AS customer_address,
    p.name AS package_name
FROM invoices i
JOIN customers c ON c.id = i.customer_id
LEFT JOIN packages p ON p.id = c.package_id
WHERE i.id = $1;

-- name: UpdateInvoice :one
-- Memperbarui data invoice (due_date, subtotal, tax, penalty, discount, credit, total, notes)
-- dan increment version untuk optimistic locking.
UPDATE invoices SET
    due_date = $2,
    subtotal = $3,
    tax_amount = $4,
    penalty_amount = $5,
    discount_amount = $6,
    credit_applied = $7,
    total_amount = $8,
    notes = $9,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInvoiceStatus :one
-- Memperbarui status invoice dengan optimistic locking via version.
-- Hanya berhasil jika version cocok (mencegah race condition).
UPDATE invoices SET
    status = $2,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $3
RETURNING *;

-- name: UpdateInvoicePaidAmount :one
-- Memperbarui jumlah yang sudah dibayar dengan optimistic locking via version.
-- Digunakan saat mencatat pembayaran.
UPDATE invoices SET
    paid_amount = $2,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1 AND version = $3
RETURNING *;

-- name: ExistsForPeriod :one
-- Mengecek apakah invoice sudah ada untuk customer dan periode tertentu (idempotency check).
SELECT EXISTS(
    SELECT 1 FROM invoices
    WHERE customer_id = $1 AND period_month = $2 AND period_year = $3
) AS exists;

-- name: ExistsForPeriodPrepaid :one
-- Mengecek apakah invoice prepaid sudah mencakup periode tertentu.
-- Invoice prepaid mencakup beberapa bulan mulai dari period_month/period_year.
SELECT EXISTS(
    SELECT 1 FROM invoices
    WHERE customer_id = $1
      AND is_prepaid = TRUE
      AND (
          (period_year * 12 + period_month) + COALESCE(prepaid_months, 0) - 1
          >= ($3 * 12 + $2)
      )
      AND (period_year * 12 + period_month) <= ($3 * 12 + $2)
      AND status != 'batal'
) AS exists;

-- name: FindOverdueInvoices :many
-- Mengambil semua invoice yang sudah melewati jatuh tempo dan masih berstatus belum_bayar.
-- Digunakan oleh cron job untuk update status ke terlambat.
SELECT *
FROM invoices
WHERE status = 'belum_bayar' AND due_date < $1;

-- name: GetInvoiceSummary :many
-- Mengambil ringkasan invoice per status (jumlah dan total nominal) untuk dashboard.
-- Digunakan untuk menampilkan statistik invoice.
SELECT status,
    COUNT(*) AS count,
    COALESCE(SUM(total_amount), 0) AS total_amount
FROM invoices
WHERE tenant_id = $1
  AND ($2::integer IS NULL OR period_month = $2)
  AND ($3::integer IS NULL OR period_year = $3)
GROUP BY status;

-- name: GetInvoicesByIDs :many
-- Mengambil beberapa invoice berdasarkan array of IDs.
-- Digunakan untuk bulk actions (reminder, cancel, PDF).
SELECT *
FROM invoices
WHERE id = ANY($1::uuid[]);

-- name: FindOpenInvoicesByCustomer :many
-- Mengambil semua invoice terbuka untuk pelanggan tertentu, diurutkan berdasarkan due_date ASC.
-- Invoice terbuka = status belum_bayar, terlambat, atau bayar_sebagian.
SELECT *
FROM invoices
WHERE customer_id = $1
  AND status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
ORDER BY due_date ASC;

-- name: FindOpenInvoicesByCustomerForUpdate :many
-- Sama seperti FindOpenInvoicesByCustomer tetapi dengan FOR UPDATE untuk row locking.
-- Harus dipanggil dalam transaksi untuk keamanan konkurensi.
SELECT *
FROM invoices
WHERE customer_id = $1
  AND status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')
ORDER BY due_date ASC
FOR UPDATE;

-- name: GetInvoicesByIDsForUpdate :many
-- Mengambil beberapa invoice berdasarkan array of IDs dengan FOR UPDATE untuk row locking.
-- Harus dipanggil dalam transaksi untuk keamanan konkurensi.
SELECT *
FROM invoices
WHERE id = ANY($1::uuid[])
FOR UPDATE;
