-- Migrasi: menambahkan kolom receipt ke tabel invoice_payments.
-- receipt_number menyimpan nomor kwitansi (format PAY-YYYY-MM-SEQ) untuk setiap pembayaran.
-- receipt_group_id menghubungkan beberapa baris invoice_payment dari satu pembayaran multi-invoice.
-- proof_image_url menyimpan path/URL ke gambar bukti transfer yang diunggah.

ALTER TABLE invoice_payments ADD COLUMN receipt_number VARCHAR(50);
ALTER TABLE invoice_payments ADD COLUMN receipt_group_id UUID;
ALTER TABLE invoice_payments ADD COLUMN proof_image_url VARCHAR(500);
