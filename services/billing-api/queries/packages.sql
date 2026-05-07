-- Query SQL untuk operasi CRUD tabel packages.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel packages dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreatePackage :one
-- Membuat paket baru dan mengembalikan semua kolom.
INSERT INTO packages (
    tenant_id, type, name, description, is_active,
    download_mbps, upload_mbps, bandwidth_type,
    burst_download_mbps, burst_upload_mbps, burst_threshold_mbps, burst_time_seconds,
    quota_type, quota_mb, quota_action, throttle_mbps,
    monthly_price, installation_fee, sell_price, reseller_price,
    duration_value, duration_unit, shared_users,
    mikrotik_profile_name, address_pool, parent_queue, hotspot_profile_name
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11, $12,
    $13, $14, $15, $16,
    $17, $18, $19, $20,
    $21, $22, $23,
    $24, $25, $26, $27
)
RETURNING *;

-- name: GetPackageByID :one
-- Mengambil paket berdasarkan ID beserta jumlah pelanggan yang masih mereferensikan paket.
SELECT p.*,
    (SELECT COUNT(*) FROM customers c
     WHERE c.package_id = p.id) AS customer_count,
    (SELECT COUNT(*) FROM customers c
     WHERE c.package_id = p.id AND c.deleted_at IS NULL) AS customer_active_count,
    (SELECT COUNT(*) FROM customers c
     WHERE c.package_id = p.id AND c.deleted_at IS NOT NULL) AS customer_deleted_count
FROM packages p
WHERE p.id = $1;

-- name: UpdatePackage :one
-- Memperbarui semua kolom mutable pada paket (kecuali id, tenant_id, type).
UPDATE packages SET
    name = $2,
    description = $3,
    download_mbps = $4,
    upload_mbps = $5,
    bandwidth_type = $6,
    burst_download_mbps = $7,
    burst_upload_mbps = $8,
    burst_threshold_mbps = $9,
    burst_time_seconds = $10,
    quota_type = $11,
    quota_mb = $12,
    quota_action = $13,
    throttle_mbps = $14,
    monthly_price = $15,
    installation_fee = $16,
    sell_price = $17,
    reseller_price = $18,
    duration_value = $19,
    duration_unit = $20,
    shared_users = $21,
    mikrotik_profile_name = $22,
    address_pool = $23,
    parent_queue = $24,
    hotspot_profile_name = $25,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePackage :exec
-- Menghapus paket secara permanen (hard delete).
DELETE FROM packages WHERE id = $1;

-- name: UpdatePackageIsActive :one
-- Memperbarui status aktif paket (aktivasi/deaktivasi).
UPDATE packages SET is_active = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: PackageNameExists :one
-- Mengecek apakah nama paket sudah ada di tenant (exclude ID tertentu).
SELECT EXISTS(
    SELECT 1 FROM packages
    WHERE tenant_id = $1 AND name = $2 AND id != $3
) AS exists;

-- name: PackageCustomerCount :one
-- Menghitung jumlah pelanggan yang masih mereferensikan paket.
SELECT COUNT(*) FROM customers
WHERE package_id = $1;

-- name: ListPackageNamesByPrefix :many
-- Mengambil daftar nama paket berdasarkan prefix (untuk generate nama duplikat).
SELECT name FROM packages
WHERE tenant_id = $1 AND name LIKE $2
ORDER BY name;
