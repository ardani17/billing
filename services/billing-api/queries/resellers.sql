-- Query SQL untuk operasi CRUD tabel resellers.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel resellers dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Query List dibangun secara dinamis di repository (sama seperti customer/package).

-- name: CreateReseller :one
-- Membuat reseller baru dan mengembalikan semua kolom.
INSERT INTO resellers (
    tenant_id, name, phone, email, address,
    password_hash, balance, daily_purchase_limit, status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
)
RETURNING *;

-- name: GetResellerByID :one
-- Mengambil reseller berdasarkan ID beserta jumlah voucher terjual (computed field).
-- total_vouchers_sold menghitung voucher yang statusnya bukan 'tersedia' dan bukan 'void'.
SELECT r.*,
    (SELECT COUNT(*) FROM vouchers v
     WHERE v.reseller_id = r.id
       AND v.status NOT IN ('tersedia', 'void')) AS total_vouchers_sold
FROM resellers r
WHERE r.id = $1;

-- name: GetResellerByPhone :one
-- Mengambil reseller berdasarkan tenant_id dan phone (untuk login).
SELECT *
FROM resellers
WHERE tenant_id = $1 AND phone = $2;

-- name: UpdateReseller :one
-- Memperbarui data reseller (name, phone, email, address, daily_purchase_limit).
UPDATE resellers SET
    name = $2,
    phone = $3,
    email = $4,
    address = $5,
    daily_purchase_limit = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateResellerStatus :one
-- Memperbarui status reseller (aktif/suspended/nonaktif).
UPDATE resellers SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateResellerPasswordHash :exec
-- Memperbarui password hash reseller (untuk reset password).
UPDATE resellers SET
    password_hash = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateResellerLastLogin :exec
-- Memperbarui timestamp last_login reseller ke waktu sekarang.
UPDATE resellers SET
    last_login = NOW()
WHERE id = $1;

-- name: UpdateResellerBalance :exec
-- Memperbarui saldo reseller (digunakan dalam transaksi atomik).
UPDATE resellers SET
    balance = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetResellerForUpdate :one
-- Mengambil reseller dengan row lock (SELECT ... FOR UPDATE).
-- Digunakan dalam transaksi untuk operasi balance atomik.
SELECT *
FROM resellers
WHERE id = $1
FOR UPDATE;

-- name: GetResellerByPhoneGlobal :one
-- Mengambil reseller berdasarkan phone saja (lintas tenant, untuk login).
SELECT * FROM resellers WHERE phone = $1;

-- name: ResellerPhoneExists :one
-- Mengecek apakah nomor telepon sudah ada di tenant (exclude ID tertentu).
SELECT EXISTS(
    SELECT 1 FROM resellers
    WHERE tenant_id = $1 AND phone = $2 AND id != $3
) AS exists;
