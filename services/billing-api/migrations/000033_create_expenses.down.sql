-- Rollback migrasi: menghapus tabel expenses beserta semua policy dan index.

DROP POLICY IF EXISTS expenses_tenant_insert ON expenses;
DROP POLICY IF EXISTS expenses_tenant_isolation ON expenses;
DROP INDEX IF EXISTS idx_expenses_tenant_period;
DROP TABLE IF EXISTS expenses;
