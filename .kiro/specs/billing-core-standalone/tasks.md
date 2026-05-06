# Tasks: Billing Core Standalone

- [x] 1. Audit Billing-only runtime
  - [x] 1.1 Verify Billing Core endpoints with optional add-ons inactive
  - [x] 1.2 Identify customer/package/network coupling leaks
  - [x] 1.3 Record implementation plan in spec

- [x] 2. Customer standalone mode
  - [x] 2.1 Add backend support for `connection_method = manual`
  - [x] 2.2 Make latitude/longitude optional for Billing-only customer records
  - [x] 2.3 Make router/PPPoE/MAC fields capability-aware in validation
  - [x] 2.4 Make ODP/ONT/ONU fields capability-aware in validation
  - [x] 2.5 Update customer create/edit UI default to manual when MikroTik is inactive
  - [x] 2.6 Hide MikroTik/Fiber fields and tabs when add-ons are inactive
  - [x] 2.7 Update customer detail labels so Billing-only tenants do not see PPPoE-only wording

- [x] 3. Package standalone mode
  - [x] 3.1 Add neutral monthly package type or compatibility display layer
  - [x] 3.2 Update package form copy from "PPPoE bulanan" to monthly billing package for Billing-only tenants
  - [x] 3.3 Hide MikroTik profile/pool/queue/hotspot fields when `mikrotik` is inactive
  - [x] 3.4 Keep voucher/reseller package flows in Billing Core without requiring MikroTik

- [x] 4. Billing and isolir behavior
  - [x] 4.1 Prevent pending router sync creation when `mikrotik` is inactive
  - [x] 4.2 Prevent RouterOS technical events from Billing API when `mikrotik` is inactive
  - [x] 4.3 Keep billing status, reminders, and walled garden page active in Billing-only mode
  - [x] 4.4 Clarify auto-isolir UI/settings copy for Billing-only versus MikroTik-enabled tenants

- [x] 5. Import/export templates
  - [x] 5.1 Add Billing-only customer import template
  - [x] 5.2 Add MikroTik columns only when `mikrotik` is active
  - [x] 5.3 Add Fiber columns only when `fiber_network` is active
  - [x] 5.4 Ensure export column chooser follows the same capability rules

- [x] 6. Dashboard and reporting
  - [x] 6.1 Stop dashboard from calling MikroTik status when `mikrotik` is inactive
  - [x] 6.2 Stop dashboard from calling OLT summary when `fiber_network` is inactive
  - [x] 6.3 Preserve `403 MODULE_NOT_ENABLED` in route-specific Next.js proxies
  - [x] 6.4 Align report module gates from old `olt` naming to `fiber_network`

- [ ] 7. Tests and smoke
  - [x] 7.1 Add backend tests for manual customer mode
  - [x] 7.2 Add backend tests for package standalone behavior
  - [x] 7.3 Smoke Billing Core only: customer, package, dashboard report, and no pending MikroTik sync
  - [x] 7.4 Smoke Billing Core + MikroTik using CHR without periodic polling
  - [ ] 7.5 Smoke Billing Core + MikroTik + Fiber Network with mock/stub fiber data until OLT hardware exists (blocked until OLT device or approved fiber stub smoke scope is available)
