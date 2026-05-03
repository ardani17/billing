-- Rollback migrasi: menghapus tabel payment_link_invoices dan payment_links beserta semua policy, constraint, dan index.

-- Hapus junction table terlebih dahulu (karena FK ke payment_links)
DROP INDEX IF EXISTS idx_payment_link_invoices_invoice_id;
DROP INDEX IF EXISTS idx_payment_link_invoices_link_id;
DROP TABLE IF EXISTS payment_link_invoices;

-- Hapus tabel payment_links beserta policy dan index
DROP POLICY IF EXISTS payment_links_tenant_insert ON payment_links;
DROP POLICY IF EXISTS payment_links_tenant_policy ON payment_links;
DROP INDEX IF EXISTS idx_payment_links_expires_at;
DROP INDEX IF EXISTS idx_payment_links_customer_active;
DROP INDEX IF EXISTS idx_payment_links_external_id;
DROP TABLE IF EXISTS payment_links;
