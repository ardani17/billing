-- Kueri SQL untuk operasi CRUD tabel report_schedules.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel report_schedules dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateReportSchedule :one
-- Membuat jadwal laporan baru dan mengembalikan semua kolom.
INSERT INTO report_schedules (
    tenant_id, report_type, schedule_type,
    format, recipients, filters, created_by_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetReportScheduleByID :one
-- Mengambil jadwal laporan berdasarkan ID.
SELECT * FROM report_schedules
WHERE id = $1;

-- name: UpdateReportSchedule :one
-- Memperbarui jadwal laporan (tipe, format, penerima, filter).
UPDATE report_schedules SET
    report_type = $2,
    schedule_type = $3,
    format = $4,
    recipients = $5,
    filters = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateReportSchedule :exec
-- Menonaktifkan jadwal laporan (atur is_active = false).
UPDATE report_schedules SET
    is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: ListReportSchedulesByTenant :many
-- Mengambil semua jadwal laporan aktif untuk tenant, diurutkan berdasarkan created_at.
SELECT * FROM report_schedules
WHERE tenant_id = $1 AND is_active = true
ORDER BY created_at DESC;

-- name: ListDueSchedules :many
-- Mengambil semua jadwal aktif berdasarkan tipe jadwal (untuk worker scheduler).
-- Digunakan oleh background worker untuk menentukan jadwal mana yang perlu dijalankan.
SELECT * FROM report_schedules
WHERE schedule_type = $1 AND is_active = true;
