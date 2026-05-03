-- Migrasi: menghapus tabel customers lama (sample dari monorepo-setup).
-- Tabel ini akan diganti dengan schema lengkap di migrasi 000008.

DROP POLICY IF EXISTS tenant_insert ON customers;
DROP POLICY IF EXISTS tenant_isolation ON customers;
DROP INDEX IF EXISTS idx_customers_status;
DROP INDEX IF EXISTS idx_customers_tenant_id;
DROP TABLE IF EXISTS customers;
