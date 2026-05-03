-- Migrasi: membuat tabel customer_recurring_items untuk menyimpan item berulang per pelanggan.
-- Item seperti sewa ONT, IP publik, dll yang otomatis ditambahkan ke invoice bulanan.
-- Setiap item dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE customer_recurring_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    customer_id   UUID NOT NULL REFERENCES customers(id),
    description   VARCHAR(500) NOT NULL,
    amount        BIGINT NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    start_date    DATE NOT NULL,
    end_date      DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_customer_recurring_items_amount CHECK (amount > 0)
);

-- Aktifkan RLS pada tabel customer_recurring_items
ALTER TABLE customer_recurring_items ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON customer_recurring_items
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON customer_recurring_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite index untuk performa query
CREATE INDEX idx_customer_recurring_items_tenant_customer_active
    ON customer_recurring_items(tenant_id, customer_id, is_active);
