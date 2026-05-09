-- Kueri SQL untuk operasi CRUD tabel olt_alarms.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel olt_alarms dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateOLTAlarm :one
INSERT INTO olt_alarms (
    tenant_id, olt_id, pon_port_index, ont_index,
    alarm_type, severity, message, source, status
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9
)
RETURNING id, tenant_id, olt_id, pon_port_index, ont_index,
    alarm_type, severity, message, source, status,
    cleared_at, created_at;

-- name: ListOLTAlarms :many
SELECT id, tenant_id, olt_id, pon_port_index, ont_index,
    alarm_type, severity, message, source, status,
    cleared_at, created_at
FROM olt_alarms
WHERE olt_id = $1
  AND (sqlc.narg('severity')::varchar IS NULL OR severity = sqlc.narg('severity')::varchar)
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOLTAlarms :one
SELECT COUNT(*) FROM olt_alarms
WHERE olt_id = $1
  AND (sqlc.narg('severity')::varchar IS NULL OR severity = sqlc.narg('severity')::varchar)
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status')::varchar);

-- name: CountActiveAlarms :one
SELECT COUNT(*) FROM olt_alarms
WHERE olt_id = $1 AND status = 'active';

-- name: CountActiveAlarmsByTenant :one
SELECT COUNT(*) FROM olt_alarms
WHERE status = 'active';

-- name: ClearOLTAlarm :exec
UPDATE olt_alarms SET
    status = 'cleared',
    cleared_at = NOW()
WHERE id = $1 AND status = 'active';

-- name: PurgeOLTAlarms :execrows
DELETE FROM olt_alarms
WHERE created_at < $1;
