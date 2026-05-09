-- Migration: create_vpn_maintenance_windows
-- Tabel jadwal maintenance VPN server.
-- Digunakan untuk notifikasi ke tenant yang terdampak sebelum maintenance.

CREATE TABLE vpn_maintenance_windows (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_endpoint   VARCHAR(255) NOT NULL,
    scheduled_start   TIMESTAMPTZ NOT NULL,
    scheduled_end     TIMESTAMPTZ NOT NULL,
    description       TEXT,
    created_by        UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index untuk kueri upcoming maintenance
CREATE INDEX idx_vpn_maintenance_scheduled
    ON vpn_maintenance_windows (scheduled_start);
