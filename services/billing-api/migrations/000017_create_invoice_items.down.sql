-- Rollback migrasi: menghapus tabel invoice_items beserta semua policy dan index.

DROP POLICY IF EXISTS tenant_insert ON invoice_items;
DROP POLICY IF EXISTS tenant_isolation ON invoice_items;
DROP INDEX IF EXISTS idx_invoice_items_tenant_invoice;
DROP TABLE IF EXISTS invoice_items;
