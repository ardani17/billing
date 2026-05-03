-- Query SQL untuk operasi CRUD tabel payment_links dan payment_link_invoices.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel payment_links dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Mendukung multi-invoice payment link melalui junction table payment_link_invoices.

-- name: CreatePaymentLink :one
-- Membuat payment link baru dan mengembalikan semua kolom.
INSERT INTO payment_links (
    tenant_id, customer_id, gateway_provider, gateway_config_id,
    external_id, payment_url, amount, status, expires_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetPaymentLinkByID :one
-- Mengambil payment link berdasarkan ID (tenant-scoped via RLS).
SELECT *
FROM payment_links
WHERE id = $1;

-- name: GetPaymentLinkByExternalID :one
-- Mengambil payment link berdasarkan external_id dari gateway.
SELECT *
FROM payment_links
WHERE external_id = $1;

-- name: GetActivePaymentLinkByCustomer :one
-- Mengambil payment link aktif (status='active') untuk customer tertentu.
SELECT *
FROM payment_links
WHERE customer_id = $1 AND status = 'active';

-- name: UpdatePaymentLinkStatus :exec
-- Memperbarui status payment link dan updated_at.
UPDATE payment_links SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdatePaymentLinkPaid :exec
-- Memperbarui payment link menjadi status 'paid' beserta metode pembayaran dan waktu bayar.
UPDATE payment_links SET
    status = 'paid',
    paid_method = $2,
    paid_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ListPaymentLinksByInvoice :many
-- Mengambil semua payment links untuk invoice tertentu melalui junction table.
SELECT pl.*
FROM payment_links pl
JOIN payment_link_invoices pli ON pl.id = pli.payment_link_id
WHERE pli.invoice_id = $1
ORDER BY pl.created_at DESC;

-- name: FindExpiredPaymentLinks :many
-- Mengambil payment links aktif yang sudah melewati waktu expires_at (untuk background job expiry).
SELECT *
FROM payment_links
WHERE status = 'active' AND expires_at < NOW()
LIMIT $1;

-- name: ExpirePaymentLinkByID :exec
-- Mengubah status payment link menjadi 'expired' jika masih aktif.
UPDATE payment_links SET
    status = 'expired',
    updated_at = NOW()
WHERE id = $1 AND status = 'active';

-- name: CreatePaymentLinkInvoice :exec
-- Membuat relasi antara payment link dan invoice di junction table.
INSERT INTO payment_link_invoices (
    payment_link_id, invoice_id
) VALUES (
    $1, $2
);

-- name: GetInvoiceIDsByPaymentLinkID :many
-- Mengambil daftar invoice_id yang terkait dengan payment link tertentu.
SELECT invoice_id
FROM payment_link_invoices
WHERE payment_link_id = $1;
