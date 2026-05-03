-- Rollback migrasi: menghapus tabel customers beserta semua dependensinya.

DROP POLICY IF EXISTS tenant_insert ON customers;
DROP POLICY IF EXISTS tenant_isolation ON customers;
DROP INDEX IF EXISTS idx_customers_active;
DROP INDEX IF EXISTS idx_customers_tenant_due_date;
DROP INDEX IF EXISTS idx_customers_tenant_package;
DROP INDEX IF EXISTS idx_customers_tenant_area;
DROP INDEX IF EXISTS idx_customers_tenant_phone;
DROP INDEX IF EXISTS idx_customers_tenant_id_seq;
DROP INDEX IF EXISTS idx_customers_tenant_status;
ALTER TABLE customers DROP CONSTRAINT IF EXISTS uq_customers_tenant_id_seq;
ALTER TABLE customers DROP CONSTRAINT IF EXISTS uq_customers_tenant_phone;
DROP TABLE IF EXISTS customers;
