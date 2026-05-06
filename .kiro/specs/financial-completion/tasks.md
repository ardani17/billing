# Implementation Plan

- [x] 1. Implement billing settings API
  - Add or complete `GET /api/v1/settings/billing`.
  - Add or complete `PUT /api/v1/settings/billing`.
  - Reuse existing billing settings model/repository where possible.
  - Validate tax rate, penalty rule, due date, grace period, and invoice prefix.
  - Add authorization guard for tenant admin or finance admin.
  - Add backend tests for load, save, validation, and unauthorized access.

- [x] 2. Replace generic billing settings UI with live form
  - Convert `/settings/billing` from placeholder/generic page into a live settings page.
  - Add sections for invoice numbering, tax, penalty, due date, reminder, and billing defaults.
  - Add loading, save, validation, empty, and permission-denied states.
  - Ensure saved settings are reloaded from API after update.

- [x] 3. Complete invoice financial operations UI
  - Add prepaid invoice creation flow.
  - Add edit, cancel, PDF, reminder, and export actions where missing.
  - Add bulk select and bulk action toolbar.
  - Show subtotal, discount, tax, penalty, paid amount, and outstanding amount consistently.
  - Ensure destructive actions require confirmation and reason where needed.

- [x] 4. Finalize invoice bulk PDF
  - Replace placeholder bulk PDF logic with final multi-invoice PDF generation.
  - Ensure selected invoices are included in the generated file.
  - Add backend test or smoke verification for valid file output.
  - Wire the finalized endpoint to the invoice bulk UI.

- [x] 5. Add credit note and debit note workflow
  - Add list/detail API if current backend only supports creation.
  - Add credit note and debit note actions from invoice detail.
  - Show note history on invoice detail.
  - Persist reason, amount, user, target invoice/customer, and audit trail.
  - Reflect credit/debit note impact in invoice balances and reports.

- [x] 6. Add customer recurring item UI
  - Add recurring item section to customer detail.
  - Support create, edit, activate, deactivate, and list.
  - Show amount, period, status, start date, and next billing inclusion.
  - Verify monthly invoice generation includes active recurring items.

- [x] 7. Complete payment operations UI
  - Add quick payment by customer and open invoice.
  - Add multi-invoice payment and pay-all flow.
  - Add receipt download/print action.
  - Add proof upload/view flow.
  - Add void payment flow with reason and audit log.
  - Add payment import result screen with per-row success/failure.

- [x] 8. Build financial reconciliation dashboard
  - Add route and page for finance reconciliation.
  - Implement period and area/cabang filters.
  - Show invoice issued, payment collected, outstanding, expense, voucher impact, credit/debit note impact, and net collection.
  - Add anomaly list for mismatch or unresolved items.
  - Add export action using active filters.

- [x] 9. Implement report settings page
  - Add `/settings/reports` route.
  - Add KPI target management.
  - Add report schedule management.
  - Add custom report template management.
  - Show schedule job result history where data exists.
  - Enforce settings permissions.

- [x] 10. Strengthen expense and profit-loss integration
  - Verify expense category and recurring expense flows end-to-end.
  - Ensure profit-loss includes revenue, discount, tax, voucher impact, expense, and net profit.
  - Add or adjust filters so revenue and expense use the same period and area scope.
  - Add audit logging for expense update/delete if missing.

- [x] 11. Add release verification
  - Run `go test ./...` for changed backend services.
  - Restore or document frontend dependency setup so web build can run.
  - Run web build after frontend changes.
  - Perform finance smoke test covering settings billing, invoice, payment, expense, reports, and reconciliation.
  - Update audit document with completion status after implementation.
