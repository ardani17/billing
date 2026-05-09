-- Migrasi: membuat tabel invoice_items untuk menyimpan baris item dalam invoice.
-- Setiap item memiliki tipe (bulanan, installation, prorate_charge, dll),
-- deskripsi, quantity, unit_price, dan nominal.
-- Setiap invoice_item dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE invoice_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    invoice_id  UUID NOT NULL REFERENCES invoices(id),
    item_type   VARCHAR(20) NOT NULL,
    description VARCHAR(500) NOT NULL,
    quantity    INTEGER NOT NULL DEFAULT 1,
    unit_price  BIGINT NOT NULL,
    amount      BIGINT NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_invoice_items_item_type CHECK (
        item_type IN ('monthly', 'installation', 'prorate_charge', 'prorate_credit', 'penalty', 'tax', 'discount', 'recurring', 'custom', 'credit_applied')
    )
);

-- Aktifkan RLS pada tabel invoice_items
ALTER TABLE invoice_items ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON invoice_items
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON invoice_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite index untuk performa kueri
CREATE INDEX idx_invoice_items_tenant_invoice ON invoice_items(tenant_id, invoice_id);
