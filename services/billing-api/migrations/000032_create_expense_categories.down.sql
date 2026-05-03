-- Rollback migrasi: menghapus tabel expense_categories beserta semua policy, constraint, dan index.

DROP POLICY IF EXISTS expense_categories_tenant_insert ON expense_categories;
DROP POLICY IF EXISTS expense_categories_tenant_isolation ON expense_categories;
DROP INDEX IF EXISTS uq_expense_categories_tenant_name;
DROP TABLE IF EXISTS expense_categories;
