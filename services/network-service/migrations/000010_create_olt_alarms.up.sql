-- Migration: create_olt_alarms_table
-- Tabel alarm OLT dengan retensi 90 hari.
-- Menyimpan alarm dari SNMP trap dan polling.

CREATE TABLE olt_alarms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    olt_id          UUID NOT NULL REFERENCES olts(id),
    pon_port_index  INTEGER,
    ont_index       INTEGER,
    alarm_type      VARCHAR(50) NOT NULL,
    severity        VARCHAR(20) NOT NULL,
    message         TEXT,
    source          VARCHAR(20) NOT NULL DEFAULT 'polling',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    cleared_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk query alarm aktif per OLT
CREATE INDEX idx_olt_alarms_olt_active
    ON olt_alarms (olt_id, status) WHERE status = 'active';

-- Index untuk query per tenant
CREATE INDEX idx_olt_alarms_tenant
    ON olt_alarms (tenant_id, created_at DESC);

-- Index untuk purge job (alarm lebih tua dari 90 hari)
CREATE INDEX idx_olt_alarms_created_at
    ON olt_alarms (created_at);

-- Row-Level Security
ALTER TABLE olt_alarms ENABLE ROW LEVEL SECURITY;

CREATE POLICY olt_alarms_tenant_isolation ON olt_alarms
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
