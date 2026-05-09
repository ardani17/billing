-- Kueri SQL untuk operasi pada tabel invoice_audit_logs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel invoice_audit_logs bersifat append-only (hanya SELECT dan INSERT).
-- Dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateInvoiceAuditLog :one
-- Membuat satu entri audit log invoice dan mengembalikan semua kolom.
INSERT INTO invoice_audit_logs (
    tenant_id, invoice_id, action, actor_id, actor_name, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListAuditLogsByInvoice :many
-- Mengambil semua audit log untuk invoice tertentu, diurutkan berdasarkan created_at.
SELECT *
FROM invoice_audit_logs
WHERE invoice_id = $1
ORDER BY created_at ASC;
