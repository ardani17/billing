-- Migrasi: menambahkan foreign key dari customers.package_id ke packages.id.
-- Kolom package_id sudah ada di tabel customers (UUID NOT NULL) dari migrasi 000008,
-- tapi belum memiliki FK karena tabel packages belum ada saat itu.

ALTER TABLE customers
    ADD CONSTRAINT fk_customers_package_id
    FOREIGN KEY (package_id) REFERENCES packages(id);
