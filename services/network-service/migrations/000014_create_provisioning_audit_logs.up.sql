-- Migration: create_provisioning_audit_logs_table
-- Tabel audit trail append-only untuk semua provisioning command.
-- Mencatat setiap CLI command yang dikirim ke OLT beserta response-nya.
-- Tidak ada operasi update atau delete pada tabel ini.

CREATE TABLE provisioning_audit_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    olt_id            UUID NOT NULL REFERENCES olts(id),
    ont_id            UUID REFERENCES onts(id),
    action            VARCHAR(50) NOT NULL,
    commands_sent     JSONB NOT NULL DEFAULT '[]',
    command_responses JSONB NOT NULL DEFAULT '[]',
    status            VARCHAR(20) NOT NULL,
    error_message     TEXT,
    performed_by      VARCHAR(100) NOT NULL,
    correlation_id    UUID NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk query per OLT
CREATE INDEX idx_audit_olt
    ON provisioning_audit_logs (olt_id, created_at DESC);

-- Index untuk query per ONT
CREATE INDEX idx_audit_ont
    ON provisioning_audit_logs (ont_id, created_at DESC)
    WHERE ont_id IS NOT NULL;

-- Index untuk query per tenant dan tanggal
CREATE INDEX idx_audit_tenant
    ON provisioning_audit_logs (tenant_id, created_at DESC);

-- Index untuk filter by action
CREATE INDEX idx_audit_action
    ON provisioning_audit_logs (action, created_at DESC);

-- Row-Level Security
ALTER TABLE provisioning_audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY audit_tenant_isolation ON provisioning_audit_logs
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
