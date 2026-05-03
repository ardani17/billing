-- Migrasi: membuat tabel kpi_targets untuk menyimpan target KPI bisnis per tenant.
-- Setiap tenant hanya memiliki satu baris kpi_targets (UNIQUE pada tenant_id).
-- Semua kolom target bersifat nullable (target opsional).
-- Dilindungi oleh RLS.

CREATE TABLE kpi_targets (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                       UUID NOT NULL UNIQUE REFERENCES tenants(id),
    monthly_revenue_target          BIGINT,
    collection_rate_target          NUMERIC(5,2),
    max_receivables                 BIGINT,
    new_customers_monthly_target    INTEGER,
    max_churn_rate                  NUMERIC(5,2),
    total_customers_target          INTEGER,
    sla_uptime_target               NUMERIC(5,2),
    max_active_alarms               INTEGER,
    min_signal_quality_percentage   NUMERIC(5,2),
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel kpi_targets
ALTER TABLE kpi_targets ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY kpi_targets_tenant_isolation ON kpi_targets
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY kpi_targets_tenant_insert ON kpi_targets
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
