-- Migration: create_static_ip_assignments
-- Managed static IP customers for MikroTik address-list and optional simple queue.

CREATE TABLE static_ip_assignments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    router_id         UUID NOT NULL REFERENCES routers(id),
    customer_id       UUID,
    ip_address        INET NOT NULL,
    address_list      VARCHAR(120) NOT NULL DEFAULT 'ISPBoss:static-active',
    queue_name        VARCHAR(120),
    rate_limit        VARCHAR(80),
    comment           TEXT NOT NULL,
    status            VARCHAR(30) NOT NULL DEFAULT 'active',
    last_sync_at      TIMESTAMPTZ,
    sync_status       VARCHAR(30) NOT NULL DEFAULT 'pending_create',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_static_ip_assignments_router_ip
    ON static_ip_assignments (router_id, ip_address)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_static_ip_assignments_tenant
    ON static_ip_assignments (tenant_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_static_ip_assignments_router
    ON static_ip_assignments (router_id)
    WHERE deleted_at IS NULL;

ALTER TABLE static_ip_assignments ENABLE ROW LEVEL SECURITY;

CREATE POLICY static_ip_assignments_tenant_isolation ON static_ip_assignments
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
