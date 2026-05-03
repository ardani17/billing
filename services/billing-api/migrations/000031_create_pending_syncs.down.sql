-- Rollback migrasi: menghapus tabel pending_syncs beserta semua index.

DROP INDEX IF EXISTS idx_pending_syncs_retry;
DROP INDEX IF EXISTS idx_pending_syncs_tenant_status;
DROP INDEX IF EXISTS idx_pending_syncs_tenant_customer;
DROP TABLE IF EXISTS pending_syncs;
