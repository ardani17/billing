# Design: Billing Core Standalone

## Current Audit Snapshot

The current implementation already keeps the main Billing API endpoints alive when optional add-ons are inactive. A local smoke in Billing-only mode returned HTTP 200 for customers, packages, invoices, payments, and dashboard reports.

Remaining gaps are mostly coupling leaks:

- Customer schema and UI still default to `pppoe` and expose PPPoE/ODP fields.
- Package schema and UI still call monthly packages `pppoe`.
- Billing status flows can publish router sync intent even when MikroTik is inactive.
- Dashboard still fetches MikroTik and OLT summaries.
- Some Next.js route-specific proxies wrap `MODULE_NOT_ENABLED` as `502 NETWORK_SERVICE_ERROR`.
- Import template still contains router/PPPoE/ODP columns by default.

## Capability Model

Use the existing module flags:

- `billing_core`: always active
- `mikrotik`: optional add-on
- `fiber_network`: optional add-on for OLT + Peta Jaringan

Frontend should read capabilities once and pass them into Billing Core forms/widgets. Backend should check entitlement before creating any add-on side effect.

## Customer Model

Add neutral customer provisioning:

- `connection_method = manual`
- PPPoE credentials optional and only required for `pppoe`
- MAC address optional and only required for `dhcp_binding`
- `router_id` optional and only meaningful when `mikrotik` is active
- ODP/ONT/ONU fields optional and only meaningful when `fiber_network` is active
- Latitude/longitude optional for Billing Core, required only when map workflows need it

Billing-only customer form sections:

- Identity
- Contact
- Billing package
- Billing dates
- Area/manual notes
- Optional manual service reference

Add-on sections:

- MikroTik provisioning
- Fiber/OLT placement

## Package Model

Introduce a neutral monthly package type or display layer:

- Preferred backend type: `monthly`
- Backward-compatible mapping: existing `pppoe` packages can be treated as monthly billing packages until migration is complete
- Voucher remains in Billing Core commercially, but live Hotspot provisioning requires MikroTik

MikroTik fields remain optional package metadata:

- `mikrotik_profile_name`
- `address_pool`
- `parent_queue`
- `hotspot_profile_name`

## Billing And Isolir

Billing Core owns financial status. MikroTik owns technical enforcement.

When `mikrotik` is inactive:

- Invoice overdue can change customer status to `isolir` according to Billing settings.
- Notifications and walled garden billing page remain available.
- No pending router sync is inserted.
- No RouterOS event should be published for technical action.
- UI labels should say "status isolir billing" or equivalent where needed.

When `mikrotik` is active:

- The same status transition can also create pending sync and network-service can execute technical isolir/unisolir.

## Dashboard And Reports

Dashboard should render Billing Core first:

- active customers
- monthly revenue
- receivables
- collection rate
- recent payment/invoice status

Network widgets are conditional:

- MikroTik card only if `mikrotik` active
- OLT/map card only if `fiber_network` active

Reports should keep finance/customer/reseller/notification reports visible. Network reports should use capability gates and the `fiber_network` flag for OLT + map sections.

## API Proxy Behavior

Generic and route-specific Next.js proxies should preserve backend add-on status:

```json
{
  "success": false,
  "error": {
    "code": "MODULE_NOT_ENABLED",
    "message": "modul belum aktif untuk tenant ini"
  }
}
```

The HTTP status should stay `403`, not `502`.

## Migration Strategy

1. Add new enum values in additive migrations.
2. Keep old `pppoe` package/customer values readable.
3. Update UI defaults for Billing-only tenants to neutral values.
4. Add capability-aware validation.
5. Add smoke tests for all entitlement combinations.

