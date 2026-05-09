-- Rollback migrasi: menghapus tabel notifikasi beserta semua dependensinya.
-- Urutan drop berdasarkan dependensi FK (notification_logs -> notification_templates -> notification_configs).

-- ============================================================
-- 1. Hapus tabel notification_logs (memiliki FK ke notification_templates)
-- ============================================================
DROP INDEX IF EXISTS uq_notif_log_dedup_active;
DROP INDEX IF EXISTS idx_notif_log_dedup;
DROP INDEX IF EXISTS idx_notif_log_tenant_created;
DROP INDEX IF EXISTS idx_notif_log_tenant_status;
DROP INDEX IF EXISTS idx_notif_log_tenant_customer;
DROP POLICY IF EXISTS tenant_delete_notif_log ON notification_logs;
DROP POLICY IF EXISTS tenant_update_notif_log ON notification_logs;
DROP POLICY IF EXISTS tenant_insert_notif_log ON notification_logs;
DROP POLICY IF EXISTS tenant_select_notif_log ON notification_logs;
DROP TABLE IF EXISTS notification_logs;

-- ============================================================
-- 2. Hapus tabel notification_templates
-- ============================================================
DROP INDEX IF EXISTS idx_notif_template_tenant_active;
DROP INDEX IF EXISTS idx_notif_template_tenant_event;
DROP POLICY IF EXISTS tenant_delete_notif_template ON notification_templates;
DROP POLICY IF EXISTS tenant_update_notif_template ON notification_templates;
DROP POLICY IF EXISTS tenant_insert_notif_template ON notification_templates;
DROP POLICY IF EXISTS tenant_select_notif_template ON notification_templates;
DROP TABLE IF EXISTS notification_templates;

-- ============================================================
-- 3. Hapus tabel notification_configs
-- ============================================================
DROP INDEX IF EXISTS idx_notif_config_tenant_enabled;
DROP POLICY IF EXISTS tenant_delete_notif_config ON notification_configs;
DROP POLICY IF EXISTS tenant_update_notif_config ON notification_configs;
DROP POLICY IF EXISTS tenant_insert_notif_config ON notification_configs;
DROP POLICY IF EXISTS tenant_select_notif_config ON notification_configs;
DROP TABLE IF EXISTS notification_configs;
