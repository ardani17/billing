-- Migration: create_olts_table
-- Tabel registrasi OLT device per tenant.
-- Menyimpan data perangkat OLT termasuk konfigurasi SNMP dan CLI.
-- Credential disimpan dalam bentuk terenkripsi (AES-256-GCM).

CREATE TABLE olts (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                     UUID NOT NULL REFERENCES tenants(id),
    name                          VARCHAR(100) NOT NULL,
    host                          VARCHAR(255) NOT NULL,
    snmp_version                  VARCHAR(5) NOT NULL DEFAULT 'v2c',
    snmp_port                     INTEGER NOT NULL DEFAULT 161,
    snmp_community_encrypted      TEXT,
    snmp_username                 VARCHAR(100),
    snmp_auth_protocol            VARCHAR(10),
    snmp_auth_password_encrypted  TEXT,
    snmp_priv_protocol            VARCHAR(10),
    snmp_priv_password_encrypted  TEXT,
    cli_protocol                  VARCHAR(10) NOT NULL DEFAULT 'ssh',
    cli_port                      INTEGER NOT NULL DEFAULT 22,
    cli_username                  VARCHAR(100) NOT NULL,
    cli_password_encrypted        TEXT NOT NULL,
    cli_enable_password_encrypted TEXT,
    brand                         VARCHAR(50),
    model                         VARCHAR(100),
    firmware_version              VARCHAR(100),
    pon_port_count                INTEGER NOT NULL DEFAULT 0,
    total_ont_count               INTEGER NOT NULL DEFAULT 0,
    status                        VARCHAR(20) NOT NULL DEFAULT 'offline',
    health_check_interval_sec     INTEGER NOT NULL DEFAULT 300,
    last_online_at                TIMESTAMPTZ,
    last_checked_at               TIMESTAMPTZ,
    failure_count                 INTEGER NOT NULL DEFAULT 0,
    notes                         TEXT,
    deleted_at                    TIMESTAMPTZ,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique constraint: nama OLT unik per tenant (exclude soft-deleted)
CREATE UNIQUE INDEX idx_olts_tenant_name
    ON olts (tenant_id, name)
    WHERE deleted_at IS NULL;

-- Index untuk query per tenant
CREATE INDEX idx_olts_tenant_id
    ON olts (tenant_id) WHERE deleted_at IS NULL;

-- Index untuk health checker (cross-tenant, by status)
CREATE INDEX idx_olts_status
    ON olts (status) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE olts ENABLE ROW LEVEL SECURITY;

CREATE POLICY olts_tenant_isolation ON olts
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
