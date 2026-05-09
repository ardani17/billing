-- Migration: create_service_profiles_table
-- Tabel service profile mapping antara paket ISPBoss dan OLT profile.
-- Digunakan saat provisioning untuk menentukan line profile dan service profile
-- yang akan diterapkan ke ONT berdasarkan paket pelanggan.

CREATE TABLE service_profiles (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    olt_id             UUID NOT NULL REFERENCES olts(id),
    name               VARCHAR(100) NOT NULL,
    line_profile_id    INTEGER NOT NULL,
    service_profile_id INTEGER NOT NULL,
    package_id         UUID,
    description        TEXT,
    deleted_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique: kombinasi profile unik per OLT (exclude hapus lunak)
CREATE UNIQUE INDEX idx_sp_olt_profiles
    ON service_profiles (olt_id, line_profile_id, service_profile_id)
    WHERE deleted_at IS NULL;

-- Index untuk kueri per OLT
CREATE INDEX idx_sp_olt
    ON service_profiles (olt_id) WHERE deleted_at IS NULL;

-- Index untuk lookup by package
CREATE INDEX idx_sp_package
    ON service_profiles (olt_id, package_id)
    WHERE deleted_at IS NULL AND package_id IS NOT NULL;

-- Row-Level Security
ALTER TABLE service_profiles ENABLE ROW LEVEL SECURITY;

CREATE POLICY sp_tenant_isolation ON service_profiles
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
