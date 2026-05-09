-- Kueri SQL untuk operasi CRUD tabel users.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel users dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateUser :one
INSERT INTO users (tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at
FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByTenantAndEmail :one
SELECT id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at
FROM users WHERE tenant_id = $1 AND email = $2;

-- name: GetUserByGoogleID :one
SELECT id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at
FROM users WHERE google_id = $1;

-- name: UpdateUser :one
UPDATE users SET name = $2, phone = $3, role = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at;

-- name: UpdateLastLogin :exec
UPDATE users SET last_login = NOW(), updated_at = NOW() WHERE id = $1;

-- name: UpdatePasswordHash :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: LinkGoogleID :exec
UPDATE users SET google_id = $2, updated_at = NOW() WHERE id = $1;

-- name: SetEmailVerified :exec
UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsersByTenant :many
SELECT id, tenant_id, name, email, phone, password_hash, role, email_verified, google_id, status, last_login, created_at, updated_at
FROM users WHERE tenant_id = $1 ORDER BY created_at DESC;

-- name: EmailExistsGlobal :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1) AS exists;
