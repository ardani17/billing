-- Query SQL untuk mengambil data pelanggan dan tenant dari tabel shared (milik billing-api).
-- Digunakan oleh delivery pipeline untuk resolusi variabel template notifikasi.
-- Tabel customers dan tenants berada di database yang sama, dibuat oleh migrasi billing-api.

-- name: GetCustomerByID :one
-- Mengambil data pelanggan berdasarkan ID untuk keperluan substitusi variabel template.
-- Hanya mengambil kolom yang dibutuhkan oleh notification service.
-- Pelanggan yang sudah dihapus (soft delete) tidak dikembalikan.
SELECT id, tenant_id, customer_id_seq, name, phone, email, package_id
FROM customers
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTenantByID :one
-- Mengambil data tenant berdasarkan ID untuk keperluan substitusi variabel template.
-- Hanya mengambil kolom yang dibutuhkan oleh notification service (nama ISP).
SELECT id, name
FROM tenants
WHERE id = $1;
