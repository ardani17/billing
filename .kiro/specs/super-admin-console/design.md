# Design: Super Admin Console

## Product Model

Super Admin is the owner console for ISPBoss. It operates above tenant scope. It manages commercial packaging and platform support, while tenant business data remains edited through tenant context or impersonation.

## Route Map

Frontend:

- `/super-admin`
- `/super-admin/tenants`
- `/super-admin/tenants/[id]`
- `/super-admin/subscriptions`
- `/super-admin/upgrade-requests`
- `/super-admin/support`
- `/super-admin/health`
- `/super-admin/audit`
- `/super-admin/settings`

Backend:

- `GET /api/v1/admin/platform/overview`
- `GET /api/v1/admin/platform/tenants`
- `POST /api/v1/admin/platform/tenants`
- `GET /api/v1/admin/platform/tenants/:id`
- `PUT /api/v1/admin/platform/tenants/:id`
- `POST /api/v1/admin/platform/tenants/:id/activate`
- `POST /api/v1/admin/platform/tenants/:id/suspend`
- `POST /api/v1/admin/platform/tenants/:id/resume`
- `POST /api/v1/admin/platform/tenants/:id/cancel`
- `POST /api/v1/admin/platform/tenants/:id/reset-owner`
- `GET /api/v1/admin/platform/tenants/:id/modules`
- `PUT /api/v1/admin/platform/tenants/:id/modules`
- `GET /api/v1/admin/platform/subscriptions`
- `PUT /api/v1/admin/platform/subscriptions/:tenant_id`
- `GET /api/v1/admin/platform/upgrade-requests`
- `POST /api/v1/admin/platform/upgrade-requests/:id/approve`
- `POST /api/v1/admin/platform/upgrade-requests/:id/reject`
- `GET /api/v1/admin/platform/support`
- `POST /api/v1/admin/platform/support`
- `GET /api/v1/admin/platform/support/:id`
- `POST /api/v1/admin/platform/support/:id/comments`
- `PUT /api/v1/admin/platform/support/:id/status`
- `GET /api/v1/admin/platform/health`
- `GET /api/v1/admin/platform/audit`
- `GET /api/v1/admin/platform/settings`
- `PUT /api/v1/admin/platform/settings`
- `POST /api/v1/admin/impersonate`
- `POST /api/v1/admin/stop-impersonate`

## Data Model

### Existing

- `tenants`
- `users`
- `tenant_modules`
- `audit_logs`

### Recommended Tables

`platform_subscriptions`

- `id`
- `tenant_id`
- `plan_code`
- `status`
- `amount`
- `currency`
- `trial_ends_at`
- `current_period_start`
- `current_period_end`
- `cancelled_at`
- `created_at`
- `updated_at`

`tenant_upgrade_requests`

- `id`
- `tenant_id`
- `requested_plan`
- `requested_modules`
- `message`
- `status`
- `processed_by`
- `processed_reason`
- `processed_at`
- `created_at`
- `updated_at`

`support_tickets`

- `id`
- `tenant_id`
- `subject`
- `description`
- `priority`
- `status`
- `assignee_id`
- `created_by`
- `created_at`
- `updated_at`

`support_ticket_comments`

- `id`
- `ticket_id`
- `author_id`
- `author_role`
- `body`
- `is_internal`
- `created_at`

`platform_settings`

- `key`
- `value_json`
- `updated_by`
- `updated_at`

## Entitlement Rules

- `billing_core` is always active for active tenants.
- `mikrotik` controls MikroTik UI/API/events.
- `fiber_network` controls OLT and Peta Jaringan UI/API/events.
- Turning an add-on off must not delete data.
- Entitlement changes must write audit logs.

## Impersonation Flow

1. Super Admin opens tenant detail.
2. UI fetches tenant admin users.
3. Super Admin selects target tenant admin and enters reason.
4. Backend verifies target role is `tenant_admin`.
5. Backend returns impersonation token pair.
6. Web stores impersonation state and routes to tenant dashboard.
7. Tenant UI shows impersonation banner.
8. Stop action calls `/api/v1/admin/stop-impersonate`.

## UI Structure

### Overview

- KPI strip.
- Tenant risk list.
- Upgrade request list.
- Support backlog.
- Health summary.
- Recent global audit.

### Tenants

- Search/filter bar.
- Tenant table.
- Row actions: detail, edit, suspend/resume, impersonate.
- Create tenant panel.

### Tenant Detail

- Tenant profile.
- Subscription card.
- Module entitlement controls.
- Owner/admin users.
- Support tickets for tenant.
- Recent audit.
- Impersonate action with reason confirmation.

### Subscriptions

- Current subscriptions.
- Renewal/trial dates.
- Plan amount.
- Add-on badges.
- Overdue/trial-expiring filters.

### Support

- Ticket queue.
- Ticket detail.
- Internal comments.
- Status/priority/assignee controls.

### Health

- Service status cards.
- Latency.
- Last error.
- Queue/worker status.
- Notification/payment provider status.

### Audit

- Filter drawer.
- Audit table.
- Detail side panel.
- Export CSV.

### Settings

- Plan defaults.
- Pricing.
- Included modules.
- Trial days.
- Tenant limits.
- Support contacts.
- Security policy.

## Security Notes

- Super Admin routes require `super_admin` role.
- Tenant data write operations must happen by impersonation, not direct cross-tenant mutation.
- Sensitive values and credentials must never be returned in global audit payloads.
- Every destructive or access-changing action requires reason and audit.
