-- Kueri SQL untuk operasi CRUD tabel password_resets dan email_verifications.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Token disimpan sebagai SHA-256 hash, plaintext hanya dikirim ke user.

-- name: CreatePasswordReset :exec
INSERT INTO password_resets (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: GetPasswordResetByHash :one
SELECT id, user_id, token_hash, expires_at, used, created_at
FROM password_resets WHERE token_hash = $1;

-- name: MarkPasswordResetUsed :exec
UPDATE password_resets SET used = true WHERE id = $1;

-- name: InvalidatePasswordResets :exec
UPDATE password_resets SET used = true WHERE user_id = $1 AND used = false;

-- name: CreateEmailVerification :exec
INSERT INTO email_verifications (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: GetEmailVerificationByHash :one
SELECT id, user_id, token_hash, expires_at, used, created_at
FROM email_verifications WHERE token_hash = $1;

-- name: MarkEmailVerificationUsed :exec
UPDATE email_verifications SET used = true WHERE id = $1;

-- name: InvalidateEmailVerifications :exec
UPDATE email_verifications SET used = true WHERE user_id = $1 AND used = false;
