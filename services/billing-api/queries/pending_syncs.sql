-- Query SQL untuk operasi CRUD tabel pending_syncs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel pending_syncs dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Digunakan oleh modul isolir untuk melacak operasi sinkronisasi router yang tertunda.

-- name: CreatePendingSync :one
-- Membuat pending sync baru dan mengembalikan semua kolom.
INSERT INTO pending_syncs (
    tenant_id, customer_id, operation_type, status,
    retry_count, max_retries, next_retry_at, metadata
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8
)
RETURNING *;

-- name: GetPendingSyncByID :one
-- Mengambil pending sync berdasarkan ID.
SELECT *
FROM pending_syncs
WHERE id = $1;

-- name: UpdatePendingSyncStatus :exec
-- Memperbarui status pending sync berdasarkan ID.
UPDATE pending_syncs SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdatePendingSyncRetry :exec
-- Memperbarui retry_count, next_retry_at, last_retry_at, error_message berdasarkan ID.
UPDATE pending_syncs SET
    retry_count = $2,
    next_retry_at = $3,
    last_retry_at = $4,
    error_message = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkPendingSyncCompleted :exec
-- Menandai pending sync sebagai completed.
UPDATE pending_syncs SET
    status = 'completed',
    updated_at = NOW()
WHERE id = $1;

-- name: MarkPendingSyncFailed :exec
-- Menandai pending sync sebagai failed dengan error message.
UPDATE pending_syncs SET
    status = 'failed',
    error_message = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: FindPendingSyncsForRetry :many
-- Mengambil pending syncs yang siap di-retry (status pending, next_retry_at <= now atau NULL).
-- Diurutkan berdasarkan created_at ASC, dibatasi batch_size.
SELECT *
FROM pending_syncs
WHERE status = 'pending'
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at ASC
LIMIT sqlc.arg('batch_size');

-- name: FindPendingSyncsByCustomer :many
-- Mengambil semua pending syncs untuk customer tertentu.
SELECT *
FROM pending_syncs
WHERE customer_id = sqlc.arg('customer_id')
ORDER BY created_at DESC;

-- name: FindPendingSyncsByTenantAndStatus :many
-- Mengambil pending syncs berdasarkan tenant dan status (opsional) dengan paginasi.
-- LEFT JOIN customers untuk mendapatkan customer_name dan customer_id_seq.
SELECT ps.*,
    c.name AS customer_name,
    c.customer_id_seq AS customer_id_seq
FROM pending_syncs ps
LEFT JOIN customers c ON c.id = ps.customer_id
WHERE ps.tenant_id = sqlc.arg('tenant_id')
  AND (sqlc.narg('status')::varchar IS NULL OR ps.status = sqlc.narg('status')::varchar)
ORDER BY ps.created_at DESC
LIMIT sqlc.arg('page_size')
OFFSET sqlc.arg('offset');

-- name: CountPendingSyncsByTenantAndStatuses :one
-- Menghitung jumlah pending syncs berdasarkan tenant dan array status.
SELECT COUNT(*)
FROM pending_syncs
WHERE tenant_id = sqlc.arg('tenant_id')
  AND status = ANY(sqlc.arg('statuses')::varchar[]);

-- name: ResetRetryForCustomer :exec
-- Mereset retry_count ke 0 dan status ke pending untuk customer tertentu.
-- Hanya mereset yang berstatus pending atau failed.
UPDATE pending_syncs SET
    retry_count = 0,
    next_retry_at = NOW(),
    status = 'pending',
    updated_at = NOW()
WHERE customer_id = sqlc.arg('customer_id')
  AND status IN ('pending', 'failed');

-- name: ResetRetryAll :execrows
-- Mereset retry_count ke 0 dan status ke pending untuk semua pending/failed di tenant.
-- Mengembalikan jumlah baris yang terpengaruh.
UPDATE pending_syncs SET
    retry_count = 0,
    next_retry_at = NOW(),
    status = 'pending',
    updated_at = NOW()
WHERE tenant_id = sqlc.arg('tenant_id')
  AND status IN ('pending', 'failed');
