DROP POLICY IF EXISTS tenant_insert ON credit_notes;
DROP POLICY IF EXISTS tenant_isolation ON credit_notes;
DROP INDEX IF EXISTS idx_credit_notes_tenant_invoice;
ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS uq_credit_notes_tenant_number;
DROP TABLE IF EXISTS credit_notes;
