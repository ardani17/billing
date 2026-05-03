-- Rollback migrasi: menghapus tabel email_verifications dan password_resets.
DROP TABLE IF EXISTS email_verifications;
DROP TABLE IF EXISTS password_resets;
