-- Migration: create_geocoding_cache_table
-- Tabel cache hasil reverse geocoding untuk FTTH Visual Mapping.
-- Menyimpan hasil geocoding selama 30 hari untuk mengurangi
-- request ke provider eksternal (Nominatim/Google).

CREATE TABLE geocoding_cache (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    lat_round  DOUBLE PRECISION NOT NULL,
    lng_round  DOUBLE PRECISION NOT NULL,
    address    TEXT NOT NULL,
    raw_json   JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique index untuk lookup by koordinat (rounded to 5 decimal places)
CREATE UNIQUE INDEX idx_geocoding_cache_coords
    ON geocoding_cache (tenant_id, lat_round, lng_round);

-- Index untuk cleanup expired cache entries
CREATE INDEX idx_geocoding_cache_expires
    ON geocoding_cache (expires_at);

-- Row-Level Security
ALTER TABLE geocoding_cache ENABLE ROW LEVEL SECURITY;

CREATE POLICY geocoding_cache_tenant_isolation ON geocoding_cache
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
