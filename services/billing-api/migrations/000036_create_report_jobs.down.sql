-- Rollback migrasi: menghapus tabel report_jobs beserta semua policy dan index.

DROP POLICY IF EXISTS report_jobs_tenant_insert ON report_jobs;
DROP POLICY IF EXISTS report_jobs_tenant_isolation ON report_jobs;
DROP INDEX IF EXISTS idx_report_jobs_tenant_status;
DROP TABLE IF EXISTS report_jobs;
