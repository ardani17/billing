-- Kueri SQL untuk audit command MikroTik.

-- name: CreateMikroTikCommandAuditLog :exec
INSERT INTO mikrotik_command_audit_logs (
    tenant_id, router_id, user_id, action, command,
    target_type, target_id, status, error_message, remote_addr
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10
);
