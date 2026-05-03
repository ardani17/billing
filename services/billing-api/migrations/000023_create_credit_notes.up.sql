-- Migrasi: membuat tabel credit_notes untuk menyimpan nota kredit penyesuaian invoice.
-- Setiap credit note dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE credit_notes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    credit_note_number  VARCHAR(50) NOT NULL,
    invoice_id          UUID NOT NULL REFERENCES invoices(id),
    amount              BIGINT NOT NULL,
    reason              TEXT NOT NULL,
    apply_to_credit     BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id       VARCHAR(255) NOT NULL,
    created_by_name     VARCHAR(255) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_credit_notes_amount CHECK (amount > 0)
);

ALTER TABLE credit_notes ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON credit_notes
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY tenant_insert ON credit_notes
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE credit_notes ADD CONSTRAINT uq_credit_notes_tenant_number
    UNIQUE (tenant_id, credit_note_number);

CREATE INDEX idx_credit_notes_tenant_invoice ON credit_notes(tenant_id, invoice_id);
