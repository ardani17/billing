-- Migration: create_pppoe_profiles_table
CREATE TABLE pppoe_profiles (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL REFERENCES tenants(id),
    package_id                UUID NOT NULL,
    profile_name              VARCHAR(100) NOT NULL,
    download_limit            VARCHAR(20) NOT NULL,
    upload_limit              VARCHAR(20) NOT NULL,
    burst_download            VARCHAR(20),
    burst_upload              VARCHAR(20),
    burst_threshold_download  VARCHAR(20),
    burst_threshold_upload    VARCHAR(20),
    burst_time                VARCHAR(20),
    address_pool              VARCHAR(100),
    local_address             VARCHAR(45) NOT NULL DEFAULT 'gateway',
    only_one                  BOOLEAN NOT NULL DEFAULT true,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique constraint: profile_name unik per tenant
CREATE UNIQUE INDEX idx_pppoe_profiles_tenant_name
    ON pppoe_profiles (tenant_id, profile_name);

-- Index untuk lookup by package_id
CREATE INDEX idx_pppoe_profiles_package_id
    ON pppoe_profiles (package_id);

-- Row-Level Security
ALTER TABLE pppoe_profiles ENABLE ROW LEVEL SECURITY;

CREATE POLICY pppoe_profiles_tenant_isolation ON pppoe_profiles
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
