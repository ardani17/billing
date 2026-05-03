-- Rollback migrasi: menghapus tabel areas beserta semua dependensinya.

DROP POLICY IF EXISTS tenant_insert ON areas;
DROP POLICY IF EXISTS tenant_isolation ON areas;
DROP INDEX IF EXISTS idx_areas_tenant_id;
ALTER TABLE areas DROP CONSTRAINT IF EXISTS uq_areas_tenant_name;
DROP TABLE IF EXISTS areas;
