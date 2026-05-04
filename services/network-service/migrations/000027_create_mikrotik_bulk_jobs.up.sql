-- Migration: create_mikrotik_bulk_jobs
-- Stores manual/on-demand bulk MikroTik action results per tenant.

CREATE TABLE mikrotik_bulk_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    action          VARCHAR(40) NOT NULL,
    status          VARCHAR(40) NOT NULL DEFAULT 'queued',
    router_ids      UUID[] NOT NULL DEFAULT '{}',
    total_count     INTEGER NOT NULL DEFAULT 0,
    success_count   INTEGER NOT NULL DEFAULT 0,
    failed_count    INTEGER NOT NULL DEFAULT 0,
    results         JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_message   TEXT,
    requested_by    UUID,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mikrotik_bulk_jobs_tenant_created
    ON mikrotik_bulk_jobs (tenant_id, created_at DESC);

CREATE INDEX idx_mikrotik_bulk_jobs_action_status
    ON mikrotik_bulk_jobs (tenant_id, action, status);

ALTER TABLE mikrotik_bulk_jobs ENABLE ROW LEVEL SECURITY;

CREATE POLICY mikrotik_bulk_jobs_tenant_isolation ON mikrotik_bulk_jobs
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

