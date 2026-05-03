-- Rollback migrasi: menghapus tabel invoice_sequences beserta semua policy dan constraint.

DROP POLICY IF EXISTS tenant_insert ON invoice_sequences;
DROP POLICY IF EXISTS tenant_isolation ON invoice_sequences;
ALTER TABLE invoice_sequences DROP CONSTRAINT IF EXISTS uq_invoice_sequences_tenant_year_month;
DROP TABLE IF EXISTS invoice_sequences;
