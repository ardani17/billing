-- Migration: create_provisioning_settings_table
-- Tabel settings provisioning per tenant.
-- Setiap tenant memiliki satu record settings yang mengontrol perilaku
-- auto-provisioning, auto-port-migration, dan strategi VLAN.

CREATE TABLE provisioning_settings (
    id                           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    UUID NOT NULL UNIQUE REFERENCES tenants(id),
    auto_provisioning_enabled    BOOLEAN NOT NULL DEFAULT false,
    auto_port_migration_enabled  BOOLEAN NOT NULL DEFAULT false,
    vlan_strategy                VARCHAR(30) NOT NULL DEFAULT 'single',
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Row-Level Security
ALTER TABLE provisioning_settings ENABLE ROW LEVEL SECURITY;

CREATE POLICY ps_tenant_isolation ON provisioning_settings
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
