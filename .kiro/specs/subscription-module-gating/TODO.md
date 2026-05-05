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

- [ ] Implement tenant module entitlement backend.
- [ ] Add backend route/event guards.
- [ ] Add frontend menu/widget gating.
- [ ] Add Super Admin entitlement UI.
- [ ] Add Tenant Admin subscription view.
- [ ] Run Billing Core smoke with add-ons disabled.

## Deferred

- [ ] Automated commercial charging integration.
- [ ] OLT real hardware validation.
