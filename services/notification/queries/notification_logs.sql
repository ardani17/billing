-- Kueri SQL untuk operasi CRUD tabel notification_logs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel notification_logs dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Digunakan untuk audit trail, retry tracking, deduplication, throttle, dan cooldown.

-- name: CreateLog :one
-- Membuat catatan log notifikasi baru dan mengembalikan log yang dibuat.
INSERT INTO notification_logs (
    tenant_id, customer_id, template_id, channel, provider, recipient,
    subject, body, status, retry_count, max_retries, error_message,
    dedup_key, metadata, sent_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12,
    $13, $14, $15
)
RETURNING *;

-- name: GetLogByID :one
-- Mengambil catatan log notifikasi berdasarkan ID.
SELECT *
FROM notification_logs
WHERE id = $1;

-- name: UpdateLog :exec
-- Memperbarui status, retry_count, error_message, sent_at, dan metadata log notifikasi.
UPDATE notification_logs
SET
    status = $2,
    retry_count = $3,
    error_message = $4,
    sent_at = $5,
    metadata = $6,
    updated_at = NOW()
WHERE id = $1;

-- name: ListLogs :many
-- Mengambil daftar log notifikasi dengan filter opsional dan paginasi.
-- Filter: tenant_id (wajib), channel, status, customer_id, template_id, date_from, date_to.
-- Diurutkan berdasarkan created_at DESC (terbaru di atas).
SELECT *
FROM notification_logs
WHERE tenant_id = $1
  AND (sqlc.narg('channel')::varchar IS NULL OR channel = sqlc.narg('channel'))
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('template_id')::uuid IS NULL OR template_id = sqlc.narg('template_id'))
  AND (sqlc.narg('date_from')::timestamptz IS NULL OR created_at >= sqlc.narg('date_from'))
  AND (sqlc.narg('date_to')::timestamptz IS NULL OR created_at <= sqlc.narg('date_to'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountLogs :one
-- Menghitung jumlah total log notifikasi dengan filter yang sama seperti ListLogs.
-- Digunakan untuk metadata paginasi (total dan total_pages).
SELECT COUNT(*)
FROM notification_logs
WHERE tenant_id = $1
  AND (sqlc.narg('channel')::varchar IS NULL OR channel = sqlc.narg('channel'))
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('template_id')::uuid IS NULL OR template_id = sqlc.narg('template_id'))
  AND (sqlc.narg('date_from')::timestamptz IS NULL OR created_at >= sqlc.narg('date_from'))
  AND (sqlc.narg('date_to')::timestamptz IS NULL OR created_at <= sqlc.narg('date_to'));

-- name: FindByDedupKey :one
-- Mencari log notifikasi berdasarkan dedup_key dalam jendela waktu tertentu.
-- Digunakan untuk pengecekan duplikasi sebelum pengiriman.
-- Hanya log dengan status aktif (bukan 'skipped' atau 'failed') yang dianggap duplikat.
SELECT *
FROM notification_logs
WHERE dedup_key = $1
  AND status NOT IN ('skipped', 'failed')
  AND created_at > NOW() - ($2 || ' hours')::interval
LIMIT 1;

-- name: CountTodayByCustomer :one
-- Menghitung jumlah notifikasi yang berhasil dikirim ke pelanggan hari ini.
-- Menggunakan timezone tenant untuk menentukan batas hari (awal hari lokal).
-- Hanya menghitung status 'sent' dan 'delivered' sesuai requirement throttle.
SELECT COUNT(*)
FROM notification_logs
WHERE tenant_id = $1
  AND customer_id = $2
  AND status IN ('sent', 'delivered')
  AND created_at >= (NOW() AT TIME ZONE $3)::date AT TIME ZONE $3;

-- name: LastSentToCustomer :one
-- Mengambil waktu pengiriman terakhir ke pelanggan tertentu.
-- Digunakan untuk pengecekan cooldown antar pesan (bawaan 30 menit).
-- Hanya mempertimbangkan notifikasi yang berhasil dikirim.
SELECT sent_at
FROM notification_logs
WHERE tenant_id = $1
  AND customer_id = $2
  AND status IN ('sent', 'delivered')
ORDER BY sent_at DESC
LIMIT 1;
