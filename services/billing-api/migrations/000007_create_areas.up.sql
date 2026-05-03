-- Migrasi: membuat tabel areas untuk grouping pelanggan per wilayah.
-- Setiap area dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE areas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    odp_id      VARCHAR(100),
    center_lat  DECIMAL(10, 7),
    center_lng  DECIMAL(10, 7),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel areas
ALTER TABLE areas ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON areas
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON areas
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: nama area unik per tenant
ALTER TABLE areas ADD CONSTRAINT uq_areas_tenant_name UNIQUE (tenant_id, name);

-- Index pada tenant_id untuk performa query
CREATE INDEX idx_areas_tenant_id ON areas(tenant_id);
