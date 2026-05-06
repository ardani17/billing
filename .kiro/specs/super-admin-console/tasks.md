# Tasks: Super Admin Console

- [x] 1. Spec and audit baseline
  - [x] 1.1 Document Super Admin product scope in `diskusi/15-super-admin.md`
  - [x] 1.2 Create requirements/design/tasks for Super Admin console
  - [x] 1.3 Add audit checklist to application review document

- [x] 2. Tenant management backend
  - [x] 2.1 Add platform tenant create/update DTOs
  - [x] 2.2 Add tenant activate/suspend/resume/cancel actions
  - [x] 2.3 Add owner reset/change action
  - [x] 2.4 Add domain verification fields/status if missing
  - [x] 2.5 Audit all tenant lifecycle changes
  - [x] 2.6 Add tests/smoke for tenant lifecycle actions

- [x] 3. Subscription and entitlement backend
  - [x] 3.1 Add `platform_subscriptions` migration
  - [x] 3.2 Add subscription handler/usecase flow
  - [x] 3.3 Add tenant module entitlement update endpoint
  - [x] 3.4 Enforce `billing_core` always active for active tenants
  - [x] 3.5 Preserve add-on data when disabling modules
  - [x] 3.6 Audit entitlement changes with reason
  - [x] 3.7 Add tests/smoke for entitlement update and audit

- [x] 4. Upgrade request flow
  - [x] 4.1 Add `tenant_upgrade_requests` migration
  - [x] 4.2 Add Tenant Admin upgrade request endpoint
  - [x] 4.3 Add Super Admin list/detail endpoints
  - [x] 4.4 Add approve/reject/cancel endpoints
  - [x] 4.5 Audit upgrade request decisions

- [x] 5. Impersonation completion
  - [x] 5.1 Expose tenant admin selector on tenant detail
  - [x] 5.2 Require reason before impersonate
  - [x] 5.3 Store/show impersonation banner in tenant UI
  - [x] 5.4 Add stop impersonate UI
  - [x] 5.5 Verify audit includes start/stop and impersonator ID
  - [x] 5.6 Add tests for forbidden targets and missing reason

- [x] 6. Support ticket console
  - [x] 6.1 Add `support_tickets` migration
  - [x] 6.2 Add `support_ticket_comments` migration
  - [x] 6.3 Add support handler/usecase flow
  - [x] 6.4 Add list/detail/comment/status endpoints
  - [x] 6.5 Build Super Admin support UI
  - [x] 6.6 Link support ticket to tenant detail and impersonation reason

- [x] 7. Service health hardening
  - [x] 7.1 Replace static Network/Notification status with real health checks
  - [x] 7.2 Add Redis and queue worker health
  - [x] 7.3 Add payment gateway status
  - [x] 7.4 Add notification delivery error rate
  - [x] 7.5 Add last error and last check timestamps

- [x] 8. Global audit improvements
  - [x] 8.1 Add query filters for tenant, actor, action, entity, and date range
  - [x] 8.2 Add audit detail endpoint
  - [x] 8.3 Add CSV export
  - [x] 8.4 Redact secrets from payloads
  - [x] 8.5 Build filterable audit UI with detail panel

- [x] 9. Platform settings
  - [x] 9.1 Add `platform_settings` storage
  - [x] 9.2 Add read/update endpoints
  - [x] 9.3 Build settings forms for plan defaults, pricing, trial days, limits, support contact, and security policy
  - [x] 9.4 Audit settings changes

- [x] 10. Frontend Super Admin UX
  - [x] 10.1 Add Upgrade Requests navigation
  - [x] 10.2 Add mobile More/drawer so all menu items are reachable
  - [x] 10.3 Add search/filter/pagination to tenants, subscriptions, support, and audit
  - [x] 10.4 Add action confirmations for suspend/cancel/entitlement changes
  - [x] 10.5 Add loading, error, and empty states for every Super Admin page
  - [x] 10.6 Run responsive smoke on desktop and mobile widths

- [x] 11. Verification
  - [x] 11.1 Run billing-api tests
  - [x] 11.2 Run web build
  - [x] 11.3 Smoke `/super-admin` overview
  - [x] 11.4 Smoke tenant entitlement update affects Tenant Admin menu/API gating
  - [x] 11.5 Smoke impersonation start/stop
  - [x] 11.6 Smoke audit filters and export

## Verification Notes

- `go test ./...` in `services/billing-api` passed.
- `npm.cmd --workspace @ispboss/web run build` passed.
- Local smoke passed for Super Admin API overview, tenants, subscriptions, upgrade requests, support, health, audit, settings, entitlement update, audit CSV export, and impersonation start/stop.
