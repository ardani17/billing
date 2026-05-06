# Requirements: Super Admin Console

## Goal

Build the ISPBoss owner console for managing the SaaS platform across all tenants. Super Admin is an internal ISPBoss role, not a tenant user. It can manage tenant lifecycle, subscription, add-on entitlements, support, impersonation, health, and global audit.

## Scope

- Platform overview.
- Tenant management.
- Subscription and module entitlement management.
- Upgrade request handling.
- Impersonation safeguards.
- Support ticket console.
- Service health monitoring.
- Global audit investigation.
- Platform settings persistence.

## Non-Goals

- No direct editing of tenant customer/package/invoice/payment records outside impersonation.
- No external billing-provider subscription charging integration in the first pass.
- No OLT hardware validation requirement.
- No destructive deletion of tenant operational data.

## Requirements

1. Super Admin SHALL access a dedicated `/super-admin` console.
2. Super Admin SHALL be internal to ISPBoss and SHALL NOT be scoped to a single tenant.
3. Super Admin SHALL view platform KPIs: tenant totals, active/trial/suspended counts, MRR, overdue subscriptions, upgrade requests, support backlog, and service health.
4. Super Admin SHALL list tenants with search, filtering, sorting, and pagination.
5. Super Admin SHALL view tenant details with owner, domain, plan, status, health, modules, counts, and recent audit.
6. Super Admin SHALL create a tenant manually for support/onboarding use.
7. Super Admin SHALL edit platform-level tenant fields: name, domain, owner, plan, status, and limits.
8. Super Admin SHALL activate, suspend, resume, and cancel tenant access.
9. Super Admin SHALL reset or change the owner tenant admin.
10. Super Admin SHALL view and update tenant module entitlements for `billing_core`, `mikrotik`, and `fiber_network`.
11. `billing_core` SHALL always remain enabled for active tenants.
12. `mikrotik` SHALL be managed as a paid add-on.
13. `fiber_network` SHALL manage OLT and Peta Jaringan together as one paid add-on.
14. Disabling an add-on SHALL preserve existing data for future reactivation.
15. Entitlement changes SHALL be audited with actor, tenant, module, old status, new status, reason, and timestamp.
16. Tenant Admin SHALL only request upgrades; it SHALL NOT activate paid add-ons directly.
17. Super Admin SHALL view, approve, reject, or cancel upgrade requests.
18. Super Admin SHALL impersonate only tenant admin users.
19. Impersonation SHALL require a reason.
20. Impersonation start and stop SHALL be audited globally.
21. The UI SHALL display a clear impersonation banner while impersonation is active.
22. Super Admin SHALL stop impersonation and return to owner console.
23. Super Admin SHALL view support tickets across tenants.
24. Super Admin SHALL filter support tickets by tenant, status, priority, and assignee.
25. Super Admin SHALL add internal comments and update ticket status.
26. Super Admin SHALL view service health for Billing API, PostgreSQL, Redis, Network Service, Notification Service, queue workers, payment gateway, and notification delivery.
27. Service health SHALL include status, latency or last check, and last error where available.
28. Super Admin SHALL view global audit logs.
29. Global audit SHALL support filtering by tenant, actor, action, entity, and date range.
30. Global audit SHALL expose detail payload for investigation without leaking secrets.
31. Platform settings SHALL persist plan defaults, pricing, included modules, trial days, default tenant limits, support contacts, and security policy.
32. Super Admin mobile navigation SHALL expose every Super Admin menu, using More/drawer if bottom nav space is limited.
33. Tests SHALL verify entitlement changes, tenant lifecycle actions, impersonation safeguards, and audit logging.
