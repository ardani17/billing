DROP TABLE IF EXISTS platform_settings;
DROP TABLE IF EXISTS platform_audit_logs;
DROP TABLE IF EXISTS support_ticket_comments;
DROP TABLE IF EXISTS support_tickets;
DROP TABLE IF EXISTS tenant_upgrade_requests;
DROP TABLE IF EXISTS platform_subscriptions;
ALTER TABLE tenants
    DROP COLUMN IF EXISTS domain_last_checked_at,
    DROP COLUMN IF EXISTS domain_verified_at,
    DROP COLUMN IF EXISTS domain_status;
