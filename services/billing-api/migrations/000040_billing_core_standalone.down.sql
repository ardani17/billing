-- Rollback dukungan Billing Core standalone.

UPDATE customers
SET connection_method = 'pppoe'
WHERE connection_method = 'manual';

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS chk_customers_connection_method;

ALTER TABLE customers
    ADD CONSTRAINT chk_customers_connection_method CHECK (
        connection_method IN ('pppoe', 'hotspot', 'dhcp_binding', 'static')
    );

UPDATE customers
SET latitude = 0
WHERE latitude IS NULL;

UPDATE customers
SET longitude = 0
WHERE longitude IS NULL;

ALTER TABLE customers
    ALTER COLUMN latitude SET NOT NULL,
    ALTER COLUMN longitude SET NOT NULL;

UPDATE packages
SET type = 'pppoe'
WHERE type = 'monthly';

ALTER TABLE packages
    DROP CONSTRAINT IF EXISTS chk_packages_type;

ALTER TABLE packages
    ADD CONSTRAINT chk_packages_type CHECK (type IN ('pppoe', 'voucher'));

