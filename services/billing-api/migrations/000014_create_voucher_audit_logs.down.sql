-- Rollback migrasi: menghapus tabel voucher_audit_logs beserta semua policy dan index.

DROP POLICY IF EXISTS tenant_insert ON voucher_audit_logs;
DROP POLICY IF EXISTS tenant_isolation ON voucher_audit_logs;
DROP INDEX IF EXISTS idx_voucher_audit_logs_tenant_created;
DROP INDEX IF EXISTS idx_voucher_audit_logs_tenant_voucher;
DROP TABLE IF EXISTS voucher_audit_logs;
