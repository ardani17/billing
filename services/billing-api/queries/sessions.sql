-- Kueri SQL untuk operasi CRUD tabel sessions.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Setiap login dari device berbeda membuat record session baru.

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, device_info, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token_hash, device_info, ip_address, expires_at, created_at;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, device_info, ip_address, expires_at, created_at
FROM sessions WHERE token_hash = $1 AND expires_at > NOW();

-- name: ListSessionsByUserID :many
SELECT id, user_id, token_hash, device_info, ip_address, expires_at, created_at
FROM sessions WHERE user_id = $1 AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: DeleteSessionByID :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteOtherSessions :exec
DELETE FROM sessions WHERE user_id = $1 AND id != $2;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= NOW();
