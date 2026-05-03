-- Migrasi: membuat tabel custom_report_templates untuk menyimpan template laporan custom per tenant.
-- Setiap template dimiliki oleh satu tenant dan dibuat oleh satu user.
-- Mendukung konfigurasi metrik (max 3), dimensi grouping, tipe tampilan, dan periode default.
-- Dilindungi oleh RLS.

CREATE TABLE custom_report_templates (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    name                  VARCHAR(255) NOT NULL,
    metrics               JSONB NOT NULL DEFAULT '[]',
    group_by              VARCHAR(50) NOT NULL,
    sub_group_by          VARCHAR(50),
    display_type          VARCHAR(20) NOT NULL CHECK (display_type IN ('table', 'bar_chart', 'line_chart', 'pie_chart')),
    default_period_range  VARCHAR(20),
    created_by_id         UUID NOT NULL REFERENCES users(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel custom_report_templates
ALTER TABLE custom_report_templates ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY custom_report_templates_tenant_isolation ON custom_report_templates
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY custom_report_templates_tenant_insert ON custom_report_templates
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index: pencarian template per tenant
CREATE INDEX idx_custom_report_templates_tenant
    ON custom_report_templates (tenant_id);
