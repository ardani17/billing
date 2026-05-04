-- Migration: create_mikrotik_command_audit_logs
-- Append-only audit trail for RouterOS changing commands.

CREATE TABLE mikrotik_command_audit_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    router_id      UUID NOT NULL REFERENCES routers(id),
    user_id        UUID,
    action         VARCHAR(80) NOT NULL,
    command        VARCHAR(160) NOT NULL,
    target_type    VARCHAR(80),
    target_id      VARCHAR(120),
    status         VARCHAR(30) NOT NULL,
    error_message  TEXT,
    remote_addr    VARCHAR(100),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mikrotik_command_audit_tenant
    ON mikrotik_command_audit_logs (tenant_id, created_at DESC);

CREATE INDEX idx_mikrotik_command_audit_router
    ON mikrotik_command_audit_logs (router_id, created_at DESC);

ALTER TABLE mikrotik_command_audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY mikrotik_command_audit_tenant_isolation ON mikrotik_command_audit_logs
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
