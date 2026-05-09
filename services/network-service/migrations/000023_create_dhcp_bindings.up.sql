-- Migration: create_dhcp_bindings
-- Managed DHCP static bindings untuk MikroTik routers.

CREATE TABLE dhcp_bindings (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    router_id          UUID NOT NULL REFERENCES routers(id),
    customer_id        UUID,
    router_lease_id    VARCHAR(64),
    server             VARCHAR(100) NOT NULL DEFAULT 'all',
    mac_address        VARCHAR(32) NOT NULL,
    ip_address         INET NOT NULL,
    host_name          VARCHAR(100),
    comment            TEXT NOT NULL,
    disabled           BOOLEAN NOT NULL DEFAULT false,
    status             VARCHAR(20) NOT NULL DEFAULT 'active',
    last_sync_at       TIMESTAMPTZ,
    sync_status        VARCHAR(30) NOT NULL DEFAULT 'pending_create',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_dhcp_bindings_router_mac
    ON dhcp_bindings (router_id, lower(mac_address))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_dhcp_bindings_router_ip
    ON dhcp_bindings (router_id, ip_address)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_dhcp_bindings_tenant_id
    ON dhcp_bindings (tenant_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_dhcp_bindings_router_id
    ON dhcp_bindings (router_id)
    WHERE deleted_at IS NULL;

ALTER TABLE dhcp_bindings ENABLE ROW LEVEL SECURITY;

CREATE POLICY dhcp_bindings_tenant_isolation ON dhcp_bindings
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
