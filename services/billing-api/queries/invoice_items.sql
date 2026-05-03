-- Query SQL untuk operasi CRUD tabel invoice_items.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel invoice_items dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: BulkCreateInvoiceItems :copyfrom
-- Bulk insert item invoice menggunakan PostgreSQL COPY protocol.
INSERT INTO invoice_items (
    tenant_id, invoice_id, item_type, description,
    quantity, unit_price, amount, sort_order, metadata
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9
);

-- name: ListInvoiceItemsByInvoice :many
-- Mengambil semua item untuk invoice tertentu, diurutkan berdasarkan sort_order.
SELECT *
FROM invoice_items
WHERE invoice_id = $1
ORDER BY sort_order ASC;

-- name: DeleteInvoiceItemsByInvoice :exec
-- Menghapus semua item untuk invoice tertentu (digunakan saat edit invoice).
DELETE FROM invoice_items
WHERE invoice_id = $1;
