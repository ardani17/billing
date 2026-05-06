# TODO Subscription Module Gating

## Product Decision

- [x] Billing Core includes notification, reporting, payment gateway, reseller/voucher, and settings.
- [x] MikroTik is the only standalone network add-on.
- [x] OLT + Peta Jaringan is one combined add-on named `fiber_network`.
- [x] OLT and Peta Jaringan are not sold separately.

## Now

- [x] Update diskusi source of truth.
- [x] Create planning spec.

## Next

- [x] Implement tenant module entitlement backend.
- [x] Add backend route/event guards.
- [x] Add frontend menu/widget gating.
- [x] Run Billing Core read smoke with add-ons disabled.
- [ ] Add Super Admin entitlement UI.
- [ ] Add Tenant Admin subscription view.
- [ ] Run Billing Core write smoke with manual customer/package data.
- [ ] Implement Billing Core Standalone spec in `.kiro/specs/billing-core-standalone`.

## Deferred

- [ ] Automated commercial charging integration.
- [ ] OLT real hardware validation.
