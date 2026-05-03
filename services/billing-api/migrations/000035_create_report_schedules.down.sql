-- Rollback migrasi: menghapus tabel report_schedules beserta semua policy dan index.

DROP POLICY IF EXISTS report_schedules_tenant_insert ON report_schedules;
DROP POLICY IF EXISTS report_schedules_tenant_isolation ON report_schedules;
DROP INDEX IF EXISTS idx_report_schedules_tenant;
DROP TABLE IF EXISTS report_schedules;
