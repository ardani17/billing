-- Migration: create_router_backups
-- Manual on-demand MikroTik export backup metadata dan content.

CREATE TABLE router_backups (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    router_id    UUID NOT NULL REFERENCES routers(id),
    file_name    VARCHAR(180) NOT NULL,
    format       VARCHAR(20) NOT NULL DEFAULT 'rsc',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    checksum     VARCHAR(80),
    content      TEXT NOT NULL,
    created_by   UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_router_backups_tenant
    ON router_backups (tenant_id, created_at DESC);

CREATE INDEX idx_router_backups_router
    ON router_backups (router_id, created_at DESC);

ALTER TABLE router_backups ENABLE ROW LEVEL SECURITY;

CREATE POLICY router_backups_tenant_isolation ON router_backups
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
