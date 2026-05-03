-- Rollback migrasi: menghapus tabel kpi_targets beserta semua policy.

DROP POLICY IF EXISTS kpi_targets_tenant_insert ON kpi_targets;
DROP POLICY IF EXISTS kpi_targets_tenant_isolation ON kpi_targets;
DROP TABLE IF EXISTS kpi_targets;
