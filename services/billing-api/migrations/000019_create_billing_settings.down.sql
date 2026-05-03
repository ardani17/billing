-- Rollback migrasi: menghapus tabel billing_settings beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS tenant_insert ON billing_settings;
DROP POLICY IF EXISTS tenant_isolation ON billing_settings;
ALTER TABLE billing_settings DROP CONSTRAINT IF EXISTS uq_billing_settings_tenant_id;
DROP TABLE IF EXISTS billing_settings;
