-- Migrasi: membuat tabel billing_settings untuk menyimpan konfigurasi billing per tenant.
-- Mendukung pengaturan generate_days, grace_period, tax, penalty, invoice prefix, timezone, dan isolir.
-- Setiap tenant hanya memiliki satu baris billing_settings (UNIQUE pada tenant_id).
-- Data dilindungi oleh RLS.

CREATE TABLE billing_settings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    generate_days         INTEGER NOT NULL DEFAULT 5,
    grace_period_days     INTEGER NOT NULL DEFAULT 7,
    suspend_days          INTEGER NOT NULL DEFAULT 30,
    tax_enabled           BOOLEAN NOT NULL DEFAULT FALSE,
    tax_rate              DECIMAL(5,2) NOT NULL DEFAULT 11.00,
    penalty_enabled       BOOLEAN NOT NULL DEFAULT FALSE,
    penalty_type          VARCHAR(20) NOT NULL DEFAULT 'fixed',
    penalty_amount        BIGINT NOT NULL DEFAULT 0,
    penalty_percentage    DECIMAL(5,2) NOT NULL DEFAULT 0,
    penalty_daily_amount  BIGINT NOT NULL DEFAULT 0,
    penalty_max_amount    BIGINT NOT NULL DEFAULT 0,
    invoice_prefix        VARCHAR(20) NOT NULL DEFAULT 'INV',
    new_customer_billing  VARCHAR(20) NOT NULL DEFAULT 'prorate',
    timezone              VARCHAR(50) NOT NULL DEFAULT 'Asia/Jakarta',
    auto_isolir           BOOLEAN NOT NULL DEFAULT TRUE,
    auto_open_isolir      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_billing_settings_penalty_type CHECK (
        penalty_type IN ('fixed', 'percentage', 'daily')
    ),
    CONSTRAINT chk_billing_settings_new_customer_billing CHECK (
        new_customer_billing IN ('prorate', 'full_month')
    ),
    CONSTRAINT chk_billing_settings_generate_days CHECK (
        generate_days >= 1 AND generate_days <= 14
    )
);

-- Aktifkan RLS pada tabel billing_settings
ALTER TABLE billing_settings ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON billing_settings
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON billing_settings
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: satu billing_settings per tenant
ALTER TABLE billing_settings ADD CONSTRAINT uq_billing_settings_tenant_id
    UNIQUE (tenant_id);
