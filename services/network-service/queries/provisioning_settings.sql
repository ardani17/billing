-- Kueri SQL untuk operasi tabel provisioning_settings.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Satu record per tenant, menggunakan upsert untuk buat/perbarui.

-- name: GetProvisioningSettingsByTenantID :one
SELECT id, tenant_id, auto_provisioning_enabled, auto_port_migration_enabled,
    vlan_strategy, created_at, updated_at
FROM provisioning_settings
WHERE tenant_id = $1;

-- name: UpsertProvisioningSettings :one
INSERT INTO provisioning_settings (
    tenant_id, auto_provisioning_enabled, auto_port_migration_enabled, vlan_strategy
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (tenant_id) DO UPDATE SET
    auto_provisioning_enabled = EXCLUDED.auto_provisioning_enabled,
    auto_port_migration_enabled = EXCLUDED.auto_port_migration_enabled,
    vlan_strategy = EXCLUDED.vlan_strategy,
    updated_at = NOW()
RETURNING id, tenant_id, auto_provisioning_enabled, auto_port_migration_enabled,
    vlan_strategy, created_at, updated_at;
