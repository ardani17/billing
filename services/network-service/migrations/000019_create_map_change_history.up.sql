-- Migration: create_map_change_history_table
-- Tabel audit trail append-only untuk riwayat perubahan map node.
-- Mencatat setiap modifikasi lokasi, kustom field, foto, dan status node.
-- Tidak ada operasi perbarui atau hapus pada tabel ini.

CREATE TABLE map_change_history (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    map_node_id  UUID NOT NULL REFERENCES map_nodes(id),
    action       VARCHAR(50) NOT NULL,
    old_value    JSONB,
    new_value    JSONB,
    performed_by VARCHAR(100) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk kueri riwayat per node (terbaru dulu)
CREATE INDEX idx_map_change_history_node
    ON map_change_history (map_node_id, created_at DESC);

-- Row-Level Security
ALTER TABLE map_change_history ENABLE ROW LEVEL SECURITY;

-- Append-only: hanya policy SELECT dan INSERT, tidak ada UPDATE atau DELETE
CREATE POLICY map_change_history_tenant_isolation ON map_change_history
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
