-- Query SQL untuk operasi CRUD tabel pppoe_profiles.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel pppoe_profiles dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Tabel ini TIDAK menggunakan soft-delete (tidak ada kolom deleted_at).

-- name: CreatePPPoEProfile :one
INSERT INTO pppoe_profiles (
    tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one
) VALUES (
    $1, $2, $3,
    $4, $5,
    $6, $7,
    $8, $9,
    $10, $11, $12, $13
)
RETURNING id, tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one,
    created_at, updated_at;

-- name: GetPPPoEProfileByID :one
SELECT id, tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one,
    created_at, updated_at
FROM pppoe_profiles
WHERE id = $1;

-- name: GetPPPoEProfileByPackageID :one
SELECT id, tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one,
    created_at, updated_at
FROM pppoe_profiles
WHERE package_id = $1;

-- name: GetPPPoEProfileByProfileName :one
SELECT id, tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one,
    created_at, updated_at
FROM pppoe_profiles
WHERE tenant_id = $1 AND profile_name = $2;

-- name: UpdatePPPoEProfile :one
UPDATE pppoe_profiles SET
    profile_name = $2,
    download_limit = $3,
    upload_limit = $4,
    burst_download = $5,
    burst_upload = $6,
    burst_threshold_download = $7,
    burst_threshold_upload = $8,
    burst_time = $9,
    address_pool = $10,
    local_address = $11,
    only_one = $12,
    updated_at = NOW()
WHERE id = $1
RETURNING id, tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one,
    created_at, updated_at;

-- name: ListPPPoEProfilesByTenant :many
SELECT id, tenant_id, package_id, profile_name,
    download_limit, upload_limit,
    burst_download, burst_upload,
    burst_threshold_download, burst_threshold_upload,
    burst_time, address_pool, local_address, only_one,
    created_at, updated_at
FROM pppoe_profiles
WHERE tenant_id = $1
ORDER BY profile_name ASC;
