-- Migration: create_map_share_links_table
-- Tabel share link read-only untuk FTTH Visual Mapping.
-- Memungkinkan admin membagikan peta ke pihak eksternal
-- dengan opsi expiry dan password protection.

CREATE TABLE map_share_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    token          VARCHAR(64) NOT NULL,
    visible_layers JSONB NOT NULL,
    expires_at     TIMESTAMPTZ,
    password_hash  VARCHAR(255),
    access_count   INTEGER NOT NULL DEFAULT 0,
    created_by     VARCHAR(100) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique index untuk lookup by token
CREATE UNIQUE INDEX idx_map_share_links_token
    ON map_share_links (token);

-- Index untuk query per tenant
CREATE INDEX idx_map_share_links_tenant
    ON map_share_links (tenant_id);

-- Row-Level Security
ALTER TABLE map_share_links ENABLE ROW LEVEL SECURITY;

CREATE POLICY map_share_links_tenant_isolation ON map_share_links
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
