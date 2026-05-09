-- Migration: create_routers_table
-- Tabel routers: menyimpan data perangkat MikroTik yang terdaftar per tenant.
-- Setiap tenant bisa memiliki banyak router dengan nama unik.

CREATE TABLE routers (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL REFERENCES tenants(id),
    name                      VARCHAR(100) NOT NULL,
    host                      VARCHAR(255) NOT NULL,
    port                      INTEGER NOT NULL DEFAULT 8728,
    username                  VARCHAR(100) NOT NULL,
    password_encrypted        TEXT NOT NULL,
    use_ssl                   BOOLEAN NOT NULL DEFAULT false,
    service_types             JSONB NOT NULL DEFAULT '["pppoe"]',
    router_os_version         VARCHAR(20),
    board_name                VARCHAR(100),
    cpu_count                 INTEGER,
    total_ram_mb              INTEGER,
    identity                  VARCHAR(255),
    status                    VARCHAR(20) NOT NULL DEFAULT 'offline',
    health_check_interval_sec INTEGER NOT NULL DEFAULT 60,
    last_online_at            TIMESTAMPTZ,
    last_checked_at           TIMESTAMPTZ,
    last_uptime_sec           BIGINT,
    failure_count             INTEGER NOT NULL DEFAULT 0,
    notes                     TEXT,
    deleted_at                TIMESTAMPTZ,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique constraint: nama router unik per tenant (exclude hapus lunak)
CREATE UNIQUE INDEX idx_routers_tenant_name
    ON routers (tenant_id, name)
    WHERE deleted_at IS NULL;

-- Index untuk kueri per tenant
CREATE INDEX idx_routers_tenant_id ON routers (tenant_id) WHERE deleted_at IS NULL;

-- Index untuk health cek kueri (router aktif)
CREATE INDEX idx_routers_status ON routers (status) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE routers ENABLE ROW LEVEL SECURITY;

CREATE POLICY routers_tenant_isolation ON routers
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
