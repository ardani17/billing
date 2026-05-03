-- Rollback migrasi: menghapus tabel resellers beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS tenant_insert ON resellers;
DROP POLICY IF EXISTS tenant_isolation ON resellers;
DROP INDEX IF EXISTS idx_resellers_tenant_phone;
DROP INDEX IF EXISTS idx_resellers_tenant_status;
ALTER TABLE resellers DROP CONSTRAINT IF EXISTS uq_resellers_tenant_phone;
DROP TABLE IF EXISTS resellers;
