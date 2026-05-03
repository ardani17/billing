-- Migrasi: menambahkan index untuk modul pembayaran manual.
-- Index ini mengoptimalkan query daftar pembayaran, filter metode, deteksi duplikat,
-- pencarian pelanggan (trigram), dan lookup invoice terbuka per pelanggan.

-- Aktifkan ekstensi pg_trgm untuk index trigram pencarian pelanggan
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Index untuk daftar pembayaran dengan filter rentang tanggal (DESC untuk tampilan terbaru dulu)
CREATE INDEX idx_invoice_payments_payment_date
    ON invoice_payments (tenant_id, payment_date DESC, created_at DESC)
    WHERE voided = false;

-- Index untuk filter pembayaran berdasarkan metode (tunai, transfer, lainnya)
CREATE INDEX idx_invoice_payments_method
    ON invoice_payments (tenant_id, payment_method)
    WHERE voided = false;

-- Index untuk deteksi duplikat pembayaran (cek kombinasi tenant, invoice, jumlah, metode, tanggal)
CREATE INDEX idx_invoice_payments_duplicate_check
    ON invoice_payments (tenant_id, invoice_id, amount, payment_method, payment_date)
    WHERE voided = false;

-- Index GIN trigram untuk pencarian pelanggan di fitur pembayaran cepat
-- Mendukung pencarian fuzzy berdasarkan nama, ID pelanggan, atau nomor telepon
CREATE INDEX idx_customers_search_payment
    ON customers USING gin (
        (name || ' ' || customer_id_seq || ' ' || phone) gin_trgm_ops
    )
    WHERE status IN ('aktif', 'isolir') AND deleted_at IS NULL;

-- Index untuk lookup invoice terbuka per pelanggan, diurutkan berdasarkan jatuh tempo (ASC untuk FIFO)
CREATE INDEX idx_invoices_open_by_customer
    ON invoices (customer_id, due_date ASC)
    WHERE status IN ('belum_bayar', 'terlambat', 'bayar_sebagian');
