-- Migration: create_vpn_tunnels
-- Tabel konfigurasi VPN tunnel per tenant.
-- Mendukung protokol: WireGuard, L2TP/IPSec, PPTP, SSTP, OpenVPN.
-- Soft-hapus via kolom deleted_at.

CREATE TABLE vpn_tunnels (
    id                           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    UUID NOT NULL REFERENCES tenants(id),
    router_id                    UUID REFERENCES routers(id),  -- nullable, standalone tunnel
    tunnel_name                  VARCHAR(100) NOT NULL,
    protocol                     VARCHAR(20) NOT NULL,  -- wireguard, l2tp_ipsec, pptp, sstp, openvpn
    vpn_ip                       VARCHAR(45) NOT NULL,  -- e.g. "10.99.1.2"
    server_endpoint              VARCHAR(255) NOT NULL, -- e.g. "vpn.ispboss.id:51820"
    server_public_key            TEXT,                   -- WireGuard server public key
    client_public_key            TEXT,                   -- WireGuard client public key
    client_private_key_encrypted TEXT,                   -- encrypted via AES-256-GCM
    pre_shared_key_encrypted     TEXT,                   -- encrypted, nullable (WireGuard PSK / IPSec PSK)
    l2tp_username                VARCHAR(100),           -- L2TP/PPTP/SSTP username
    l2tp_password_encrypted      TEXT,                   -- encrypted L2TP/PPTP/SSTP password
    status                       VARCHAR(20) NOT NULL DEFAULT 'pending',
    listen_port                  INTEGER NOT NULL DEFAULT 51820,
    allowed_addresses            TEXT NOT NULL DEFAULT '10.99.0.0/16',
    persistent_keepalive         INTEGER NOT NULL DEFAULT 25,
    last_handshake_at            TIMESTAMPTZ,
    latency_ms                   INTEGER,
    bandwidth_cap_mbps           INTEGER,
    rate_limit_pps               INTEGER NOT NULL DEFAULT 100, -- packets per second rate limit
    active_endpoint              VARCHAR(255),  -- endpoint aktif saat ini (primary/secondary)
    notes                        TEXT,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at                   TIMESTAMPTZ
);

-- Unique constraint: tunnel_name unik per tenant (exclude hapus lunak)
CREATE UNIQUE INDEX idx_vpn_tunnels_tenant_name
    ON vpn_tunnels (tenant_id, tunnel_name)
    WHERE deleted_at IS NULL;

-- Unique constraint: vpn_ip unik per tenant (exclude hapus lunak)
CREATE UNIQUE INDEX idx_vpn_tunnels_tenant_vpn_ip
    ON vpn_tunnels (tenant_id, vpn_ip)
    WHERE deleted_at IS NULL;

-- Index untuk kueri per tenant
CREATE INDEX idx_vpn_tunnels_tenant_id
    ON vpn_tunnels (tenant_id) WHERE deleted_at IS NULL;

-- Index untuk health monitor (cross-tenant, by status)
CREATE INDEX idx_vpn_tunnels_status
    ON vpn_tunnels (status) WHERE deleted_at IS NULL;

-- Index untuk lookup by router_id
CREATE INDEX idx_vpn_tunnels_router_id
    ON vpn_tunnels (router_id) WHERE deleted_at IS NULL;

-- Row-Level Security
ALTER TABLE vpn_tunnels ENABLE ROW LEVEL SECURITY;

CREATE POLICY vpn_tunnels_tenant_isolation ON vpn_tunnels
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
