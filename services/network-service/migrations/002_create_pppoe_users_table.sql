-- Migration: create_pppoe_users_table
CREATE TABLE pppoe_users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    customer_id       UUID NOT NULL,
    router_id         UUID NOT NULL REFERENCES routers(id),
    username          VARCHAR(100) NOT NULL,
    password_encrypted TEXT NOT NULL,
    profile_name      VARCHAR(100) NOT NULL,
    service           VARCHAR(20) NOT NULL DEFAULT 'pppoe',
    remote_address    VARCHAR(45),
    comment           TEXT NOT NULL,
    disabled          BOOLEAN NOT NULL DEFAULT false,
    use_simple_queue  BOOLEAN NOT NULL DEFAULT false,
    status            VARCHAR(20) NOT NULL DEFAULT 'active',
    last_sync_at      TIMESTAMPTZ,
    sync_status       VARCHAR(20) NOT NULL DEFAULT 'pending_create',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

-- Unique constraint: username unik per router (exclude hapus lunak)
CREATE UNIQUE INDEX idx_pppoe_users_router_username
    ON pppoe_users (router_id, username)
    WHERE deleted_at IS NULL;

-- Index untuk kueri per tenant
CREATE INDEX idx_pppoe_users_tenant_id
    ON pppoe_users (tenant_id) WHERE deleted_at IS NULL;

-- Index untuk lookup by customer_id
CREATE INDEX idx_pppoe_users_customer_id
    ON pppoe_users (customer_id) WHERE deleted_at IS NULL;

-- Index untuk sync job (per router)
CREATE INDEX idx_pppoe_users_router_sync
    ON pppoe_users (router_id, sync_status) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE pppoe_users ENABLE ROW LEVEL SECURITY;

CREATE POLICY pppoe_users_tenant_isolation ON pppoe_users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
