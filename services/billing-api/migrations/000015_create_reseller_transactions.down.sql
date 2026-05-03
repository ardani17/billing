-- Rollback migrasi: menghapus tabel reseller_transactions beserta semua policy dan index.

DROP POLICY IF EXISTS tenant_insert ON reseller_transactions;
DROP POLICY IF EXISTS tenant_isolation ON reseller_transactions;
DROP INDEX IF EXISTS idx_reseller_tx_reseller_created;
DROP INDEX IF EXISTS idx_reseller_tx_reseller;
DROP TABLE IF EXISTS reseller_transactions;
