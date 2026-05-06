DROP POLICY IF EXISTS tenant_insert ON cashflow_manual_transactions;
DROP POLICY IF EXISTS tenant_isolation ON cashflow_manual_transactions;
DROP INDEX IF EXISTS idx_cashflow_manual_tenant_direction;
DROP INDEX IF EXISTS idx_cashflow_manual_tenant_date;
DROP TABLE IF EXISTS cashflow_manual_transactions;

ALTER TABLE inventory_movements
    DROP CONSTRAINT IF EXISTS chk_inventory_movement_type,
    ADD CONSTRAINT chk_inventory_movement_type CHECK (
        movement_type IN ('purchase','install','return','transfer','adjustment','damaged','lost')
    );
