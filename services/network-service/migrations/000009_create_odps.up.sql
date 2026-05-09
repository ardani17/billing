-- Migration: create_odps_table
-- Tabel ODP (Optical Distribution Point) / splitter per tenant.
-- Setiap ODP terhubung ke satu OLT pada PON port tertentu.

CREATE TABLE odps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    olt_id          UUID NOT NULL REFERENCES olts(id),
    pon_port_index  INTEGER NOT NULL,
    name            VARCHAR(100) NOT NULL,
    splitter_type   VARCHAR(10) NOT NULL,
    capacity        INTEGER NOT NULL,
    used_ports      INTEGER NOT NULL DEFAULT 0,
    address         TEXT,
    latitude        DECIMAL(10,7),
    longitude       DECIMAL(10,7),
    notes           TEXT,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique constraint: nama ODP unik per tenant (exclude hapus lunak)
CREATE UNIQUE INDEX idx_odps_tenant_name
    ON odps (tenant_id, name)
    WHERE deleted_at IS NULL;

-- Index untuk kueri per OLT dan port
CREATE INDEX idx_odps_olt_port
    ON odps (olt_id, pon_port_index) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE odps ENABLE ROW LEVEL SECURITY;

CREATE POLICY odps_tenant_isolation ON odps
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
