-- Rollback migrasi: menghapus tabel packages beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS tenant_insert ON packages;
DROP POLICY IF EXISTS tenant_isolation ON packages;
DROP INDEX IF EXISTS idx_packages_tenant_type_active;
DROP INDEX IF EXISTS idx_packages_tenant_active;
DROP INDEX IF EXISTS idx_packages_tenant_type;
DROP TABLE IF EXISTS packages;
