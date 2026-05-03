-- Rollback migrasi: menghapus tabel invoice_payments beserta semua policy dan index.

DROP POLICY IF EXISTS tenant_insert ON invoice_payments;
DROP POLICY IF EXISTS tenant_isolation ON invoice_payments;
DROP INDEX IF EXISTS idx_invoice_payments_tenant_payment_date;
DROP INDEX IF EXISTS idx_invoice_payments_tenant_invoice;
DROP TABLE IF EXISTS invoice_payments;
