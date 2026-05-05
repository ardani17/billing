# Requirements: Subscription Module Gating

## Goal

ISPBoss must support commercial packaging where Billing Core is always available, while MikroTik and Fiber Network are paid add-ons. Tenants that do not buy an add-on must not see, call, or accidentally trigger that add-on, and Billing Core must keep working normally.

## Product Packages

| Commercial Package | Module Flag | Contents |
|---|---|---|
| Billing Core | `billing_core` | Auth, tenant, customers, packages, invoice, payment, payment gateway, notification, reporting, reseller/voucher, settings |
| Add-on MikroTik | `mikrotik` | Router, PPPoE/Hotspot, isolir technical action, session, traffic, VPN, backup/firmware, sync |
| Add-on OLT + Peta Jaringan | `fiber_network` | OLT, ONT, ODP, provisioning, alarm, FTTH mapping, topology/map |

## Scope

- Tenant module entitlement storage.
- Backend guard for add-on APIs.
- Event guard so inactive add-ons are skipped safely.
- Frontend menu/widget gating.
- Super Admin management of tenant package/add-ons.
- Tenant Admin subscription view and upgrade request path.

## Non-Goals

- No billing-provider subscription charging integration in this first pass.
- No OLT real hardware validation until hardware is available.
- No new MikroTik polling or scheduler.
- No removal of existing data when an add-on is disabled.

## Requirements

1. Billing Core SHALL always be enabled for every active tenant.
2. Notification, reporting, payment gateway, reseller/voucher, and billing settings SHALL be treated as Billing Core features, not separate commercial add-ons.
3. MikroTik features SHALL require the `mikrotik` module flag.
4. OLT and Peta Jaringan features SHALL require the single `fiber_network` module flag.
5. OLT and Peta Jaringan SHALL NOT be sold or enabled independently.
6. Frontend navigation SHALL hide MikroTik menus when `mikrotik` is inactive.
7. Frontend navigation SHALL hide OLT and Peta Jaringan menus when `fiber_network` is inactive.
8. Backend MikroTik APIs SHALL return `403 MODULE_NOT_ENABLED` when `mikrotik` is inactive.
9. Backend OLT, ODP, and network map APIs SHALL return `403 MODULE_NOT_ENABLED` when `fiber_network` is inactive.
10. Billing workflows SHALL continue without error when either add-on is inactive.
11. Billing-to-MikroTik events SHALL be ignored safely when `mikrotik` is inactive.
12. Fiber events/jobs SHALL be ignored safely when `fiber_network` is inactive.
13. Disabling an add-on SHALL preserve existing data and credentials for future reactivation.
14. Super Admin SHALL be able to view and update tenant add-on entitlements.
15. Tenant Admin SHALL be able to view current package/add-ons and request upgrade, but SHALL NOT directly activate paid add-ons.
16. Audit logs SHALL record add-on entitlement changes.
17. Tests SHALL cover enabled and disabled add-on paths.
