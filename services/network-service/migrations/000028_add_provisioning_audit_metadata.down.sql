DROP INDEX IF EXISTS idx_provisioning_audit_logs_transport_created;

ALTER TABLE provisioning_audit_logs
    DROP COLUMN IF EXISTS operation,
    DROP COLUMN IF EXISTS transport,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS brand;
