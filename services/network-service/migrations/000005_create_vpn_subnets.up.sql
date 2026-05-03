-- Migration: create_vpn_subnets
-- Tabel alokasi subnet VPN per tenant. Setiap tenant mendapat 1 subnet /24.
-- Format: 10.99.{tenant_seq}.0/24, server IP di .1, client mulai dari .2

CREATE TABLE vpn_subnets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL UNIQUE REFERENCES tenants(id),
    subnet_prefix       VARCHAR(18) NOT NULL,   -- e.g. "10.99.1.0/24"
    tenant_seq          INTEGER NOT NULL UNIQUE, -- nomor urut tenant untuk subnet
    server_ip           VARCHAR(45) NOT NULL,    -- e.g. "10.99.1.1"
    next_client_ip_seq  INTEGER NOT NULL DEFAULT 2, -- seq berikutnya untuk client IP
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Row-Level Security
ALTER TABLE vpn_subnets ENABLE ROW LEVEL SECURITY;

CREATE POLICY vpn_subnets_tenant_isolation ON vpn_subnets
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
