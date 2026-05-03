-- Migration: create_map_label_settings_table
-- Tabel konfigurasi label per tenant untuk FTTH Visual Mapping.
-- Setiap tenant memiliki satu record settings yang menentukan
-- informasi apa yang tampil di label node di peta.

CREATE TABLE map_label_settings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL UNIQUE REFERENCES tenants(id),
    olt_labels     JSONB NOT NULL DEFAULT '["name","brand_model","ont_count"]',
    odp_labels     JSONB NOT NULL DEFAULT '["name","splitter_type","capacity"]',
    ont_labels     JSONB NOT NULL DEFAULT '["customer_name","package"]',
    min_zoom_level INTEGER NOT NULL DEFAULT 15,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Row-Level Security
ALTER TABLE map_label_settings ENABLE ROW LEVEL SECURITY;

CREATE POLICY map_label_settings_tenant_isolation ON map_label_settings
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
