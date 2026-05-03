# Implementation Plan: Invoice Generation Module

## Overview

Bottom-up implementation of the Invoice Generation module for ISPBoss billing-api. Starts with database migrations (7 tables), then domain entities (invoice state machine, prorate calculation, invoice number formatting, late fee calculation, billing settings, credit/debit notes), sqlc queries, repositories, usecases (InvoiceUsecase, InvoiceActionUsecase, InvoiceCronUsecase, RecurringItemUsecase, CreditNoteUsecase, DebitNoteUsecase), handlers (InvoiceHandler, InvoiceActionHandler, RecurringItemHandler, CreditNoteHandler, DebitNoteHandler), router wiring, and finally the asynq worker for cron jobs. Each task builds on the previous and is independently testable. All code is Go, using existing patterns from the customer/package/reseller modules (Fiber, sqlc, pgx, asynq, go-playground/validator, rapid). Monetary values are BIGINT (Rupiah). Optimistic locking via `version` field prevents double payment. Invoice number generation is atomic via `SELECT FOR UPDATE` on `invoice_sequences`.

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Create migration: create invoices table
    - Create `services/billing-api/migrations/000016_create_invoices.up.sql` — `invoices` table with 21 columns (`id`, `tenant_id`, `customer_id`, `invoice_number`, `period_month`, `period_year`, `due_date`, `subtotal`, `tax_amount`, `penalty_amount`, `discount_amount`, `credit_applied`, `total_amount`, `paid_amount`, `status`, `notes`, `is_prepaid`, `prepaid_months`, `version`, `created_at`, `updated_at`), FK to `customers(id)`, CHECK constraints (`status` IN belum_bayar/terlambat/lunas/bayar_sebagian/batal/prorate, monetary columns >= 0, `period_month` 1-12), RLS policies (tenant_isolation + tenant_insert), unique constraint `(tenant_id, invoice_number)`, composite indexes on `(tenant_id, status)`, `(tenant_id, customer_id)`, `(tenant_id, period_year, period_month)`, `(tenant_id, due_date, status)`
    - Create `services/billing-api/migrations/000016_create_invoices.down.sql` — drop policies, constraints, indexes, and table
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_

  - [x] 1.2 Create migration: create invoice_items table
    - Create `services/billing-api/migrations/000017_create_invoice_items.up.sql` — `invoice_items` table with 11 columns (`id`, `tenant_id`, `invoice_id`, `item_type`, `description`, `quantity`, `unit_price`, `amount`, `sort_order`, `metadata`, `created_at`), FK to `invoices(id)`, CHECK constraint (`item_type` IN monthly/installation/prorate_charge/prorate_credit/penalty/tax/discount/recurring/custom/credit_applied), RLS policies, composite index on `(tenant_id, invoice_id)`
    - Create `services/billing-api/migrations/000017_create_invoice_items.down.sql` — drop policies, indexes, and table
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 1.3 Create migration: create invoice_payments table
    - Create `services/billing-api/migrations/000018_create_invoice_payments.up.sql` — `invoice_payments` table with 15 columns (`id`, `tenant_id`, `invoice_id`, `amount`, `payment_method`, `payment_date`, `reference_number`, `notes`, `recorded_by_id`, `recorded_by_name`, `voided`, `voided_at`, `voided_by`, `void_reason`, `created_at`), FK to `invoices(id)`, CHECK constraints (`payment_method` IN tunai/transfer/xendit/midtrans/lainnya, `amount` > 0), RLS policies, composite indexes on `(tenant_id, invoice_id)` and `(tenant_id, payment_date)`
    - Create `services/billing-api/migrations/000018_create_invoice_payments.down.sql` — drop policies, indexes, and table
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

  - [x] 1.4 Create migration: create billing_settings table
    - Create `services/billing-api/migrations/000019_create_billing_settings.up.sql` — `billing_settings` table with 20 columns (`id`, `tenant_id`, `generate_days`, `grace_period_days`, `suspend_days`, `tax_enabled`, `tax_rate`, `penalty_enabled`, `penalty_type`, `penalty_amount`, `penalty_percentage`, `penalty_daily_amount`, `penalty_max_amount`, `invoice_prefix`, `new_customer_billing`, `timezone`, `auto_isolir`, `auto_open_isolir`, `created_at`, `updated_at`), UNIQUE constraint on `tenant_id`, CHECK constraints (`penalty_type` IN fixed/percentage/daily, `new_customer_billing` IN prorate/full_month, `generate_days` BETWEEN 1 AND 14), RLS policies
    - Create `services/billing-api/migrations/000019_create_billing_settings.down.sql` — drop policies, constraints, and table
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x] 1.5 Create migration: create customer_recurring_items table
    - Create `services/billing-api/migrations/000020_create_customer_recurring_items.up.sql` — `customer_recurring_items` table with 10 columns (`id`, `tenant_id`, `customer_id`, `description`, `amount`, `is_active`, `start_date`, `end_date`, `created_at`, `updated_at`), FK to `customers(id)`, CHECK constraint (`amount` > 0), RLS policies, composite index on `(tenant_id, customer_id, is_active)`
    - Create `services/billing-api/migrations/000020_create_customer_recurring_items.down.sql` — drop policies, indexes, and table
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

  - [x] 1.6 Create migration: create invoice_audit_logs table
    - Create `services/billing-api/migrations/000021_create_invoice_audit_logs.up.sql` — `invoice_audit_logs` table with 8 columns (`id`, `tenant_id`, `invoice_id`, `action`, `actor_id`, `actor_name`, `metadata`, `created_at`), FK to `invoices(id)`, RLS policies (SELECT + INSERT only — append-only table, no UPDATE/DELETE), composite index on `(tenant_id, invoice_id)`
    - Create `services/billing-api/migrations/000021_create_invoice_audit_logs.down.sql` — drop policies, indexes, and table
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [x] 1.7 Create migration: create invoice_sequences table
    - Create `services/billing-api/migrations/000022_create_invoice_sequences.up.sql` — `invoice_sequences` table with 7 columns (`id`, `tenant_id`, `year`, `month`, `last_seq`, `created_at`, `updated_at`), UNIQUE constraint on `(tenant_id, year, month)`, RLS policies
    - Create `services/billing-api/migrations/000022_create_invoice_sequences.down.sql` — drop constraints and table
    - _Requirements: 7.1, 7.2_

