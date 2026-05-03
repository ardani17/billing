-- Query SQL untuk operasi CRUD tabel report_jobs.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel report_jobs dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreateReportJob :one
-- Membuat job export laporan baru dan mengembalikan semua kolom.
INSERT INTO report_jobs (
    tenant_id, report_type, format,
    filters, requested_by
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetReportJobByID :one
-- Mengambil job export laporan berdasarkan ID.
SELECT * FROM report_jobs
WHERE id = $1;

-- name: UpdateReportJobStatus :exec
-- Memperbarui status job export (pending → processing → completed/failed).
-- Juga mengisi download_url dan error jika ada.
UPDATE report_jobs SET
    status = $2,
    download_url = $3,
    error = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: CleanupOldReportJobs :exec
-- Menghapus job export lama yang sudah melewati batas waktu retensi.
-- Digunakan oleh background worker untuk membersihkan data lama.
DELETE FROM report_jobs
WHERE created_at < $1;
