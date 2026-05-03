-- Rollback migrasi: menghapus tabel receipt_sequences beserta semua policy, index, dan constraint.

DROP POLICY IF EXISTS receipt_sequences_tenant_insert ON receipt_sequences;
DROP POLICY IF EXISTS receipt_sequences_tenant_policy ON receipt_sequences;
DROP INDEX IF EXISTS idx_receipt_sequences_tenant_year_month;
ALTER TABLE receipt_sequences DROP CONSTRAINT IF EXISTS uq_receipt_sequences_tenant_year_month;
DROP TABLE IF EXISTS receipt_sequences;
