-- Migrasi: membuat tabel reseller_transactions untuk mencatat semua transaksi keuangan reseller.
-- Setiap transaksi mencatat balance_before dan balance_after untuk audit trail.

CREATE TABLE reseller_transactions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    reseller_id    UUID NOT NULL REFERENCES resellers(id),
    type           VARCHAR(20) NOT NULL,
    amount         BIGINT NOT NULL,
    balance_before BIGINT NOT NULL,
    balance_after  BIGINT NOT NULL,
    reference_id   UUID,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_reseller_tx_type CHECK (
        type IN ('deposit', 'purchase', 'refund', 'withdraw')
    )
);

-- Aktifkan RLS pada tabel reseller_transactions
ALTER TABLE reseller_transactions ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON reseller_transactions
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON reseller_transactions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite indexes untuk performa kueri
CREATE INDEX idx_reseller_tx_reseller ON reseller_transactions(tenant_id, reseller_id);
CREATE INDEX idx_reseller_tx_reseller_created ON reseller_transactions(tenant_id, reseller_id, created_at);
