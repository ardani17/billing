-- Rollback migrasi: menghapus foreign key dari customers.package_id.

ALTER TABLE customers DROP CONSTRAINT IF EXISTS fk_customers_package_id;
