CREATE TABLE IF NOT EXISTS platform_subscriptions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_code            VARCHAR(50) NOT NULL DEFAULT 'starter',
    status               VARCHAR(50) NOT NULL DEFAULT 'trial',
    amount               BIGINT NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency             VARCHAR(10) NOT NULL DEFAULT 'IDR',
    trial_ends_at        TIMESTAMPTZ,
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end   TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 month'),
    cancelled_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_platform_subscriptions_tenant UNIQUE (tenant_id),
    CONSTRAINT chk_platform_subscriptions_status CHECK (status IN ('trial', 'active', 'overdue', 'suspended', 'cancelled'))
);

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS domain_status VARCHAR(50) NOT NULL DEFAULT 'unverified',
    ADD COLUMN IF NOT EXISTS domain_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS domain_last_checked_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_platform_subscriptions_status
    ON platform_subscriptions(status, current_period_end);

INSERT INTO platform_subscriptions (tenant_id, plan_code, status, amount, trial_ends_at, current_period_start, current_period_end)
SELECT
    t.id,
    t.plan,
    CASE
        WHEN t.status = 'trial' THEN 'trial'
        WHEN t.status = 'suspended' THEN 'suspended'
        WHEN t.status = 'cancelled' THEN 'cancelled'
        ELSE 'active'
    END,
    CASE
        WHEN t.plan IN ('growth', 'pro') THEN 799000
        WHEN t.plan IN ('scale', 'enterprise') THEN 1499000
        ELSE 299000
    END,
    CASE WHEN t.status = 'trial' THEN t.created_at + INTERVAL '14 days' ELSE NULL END,
    t.created_at,
    t.created_at + INTERVAL '1 month'
FROM tenants t
ON CONFLICT (tenant_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS tenant_upgrade_requests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requested_plan    VARCHAR(50),
    requested_modules JSONB NOT NULL DEFAULT '[]'::jsonb,
    message           TEXT,
    status            VARCHAR(50) NOT NULL DEFAULT 'pending',
    processed_by      UUID,
    processed_reason  TEXT,
    processed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tenant_upgrade_requests_status CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_tenant_upgrade_requests_status
    ON tenant_upgrade_requests(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_upgrade_requests_tenant
    ON tenant_upgrade_requests(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS support_tickets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id) ON DELETE SET NULL,
    subject     VARCHAR(255) NOT NULL,
    description TEXT,
    priority    VARCHAR(50) NOT NULL DEFAULT 'normal',
    status      VARCHAR(50) NOT NULL DEFAULT 'open',
    assignee_id UUID,
    created_by  UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_support_tickets_priority CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT chk_support_tickets_status CHECK (status IN ('open', 'in_progress', 'waiting_tenant', 'resolved', 'closed'))
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_tenant
    ON support_tickets(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_support_tickets_status_priority
    ON support_tickets(status, priority, created_at DESC);

CREATE TABLE IF NOT EXISTS support_ticket_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    author_id   UUID,
    author_role VARCHAR(50) NOT NULL DEFAULT 'super_admin',
    body        TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_support_ticket_comments_ticket
    ON support_ticket_comments(ticket_id, created_at ASC);

CREATE TABLE IF NOT EXISTS platform_settings (
    key        VARCHAR(100) PRIMARY KEY,
    value_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS platform_audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id   TEXT NOT NULL,
    action      VARCHAR(150) NOT NULL,
    actor_id    UUID,
    actor_name  VARCHAR(150) NOT NULL DEFAULT 'Super Admin',
    changes     JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_created
    ON platform_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_action
    ON platform_audit_logs(action, created_at DESC);

INSERT INTO platform_settings (key, value_json)
VALUES
    ('plan_defaults', '{"plans":[{"code":"billing_core","name":"Billing Core","amount":299000,"modules":["billing_core"]},{"code":"growth","name":"Billing Core + MikroTik","amount":799000,"modules":["billing_core","mikrotik"]},{"code":"scale","name":"Billing Core + MikroTik + Fiber","amount":1499000,"modules":["billing_core","mikrotik","fiber_network"]}]}'::jsonb),
    ('security_policy', '{"impersonate_reason_required":true,"audit_retention_months":24,"super_admin_mfa_required":true}'::jsonb),
    ('tenant_limits', '{"customer_limit":1000,"router_limit":5,"olt_limit":2,"reseller_limit":50}'::jsonb),
    ('support_contact', '{"email":"support@ispboss.id","whatsapp":"+6280000000000"}'::jsonb)
ON CONFLICT (key) DO NOTHING;
