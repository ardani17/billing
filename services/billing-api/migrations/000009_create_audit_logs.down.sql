-- Rollback migrasi: menghapus tabel audit_logs beserta semua dependensinya.

DROP POLICY IF EXISTS tenant_insert ON audit_logs;
DROP POLICY IF EXISTS tenant_isolation ON audit_logs;
DROP INDEX IF EXISTS idx_audit_logs_created;
DROP INDEX IF EXISTS idx_audit_logs_entity;
DROP TABLE IF EXISTS audit_logs;
