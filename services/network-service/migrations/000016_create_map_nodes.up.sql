-- Migration: create_map_nodes_table
-- Tabel map node untuk FTTH Visual Mapping per tenant.
-- Setiap map node merepresentasikan titik di peta (OLT, ODP, atau ONT)
-- yang terhubung ke entitas jaringan via reference_id.

CREATE TABLE map_nodes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    node_type     VARCHAR(20) NOT NULL,
    reference_id  UUID NOT NULL,
    latitude      DOUBLE PRECISION NOT NULL,
    longitude     DOUBLE PRECISION NOT NULL,
    custom_fields JSONB,
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique constraint: satu entitas jaringan hanya boleh punya satu map node aktif per tenant
CREATE UNIQUE INDEX idx_map_nodes_tenant_type_ref
    ON map_nodes (tenant_id, node_type, reference_id)
    WHERE deleted_at IS NULL;

-- Index untuk query berdasarkan lokasi (bounding box)
CREATE INDEX idx_map_nodes_tenant_location
    ON map_nodes (tenant_id, latitude, longitude);

-- Index untuk query berdasarkan tipe node
CREATE INDEX idx_map_nodes_tenant_type
    ON map_nodes (tenant_id, node_type);

-- Row-Level Security
ALTER TABLE map_nodes ENABLE ROW LEVEL SECURITY;

CREATE POLICY map_nodes_tenant_isolation ON map_nodes
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
