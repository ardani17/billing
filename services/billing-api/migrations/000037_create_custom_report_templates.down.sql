-- Rollback migrasi: menghapus tabel custom_report_templates beserta semua policy dan index.

DROP POLICY IF EXISTS custom_report_templates_tenant_insert ON custom_report_templates;
DROP POLICY IF EXISTS custom_report_templates_tenant_isolation ON custom_report_templates;
DROP INDEX IF EXISTS idx_custom_report_templates_tenant;
DROP TABLE IF EXISTS custom_report_templates;
