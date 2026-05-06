# Requirements: Billing Core Standalone

## Goal

Billing Core must run as a complete ISP billing application without requiring the MikroTik add-on or the Fiber Network add-on. Tenants that only buy Billing Core can manage customers, packages, invoices, payments, vouchers, reports, notifications, and basic settings using manual operational data.

## Module Boundary

| Area | Billing Core only | With MikroTik add-on | With Fiber Network add-on |
|---|---|---|---|
| Customers | Manual customer/service records | Router, PPPoE, Hotspot, DHCP, static IP actions | ODP, ONT/ONU, OLT, fiber location data |
| Packages | Monthly billing packages and voucher products | MikroTik profile, pool, queue, hotspot profile | Optional fiber capacity mapping |
| Billing | Invoice, payment, reminder, status, walled garden billing page | Technical isolir/unisolir on router | Fiber operational references |
| Dashboard | Finance, receivables, customers, collection rate | Router status/session widgets | OLT/map status widgets |
| Reports | Finance, customer, reseller, notification, operational audit | Network traffic/uptime/session reports | OLT signal/alarm/map reports |

## Requirements

1. Billing Core SHALL allow customer creation without `router_id`, PPPoE credentials, MAC address, ODP, ONT/ONU, OLT, or map coordinates.
2. Billing Core SHALL provide a neutral connection mode such as `manual` for customers whose technical provisioning is handled outside ISPBoss.
3. MikroTik-specific customer fields SHALL only be shown, validated, imported, and synced when `mikrotik` is active.
4. Fiber-specific customer fields SHALL only be shown, validated, imported, and linked when `fiber_network` is active.
5. Billing Core SHALL support monthly packages without naming them PPPoE packages.
6. MikroTik profile, address pool, parent queue, burst, and hotspot profile fields SHALL be optional add-on fields and hidden when `mikrotik` is inactive.
7. Billing Core invoice, payment, voucher, reseller, notification, and report flows SHALL not call network-service when both optional add-ons are inactive.
8. Billing Core isolir/unisolir SHALL update billing/customer status and notification state without creating router sync work when `mikrotik` is inactive.
9. Auto-isolir and auto-open-isolir settings SHALL have a clear billing-only behavior: status change and notification only, no router action.
10. Pending router sync records SHALL only be created for tenants with `mikrotik` active.
11. Dashboard SHALL hide network status widgets when corresponding add-ons are inactive.
12. Add-on API proxy errors SHALL preserve `403 MODULE_NOT_ENABLED` instead of wrapping them as generic `502` errors.
13. Import/export templates SHALL be capability-aware: base billing columns always, MikroTik columns only with `mikrotik`, fiber columns only with `fiber_network`.
14. Customer detail SHALL hide Network and Fiber tabs/fields when add-ons are inactive.
15. Reports SHALL use `fiber_network` as the commercial flag for OLT + Peta Jaringan sections.
16. Smoke tests SHALL cover Billing Core only, Billing Core + MikroTik, and Billing Core + MikroTik + Fiber Network entitlement combinations.

## Non-Goals

- No real OLT hardware validation in this spec.
- No removal of existing MikroTik or fiber data when an add-on is disabled.
- No commercial subscription charging automation in this spec.
- No periodic MikroTik polling.

