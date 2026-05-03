-- Rollback migrasi: menghapus tabel invoices beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS tenant_insert ON invoices;
DROP POLICY IF EXISTS tenant_isolation ON invoices;
DROP INDEX IF EXISTS idx_invoices_tenant_due_date_status;
DROP INDEX IF EXISTS idx_invoices_tenant_period;
DROP INDEX IF EXISTS idx_invoices_tenant_customer;
DROP INDEX IF EXISTS idx_invoices_tenant_status;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS uq_invoices_tenant_invoice_number;
DROP TABLE IF EXISTS invoices;
