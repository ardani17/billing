-- Migration: create_onts_table
-- Tabel ONT entity per tenant, linked ke OLT, PON port, customer, ODP, VLAN, service profile.
-- Menyimpan data lifecycle ONT termasuk status provisioning.

CREATE TABLE onts (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id),
    olt_id                 UUID NOT NULL REFERENCES olts(id),
    pon_port_index         INTEGER NOT NULL,
    ont_index              INTEGER NOT NULL,
    serial_number          VARCHAR(50) NOT NULL,
    customer_id            UUID,
    odp_id                 UUID REFERENCES odps(id),
    vlan_id                UUID REFERENCES vlans(id),
    service_profile_id     UUID REFERENCES service_profiles(id),
    status                 VARCHAR(30) NOT NULL DEFAULT 'registered',
    provisioning_state     VARCHAR(20) NOT NULL DEFAULT 'pending',
    description            TEXT,
    last_provisioned_at    TIMESTAMPTZ,
    last_decommissioned_at TIMESTAMPTZ,
    deleted_at             TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique: serial number unik per tenant (exclude soft-deleted)
CREATE UNIQUE INDEX idx_onts_tenant_sn
    ON onts (tenant_id, serial_number)
    WHERE deleted_at IS NULL;

-- Unique: posisi ONT unik per OLT port (exclude soft-deleted)
CREATE UNIQUE INDEX idx_onts_olt_port_index
    ON onts (olt_id, pon_port_index, ont_index)
    WHERE deleted_at IS NULL;

-- Index untuk query per OLT dan status
CREATE INDEX idx_onts_olt_status
    ON onts (olt_id, status) WHERE deleted_at IS NULL;

-- Index untuk query per customer
CREATE INDEX idx_onts_customer
    ON onts (customer_id) WHERE deleted_at IS NULL AND customer_id IS NOT NULL;

-- Index untuk query per tenant
CREATE INDEX idx_onts_tenant
    ON onts (tenant_id) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE onts ENABLE ROW LEVEL SECURITY;

CREATE POLICY onts_tenant_isolation ON onts
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
