-- Migrasi: membuat tabel audit_logs untuk mencatat semua perubahan entitas.
-- Tabel ini bersifat shared (digunakan oleh semua modul) dan dilindungi RLS.

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    entity_type VARCHAR(50) NOT NULL,
    entity_id   UUID NOT NULL,
    action      VARCHAR(100) NOT NULL,
    actor_id    UUID NOT NULL,
    actor_name  VARCHAR(255) NOT NULL,
    changes     JSONB,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel audit_logs
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant
CREATE POLICY tenant_isolation ON audit_logs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON audit_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Composite indexes untuk performa kueri
CREATE INDEX idx_audit_logs_entity ON audit_logs(tenant_id, entity_type, entity_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(tenant_id, created_at);
