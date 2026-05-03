-- Migrasi: membuat tabel report_jobs untuk menyimpan job export laporan per tenant.
-- Setiap job dimiliki oleh satu tenant dan diminta oleh satu user.
-- Mendukung status tracking (pending, processing, completed, failed) dan download URL.
-- Dilindungi oleh RLS.

CREATE TABLE report_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    report_type     VARCHAR(50) NOT NULL,
    format          VARCHAR(10) NOT NULL,
    filters         JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    download_url    TEXT,
    error           TEXT,
    requested_by    UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel report_jobs
ALTER TABLE report_jobs ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY report_jobs_tenant_isolation ON report_jobs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY report_jobs_tenant_insert ON report_jobs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index: pencarian job per tenant dan status
CREATE INDEX idx_report_jobs_tenant_status
    ON report_jobs (tenant_id, status);
