# Design: Subscription Module Gating

## Product Decision

The product has one always-on base package and two paid add-ons:

- Billing Core
- Add-on MikroTik
- Add-on OLT + Peta Jaringan

`notification` is not a commercial module flag. It remains part of Billing Core. `olt` and `ftth_mapping` are represented commercially by one flag: `fiber_network`.

## Data Model

Add tenant entitlement storage in Billing API because tenant/subscription ownership belongs to Billing Core.

Recommended table:

`tenant_modules`

- `id`
- `tenant_id`
- `module_key`
- `status`
- `source`
- `enabled_at`
- `disabled_at`
- `updated_by`
- `created_at`
- `updated_at`

Allowed `module_key` values:

- `billing_core`
- `mikrotik`
- `fiber_network`

Allowed statuses:

- `active`
- `inactive`

`billing_core` is inserted automatically and cannot be disabled.

## Backend Guard

Create shared entitlement lookup:

- `IsModuleEnabled(ctx, tenantID, moduleKey)`
- `RequireModule(moduleKey)` middleware/helper

Guard rules:

- Billing API core routes do not need add-on guards.
- Network Service MikroTik route groups require `mikrotik`.
- Network Service OLT, ODP, and network map route groups require `fiber_network`.
- Event workers check entitlement before running network side effects.

Error response:

```json
{
  "success": false,
  "error": {
    "code": "MODULE_NOT_ENABLED",
    "message": "modul tidak aktif untuk tenant ini"
  }
}
```

## Frontend Gating

Frontend should fetch session/tenant capabilities once and expose a small helper:

- `canUse('mikrotik')`
- `canUse('fiber_network')`

Gating:

- Hide MikroTik sidebar/nav/widget when `mikrotik` inactive.
- Hide OLT and Peta Jaringan sidebar/nav/widget when `fiber_network` inactive.
- Hide customer detail network shortcuts that depend on inactive add-ons.
- Keep Billing, Notification, Reports, Reseller/Voucher, and Settings visible according to RBAC.

## Super Admin

Owner app can manage tenant subscription:

- package name
- customer limit tier
- add-on MikroTik active/inactive
- add-on OLT + Peta Jaringan active/inactive
- effective date
- audit note

## Tenant Admin

Tenant admin sees:

- current Billing Core package/tier
- active add-ons
- inactive add-ons with upgrade CTA
- upgrade request history

Tenant admin cannot directly activate paid add-ons without owner approval.

## Graceful Degradation

- Disabled add-on menus are hidden.
- Disabled add-on APIs return `MODULE_NOT_ENABLED`.
- Disabled add-on events are skipped and logged.
- Existing add-on data is preserved.
- Billing Core never requires MikroTik, OLT, or Peta Jaringan to complete invoice/payment/customer flows.
