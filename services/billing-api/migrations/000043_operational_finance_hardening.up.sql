ALTER TABLE inventory_movements
    DROP CONSTRAINT IF EXISTS chk_inventory_movement_type,
    ADD CONSTRAINT chk_inventory_movement_type CHECK (
        movement_type IN ('purchase','install','return','transfer','adjustment','damaged','lost','rma','retired')
    );

CREATE TABLE IF NOT EXISTS cashflow_manual_transactions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    direction        VARCHAR(10) NOT NULL CHECK (direction IN ('in','out')),
    category         VARCHAR(100) NOT NULL,
    description      TEXT NOT NULL,
    amount           BIGINT NOT NULL CHECK (amount > 0),
    transaction_date DATE NOT NULL,
    created_by_id    UUID NOT NULL REFERENCES users(id),
    deleted_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cashflow_manual_tenant_date
    ON cashflow_manual_transactions(tenant_id, transaction_date DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cashflow_manual_tenant_direction
    ON cashflow_manual_transactions(tenant_id, direction, transaction_date DESC)
    WHERE deleted_at IS NULL;

ALTER TABLE cashflow_manual_transactions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON cashflow_manual_transactions;
CREATE POLICY tenant_isolation ON cashflow_manual_transactions
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
DROP POLICY IF EXISTS tenant_insert ON cashflow_manual_transactions;
CREATE POLICY tenant_insert ON cashflow_manual_transactions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
