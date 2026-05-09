-- Kueri SQL untuk operasi CRUD tabel vouchers.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel vouchers dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: BulkCreateVouchers :copyfrom
-- Bulk insert voucher baru menggunakan PostgreSQL COPY protocol.
INSERT INTO vouchers (
    tenant_id, code, package_id, status
) VALUES (
    $1, $2, $3, $4
);

-- name: GetVoucherByID :one
-- Mengambil voucher berdasarkan ID beserta nama paket dan nama reseller (joined).
-- LEFT JOIN pada resellers karena reseller_id bisa NULL (voucher belum terjual).
SELECT v.*,
    p.name AS package_name,
    r.name AS reseller_name
FROM vouchers v
JOIN packages p ON p.id = v.package_id
LEFT JOIN resellers r ON r.id = v.reseller_id
WHERE v.id = $1;

-- name: GetVoucherByCode :one
-- Mengambil voucher berdasarkan tenant_id dan code (untuk aktivasi/lookup).
SELECT *
FROM vouchers
WHERE tenant_id = $1 AND code = $2;

-- name: UpdateVoucherStatus :one
-- Memperbarui status voucher dan updated_at.
UPDATE vouchers SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateVoucherVoid :one
-- Memperbarui voucher menjadi void: atur status, voided_at, dan updated_at.
UPDATE vouchers SET
    status = 'void',
    voided_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AssignVoucherToReseller :one
-- Meng-assign voucher ke reseller saat pembelian (atur snapshot, purchased_at, expires_at).
UPDATE vouchers SET
    reseller_id = $2,
    status = 'terjual',
    sell_price_snapshot = $3,
    reseller_price_snapshot = $4,
    purchased_at = $5,
    expires_at = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AdminAssignVoucher :one
-- Meng-assign voucher ke reseller oleh admin (tanpa snapshot harga).
UPDATE vouchers SET
    reseller_id = $2,
    status = 'terjual',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetExpiredVouchers :many
-- Mengambil voucher terjual yang sudah melewati expires_at (untuk cron expiry).
SELECT *
FROM vouchers
WHERE status = 'terjual' AND expires_at < NOW()
LIMIT $1;

-- name: VoucherCodeExists :one
-- Mengecek apakah kode voucher sudah ada di tenant (untuk collision cek).
SELECT EXISTS(
    SELECT 1 FROM vouchers
    WHERE tenant_id = $1 AND code = $2
) AS exists;

-- name: GetVouchersByIDs :many
-- Mengambil beberapa voucher berdasarkan array of IDs.
SELECT *
FROM vouchers
WHERE id = ANY($1::uuid[]);

-- name: CountVouchersByResellerAndStatus :one
-- Menghitung jumlah voucher per reseller dan array status.
SELECT COUNT(*)
FROM vouchers
WHERE reseller_id = $1 AND status = ANY(sqlc.arg('status')::varchar[]);

-- name: CountVouchersSoldToday :one
-- Menghitung jumlah voucher yang dibeli reseller hari ini (berdasarkan purchased_at).
SELECT COUNT(*)
FROM vouchers
WHERE reseller_id = $1
  AND purchased_at >= CURRENT_DATE
  AND purchased_at < CURRENT_DATE + INTERVAL '1 day';

-- name: UpdateVoucherExpired :one
-- Memperbarui voucher menjadi expired: atur status dan updated_at.
UPDATE vouchers SET
    status = 'expired',
    updated_at = NOW()
WHERE id = $1
RETURNING *;