- [x] 2. Domain entities — Invoice
  - [x] 2.1 Create domain/invoice.go with Invoice entity, state machine, and errors
    - Create `services/billing-api/internal/domain/invoice.go` with: `InvoiceStatus` type and constants (`belum_bayar`, `terlambat`, `lunas`, `bayar_sebagian`, `batal`, `prorate`), `InvoiceItemType` type and constants (`monthly`, `installation`, `prorate_charge`, `prorate_credit`, `penalty`, `tax`, `discount`, `recurring`, `custom`, `credit_applied`), `PenaltyType` type and constants (`fixed`, `percentage`, `daily`), `Invoice` struct (21 fields + joined fields), `InvoiceItem` struct, `InvoicePayment` struct, `InvoiceAuditLog` struct, `ValidInvoiceTransitions` map, `CanInvoiceTransition`, `InvoiceTransition`, `AllowedInvoiceTargets` functions, `CalculateLateFee` function, domain error variables (`ErrInvoiceNotFound`, `ErrInvalidInvoiceStatusTransition`, `ErrInvoiceNotEditable`, `ErrInvoiceNotCancellable`, `ErrInvoiceConfirmationMismatch`, `ErrInvoiceDuplicate`, `ErrCreditNoteNotFound`, `ErrDebitNoteNotFound`, `ErrRecurringItemNotFound`, `ErrBillingSettingsNotFound`)
    - _Requirements: 9.1, 9.2, 9.3, 19.1, 19.2, 19.3, 19.4, 19.5_

  - [x] 2.2 Write property test: Invoice State Machine Determinism (Property 3)
    - **Property 3: Invoice status state machine determinism**
    - In `services/billing-api/internal/domain/invoice_test.go`, use `rapid.Check` to verify that for any valid `InvoiceStatus` and any target status, `InvoiceTransition` is deterministic: valid transitions yield the target status, invalid transitions return error and status remains unchanged. Terminal states (`lunas`, `batal`) have no valid outgoing transitions.
    - **Validates: Requirements 9.1, 9.3**

  - [x] 2.3 Write property test: Late Fee Capped by Max Amount (Property 11)
    - **Property 11: Late fee capped by max amount**
    - In `services/billing-api/internal/domain/invoice_test.go`, use `rapid.Check` to verify that for any billing settings with `penalty_enabled = true` and `penalty_max_amount > 0`, and for any subtotal and days_overdue, `CalculateLateFee(settings, subtotal, daysOverdue)` returns a value less than or equal to `penalty_max_amount`.
    - **Validates: Requirements 19.5**

- [x] 3. Domain entities — Billing Settings and Credit/Debit Notes
  - [x] 3.1 Create domain/billing_settings.go with BillingSettings entity
    - Create `services/billing-api/internal/domain/billing_settings.go` with: `BillingSettings` struct (20 fields including generate_days, grace_period_days, suspend_days, tax_enabled, tax_rate, penalty_enabled, penalty_type, penalty_amount, penalty_percentage, penalty_daily_amount, penalty_max_amount, invoice_prefix, new_customer_billing, timezone, auto_isolir, auto_open_isolir)
    - _Requirements: 4.1, 8.1, 8.5, 19.1, 20.1_

  - [x] 3.2 Create domain/credit_note.go with CreditNote, DebitNote, and CustomerRecurringItem entities
    - Create `services/billing-api/internal/domain/credit_note.go` with: `CreditNote` struct, `DebitNote` struct, `DebitNoteItem` struct, `CustomerRecurringItem` struct
    - _Requirements: 5.1, 22.1, 26.1, 27.1_

- [x] 4. Domain entities — Prorate and Invoice Number
  - [x] 4.1 Create domain/invoice_prorate.go with prorate calculation functions
    - Create `services/billing-api/internal/domain/invoice_prorate.go` with: `CalculateProrate(monthlyPrice int64, remainingDays int) int64`, `CalculateProrateCredit(monthlyPrice int64, remainingDays int) int64`, `RoundUpTo500(amount int64) int64`, `RoundDownTo500(amount int64) int64`, `CalculateRemainingDays(changeDate, nextDueDate time.Time) int`
    - Uses fixed 30-day month. Charge rounds up to Rp 500, credit rounds down to Rp 500.
    - _Requirements: 17.1, 17.2, 17.3, 18.1, 18.2, 18.3, 18.4, 18.5_

  - [x] 4.2 Write property test: Prorate Calculation Correctness (Property 9)
    - **Property 9: Prorate calculation correctness**
    - In `services/billing-api/internal/domain/invoice_prorate_test.go`, use `rapid.Check` to verify that for any monthly_price (positive), remaining_days (1-30): `CalculateProrate(monthly_price, remaining_days)` returns `RoundUpTo500(monthly_price * remaining_days / 30)` and is non-negative. For upgrade (new > old): charge = `RoundUpTo500((new - old) * remaining_days / 30)` is non-negative. For downgrade (old > new): credit = `RoundDownTo500((old - new) * remaining_days / 30)` is non-negative.
    - **Validates: Requirements 17.1, 18.1, 18.2, 18.7**

  - [x] 4.3 Write property test: Rounding Functions Correctness (Property 10)
    - **Property 10: Rounding functions correctness**
    - In `services/billing-api/internal/domain/invoice_prorate_test.go`, use `rapid.Check` to verify that for any non-negative integer amount: `RoundUpTo500(amount)` >= amount, is a multiple of 500, and `RoundUpTo500(amount) - amount < 500`. `RoundDownTo500(amount)` <= amount, is a multiple of 500, and `amount - RoundDownTo500(amount) < 500`. Both are idempotent.
    - **Validates: Requirements 17.3, 18.5**

  - [x] 4.4 Create domain/invoice_number.go with invoice number formatting functions
    - Create `services/billing-api/internal/domain/invoice_number.go` with: `FormatInvoiceNumber(prefix string, year, month, seq int) string`, `FormatCreditNoteNumber(year, month, seq int) string`, `FormatDebitNoteNumber(year, month, seq int) string`
    - Format: `{prefix}-{YYYY}-{MM}-{SEQ}` with SEQ zero-padded to minimum 3 digits.
    - _Requirements: 7.3, 7.4, 26.3, 27.3_

  - [x] 4.5 Write property test: Invoice Number Format Round-Trip (Property 2)
    - **Property 2: Invoice number format round-trip**
    - In `services/billing-api/internal/domain/invoice_number_test.go`, use `rapid.Check` to verify that for any valid prefix (non-empty alphanumeric), year (2000-2099), month (1-12), and sequence (positive integer), `FormatInvoiceNumber(prefix, year, month, seq)` produces a string matching `{prefix}-{YYYY}-{MM}-{SEQ}` where SEQ is zero-padded to at least 3 digits, and parsing components back yields the original values.
    - **Validates: Requirements 7.4**

