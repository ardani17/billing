-- Migrasi: membuat tabel invoice_audit_logs untuk menyimpan catatan lifecycle invoice.
-- Tabel ini bersifat append-only — hanya operasi SELECT dan INSERT yang diizinkan.
-- Tidak ada UPDATE atau DELETE untuk menjaga integritas audit trail.
-- Setiap audit log dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE invoice_audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    invoice_id  UUID NOT NULL REFERENCES invoices(id),
    action      VARCHAR(100) NOT NULL,
    actor_id    VARCHAR(255) NOT NULL,
    actor_name  VARCHAR(255) NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel invoice_audit_logs
ALTER TABLE invoice_audit_logs ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT only — append-only table)
CREATE POLICY tenant_select ON invoice_audit_logs
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session (append-only table)
CREATE POLICY tenant_insert ON invoice_audit_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite index untuk performa query
CREATE INDEX idx_invoice_audit_logs_tenant_invoice ON invoice_audit_logs(tenant_id, invoice_id);
