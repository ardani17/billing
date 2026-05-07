-- Kueri SQL untuk operasi tabel provisioning_audit_logs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel ini append-only: hanya Buat dan List, tidak ada Perbarui atau Hapus.
-- Tabel dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateAuditLog :one
INSERT INTO provisioning_audit_logs (
    tenant_id, olt_id, ont_id, action, commands_sent,
    command_responses, status, error_message, performed_by,
    brand, model, transport, operation, correlation_id
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14
)
RETURNING id, tenant_id, olt_id, ont_id, action, commands_sent,
    command_responses, status, error_message, performed_by,
    brand, model, transport, operation,
    correlation_id, created_at;

-- name: ListAuditLogs :many
SELECT id, tenant_id, olt_id, ont_id, action, commands_sent,
    command_responses, status, error_message, performed_by,
    brand, model, transport, operation,
    correlation_id, created_at
FROM provisioning_audit_logs
WHERE (sqlc.narg('olt_id')::uuid IS NULL OR olt_id = sqlc.narg('olt_id')::uuid)
  AND (sqlc.narg('ont_id')::uuid IS NULL OR ont_id = sqlc.narg('ont_id')::uuid)
  AND (sqlc.narg('action')::varchar IS NULL OR action = sqlc.narg('action')::varchar)
  AND (sqlc.narg('date_from')::timestamptz IS NULL OR created_at >= sqlc.narg('date_from')::timestamptz)
  AND (sqlc.narg('date_to')::timestamptz IS NULL OR created_at <= sqlc.narg('date_to')::timestamptz)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM provisioning_audit_logs
WHERE (sqlc.narg('olt_id')::uuid IS NULL OR olt_id = sqlc.narg('olt_id')::uuid)
  AND (sqlc.narg('ont_id')::uuid IS NULL OR ont_id = sqlc.narg('ont_id')::uuid)
  AND (sqlc.narg('action')::varchar IS NULL OR action = sqlc.narg('action')::varchar)
  AND (sqlc.narg('date_from')::timestamptz IS NULL OR created_at >= sqlc.narg('date_from')::timestamptz)
  AND (sqlc.narg('date_to')::timestamptz IS NULL OR created_at <= sqlc.narg('date_to')::timestamptz);
