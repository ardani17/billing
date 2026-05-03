-- Rollback migrasi: menghapus tabel webhook_logs beserta semua index.

DROP INDEX IF EXISTS idx_webhook_logs_tenant;
DROP INDEX IF EXISTS idx_webhook_logs_cleanup;
DROP INDEX IF EXISTS idx_webhook_logs_external_id;
DROP INDEX IF EXISTS idx_webhook_logs_idempotency;
DROP TABLE IF EXISTS webhook_logs;
