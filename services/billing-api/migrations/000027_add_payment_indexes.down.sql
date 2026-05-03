-- Rollback migrasi: menghapus semua index yang ditambahkan untuk modul pembayaran manual.

DROP INDEX IF EXISTS idx_invoices_open_by_customer;
DROP INDEX IF EXISTS idx_customers_search_payment;
DROP INDEX IF EXISTS idx_invoice_payments_duplicate_check;
DROP INDEX IF EXISTS idx_invoice_payments_method;
DROP INDEX IF EXISTS idx_invoice_payments_payment_date;

-- Catatan: ekstensi pg_trgm tidak dihapus karena mungkin digunakan oleh komponen lain.
