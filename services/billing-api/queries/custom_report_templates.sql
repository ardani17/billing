-- Kueri SQL untuk operasi CRUD tabel custom_report_templates.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel custom_report_templates dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateCustomReportTemplate :one
-- Membuat template laporan kustom baru dan mengembalikan semua kolom.
INSERT INTO custom_report_templates (
    tenant_id, name, metrics, group_by,
    sub_group_by, display_type, default_period_range,
    created_by_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetCustomReportTemplateByID :one
-- Mengambil template laporan kustom berdasarkan ID.
SELECT * FROM custom_report_templates
WHERE id = $1;

-- name: DeleteCustomReportTemplate :exec
-- Menghapus template laporan kustom secara permanen.
DELETE FROM custom_report_templates
WHERE id = $1;

-- name: ListCustomReportTemplatesByTenant :many
-- Mengambil semua template laporan kustom untuk tenant, diurutkan berdasarkan nama.
SELECT * FROM custom_report_templates
WHERE tenant_id = $1
ORDER BY name ASC;
