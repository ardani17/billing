-- Migrasi: membuat tabel payment_gateway_configs untuk menyimpan konfigurasi payment gateway per tenant.
-- Mendukung dua provider: Xendit dan Midtrans. Setiap tenant bisa memiliki kedua provider aktif.
-- API key dan webhook secret disimpan dalam bentuk terenkripsi (AES-256-GCM).
-- Data dilindungi oleh RLS.

CREATE TABLE payment_gateway_configs (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL REFERENCES tenants(id),
    gateway_provider          VARCHAR(20) NOT NULL,
    is_active                 BOOLEAN NOT NULL DEFAULT true,
    api_key_encrypted         TEXT NOT NULL,
    webhook_secret_encrypted  TEXT NOT NULL,
    enabled_methods           JSONB NOT NULL DEFAULT '[]'::jsonb,
    payment_link_expiry_days  INTEGER NOT NULL DEFAULT 7,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_payment_gateway_configs_provider CHECK (
        gateway_provider IN ('xendit', 'midtrans')
    ),
    CONSTRAINT chk_payment_gateway_configs_expiry_days CHECK (
        payment_link_expiry_days >= 1 AND payment_link_expiry_days <= 30
    )
);

-- Aktifkan RLS pada tabel payment_gateway_configs
ALTER TABLE payment_gateway_configs ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY payment_gateway_configs_tenant_policy ON payment_gateway_configs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY payment_gateway_configs_tenant_insert ON payment_gateway_configs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: satu konfigurasi per provider per tenant
ALTER TABLE payment_gateway_configs ADD CONSTRAINT uq_payment_gateway_configs_tenant_provider
    UNIQUE (tenant_id, gateway_provider);

-- Partial index: konfigurasi aktif per tenant untuk query cepat
CREATE INDEX idx_payment_gateway_configs_tenant_active
    ON payment_gateway_configs(tenant_id, is_active)
    WHERE is_active = true;
