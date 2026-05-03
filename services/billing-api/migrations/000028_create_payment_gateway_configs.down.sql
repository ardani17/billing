-- Rollback migrasi: menghapus tabel payment_gateway_configs beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS payment_gateway_configs_tenant_insert ON payment_gateway_configs;
DROP POLICY IF EXISTS payment_gateway_configs_tenant_policy ON payment_gateway_configs;
DROP INDEX IF EXISTS idx_payment_gateway_configs_tenant_active;
ALTER TABLE payment_gateway_configs DROP CONSTRAINT IF EXISTS uq_payment_gateway_configs_tenant_provider;
DROP TABLE IF EXISTS payment_gateway_configs;
