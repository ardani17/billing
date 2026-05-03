-- Migrasi: membuat tabel debit_notes dan debit_note_items untuk menyimpan nota debit.
-- Setiap debit note dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE debit_notes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    debit_note_number   VARCHAR(50) NOT NULL,
    customer_id         UUID NOT NULL REFERENCES customers(id),
    due_date            DATE NOT NULL,
    total_amount        BIGINT NOT NULL DEFAULT 0,
    invoice_id          UUID REFERENCES invoices(id),
    created_by_id       VARCHAR(255) NOT NULL,
    created_by_name     VARCHAR(255) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_debit_notes_total_amount CHECK (total_amount >= 0)
);

ALTER TABLE debit_notes ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON debit_notes
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY tenant_insert ON debit_notes
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE debit_notes ADD CONSTRAINT uq_debit_notes_tenant_number
    UNIQUE (tenant_id, debit_note_number);

CREATE INDEX idx_debit_notes_tenant_customer ON debit_notes(tenant_id, customer_id);

-- Tabel debit_note_items untuk menyimpan item dalam debit note
CREATE TABLE debit_note_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    debit_note_id   UUID NOT NULL REFERENCES debit_notes(id),
    description     VARCHAR(500) NOT NULL,
    amount          BIGINT NOT NULL,

    CONSTRAINT chk_debit_note_items_amount CHECK (amount > 0)
);

CREATE INDEX idx_debit_note_items_debit_note ON debit_note_items(debit_note_id);
