-- Migrasi: membuat tabel payment_links dan payment_link_invoices.
-- payment_links menyimpan payment link yang digenerate via Xendit/Midtrans.
-- payment_link_invoices adalah junction table untuk mendukung multi-invoice payment link.
-- Data dilindungi oleh RLS.

CREATE TABLE payment_links (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    customer_id       UUID NOT NULL REFERENCES customers(id),
    gateway_provider  VARCHAR(20) NOT NULL,
    gateway_config_id UUID NOT NULL REFERENCES payment_gateway_configs(id),
    external_id       VARCHAR(255) NOT NULL,
    payment_url       TEXT NOT NULL,
    amount            BIGINT NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at        TIMESTAMPTZ NOT NULL,
    paid_at           TIMESTAMPTZ,
    paid_method       VARCHAR(50),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_payment_links_provider CHECK (
        gateway_provider IN ('xendit', 'midtrans')
    ),
    CONSTRAINT chk_payment_links_amount CHECK (
        amount > 0
    ),
    CONSTRAINT chk_payment_links_status CHECK (
        status IN ('active', 'expired', 'paid', 'failed')
    )
);

-- Aktifkan RLS pada tabel payment_links
ALTER TABLE payment_links ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY payment_links_tenant_policy ON payment_links
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY payment_links_tenant_insert ON payment_links
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique index: external_id harus unik (ID dari gateway)
CREATE UNIQUE INDEX idx_payment_links_external_id ON payment_links(external_id);

-- Partial index: payment link aktif per customer untuk query cepat
CREATE INDEX idx_payment_links_customer_active
    ON payment_links(customer_id, status)
    WHERE status = 'active';

-- Partial index: payment link aktif yang akan expired untuk background job
CREATE INDEX idx_payment_links_expires_at
    ON payment_links(expires_at)
    WHERE status = 'active';

-- Junction table: relasi many-to-many antara payment_links dan invoices
CREATE TABLE payment_link_invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_link_id UUID NOT NULL REFERENCES payment_links(id) ON DELETE CASCADE,
    invoice_id      UUID NOT NULL REFERENCES invoices(id),

    -- Satu invoice hanya bisa terhubung sekali ke satu payment link
    CONSTRAINT uq_payment_link_invoices UNIQUE (payment_link_id, invoice_id)
);

-- Index pada FK payment_link_id untuk join query
CREATE INDEX idx_payment_link_invoices_link_id ON payment_link_invoices(payment_link_id);

-- Index pada FK invoice_id untuk reverse lookup
CREATE INDEX idx_payment_link_invoices_invoice_id ON payment_link_invoices(invoice_id);
