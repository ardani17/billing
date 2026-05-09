-- Migrasi: membuat tabel invoices untuk menyimpan tagihan pelanggan.
-- Mendukung status lifecycle (belum_bayar, terlambat, lunas, bayar_sebagian, batal, prorate),
-- optimistic locking via version, dan prepaid billing.
-- Setiap invoice dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE invoices (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    customer_id       UUID NOT NULL REFERENCES customers(id),
    invoice_number    VARCHAR(50) NOT NULL,
    period_month      INTEGER NOT NULL,
    period_year       INTEGER NOT NULL,
    due_date          DATE NOT NULL,
    subtotal          BIGINT NOT NULL DEFAULT 0,
    tax_amount        BIGINT NOT NULL DEFAULT 0,
    penalty_amount    BIGINT NOT NULL DEFAULT 0,
    discount_amount   BIGINT NOT NULL DEFAULT 0,
    credit_applied    BIGINT NOT NULL DEFAULT 0,
    total_amount      BIGINT NOT NULL DEFAULT 0,
    paid_amount       BIGINT NOT NULL DEFAULT 0,
    status            VARCHAR(20) NOT NULL DEFAULT 'belum_bayar',
    notes             TEXT,
    is_prepaid        BOOLEAN NOT NULL DEFAULT FALSE,
    prepaid_months    INTEGER,
    version           INTEGER NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_invoices_status CHECK (
        status IN ('belum_bayar', 'terlambat', 'lunas', 'bayar_sebagian', 'batal', 'prorate')
    ),
    CONSTRAINT chk_invoices_subtotal CHECK (subtotal >= 0),
    CONSTRAINT chk_invoices_tax_amount CHECK (tax_amount >= 0),
    CONSTRAINT chk_invoices_penalty_amount CHECK (penalty_amount >= 0),
    CONSTRAINT chk_invoices_discount_amount CHECK (discount_amount >= 0),
    CONSTRAINT chk_invoices_credit_applied CHECK (credit_applied >= 0),
    CONSTRAINT chk_invoices_total_amount CHECK (total_amount >= 0),
    CONSTRAINT chk_invoices_paid_amount CHECK (paid_amount >= 0),
    CONSTRAINT chk_invoices_period_month CHECK (period_month >= 1 AND period_month <= 12)
);

-- Aktifkan RLS pada tabel invoices
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON invoices
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON invoices
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: nomor invoice unik per tenant
ALTER TABLE invoices ADD CONSTRAINT uq_invoices_tenant_invoice_number
    UNIQUE (tenant_id, invoice_number);

-- Composite indexes untuk performa kueri
CREATE INDEX idx_invoices_tenant_status ON invoices(tenant_id, status);
CREATE INDEX idx_invoices_tenant_customer ON invoices(tenant_id, customer_id);
CREATE INDEX idx_invoices_tenant_period ON invoices(tenant_id, period_year, period_month);
CREATE INDEX idx_invoices_tenant_due_date_status ON invoices(tenant_id, due_date, status);
