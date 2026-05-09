-- Kueri SQL untuk operasi CRUD tabel notification_templates.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel notification_templates dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant memiliki slug unik per template (UNIQUE pada tenant_id, slug).

-- name: CreateTemplate :one
-- Membuat template notifikasi baru dan mengembalikan template yang dibuat.
INSERT INTO notification_templates (
    tenant_id, slug, name, category, event_type, channels,
    body_whatsapp, body_sms, body_email_subject, body_email_html,
    variables, is_active, is_default
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10,
    $11, $12, $13
)
RETURNING *;

-- name: GetTemplateByID :one
-- Mengambil template notifikasi berdasarkan ID.
SELECT *
FROM notification_templates
WHERE id = $1;

-- name: GetTemplateBySlug :one
-- Mengambil template notifikasi berdasarkan tenant_id dan slug yang masih aktif.
SELECT *
FROM notification_templates
WHERE tenant_id = $1 AND slug = $2 AND is_active = true;

-- name: GetTemplateByEventType :one
-- Mengambil template notifikasi berdasarkan tenant_id dan event_type yang masih aktif.
-- Digunakan oleh delivery pipeline untuk resolusi template dari event.
SELECT *
FROM notification_templates
WHERE tenant_id = $1 AND event_type = $2 AND is_active = true;

-- name: UpdateTemplate :one
-- Memperbarui template notifikasi dan mengembalikan template yang diperbarui.
UPDATE notification_templates
SET
    name = $2,
    channels = $3,
    body_whatsapp = $4,
    body_sms = $5,
    body_email_subject = $6,
    body_email_html = $7,
    is_active = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteTemplate :exec
-- Menonaktifkan template dengan mengatur is_active menjadi false.
-- Hanya template kustom (is_default=false) yang boleh dihapus (validasi di layer repositori).
UPDATE notification_templates
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ListTemplatesByTenant :many
-- Mengambil semua template notifikasi untuk tenant tertentu, diurutkan berdasarkan waktu pembuatan.
SELECT *
FROM notification_templates
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: BulkCreateTemplates :one
-- Membuat satu template notifikasi (dipanggil dalam loop oleh repositori untuk bulk insert).
INSERT INTO notification_templates (
    tenant_id, slug, name, category, event_type, channels,
    body_whatsapp, body_sms, body_email_subject, body_email_html,
    variables, is_active, is_default
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10,
    $11, $12, $13
)
RETURNING *;

-- name: SlugExists :one
-- Mengecek apakah slug sudah ada di tenant tertentu (exclude ID tertentu untuk keperluan perbarui).
-- Mengembalikan true jika slug sudah dipakai oleh template lain yang masih aktif.
SELECT EXISTS(
    SELECT 1
    FROM notification_templates
    WHERE tenant_id = $1 AND slug = $2 AND id != $3 AND is_active = true
);
