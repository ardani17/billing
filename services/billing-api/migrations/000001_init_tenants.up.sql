-- Migrasi awal: membuat tabel tenants.
-- Tabel tenants menyimpan data operator ISP/RT-RW Net yang berlangganan ISPBoss.
-- Setiap tenant memiliki data terisolasi via tenant_id dan RLS.

CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    domain     VARCHAR(255),
    plan       VARCHAR(50) NOT NULL DEFAULT 'starter',
    status     VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk lookup berdasarkan domain (white label)
CREATE INDEX idx_tenants_domain ON tenants(domain);

-- Index untuk filter berdasarkan status tenant
CREATE INDEX idx_tenants_status ON tenants(status);
