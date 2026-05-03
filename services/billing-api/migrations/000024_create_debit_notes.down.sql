DROP INDEX IF EXISTS idx_debit_note_items_debit_note;
DROP TABLE IF EXISTS debit_note_items;
DROP POLICY IF EXISTS tenant_insert ON debit_notes;
DROP POLICY IF EXISTS tenant_isolation ON debit_notes;
DROP INDEX IF EXISTS idx_debit_notes_tenant_customer;
ALTER TABLE debit_notes DROP CONSTRAINT IF EXISTS uq_debit_notes_tenant_number;
DROP TABLE IF EXISTS debit_notes;
