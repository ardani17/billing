-- Rollback migrasi: menghapus tabel customer_recurring_items beserta semua policy dan index.

DROP POLICY IF EXISTS tenant_insert ON customer_recurring_items;
DROP POLICY IF EXISTS tenant_isolation ON customer_recurring_items;
DROP INDEX IF EXISTS idx_customer_recurring_items_tenant_customer_active;
DROP TABLE IF EXISTS customer_recurring_items;
