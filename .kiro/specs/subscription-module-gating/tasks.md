# Tasks: Subscription Module Gating

- [x] 1. Finalize entitlement model
  - [x] 1.1 Add `tenant_modules` migration in Billing API
  - [x] 1.2 Add domain constants for `billing_core`, `mikrotik`, `fiber_network`
  - [x] 1.3 Add repository and usecase for tenant module lookup

- [x] 2. Backend API guards
  - [x] 2.1 Add shared `MODULE_NOT_ENABLED` error response
  - [x] 2.2 Guard MikroTik routes with `mikrotik`
  - [x] 2.3 Guard OLT, ODP, and network map routes with `fiber_network`
  - [x] 2.4 Keep Billing Core APIs unguarded by add-on flags

- [x] 3. Event guards
  - [x] 3.1 Skip billing-to-MikroTik events when `mikrotik` is inactive
  - [x] 3.2 Skip fiber jobs/events when `fiber_network` is inactive
  - [x] 3.3 Log skipped add-on events without marking Billing Core flow failed

- [ ] 4. Frontend capability gating
  - [x] 4.1 Expose tenant module capabilities to web app
  - [x] 4.2 Hide MikroTik menu/widgets when inactive
  - [x] 4.3 Hide OLT and Peta Jaringan menu/widgets when inactive
  - [ ] 4.4 Hide customer/package fields that depend on inactive add-ons
  - [x] 4.5 Keep Notification, Reports, Payment Gateway, and Reseller/Voucher under Billing Core

- [ ] 5. Subscription UI
  - [ ] 5.1 Add Tenant Admin subscription view with current add-ons
  - [ ] 5.2 Add upgrade request CTA/history
  - [ ] 5.3 Add Super Admin tenant entitlement controls
  - [ ] 5.4 Write audit log for entitlement changes

- [ ] 6. Tests and smoke
  - [x] 6.1 Unit test module lookup and route guards via service test suites/build
  - [x] 6.2 Integration test disabled MikroTik API returns `MODULE_NOT_ENABLED`
  - [x] 6.3 Integration test disabled Fiber Network API returns `MODULE_NOT_ENABLED`
  - [ ] 6.4 Smoke Billing Core flows with both add-ons disabled
  - [ ] 6.5 Smoke MikroTik flows with `mikrotik` enabled
