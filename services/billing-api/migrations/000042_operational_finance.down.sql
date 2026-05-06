DROP POLICY IF EXISTS tenant_insert ON inventory_movements;
DROP POLICY IF EXISTS tenant_isolation ON inventory_movements;
DROP INDEX IF EXISTS idx_inventory_movements_tenant_type;
DROP INDEX IF EXISTS idx_inventory_movements_tenant_item;
DROP TABLE IF EXISTS inventory_movements;

DROP POLICY IF EXISTS tenant_insert ON inventory_assets;
DROP POLICY IF EXISTS tenant_isolation ON inventory_assets;
DROP INDEX IF EXISTS idx_inventory_assets_tenant_status;
DROP INDEX IF EXISTS idx_inventory_assets_tenant_item;
DROP INDEX IF EXISTS uq_inventory_assets_tenant_serial;
DROP TABLE IF EXISTS inventory_assets;

DROP POLICY IF EXISTS tenant_insert ON inventory_items;
DROP POLICY IF EXISTS tenant_isolation ON inventory_items;
DROP INDEX IF EXISTS idx_inventory_items_tenant_category;
DROP INDEX IF EXISTS uq_inventory_items_tenant_name;
DROP TABLE IF EXISTS inventory_items;

ALTER TABLE expenses
    DROP COLUMN IF EXISTS inventory_movement_id,
    DROP COLUMN IF EXISTS attachment_url,
    DROP COLUMN IF EXISTS reference_number,
    DROP COLUMN IF EXISTS vendor_name,
    DROP COLUMN IF EXISTS payment_method;
