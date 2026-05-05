-- Migrasi: entitlement modul per tenant.
-- Billing Core selalu aktif. MikroTik dan OLT+Peta Jaringan adalah add-on opsional.

CREATE TABLE tenant_modules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_code VARCHAR(50) NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'inactive',
    activated_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_modules_tenant_code UNIQUE (tenant_id, module_code),
    CONSTRAINT chk_tenant_modules_code CHECK (module_code IN ('billing_core', 'mikrotik', 'fiber_network')),
    CONSTRAINT chk_tenant_modules_status CHECK (status IN ('active', 'inactive', 'suspended'))
);

CREATE INDEX idx_tenant_modules_tenant_id ON tenant_modules(tenant_id);
CREATE INDEX idx_tenant_modules_module_status ON tenant_modules(module_code, status);

INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
SELECT id, 'billing_core', 'active', NOW()
FROM tenants
ON CONFLICT (tenant_id, module_code) DO NOTHING;

DO $$
BEGIN
    IF to_regclass('public.routers') IS NOT NULL THEN
        INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
        SELECT DISTINCT tenant_id, 'mikrotik', 'active', NOW()
        FROM routers
        ON CONFLICT (tenant_id, module_code) DO UPDATE
        SET status = 'active',
            activated_at = COALESCE(tenant_modules.activated_at, NOW()),
            updated_at = NOW();
    END IF;

    IF to_regclass('public.olts') IS NOT NULL THEN
        INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
        SELECT DISTINCT tenant_id, 'fiber_network', 'active', NOW()
        FROM olts
        ON CONFLICT (tenant_id, module_code) DO UPDATE
        SET status = 'active',
            activated_at = COALESCE(tenant_modules.activated_at, NOW()),
            updated_at = NOW();
    END IF;

    IF to_regclass('public.map_nodes') IS NOT NULL THEN
        INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
        SELECT DISTINCT tenant_id, 'fiber_network', 'active', NOW()
        FROM map_nodes
        ON CONFLICT (tenant_id, module_code) DO UPDATE
        SET status = 'active',
            activated_at = COALESCE(tenant_modules.activated_at, NOW()),
            updated_at = NOW();
    END IF;
END $$;
