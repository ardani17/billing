-- Query SQL untuk operasi CRUD tabel webhook_logs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel webhook_logs TIDAK menggunakan RLS karena webhook diterima sebelum identifikasi tenant.
-- Bersifat append-only, menyimpan seluruh request termasuk yang gagal verifikasi.

-- name: CreateWebhookLog :one
-- Membuat log webhook baru dan mengembalikan semua kolom.
INSERT INTO webhook_logs (
    tenant_id, gateway_provider, event_type,
    external_id, request_body, source_ip,
    signature_valid, processing_status, error_message
) VALUES (
    $1, $2, $3,
    $4, $5, $6,
    $7, $8, $9
)
RETURNING *;

-- name: GetWebhookLogByID :one
-- Mengambil webhook log berdasarkan ID.
SELECT *
FROM webhook_logs
WHERE id = $1;

-- name: UpdateWebhookLogStatus :exec
-- Memperbarui status pemrosesan dan pesan error pada webhook log.
UPDATE webhook_logs SET
    processing_status = $2,
    error_message = $3
WHERE id = $1;

-- name: UpdateWebhookLogSignatureValid :exec
-- Memperbarui flag signature_valid pada webhook log.
UPDATE webhook_logs SET
    signature_valid = $2
WHERE id = $1;

-- name: IsWebhookAlreadyProcessed :one
-- Mengecek apakah webhook dengan external_id dan event_type sudah berhasil diproses.
-- Digunakan untuk idempotency check sebelum memproses webhook.
SELECT EXISTS(
    SELECT 1 FROM webhook_logs
    WHERE external_id = $1 AND event_type = $2 AND processing_status = 'processed'
) AS exists;

-- name: ListWebhookLogsByExternalID :many
-- Mengambil semua webhook logs berdasarkan external_id dengan urutan terbaru.
-- Digunakan untuk melihat riwayat webhook terkait payment link tertentu.
SELECT *
FROM webhook_logs
WHERE external_id = $1
ORDER BY created_at DESC;

-- name: DeleteWebhookLogsOlderThan :execrows
-- Menghapus webhook logs yang lebih tua dari waktu yang ditentukan.
-- Tidak menghapus logs dengan status 'failed' atau signature_valid = false (log keamanan dipertahankan).
-- Mengembalikan jumlah baris yang dihapus.
DELETE FROM webhook_logs
WHERE created_at < $1
  AND processing_status != 'failed'
  AND (signature_valid IS NULL OR signature_valid = true);
