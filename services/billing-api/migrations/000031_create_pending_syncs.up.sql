-- Migrasi: membuat tabel pending_syncs untuk melacak operasi sinkronisasi router yang tertunda.
-- Tabel ini menyimpan status isolir/un_isolir/suspend yang perlu disinkronkan ke router MikroTik.
-- Menggunakan RLS karena data bersifat tenant-scoped.

CREATE TABLE pending_syncs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    customer_id     UUID NOT NULL REFERENCES customers(id),
    operation_type  VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 5,
    last_retry_at   TIMESTAMPTZ,
    next_retry_at   TIMESTAMPTZ,
    error_message   TEXT,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_pending_syncs_operation_type CHECK (
        operation_type IN ('isolir', 'un_isolir', 'suspend')
    ),
    CONSTRAINT chk_pending_syncs_status CHECK (
        status IN ('pending', 'in_progress', 'completed', 'failed')
    ),
    CONSTRAINT chk_pending_syncs_retry_count CHECK (
        retry_count >= 0 AND retry_count <= max_retries
    )
);

-- Mengaktifkan RLS: pending_syncs bersifat tenant-scoped.
ALTER TABLE pending_syncs ENABLE ROW LEVEL SECURITY;

-- Policy untuk SELECT, UPDATE, DELETE: hanya bisa akses data tenant sendiri
CREATE POLICY pending_syncs_tenant_policy ON pending_syncs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy untuk INSERT: hanya bisa insert data tenant sendiri
CREATE POLICY pending_syncs_tenant_insert ON pending_syncs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index: lookup berdasarkan tenant dan customer
CREATE INDEX idx_pending_syncs_tenant_customer
    ON pending_syncs(tenant_id, customer_id);

-- Index: lookup berdasarkan tenant dan status
CREATE INDEX idx_pending_syncs_tenant_status
    ON pending_syncs(tenant_id, status);

-- Partial index: untuk periodic sync job (hanya ambil yang pending dan siap retry)
CREATE INDEX idx_pending_syncs_retry
    ON pending_syncs(status, next_retry_at)
    WHERE status = 'pending';
