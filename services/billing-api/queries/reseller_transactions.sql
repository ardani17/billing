-- Query SQL untuk operasi tabel reseller_transactions.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel reseller_transactions mencatat semua transaksi keuangan reseller (deposit, purchase, refund, withdraw).
-- Tabel dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreateResellerTransaction :one
-- Membuat catatan transaksi reseller baru dan mengembalikan semua kolom.
INSERT INTO reseller_transactions (
    tenant_id, reseller_id, type, amount, balance_before, balance_after, reference_id, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ListResellerTransactions :many
-- Mengambil daftar transaksi reseller dengan pagination, diurutkan berdasarkan waktu terbaru.
SELECT *
FROM reseller_transactions
WHERE tenant_id = $1 AND reseller_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountResellerTransactions :one
-- Menghitung total transaksi reseller (untuk pagination metadata).
SELECT COUNT(*)
FROM reseller_transactions
WHERE tenant_id = $1 AND reseller_id = $2;

-- name: ListResellerDeposits :many
-- Mengambil daftar transaksi deposit reseller dengan pagination, diurutkan berdasarkan waktu terbaru.
SELECT *
FROM reseller_transactions
WHERE tenant_id = $1 AND reseller_id = $2 AND type = 'deposit'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountResellerDeposits :one
-- Menghitung total transaksi deposit reseller (untuk pagination metadata).
SELECT COUNT(*)
FROM reseller_transactions
WHERE tenant_id = $1 AND reseller_id = $2 AND type = 'deposit';
