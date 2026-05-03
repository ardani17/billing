-- Migrasi: membuat tabel voucher_audit_logs untuk menyimpan catatan lifecycle voucher.
-- Tabel ini bersifat append-only — hanya INSERT dan SELECT yang diizinkan.
-- Setiap log dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE voucher_audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    voucher_id UUID NOT NULL REFERENCES vouchers(id),
    action     VARCHAR(255) NOT NULL,
    actor_id   VARCHAR(255) NOT NULL,
    actor_name VARCHAR(255) NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel voucher_audit_logs
ALTER TABLE voucher_audit_logs ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT only — append-only table)
CREATE POLICY tenant_isolation ON voucher_audit_logs
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session (append-only table)
CREATE POLICY tenant_insert ON voucher_audit_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite indexes untuk performa query
CREATE INDEX idx_voucher_audit_logs_tenant_voucher ON voucher_audit_logs(tenant_id, voucher_id);
CREATE INDEX idx_voucher_audit_logs_tenant_created ON voucher_audit_logs(tenant_id, created_at);
