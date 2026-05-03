-- Migrasi: membuat tabel invoice_sequences untuk menyimpan sequence nomor invoice per tenant per bulan.
-- Digunakan untuk auto-increment nomor invoice secara atomik (SELECT FOR UPDATE).
-- Satu row per kombinasi tenant/year/month, dijamin unik via UNIQUE constraint.
-- Setiap sequence dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE invoice_sequences (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    year        INTEGER NOT NULL,
    month       INTEGER NOT NULL,
    last_seq    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel invoice_sequences
ALTER TABLE invoice_sequences ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON invoice_sequences
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON invoice_sequences
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: satu sequence per tenant per bulan per tahun
ALTER TABLE invoice_sequences ADD CONSTRAINT uq_invoice_sequences_tenant_year_month
    UNIQUE (tenant_id, year, month);
