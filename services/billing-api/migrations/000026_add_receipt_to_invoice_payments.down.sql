-- Rollback migrasi: menghapus kolom receipt dari tabel invoice_payments.

ALTER TABLE invoice_payments DROP COLUMN IF EXISTS receipt_number;
ALTER TABLE invoice_payments DROP COLUMN IF EXISTS receipt_group_id;
ALTER TABLE invoice_payments DROP COLUMN IF EXISTS proof_image_url;
