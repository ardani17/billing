-- Migrasi: membuat tabel packages untuk menyimpan paket internet.
-- Mendukung dua jenis paket: PPPoE/Static (bulanan) dan Hotspot/Voucher (durasi).
-- Setiap paket dimiliki oleh satu tenant dan dilindungi oleh RLS.

CREATE TABLE packages (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    type                  VARCHAR(20) NOT NULL,
    name                  VARCHAR(255) NOT NULL,
    description           TEXT,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    download_mbps         INTEGER NOT NULL,
    upload_mbps           INTEGER NOT NULL,
    bandwidth_type        VARCHAR(20),
    burst_download_mbps   INTEGER,
    burst_upload_mbps     INTEGER,
    burst_threshold_mbps  INTEGER,
    burst_time_seconds    INTEGER,
    quota_type            VARCHAR(20) NOT NULL,
    quota_mb              INTEGER,
    quota_action          VARCHAR(20),
    throttle_mbps         INTEGER,
    monthly_price         BIGINT,
    installation_fee      BIGINT NOT NULL DEFAULT 0,
    sell_price            BIGINT,
    reseller_price        BIGINT,
    duration_value        INTEGER,
    duration_unit         VARCHAR(20),
    shared_users          INTEGER NOT NULL DEFAULT 1,
    mikrotik_profile_name VARCHAR(255),
    address_pool          VARCHAR(255),
    parent_queue          VARCHAR(255),
    hotspot_profile_name  VARCHAR(255),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_packages_type CHECK (type IN ('pppoe', 'voucher')),
    CONSTRAINT chk_packages_quota_type CHECK (
        quota_type IN ('unlimited', 'monthly_quota', 'fup', 'quota')
    ),
    CONSTRAINT chk_packages_download_mbps CHECK (download_mbps > 0),
    CONSTRAINT chk_packages_upload_mbps CHECK (upload_mbps > 0)
);

-- Aktifkan RLS pada tabel packages
ALTER TABLE packages ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON packages
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON packages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: nama paket unik per tenant
ALTER TABLE packages ADD CONSTRAINT uq_packages_tenant_name UNIQUE (tenant_id, name);

-- Composite indexes untuk performa query
CREATE INDEX idx_packages_tenant_type ON packages(tenant_id, type);
CREATE INDEX idx_packages_tenant_active ON packages(tenant_id, is_active);
CREATE INDEX idx_packages_tenant_type_active ON packages(tenant_id, type, is_active);
