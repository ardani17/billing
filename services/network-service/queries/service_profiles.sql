-- Kueri SQL untuk operasi CRUD tabel service_profiles.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel service_profiles dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateServiceProfile :one
INSERT INTO service_profiles (
    tenant_id, olt_id, name, line_profile_id, service_profile_id,
    package_id, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, tenant_id, olt_id, name, line_profile_id, service_profile_id,
    package_id, description, deleted_at, created_at, updated_at;

-- name: GetServiceProfileByID :one
SELECT id, tenant_id, olt_id, name, line_profile_id, service_profile_id,
    package_id, description, deleted_at, created_at, updated_at
FROM service_profiles
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateServiceProfile :one
UPDATE service_profiles SET
    name = $2,
    line_profile_id = $3,
    service_profile_id = $4,
    package_id = $5,
    description = $6,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, olt_id, name, line_profile_id, service_profile_id,
    package_id, description, deleted_at, created_at, updated_at;

-- name: SoftDeleteServiceProfile :exec
UPDATE service_profiles SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListServiceProfiles :many
SELECT id, tenant_id, olt_id, name, line_profile_id, service_profile_id,
    package_id, description, deleted_at, created_at, updated_at
FROM service_profiles
WHERE olt_id = $1 AND deleted_at IS NULL
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: CountServiceProfiles :one
SELECT COUNT(*) FROM service_profiles
WHERE olt_id = $1 AND deleted_at IS NULL;

-- name: GetServiceProfileByPackageAndOLT :one
SELECT id, tenant_id, olt_id, name, line_profile_id, service_profile_id,
    package_id, description, deleted_at, created_at, updated_at
FROM service_profiles
WHERE olt_id = $1 AND package_id = $2 AND deleted_at IS NULL;

-- name: ServiceProfileExists :one
SELECT EXISTS(
    SELECT 1 FROM service_profiles
    WHERE olt_id = $1 AND line_profile_id = $2 AND service_profile_id = $3
      AND id != $4 AND deleted_at IS NULL
) AS exists;

-- name: CountServiceProfileActiveONTs :one
SELECT COUNT(*) FROM onts
WHERE service_profile_id = $1 AND deleted_at IS NULL AND status != 'decommissioned';
