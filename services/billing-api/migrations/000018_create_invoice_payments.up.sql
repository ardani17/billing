-- Migrasi: membuat tabel invoice_payments untuk menyimpan catatan pembayaran terhadap invoice.
-- Mendukung berbagai metode pembayaran (tunai, transfer, xendit, midtrans, lainnya),
-- void/pembatalan pembayaran, dan pelacakan siapa yang mencatat pembayaran.
-- Setiap invoice_payment dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE invoice_payments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    invoice_id       UUID NOT NULL REFERENCES invoices(id),
    amount           BIGINT NOT NULL,
    payment_method   VARCHAR(20) NOT NULL,
    payment_date     DATE NOT NULL,
    reference_number VARCHAR(255),
    notes            TEXT,
    recorded_by_id   UUID NOT NULL,
    recorded_by_name VARCHAR(255) NOT NULL,
    voided           BOOLEAN NOT NULL DEFAULT FALSE,
    voided_at        TIMESTAMPTZ,
    voided_by        VARCHAR(255),
    void_reason      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_invoice_payments_payment_method CHECK (
        payment_method IN ('tunai', 'transfer', 'xendit', 'midtrans', 'lainnya')
    ),
    CONSTRAINT chk_invoice_payments_amount CHECK (amount > 0)
);

-- Aktifkan RLS pada tabel invoice_payments
ALTER TABLE invoice_payments ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON invoice_payments
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON invoice_payments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite indexes untuk performa kueri
CREATE INDEX idx_invoice_payments_tenant_invoice ON invoice_payments(tenant_id, invoice_id);
CREATE INDEX idx_invoice_payments_tenant_payment_date ON invoice_payments(tenant_id, payment_date);
