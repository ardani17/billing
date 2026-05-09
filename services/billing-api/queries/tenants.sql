-- Kueri SQL untuk operasi CRUD tabel tenants.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.

-- name: GetTenant :one
SELECT id, name, domain, plan, status, created_at, updated_at
FROM tenants WHERE id = $1;

-- name: ListTenants :many
SELECT id, name, domain, plan, status, created_at, updated_at
FROM tenants ORDER BY created_at DESC;

-- name: CreateTenant :one
INSERT INTO tenants (name, domain, plan, status)
VALUES ($1, $2, $3, $4)
RETURNING id, name, domain, plan, status, created_at, updated_at;

-- name: UpdateTenant :one
UPDATE tenants SET name = $2, domain = $3, plan = $4, status = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, name, domain, plan, status, created_at, updated_at;

-- name: DeleteTenant :exec
DELETE FROM tenants WHERE id = $1;

-- name: GetTenantByDomain :one
SELECT id, name, domain, plan, status, created_at, updated_at
FROM tenants WHERE domain = $1;
