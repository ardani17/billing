-- Migrasi: membuat tabel vouchers untuk menyimpan kode voucher internet.
-- Voucher memiliki lifecycle status (tersedia → terjual → aktif → selesai/expired/void)
-- dengan snapshot harga saat pembelian dan audit trail.
-- Setiap voucher dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE vouchers (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id),
    code                    VARCHAR(255) NOT NULL,
    package_id              UUID NOT NULL REFERENCES packages(id),
    reseller_id             UUID REFERENCES resellers(id),
    status                  VARCHAR(20) NOT NULL DEFAULT 'tersedia',
    sell_price_snapshot     BIGINT,
    reseller_price_snapshot BIGINT,
    purchased_at            TIMESTAMPTZ,
    activated_at            TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ,
    voided_at               TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_vouchers_status CHECK (
        status IN ('tersedia', 'terjual', 'aktif', 'selesai', 'expired', 'void')
    )
);

-- Aktifkan RLS pada tabel vouchers
ALTER TABLE vouchers ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON vouchers
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON vouchers
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: kode voucher unik per tenant
ALTER TABLE vouchers ADD CONSTRAINT uq_vouchers_tenant_code
    UNIQUE (tenant_id, code);

-- Composite indexes untuk performa query
CREATE INDEX idx_vouchers_tenant_status ON vouchers(tenant_id, status);
CREATE INDEX idx_vouchers_tenant_package ON vouchers(tenant_id, package_id);
CREATE INDEX idx_vouchers_tenant_reseller ON vouchers(tenant_id, reseller_id);
CREATE INDEX idx_vouchers_tenant_status_expires ON vouchers(tenant_id, status, expires_at);
