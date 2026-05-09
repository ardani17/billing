-- Migration: create_cable_routes_table
-- Tabel cable route untuk FTTH Visual Mapping per tenant.
-- Setiap cable route merepresentasikan jalur kabel fiber antara dua map node
-- dengan koordinat waypoints dan jarak yang dihitung otomatis.

CREATE TABLE cable_routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    from_node_id    UUID NOT NULL REFERENCES map_nodes(id),
    to_node_id      UUID NOT NULL REFERENCES map_nodes(id),
    route_type      VARCHAR(20) NOT NULL,
    coordinates     JSONB NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL,
    core_count      INTEGER,
    description     TEXT,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk kueri berdasarkan node endpoints (exclude hapus lunak)
CREATE INDEX idx_cable_routes_tenant_nodes
    ON cable_routes (tenant_id, from_node_id, to_node_id)
    WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE cable_routes ENABLE ROW LEVEL SECURITY;

CREATE POLICY cable_routes_tenant_isolation ON cable_routes
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