- [x] 5. Domain entities — Event payloads and repository interfaces
  - [x] 5.1 Create domain/invoice_event.go with event payload types
    - Create `services/billing-api/internal/domain/invoice_event.go` with event payload structs: `InvoiceCreatedPayload`, `InvoiceOverduePayload`, `InvoiceCancelledPayload`, `InvoiceReminderPayload`
    - _Requirements: 8.9, 10.4, 15.7, 23.1_

  - [x] 5.2 Append repository interfaces and DTOs to domain/repository.go
    - Append to `services/billing-api/internal/domain/repository.go`: `InvoiceRepository` interface (11 methods), `InvoiceItemRepository` interface (3 methods), `InvoicePaymentRepository` interface (3 methods), `InvoiceAuditLogRepository` interface (2 methods), `InvoiceSequenceRepository` interface (1 method: `NextSequence`), `BillingSettingsRepository` interface (3 methods), `CustomerRecurringItemRepository` interface (6 methods), `CreditNoteRepository` interface (3 methods), `DebitNoteRepository` interface (3 methods)
    - Add DTOs: `CreateInvoiceRequest`, `CreateInvoiceItemRequest`, `EditInvoiceRequest`, `CancelInvoiceRequest`, `RecordPaymentRequest`, `CreatePrepaidInvoiceRequest`, `BulkInvoiceIDsRequest`, `BulkCancelRequest`, `InvoiceListParams`, `InvoiceListResult`, `InvoiceDetail`, `InvoiceSummary`, `InvoiceSummaryStat`, `CreateRecurringItemRequest`, `UpdateRecurringItemRequest`, `CreateCreditNoteRequest`, `CreateDebitNoteRequest`, `DebitNoteItemRequest`, `BulkActionResult`
    - _Requirements: 11.1, 11.2, 12.1, 13.1, 14.1, 15.1, 16.1, 22.1, 23.1, 25.1, 26.1, 27.1, 28.1_

- [x] 6. Checkpoint — Domain layer complete
  - Ensure all domain files compile (`go build ./...` in `services/billing-api`). Ensure property tests pass. Ask the user if questions arise.


- [x] 7. sqlc queries
  - [x] 7.1 Create queries/invoices.sql with invoice queries
    - Create `services/billing-api/queries/invoices.sql` with sqlc queries for: `CreateInvoice` (:one), `GetInvoiceByID` (:one, with JOIN to customers for name/customer_id_seq/phone/address and packages for package_name), `UpdateInvoice` (:one), `UpdateInvoiceStatus` (:one, with WHERE version = $version for optimistic locking), `UpdateInvoicePaidAmount` (:one, with WHERE version = $version), `ExistsForPeriod` (:one, SELECT EXISTS for customer_id + period_month + period_year), `ExistsForPeriodPrepaid` (:one, SELECT EXISTS for prepaid invoices covering a period), `FindOverdueInvoices` (:many, status = 'belum_bayar' AND due_date < $current_date), `GetInvoiceSummary` (:many, GROUP BY status with COUNT and SUM), `GetInvoicesByIDs` (:many)
    - Note: `List` query is built dynamically in repository (same pattern as customer/package) — not in sqlc
    - _Requirements: 1.1, 8.2, 8.7, 10.1, 12.1, 13.1, 14.1, 25.7, 28.1_

  - [x] 7.2 Create queries/invoice_items.sql with invoice item queries
    - Create `services/billing-api/queries/invoice_items.sql` with sqlc queries for: `BulkCreateInvoiceItems` (:copyfrom), `ListInvoiceItemsByInvoice` (:many, ORDER BY sort_order ASC), `DeleteInvoiceItemsByInvoice` (:exec)
    - _Requirements: 2.1, 13.1, 14.5_

  - [x] 7.3 Create queries/invoice_payments.sql with payment queries
    - Create `services/billing-api/queries/invoice_payments.sql` with sqlc queries for: `CreateInvoicePayment` (:one), `ListPaymentsByInvoice` (:many, WHERE voided = false ORDER BY created_at ASC), `VoidPayment` (:exec, SET voided = true, voided_at, voided_by, void_reason)
    - _Requirements: 3.1, 13.3_

  - [x] 7.4 Create queries/billing_settings.sql with billing settings queries
    - Create `services/billing-api/queries/billing_settings.sql` with sqlc queries for: `GetBillingSettingsByTenantID` (:one), `UpsertBillingSettings` (:one, INSERT ON CONFLICT (tenant_id) DO UPDATE), `ListAllBillingSettings` (:many)
    - _Requirements: 4.1, 8.1_

  - [x] 7.5 Create queries/customer_recurring_items.sql with recurring item queries
    - Create `services/billing-api/queries/customer_recurring_items.sql` with sqlc queries for: `CreateRecurringItem` (:one), `GetRecurringItemByID` (:one), `UpdateRecurringItem` (:one), `DeactivateRecurringItem` (:exec, SET is_active = false), `ListRecurringItemsByCustomer` (:many), `ListActiveRecurringItemsByCustomer` (:many, WHERE is_active = true AND start_date <= $period_date AND (end_date IS NULL OR end_date > $period_date))
    - _Requirements: 5.1, 8.4, 22.1, 22.4, 22.5, 22.6, 22.7_

  - [x] 7.6 Create queries/invoice_audit_logs.sql with audit log queries
    - Create `services/billing-api/queries/invoice_audit_logs.sql` with sqlc queries for: `CreateInvoiceAuditLog` (:one), `ListAuditLogsByInvoice` (:many, ORDER BY created_at ASC)
    - _Requirements: 6.1, 24.5_

  - [x] 7.7 Create queries/invoice_sequences.sql with sequence queries
    - Create `services/billing-api/queries/invoice_sequences.sql` with sqlc queries for: `NextInvoiceSequence` (:one, INSERT ON CONFLICT DO UPDATE SET last_seq = last_seq + 1 RETURNING last_seq — or use SELECT FOR UPDATE + UPDATE pattern for atomicity)
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 7.8 Run sqlc generate to produce Go code
    - Run `sqlc generate` in `services/billing-api/` to regenerate `internal/repository/` files (adds `invoices.sql.go`, `invoice_items.sql.go`, `invoice_payments.sql.go`, `billing_settings.sql.go`, `customer_recurring_items.sql.go`, `invoice_audit_logs.sql.go`, `invoice_sequences.sql.go`, updates `models.go`)
    - Verify generated code compiles
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1, 6.1, 7.1_

