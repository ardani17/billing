ALTER TABLE provisioning_audit_logs
    ADD COLUMN brand VARCHAR(50),
    ADD COLUMN model VARCHAR(100),
    ADD COLUMN transport VARCHAR(50),
    ADD COLUMN operation VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_provisioning_audit_logs_transport_created
    ON provisioning_audit_logs (transport, created_at DESC);
