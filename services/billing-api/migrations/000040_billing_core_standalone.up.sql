-- Migrasi: dukungan Billing Core standalone tanpa add-on jaringan.
-- Menambahkan mode pelanggan manual dan paket bulanan netral.

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS chk_customers_connection_method;

ALTER TABLE customers
    ADD CONSTRAINT chk_customers_connection_method CHECK (
        connection_method IN ('manual', 'pppoe', 'hotspot', 'dhcp_binding', 'static')
    );

ALTER TABLE customers
    ALTER COLUMN latitude DROP NOT NULL,
    ALTER COLUMN longitude DROP NOT NULL;

ALTER TABLE packages
    DROP CONSTRAINT IF EXISTS chk_packages_type;

ALTER TABLE packages
    ADD CONSTRAINT chk_packages_type CHECK (type IN ('monthly', 'pppoe', 'voucher'));

