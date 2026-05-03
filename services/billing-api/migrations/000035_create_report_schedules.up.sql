-- Migrasi: membuat tabel report_schedules untuk menyimpan jadwal laporan otomatis per tenant.
-- Setiap jadwal dimiliki oleh satu tenant dan dibuat oleh satu user.
-- Mendukung tipe jadwal harian/mingguan/bulanan, format PDF/Excel, dan daftar penerima.
-- Dilindungi oleh RLS.

CREATE TABLE report_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    report_type     VARCHAR(50) NOT NULL,
    schedule_type   VARCHAR(20) NOT NULL CHECK (schedule_type IN ('daily', 'weekly', 'monthly')),
    format          VARCHAR(10) NOT NULL CHECK (format IN ('pdf', 'xlsx')),
    recipients      JSONB NOT NULL DEFAULT '[]',
    filters         JSONB NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_by_id   UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel report_schedules
ALTER TABLE report_schedules ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY report_schedules_tenant_isolation ON report_schedules
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY report_schedules_tenant_insert ON report_schedules
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index: pencarian jadwal aktif per tenant
CREATE INDEX idx_report_schedules_tenant
    ON report_schedules (tenant_id)
    WHERE is_active = true;
