-- Rollback migrasi: menghapus tabel vouchers beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS tenant_insert ON vouchers;
DROP POLICY IF EXISTS tenant_isolation ON vouchers;
DROP INDEX IF EXISTS idx_vouchers_tenant_status_expires;
DROP INDEX IF EXISTS idx_vouchers_tenant_reseller;
DROP INDEX IF EXISTS idx_vouchers_tenant_package;
DROP INDEX IF EXISTS idx_vouchers_tenant_status;
ALTER TABLE vouchers DROP CONSTRAINT IF EXISTS uq_vouchers_tenant_code;
DROP TABLE IF EXISTS vouchers;
