-- Migration: create_vlans_table
-- Tabel VLAN per OLT per tenant.
-- Menyimpan daftar VLAN yang tersedia pada setiap OLT.
-- Digunakan saat provisioning ONT untuk assignment VLAN.

CREATE TABLE vlans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    olt_id      UUID NOT NULL REFERENCES olts(id),
    vlan_id     INTEGER NOT NULL,
    name        VARCHAR(100) NOT NULL,
    vlan_type   VARCHAR(30) NOT NULL DEFAULT 'data',
    description TEXT,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique: VLAN ID unik per OLT (exclude hapus lunak)
CREATE UNIQUE INDEX idx_vlans_olt_vlanid
    ON vlans (olt_id, vlan_id)
    WHERE deleted_at IS NULL;

-- Index untuk kueri per OLT
CREATE INDEX idx_vlans_olt
    ON vlans (olt_id) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE vlans ENABLE ROW LEVEL SECURITY;

CREATE POLICY vlans_tenant_isolation ON vlans
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