- [x] 8. Checkpoint — sqlc layer complete
  - Ensure all generated files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 9. Repository implementations
  - [x] 9.1 Create repository/invoice_repo.go
    - Create `services/billing-api/internal/repository/invoice_repo.go` implementing `domain.InvoiceRepository` — wraps sqlc-generated queries, handles `List` with dynamic filtering/search/sorting/pagination (build query manually since sqlc doesn't support dynamic WHERE), implements `UpdateStatus` and `UpdatePaidAmount` with optimistic locking (WHERE version = $version), `ExistsForPeriod` and `ExistsForPeriodPrepaid` for idempotency checks, `FindOverdue` for cron job, `GetSummary` for dashboard stats, `GetByIDs` for bulk operations
    - Dynamic list query supports: filter by `status`, `period_month`/`period_year`, `package_id` (via customer JOIN), `area_id` (via customer JOIN), `search` (ILIKE on invoice_number, customer name, customer_id_seq), sorting by `invoice_number`/`due_date`/`total_amount`/`status`/`created_at`, pagination with joined customer_name, customer_id_seq, package_name
    - _Requirements: 8.2, 8.7, 10.1, 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7, 12.8, 12.9, 13.1, 25.7, 28.1_

  - [x] 9.2 Create repository/invoice_item_repo.go
    - Create `services/billing-api/internal/repository/invoice_item_repo.go` implementing `domain.InvoiceItemRepository` — wraps sqlc-generated queries, implements `BulkCreate` using sqlc copyfrom, `ListByInvoice` ordered by sort_order, `DeleteByInvoice` for edit flow
    - _Requirements: 2.1, 13.1, 14.5_

  - [x] 9.3 Create repository/invoice_payment_repo.go
    - Create `services/billing-api/internal/repository/invoice_payment_repo.go` implementing `domain.InvoicePaymentRepository` — wraps sqlc-generated queries, implements `Create`, `ListByInvoice` (non-voided only), `VoidPayment`
    - _Requirements: 3.1, 13.3_

  - [x] 9.4 Create repository/invoice_sequence_repo.go
    - Create `services/billing-api/internal/repository/invoice_sequence_repo.go` implementing `domain.InvoiceSequenceRepository` — implements `NextSequence` with atomic increment (INSERT ON CONFLICT or SELECT FOR UPDATE + UPDATE pattern), ensures no race conditions for concurrent invoice creation
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 9.5 Create repository/invoice_audit_repo.go
    - Create `services/billing-api/internal/repository/invoice_audit_repo.go` implementing `domain.InvoiceAuditLogRepository` — wraps sqlc-generated queries, implements `Create` and `ListByInvoice`
    - _Requirements: 6.1, 6.4, 24.1, 24.4, 24.5_

  - [x] 9.6 Create repository/billing_settings_repo.go
    - Create `services/billing-api/internal/repository/billing_settings_repo.go` implementing `domain.BillingSettingsRepository` — wraps sqlc-generated queries, implements `GetByTenantID`, `Upsert`, `ListAll`
    - _Requirements: 4.1, 8.1_

  - [x] 9.7 Create repository/recurring_item_repo.go
    - Create `services/billing-api/internal/repository/recurring_item_repo.go` implementing `domain.CustomerRecurringItemRepository` — wraps sqlc-generated queries, implements `Create`, `GetByID`, `Update`, `Deactivate`, `ListByCustomer`, `ListActiveByCustomer`
    - _Requirements: 5.1, 22.1, 22.4, 22.5, 22.6, 22.7_

  - [x] 9.8 Create repository/credit_note_repo.go
    - Create `services/billing-api/internal/repository/credit_note_repo.go` implementing `domain.CreditNoteRepository` and `domain.DebitNoteRepository` — wraps sqlc-generated queries for credit note and debit note CRUD
    - _Requirements: 26.1, 27.1_

- [x] 10. Checkpoint — Data layer complete
  - Ensure all repository files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.


- [x] 11. Usecase layer — InvoiceUsecase (CRUD)
  - [x] 11.1 Create usecase/invoice_usecase.go with InvoiceUsecase
    - Create `services/billing-api/internal/usecase/invoice_usecase.go` implementing `Create`, `CreatePrepaid`, `GetByID`, `Edit`, `List`, `Summary` methods
    - `Create`: validate customer exists and is active → generate invoice number via `InvoiceSequenceRepository.NextSequence` → calculate subtotal from items → if apply_tax: calculate tax = `subtotal * tax_rate / 100` → if apply_credit and customer has credit_balance: apply credit = `min(credit_balance, total)` → atomically reduce customer credit_balance → create invoice with status `belum_bayar` → bulk create invoice items → write audit log (`invoice.created_manual`) → publish `invoice.created` event → return invoice
    - `CreatePrepaid`: validate customer → generate invoice number → create line items for each month → if discount_months > 0: add discount item → set `is_prepaid=true`, `prepaid_months=months` → create invoice → write audit log (`invoice.created_prepaid`) → return invoice
    - `GetByID`: fetch invoice → fetch items → fetch payments (non-voided) → optionally fetch audit logs → return `InvoiceDetail`
    - `Edit`: fetch invoice → verify status == `belum_bayar` (else return `ErrInvoiceNotEditable`) → delete old items → recalculate subtotal/tax/credit/total → update invoice (increment version) → bulk create new items → write audit log (`invoice.edited`) → return invoice
    - `List`: delegate to repository with params, apply defaults (page=1, page_size=25)
    - `Summary`: delegate to repository `GetSummary`
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7, 11.8, 12.1, 12.2, 12.8, 13.1, 13.2, 13.3, 13.4, 13.5, 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 25.1, 25.2, 25.3, 25.4, 25.5, 25.6, 25.7, 25.8, 28.1, 28.2, 28.3, 28.4_

  - [x] 11.2 Write property test: Invoice Item Amount Consistency (Property 1)
    - **Property 1: Invoice item amount consistency**
    - In `services/billing-api/internal/domain/invoice_test.go`, use `rapid.Check` to verify that for any invoice item with positive quantity and positive unit_price, the computed `amount` equals `quantity * unit_price`.
    - **Validates: Requirements 2.5**

  - [x] 11.3 Write property test: Invoice Total Amount Invariant (Property 4)
    - **Property 4: Invoice total amount invariant**
    - In `services/billing-api/internal/domain/invoice_test.go`, use `rapid.Check` to verify that for any invoice with non-negative subtotal, tax_amount, penalty_amount, discount_amount, and credit_applied, the `total_amount` equals `subtotal + tax_amount + penalty_amount - discount_amount - credit_applied`, and `total_amount` >= 0.
    - **Validates: Requirements 14.5**

  - [x] 11.4 Write property test: Only belum_bayar Invoices Are Editable (Property 12)
    - **Property 12: Only belum_bayar invoices are editable**
    - In `services/billing-api/internal/usecase/invoice_usecase_test.go`, use `rapid.Check` to verify that for any invoice whose status is NOT `belum_bayar`, attempting to edit returns `ErrInvoiceNotEditable` and the invoice remains unchanged.
    - **Validates: Requirements 14.2**

- [x] 12. Usecase layer — InvoiceActionUsecase
  - [x] 12.1 Create usecase/invoice_action.go with InvoiceActionUsecase
    - Create `services/billing-api/internal/usecase/invoice_action.go` implementing `Cancel`, `RecordPayment`, `BulkReminder`, `BulkCancel`, `BulkPDF`, `ExportCSV` methods
    - `Cancel`: fetch invoice → verify status is cancellable (not `lunas` or `batal`, else return `ErrInvoiceNotCancellable`) → verify `confirmation_number` matches `invoice_number` (else return `ErrInvoiceConfirmationMismatch`) → if `credit_applied > 0`: restore credit to customer's `credit_balance` → transition status to `batal` → write audit log (`invoice.cancelled`, reason) → publish `invoice.cancelled` event → return invoice
    - `RecordPayment`: fetch invoice → verify status allows payment (not `lunas`, not `batal`) → if invoice is `terlambat` and penalty_enabled: calculate late fee via `CalculateLateFee` → add penalty item → recalculate total → create payment record → update `paid_amount` (with optimistic locking via version) → determine new status: if paid_amount >= total_amount → `lunas` (excess → customer credit_balance), elif paid_amount > 0 → `bayar_sebagian` → update status → write audit log (`invoice.payment_recorded`) → return invoice
    - `BulkReminder`: fetch invoices by IDs → for each eligible (status `belum_bayar` or `terlambat`): publish `invoice.reminder` event → write audit log (`invoice.reminder_sent`) → return `BulkActionResult`
    - `BulkCancel`: for each invoice ID: attempt cancel (same logic as single cancel) → collect successes/failures → return `BulkActionResult`
    - `BulkPDF`: fetch invoices by IDs → generate PDF for each → package into ZIP → return bytes
    - `ExportCSV`: fetch invoices with filters → format as CSV (invoice_number, customer_name, customer_id_seq, period, due_date, subtotal, tax, penalty, total, paid, status) → return bytes
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7, 16.1, 16.2, 16.3, 16.4, 16.5, 19.1, 19.2, 19.3, 19.4, 19.5, 19.6, 19.7, 21.1, 23.1, 23.2, 23.3, 23.4, 23.5, 23.6, 24.1_

  - [x] 12.2 Write property test: Credit Restoration on Cancel Round-Trip (Property 8)
    - **Property 8: Credit restoration on cancel round-trip**
    - In `services/billing-api/internal/usecase/invoice_action_test.go`, use `rapid.Check` to verify that for any invoice with `credit_applied > 0`, cancelling the invoice increases the customer's `credit_balance` by exactly the `credit_applied` amount.
    - **Validates: Requirements 15.4, 21.4**

  - [x] 12.3 Write property test: Credit Application Bounded (Property 6)
    - **Property 6: Credit application bounded**
    - In `services/billing-api/internal/usecase/invoice_action_test.go`, use `rapid.Check` to verify that for any customer with non-negative credit_balance and any invoice with positive total_amount, the credit applied equals `min(credit_balance, total_amount)`, and the resulting customer credit_balance equals `original - credit_applied`.
    - **Validates: Requirements 8.6, 21.2**

  - [x] 12.4 Write property test: Overpayment Becomes Credit (Property 13)
    - **Property 13: Overpayment becomes credit**
    - In `services/billing-api/internal/usecase/invoice_action_test.go`, use `rapid.Check` to verify that for any invoice with remaining balance R > 0 and payment amount P > R, the excess `P - R` is added to customer's `credit_balance`, invoice `paid_amount` becomes `total_amount`, and status transitions to `lunas`.
    - **Validates: Requirements 21.1**

  - [x] 12.5 Write property test: Credit Balance Non-Negative Invariant (Property 7)
    - **Property 7: Credit balance non-negative invariant**
    - In `services/billing-api/internal/usecase/invoice_action_test.go`, use `rapid.Check` to verify that for any sequence of credit operations (apply, restore on cancel, add overpayment), the customer's `credit_balance` remains >= 0 at all times.
    - **Validates: Requirements 21.5**

- [x] 13. Usecase layer — InvoiceCronUsecase
  - [x] 13.1 Create usecase/invoice_cron.go with InvoiceCronUsecase
    - Create `services/billing-api/internal/usecase/invoice_cron.go` implementing `ProcessAutoGenerate` and `ProcessOverdueUpdate` methods
    - `ProcessAutoGenerate`: list all billing settings → for each tenant: find eligible customers (current_date == due_date - generate_days, status = aktif) → for each customer: check `ExistsForPeriod` (idempotent) → check `ExistsForPeriodPrepaid` (skip if prepaid covers period) → generate invoice number → fetch package (capture `monthly_price` as price snapshot) → if first invoice for customer AND package has `installation_fee > 0`: add installation item (`item_type=installation`) → fetch active recurring items → calculate subtotal (monthly + installation + recurring items) → if tax_enabled: add tax item → if customer has credit_balance: apply credit → create invoice with status `belum_bayar` → create items → write audit log (`invoice.generated`, actor=System) → publish `invoice.created` event
    - `ProcessOverdueUpdate`: find all invoices with status `belum_bayar` and due_date < current_date → for each: transition to `terlambat` → write audit log (`invoice.overdue`, actor=System) → publish `invoice.overdue` event
    - Process each tenant/customer independently — failure for one does not block others. Errors logged with zerolog.
    - IMPORTANT: All prices captured at generation time are point-in-time snapshots — subsequent package price changes do NOT affect existing invoices (Req 30)
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8, 8.9, 10.1, 10.2, 10.3, 10.4, 29.1, 29.2, 29.3, 29.4, 30.1, 30.2_

  - [x] 13.2 Write property test: Tax Calculated on Subtotal Excluding Penalty (Property 5)
    - **Property 5: Tax calculated on subtotal excluding penalty**
    - In `services/billing-api/internal/usecase/invoice_cron_test.go`, use `rapid.Check` to verify that for any positive subtotal and positive tax_rate, the tax amount equals `round(subtotal * tax_rate / 100)` where subtotal is the sum of non-tax, non-penalty line items. Penalty does not affect tax calculation.
    - **Validates: Requirements 8.5, 20.2, 20.4**

- [x] 14. Usecase layer — RecurringItemUsecase
  - [x] 14.1 Create usecase/recurring_item_usecase.go with RecurringItemUsecase
    - Create `services/billing-api/internal/usecase/recurring_item_usecase.go` implementing `Create`, `List`, `Update`, `Delete` methods
    - `Create`: validate customer exists → create recurring item with `is_active=true` → return item
    - `List`: delegate to repository `ListByCustomer`
    - `Update`: fetch item → verify belongs to customer → update fields → return item
    - `Delete`: fetch item → verify belongs to customer → deactivate (set `is_active=false`) → soft delete
    - _Requirements: 22.1, 22.2, 22.3, 22.4, 22.5, 22.6_

- [x] 15. Usecase layer — CreditNoteUsecase and DebitNoteUsecase
  - [x] 15.1 Create usecase/credit_note_usecase.go with CreditNoteUsecase
    - Create `services/billing-api/internal/usecase/credit_note_usecase.go` implementing `Create` method
    - `Create`: validate invoice exists → generate credit note number (CN-{YYYY}-{MM}-{SEQ}) via sequence → create credit note → if `apply_to_credit` is true: atomically increase customer's `credit_balance` → write audit log (`credit_note.created` on referenced invoice) → return credit note
    - _Requirements: 26.1, 26.2, 26.3, 26.4, 26.5, 26.6_

  - [x] 15.2 Create usecase/debit_note_usecase.go with DebitNoteUsecase
    - Create `services/billing-api/internal/usecase/debit_note_usecase.go` implementing `Create` method
    - `Create`: validate customer exists → generate debit note number (DN-{YYYY}-{MM}-{SEQ}) via sequence → create debit note with items → if `create_invoice` is true: create corresponding invoice with debit note items → write audit log (`debit_note.created`) → return debit note
    - _Requirements: 27.1, 27.2, 27.3, 27.4, 27.5_

- [x] 16. Checkpoint — Usecase layer complete
  - Ensure all usecase files compile (`go build ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.


- [x] 17. HTTP handlers — InvoiceHandler (CRUD)
  - [x] 17.1 Create handler/invoice_handler.go
    - Create `services/billing-api/internal/handler/invoice_handler.go` with `InvoiceHandler` struct (depends on `InvoiceUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewInvoiceHandler`, and methods: `List`, `Get`, `Create`, `CreatePrepaid`, `Edit`, `Summary`, `PDF`, `AuditLogs`
    - `List`: parse query params (status, period_month, period_year, package_id, area_id, search, sort_by, sort_order, page, page_size) → validate → call usecase → return paginated response
    - `Get`: parse ID + `include` query param → call usecase → return `InvoiceDetail`
    - `Create`: parse body → validate (customer_id, due_date, items, notes, apply_tax, apply_credit) → call usecase → return 201
    - `CreatePrepaid`: parse body → validate (customer_id, months, start_period_month, start_period_year, discount_months) → call usecase → return 201
    - `Edit`: parse ID + body → validate → call usecase → return 200
    - `Summary`: parse optional period_month/period_year query params → call usecase → return summary
    - `PDF`: parse ID → call usecase `GeneratePDF` → return PDF with `Content-Type: application/pdf`
    - `AuditLogs`: parse ID → call usecase (include audit logs) → return audit logs array
    - Map domain errors to HTTP responses using the error mapping table from design
    - _Requirements: 11.1, 11.2, 11.3, 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7, 12.8, 12.9, 13.1, 13.2, 13.3, 13.4, 13.5, 14.1, 14.2, 14.3, 14.4, 14.7, 16.1, 16.5, 24.5, 25.1, 25.2, 28.1, 28.3_

- [x] 18. HTTP handlers — InvoiceActionHandler
  - [x] 18.1 Create handler/invoice_action_handler.go
    - Create `services/billing-api/internal/handler/invoice_action_handler.go` with `InvoiceActionHandler` struct (depends on `InvoiceActionUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewInvoiceActionHandler`, and methods: `Cancel`, `RecordPayment`, `BulkReminder`, `BulkCancel`, `BulkPDF`, `ExportCSV`
    - `Cancel`: parse ID + body (confirmation_number, reason) → validate → call usecase → map errors (CONFIRMATION_MISMATCH, INVOICE_NOT_CANCELLABLE) → return 200
    - `RecordPayment`: parse ID + body (amount, payment_method, payment_date, reference_number, notes) → validate → call usecase → map errors (VERSION_CONFLICT) → return 200
    - `BulkReminder`: parse body (invoice_ids) → validate → call usecase → return 200 with `BulkActionResult`
    - `BulkCancel`: parse body (invoice_ids, reason) → validate → call usecase → return 200 with `BulkActionResult`
    - `BulkPDF`: parse body (invoice_ids) → validate → call usecase → return ZIP with `Content-Type: application/zip`
    - `ExportCSV`: parse query params → call usecase → return CSV with `Content-Disposition` header
    - _Requirements: 15.1, 15.2, 15.3, 15.5, 16.1, 19.6, 19.7, 23.1, 23.2, 23.3, 23.4, 23.5_

- [x] 19. HTTP handlers — RecurringItemHandler
  - [x] 19.1 Create handler/recurring_item_handler.go
    - Create `services/billing-api/internal/handler/recurring_item_handler.go` with `RecurringItemHandler` struct (depends on `RecurringItemUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewRecurringItemHandler`, and methods: `List`, `Create`, `Update`, `Delete`
    - `List`: parse customer ID from URL → call usecase → return list
    - `Create`: parse customer ID + body (description, amount, start_date, end_date) → validate → call usecase → return 201
    - `Update`: parse customer ID + item ID + body → validate → call usecase → return 200
    - `Delete`: parse customer ID + item ID → call usecase → return 200
    - Map domain errors: `ErrRecurringItemNotFound` → 404
    - _Requirements: 22.1, 22.2, 22.3, 22.4, 22.5, 22.6_

- [x] 20. HTTP handlers — CreditNoteHandler and DebitNoteHandler
  - [x] 20.1 Create handler/credit_note_handler.go
    - Create `services/billing-api/internal/handler/credit_note_handler.go` with `CreditNoteHandler` struct (depends on `CreditNoteUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewCreditNoteHandler`, and method: `Create`
    - `Create`: parse body (invoice_id, amount, reason, apply_to_credit) → validate → call usecase → return 201
    - Map domain errors: `ErrInvoiceNotFound` → 404, `ErrCreditNoteNotFound` → 404
    - _Requirements: 26.1, 26.2, 26.4_

  - [x] 20.2 Create handler/debit_note_handler.go
    - Create `services/billing-api/internal/handler/debit_note_handler.go` with `DebitNoteHandler` struct (depends on `DebitNoteUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewDebitNoteHandler`, and method: `Create`
    - `Create`: parse body (customer_id, items, due_date, create_invoice) → validate → call usecase → return 201
    - Map domain errors: `ErrCustomerNotFound` → 404, `ErrDebitNoteNotFound` → 404
    - _Requirements: 27.1, 27.2, 27.4_

- [x] 21. Checkpoint — Handler layer complete
  - Ensure all handler files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 22. Router wiring and main.go dependency injection
  - [x] 22.1 Update handler/router.go with invoice, recurring item, credit/debit note routes
    - Modify `services/billing-api/internal/handler/router.go`: add `InvoiceHandler`, `InvoiceActionHandler`, `RecurringItemHandler`, `CreditNoteHandler`, `DebitNoteHandler` to `RouterConfig` struct
    - Register invoice read routes (admin + operator + kasir GET-only): `GET /v1/invoices`, `GET /v1/invoices/summary`, `GET /v1/invoices/:id`, `GET /v1/invoices/:id/pdf`, `GET /v1/invoices/:id/audit-logs`
    - Register invoice write routes (admin + kasir): `POST /v1/invoices/:id/payment`
    - Register invoice admin-only routes (tenant_admin): `POST /v1/invoices`, `POST /v1/invoices/prepaid`, `PUT /v1/invoices/:id`, `POST /v1/invoices/:id/cancel`, `POST /v1/invoices/bulk/reminder`, `POST /v1/invoices/bulk/cancel`, `POST /v1/invoices/bulk/pdf`, `GET /v1/invoices/export`
    - Register recurring item routes (admin-only, nested under customers): `GET /v1/customers/:id/recurring-items`, `POST /v1/customers/:id/recurring-items`, `PUT /v1/customers/:id/recurring-items/:item_id`, `DELETE /v1/customers/:id/recurring-items/:item_id`
    - Register credit/debit note routes (admin-only): `POST /v1/credit-notes`, `POST /v1/debit-notes`
    - _Requirements: 11.1, 12.1, 13.1, 14.1, 15.1, 16.1, 22.1, 23.1, 23.2, 23.3, 23.4, 25.1, 26.1, 27.1, 28.1_

  - [x] 22.2 Update cmd/main.go to wire all new dependencies
    - Modify `services/billing-api/cmd/main.go`: instantiate all new repositories (`InvoiceRepo`, `InvoiceItemRepo`, `InvoicePaymentRepo`, `InvoiceAuditLogRepo`, `InvoiceSequenceRepo`, `BillingSettingsRepo`, `RecurringItemRepo`, `CreditNoteRepo`, `DebitNoteRepo`), all new usecases (`InvoiceUsecase`, `InvoiceActionUsecase`, `InvoiceCronUsecase`, `RecurringItemUsecase`, `CreditNoteUsecase`, `DebitNoteUsecase`), all new handlers (`InvoiceHandler`, `InvoiceActionHandler`, `RecurringItemHandler`, `CreditNoteHandler`, `DebitNoteHandler`), and pass all to `RouterConfig`
    - Follow the same dependency injection pattern as existing customer/package/reseller wiring
    - _Requirements: 8.1, 11.1, 12.1, 22.1, 26.1, 27.1_

- [x] 23. Checkpoint — Full module compiles and routes registered
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 24. Worker — Invoice cron jobs
  - [x] 24.1 Create worker/invoice_worker.go
    - Create `services/billing-api/internal/worker/invoice_worker.go` with `InvoiceWorker` struct that registers two asynq handlers: (1) `invoice.generate_cron` task handler — calls `InvoiceCronUsecase.ProcessAutoGenerate`, (2) `invoice.overdue_cron` task handler — calls `InvoiceCronUsecase.ProcessOverdueUpdate`
    - Register the worker in `cmd/main.go` — register periodic tasks with asynq scheduler: `invoice.generate_cron` daily at 00:01 (tenant timezone), `invoice.overdue_cron` daily at 00:05
    - _Requirements: 8.1, 10.1_

- [x] 25. Final checkpoint — All tests pass
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ensure all property tests pass (`go test ./...`). Ask the user if questions arise.

- [x] 26. Write unit tests for handlers and usecases
  - [x] 26.1 Write unit tests for InvoiceHandler
    - In `services/billing-api/internal/handler/invoice_handler_test.go`, test HTTP status codes, request parsing, response format for all CRUD endpoints, including error cases (404 INVOICE_NOT_FOUND, 422 INVOICE_NOT_EDITABLE, 400 VALIDATION_ERROR)
    - _Requirements: 11.2, 13.5, 14.3, 14.7_

  - [x] 26.2 Write unit tests for InvoiceActionHandler
    - In `services/billing-api/internal/handler/invoice_action_handler_test.go`, test HTTP status codes for cancel, record payment, bulk reminder, bulk cancel, export endpoints, including error cases (400 CONFIRMATION_MISMATCH, 422 INVOICE_NOT_CANCELLABLE, 409 VERSION_CONFLICT)
    - _Requirements: 15.2, 15.3, 23.5_

  - [x] 26.3 Write unit tests for InvoiceUsecase
    - In `services/billing-api/internal/usecase/invoice_usecase_test.go`, test business logic: manual creation with tax/credit, prepaid creation with discount, edit only belum_bayar, list with pagination
    - _Requirements: 11.4, 11.5, 11.6, 14.2, 14.5, 25.5, 25.6_

  - [x] 26.4 Write unit tests for InvoiceActionUsecase
    - In `services/billing-api/internal/usecase/invoice_action_test.go`, test cancel with credit restoration, record payment with late fee, overpayment to credit, optimistic locking conflict, bulk operations
    - _Requirements: 15.4, 19.1, 19.5, 21.1, 23.5_

  - [x] 26.5 Write unit tests for InvoiceCronUsecase
    - In `services/billing-api/internal/usecase/invoice_cron_test.go`, test auto-generate idempotency, skip prepaid periods, include recurring items, tax calculation, credit application, overdue status update
    - _Requirements: 8.2, 8.4, 8.5, 8.6, 8.7, 10.1, 10.2_

  - [x] 26.6 Write unit tests for RecurringItemUsecase
    - In `services/billing-api/internal/usecase/recurring_item_usecase_test.go`, test CRUD operations, soft delete, validation
    - _Requirements: 22.1, 22.2, 22.6_

  - [x] 26.7 Write unit tests for CreditNoteUsecase and DebitNoteUsecase
    - In `services/billing-api/internal/usecase/credit_note_usecase_test.go` and `debit_note_usecase_test.go`, test creation with credit balance update, debit note with invoice creation
    - _Requirements: 26.4, 26.5, 27.4_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after each layer
- Property tests validate universal correctness properties from the design document (15 properties total, including price snapshot immutability and installation fee first-invoice-only)
- Migration numbering starts at 000016 (after reseller/voucher migrations 000012-000015)
- The `invoices` table depends on `customers` (FK) — existing table
- The `invoice_audit_logs` table is SEPARATE from the shared `audit_logs` table — different schema, append-only, invoice-specific
- Monetary values are BIGINT (Rupiah) — no floating point for money
- Prorate uses fixed 30-day month — no calendar complexity
- Invoice number generation is atomic via `SELECT FOR UPDATE` on `invoice_sequences` — prevents race conditions
- Optimistic locking via `version` field prevents double payment when webhook and manual recording happen simultaneously
- PDF generation uses `maroto` or `gofpdf` library — can start with placeholder interface
- All code comments MUST be in Indonesian; variable/function names in English
- Max 200 lines per file — split handlers and usecases into multiple files as shown in the file structure
- Dynamic `List` queries in repositories follow the same pattern as `customer_repo.go` and `package_repo.go`
- Credit balance operations MUST be atomic: use DB transactions
- Cron jobs process each tenant/customer independently — failure for one does not block others
- The existing `CustomerRepository` is reused for customer lookups and credit_balance updates
- The existing `PackageRepository` is reused for package price lookups
- `domain/repository.go` is appended (not replaced) — existing auth/customer/area/package/reseller/voucher interfaces remain
- Installation fee (`installation_fee` from package) is automatically added to the FIRST invoice only for new customers (Req 29). Subsequent invoices do not include it.
- All invoice item `unit_price` values are point-in-time price snapshots captured at generation time. Package price changes do NOT retroactively affect existing invoices (Req 30).
- `domain/repository.go` is appended (not replaced) — existing auth/customer/area/package/reseller/voucher interfaces remain
