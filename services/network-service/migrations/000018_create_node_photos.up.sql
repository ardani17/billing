-- Migration: create_node_photos_table
-- Tabel foto per map node untuk dokumentasi instalasi.
-- Setiap node bisa memiliki maksimal 5 foto (enforced di application layer).

CREATE TABLE node_photos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    map_node_id     UUID NOT NULL REFERENCES map_nodes(id),
    file_path       VARCHAR(500) NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    caption         VARCHAR(200),
    uploaded_by     VARCHAR(100) NOT NULL,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk kueri foto per node (exclude hapus lunak)
CREATE INDEX idx_node_photos_node
    ON node_photos (map_node_id)
    WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE node_photos ENABLE ROW LEVEL SECURITY;

CREATE POLICY node_photos_tenant_isolation ON node_photos
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
