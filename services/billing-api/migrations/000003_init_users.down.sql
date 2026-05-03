-- Rollback migrasi: menghapus tabel users beserta semua dependensinya.
-- CASCADE akan menghapus policy RLS yang terkait secara otomatis.
DROP TABLE IF EXISTS users CASCADE;
