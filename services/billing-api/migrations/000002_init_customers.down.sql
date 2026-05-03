-- Rollback migrasi: menghapus tabel customers beserta semua dependensinya.
-- CASCADE akan menghapus policy RLS yang terkait secara otomatis.
DROP TABLE IF EXISTS customers CASCADE;
