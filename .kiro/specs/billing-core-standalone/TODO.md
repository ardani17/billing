# TODO Billing Core Standalone

## Confirmed

- [x] Billing Core endpoint smoke passes with optional add-ons inactive for customers, packages, invoices, payments, and dashboard report.
- [x] MikroTik and Fiber Network are commercial add-ons only.
- [x] OLT + Peta Jaringan must stay one combined add-on: `fiber_network`.
- [x] Billing Core must support manual customer data entry without router/OLT dependencies.

## Found Gaps

- [x] Customer creation still defaults to PPPoE in the web UI.
- [x] Customer connection enum has no neutral/manual mode.
- [x] Customer coordinates are still required in the original customer table design.
- [x] Customer import template still includes PPPoE/router/ODP columns by default.
- [x] Package type is still named `pppoe` for monthly billing packages.
- [x] Package form still shows MikroTik profile even when MikroTik is inactive.
- [x] Billing isolir/suspend flows still need a clean no-router path when MikroTik is inactive.
- [x] Dashboard still calls network summaries when add-ons are inactive.
- [x] MikroTik route-specific proxy currently wraps module-disabled response as 502.
- [x] Report gates need consistent `fiber_network` naming for OLT + Peta Jaringan.

## Remaining Follow-up

- [x] Make import/export columns dynamically capability-aware, not only the base Billing-only template.
- [x] Clarify auto-isolir settings copy for tenants without MikroTik.
- [x] Add focused backend tests for manual customer and monthly package behavior.
- [x] Run a separate CHR smoke with MikroTik module active after Billing-only work is merged.

## Latest Verification

- [x] `go test ./internal/domain ./internal/usecase ./internal/handler` passes in `services/billing-api`.
- [x] `go test ./internal/domain ./internal/usecase ./internal/handler` passes in `services/network-service`.
- [x] `npm.cmd --prefix apps/web run build` passes.
- [x] Local Billing-only smoke passes with MikroTik inactive: import template, export job, settings billing, dashboard report, and module-disabled MikroTik response.
- [x] CHR MikroTik smoke passes with module enabled temporarily: status summary, API test, API-SSL test, PPPoE users list, and session count.
- [x] MikroTik module restored to inactive after smoke and pending sync count remains `0`.

## Recommended Work Order

1. Customer standalone mode.
2. Package standalone mode.
3. Billing isolir no-router behavior.
4. Import/export capability-aware templates.
5. Dashboard/report capability cleanup.
6. Smoke tests for all entitlement combinations.
