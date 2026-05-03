-- Rollback migrasi: menghapus tabel tenants beserta semua dependensinya.
DROP TABLE IF EXISTS tenants CASCADE;
