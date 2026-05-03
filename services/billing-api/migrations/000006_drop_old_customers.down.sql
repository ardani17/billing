-- Rollback migrasi: membuat ulang tabel customers lama (sample dari monorepo-setup).
-- Mengembalikan schema asli dari migrasi 000002.

CREATE TABLE customers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255),
    phone      VARCHAR(50),
    status     VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel customers untuk isolasi data antar tenant
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;

-- Policy: hanya baris dengan tenant_id yang cocok dengan session variable
-- app.tenant_id yang bisa diakses (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON customers
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy untuk INSERT: memastikan tenant_id yang di-insert sesuai dengan session variable
CREATE POLICY tenant_insert ON customers
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index pada tenant_id untuk performa query dan RLS filtering
CREATE INDEX idx_customers_tenant_id ON customers(tenant_id);

-- Index komposit untuk query pelanggan berdasarkan status per tenant
CREATE INDEX idx_customers_status ON customers(tenant_id, status);
