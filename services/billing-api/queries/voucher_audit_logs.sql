-- Query SQL untuk operasi tabel voucher_audit_logs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel voucher_audit_logs bersifat append-only — hanya INSERT dan SELECT yang diizinkan.
-- Tabel dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreateVoucherAuditLog :one
-- Membuat catatan audit log voucher baru dan mengembalikan semua kolom.
INSERT INTO voucher_audit_logs (
    tenant_id, voucher_id, action, actor_id, actor_name, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListVoucherAuditLogsByVoucher :many
-- Mengambil semua audit log untuk voucher tertentu, diurutkan berdasarkan waktu pembuatan (ASC).
SELECT *
FROM voucher_audit_logs
WHERE voucher_id = $1
ORDER BY created_at ASC;
