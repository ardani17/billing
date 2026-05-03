-- Rollback migrasi: menghapus tabel invoice_audit_logs beserta semua policy dan index.

DROP POLICY IF EXISTS tenant_insert ON invoice_audit_logs;
DROP POLICY IF EXISTS tenant_select ON invoice_audit_logs;
DROP INDEX IF EXISTS idx_invoice_audit_logs_tenant_invoice;
DROP TABLE IF EXISTS invoice_audit_logs;
