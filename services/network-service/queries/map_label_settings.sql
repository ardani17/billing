-- Kueri SQL untuk operasi pada tabel map_label_settings.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel map_label_settings dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Setiap tenant memiliki satu record settings (UNIQUE tenant_id).

-- name: GetMapLabelSettingsByTenantID :one
SELECT id, tenant_id, olt_labels, odp_labels, ont_labels,
    min_zoom_level, created_at, updated_at
FROM map_label_settings
WHERE tenant_id = $1;

-- name: UpsertMapLabelSettings :one
INSERT INTO map_label_settings (
    tenant_id, olt_labels, odp_labels, ont_labels, min_zoom_level
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (tenant_id) DO UPDATE SET
    olt_labels = EXCLUDED.olt_labels,
    odp_labels = EXCLUDED.odp_labels,
    ont_labels = EXCLUDED.ont_labels,
    min_zoom_level = EXCLUDED.min_zoom_level,
    updated_at = NOW()
RETURNING id, tenant_id, olt_labels, odp_labels, ont_labels,
    min_zoom_level, created_at, updated_at;
