-- Migrasi: membuat tabel resellers untuk menyimpan data reseller voucher.
-- Reseller adalah pihak ketiga yang membeli voucher dan menjualnya ke end-user.
-- Setiap reseller dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE resellers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id),
    name                 VARCHAR(255) NOT NULL,
    phone                VARCHAR(20) NOT NULL,
    email                VARCHAR(255),
    address              TEXT,
    password_hash        VARCHAR(255) NOT NULL,
    balance              BIGINT NOT NULL DEFAULT 0,
    daily_purchase_limit INTEGER NOT NULL DEFAULT 0,
    status               VARCHAR(20) NOT NULL DEFAULT 'aktif',
    last_login           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_resellers_status CHECK (
        status IN ('aktif', 'suspended', 'nonaktif')
    ),
    CONSTRAINT chk_resellers_balance CHECK (balance >= 0),
    CONSTRAINT chk_resellers_daily_purchase_limit CHECK (daily_purchase_limit >= 0)
);

-- Aktifkan RLS pada tabel resellers
ALTER TABLE resellers ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON resellers
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON resellers
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: nomor telepon unik per tenant
ALTER TABLE resellers ADD CONSTRAINT uq_resellers_tenant_phone
    UNIQUE (tenant_id, phone);

-- Composite indexes untuk performa kueri
CREATE INDEX idx_resellers_tenant_status ON resellers(tenant_id, status);
CREATE INDEX idx_resellers_tenant_phone ON resellers(tenant_id, phone);
