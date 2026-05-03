# Implementation Plan: Manual Payment Recording Module

## Overview

Bottom-up implementation of the Manual Payment Recording module for ISPBoss billing-api. Starts with database migrations (receipt_sequences table, invoice_payments column additions, new indexes), then domain entities (AllocatePaymentFIFO pure function, FormatReceiptNumber, DeterminePostVoidStatus, payment DTOs, domain errors), property tests for pure domain functions, repository extensions (new methods on existing interfaces + ReceiptSequenceRepository), sqlc queries (receipt_sequences, payment list dynamic query), usecase layer (PaymentUsecase with methods split across files), handler layer (PaymentHandler), router wiring and main.go DI, and unit tests. Each task builds on the previous and is independently testable. All code is Go, using existing patterns (Fiber, sqlc, pgx, asynq, go-playground/validator, rapid). Monetary values are BIGINT (Rupiah). FIFO allocation is a pure domain function. Receipt sequence uses the same `SELECT FOR UPDATE` pattern as invoice_sequences. The module EXTENDS existing interfaces (InvoiceRepository, InvoicePaymentRepository, CustomerRepository) — no new interfaces for existing repos.

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Create migration: create receipt_sequences table
    - Create `services/billing-api/migrations/000025_create_receipt_sequences.up.sql` — `receipt_sequences` table with 7 columns (`id` UUID PK DEFAULT gen_random_uuid(), `tenant_id` UUID NOT NULL FK to tenants(id), `year` INTEGER NOT NULL, `month` INTEGER NOT NULL, `last_seq` INTEGER NOT NULL DEFAULT 0, `created_at` TIMESTAMPTZ DEFAULT NOW(), `updated_at` TIMESTAMPTZ DEFAULT NOW()), UNIQUE constraint on `(tenant_id, year, month)`, RLS policy `receipt_sequences_tenant_policy`, index `idx_receipt_sequences_tenant_year_month` on `(tenant_id, year, month)`
    - Create `services/billing-api/migrations/000025_create_receipt_sequences.down.sql` — drop policies, indexes, and table
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 1.2 Create migration: extend invoice_payments table with receipt columns
    - Create `services/billing-api/migrations/000026_add_receipt_to_invoice_payments.up.sql` — ALTER TABLE invoice_payments ADD COLUMN `receipt_number` VARCHAR(50), ADD COLUMN `receipt_group_id` UUID (links multiple invoice_payment rows from a single multi-invoice payment), ADD COLUMN `proof_image_url` VARCHAR(500) (stores path/URL to uploaded proof of transfer image)
    - Create `services/billing-api/migrations/000026_add_receipt_to_invoice_payments.down.sql` — ALTER TABLE invoice_payments DROP COLUMN receipt_number, DROP COLUMN receipt_group_id, DROP COLUMN proof_image_url
    - _Requirements: 8.1, 8.4, 8.5, 17.5_

  - [x] 1.3 Create migration: add payment-related indexes
    - Create `services/billing-api/migrations/000027_add_payment_indexes.up.sql` — create indexes: `idx_invoice_payments_payment_date` on `(tenant_id, payment_date DESC, created_at DESC) WHERE voided = false`, `idx_invoice_payments_method` on `(tenant_id, payment_method) WHERE voided = false`, `idx_invoice_payments_duplicate_check` on `(tenant_id, invoice_id, amount, payment_method, payment_date) WHERE voided = false`, `idx_customers_search_payment` GIN trigram index on `(name || ' ' || customer_id_seq || ' ' || phone) WHERE status IN ('aktif', 'isolir') AND deleted_at IS NULL`, `idx_invoices_open_by_customer` on `(customer_id, due_date ASC) WHERE status IN ('belum_bayar', 'terlambat', 'bayar_sebagian')`
    - Create `services/billing-api/migrations/000027_add_payment_indexes.down.sql` — drop all indexes
    - _Requirements: 1.1, 3.1, 4.1, 5.1, 13.1_

- [x] 2. Domain entities — Payment
  - [x] 2.1 Create domain/payment.go with AllocatePaymentFIFO pure function, types, and errors
    - Create `services/billing-api/internal/domain/payment.go` with: `PaymentAllocation` struct (InvoiceID, InvoiceNumber, AllocatedAmt, NewPaidAmount, NewStatus), `FIFOInput` struct (InvoiceID, InvoiceNumber, TotalAmount, PaidAmount, Status), `FIFOResult` struct (Allocations, TotalAllocated, ExcessToCredit), `AllocatePaymentFIFO(invoices []FIFOInput, amount int64) FIFOResult` pure function, `DeterminePostVoidStatus(paidAmount, totalAmount int64, dueDate time.Time, now time.Time) InvoiceStatus` pure function, domain error variables (`ErrPaymentNotFound`, `ErrPaymentAlreadyVoided`, `ErrVoidTimeLimitExceeded`, `ErrNoOpenInvoices`, `ErrInvalidInvoiceSelection`, `ErrSearchTermTooShort`, `ErrCSVTooLarge`, `ErrConcurrentModification`, `ErrFileTooLarge`, `ErrInvalidFileFormat`, `ErrProofNotFound`)
    - AllocatePaymentFIFO invariants: TotalAllocated + ExcessToCredit == amount; for each allocation, AllocatedAmt <= (TotalAmount - PaidAmount); if fully paid → NewStatus = lunas; if partially paid → NewStatus = bayar_sebagian
    - DeterminePostVoidStatus: if paidAmount == 0 && dueDate after now → belum_bayar; if paidAmount == 0 && dueDate before/equal now → terlambat; if 0 < paidAmount < totalAmount → bayar_sebagian
    - _Requirements: 5.1, 5.5, 5.6, 5.7, 5.8, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 11.1, 11.2, 11.3, 11.4, 16.5_

  - [x] 2.2 Create domain/receipt.go with FormatReceiptNumber, ParseReceiptNumber, and event payloads
    - Create `services/billing-api/internal/domain/receipt.go` with: `FormatReceiptNumber(year, month, seq int) string` (format PAY-{YYYY}-{MM}-{SEQ} with 4-digit minimum zero-padded seq), `zeroPadReceiptSeq(seq int) string`, `ParseReceiptNumber(receiptNumber string) (int, int, int, error)`, `PaymentRecordedPayload` struct, `PaymentVoidedReIsolirPayload` struct
    - _Requirements: 7.4, 8.1, 8.2, 8.6, 11.8, 14.1, 14.2_

  - [x] 2.3 Add payment DTOs to domain/repository.go
    - Append to `services/billing-api/internal/domain/repository.go`: `PaymentListParams` struct (TenantID, Page, PageSize, PaymentMethod, DateFrom, DateTo, RecordedBy, Search, IncludeVoided), `PaymentListItem` struct (ID, InvoiceID, InvoiceNumber, CustomerName, CustomerIDSeq, Amount, PaymentMethod, PaymentDate, ReferenceNumber, ReceiptNumber, RecordedByName, Voided, VoidReason, ProofImageURL, CreatedAt), `PaymentListResult` struct (Data, Pagination), `PaymentSummary` struct (Today, ThisMonth, ByMethod), `PaymentSummaryStat` struct (Count, TotalAmount), `OpenInvoicesResponse` struct (Invoices, TotalArrears), `OpenInvoiceItem` struct, `MultiPaymentRequest` struct, `MultiPaymentResponse` struct, `PayAllRequest` struct, `VoidPaymentRequest` struct, `VoidPaymentResponse` struct, `ReceiptData` struct, `ReceiptInvoice` struct, `BulkImportResponse` struct, `BulkImportResult` struct, `ActorInfo` struct (ID, Name)
    - _Requirements: 1.1, 1.2, 1.7, 1.8, 2.1, 2.2, 2.3, 2.4, 4.1, 4.2, 4.3, 5.2, 5.3, 5.12, 6.2, 6.5, 9.1, 9.3, 12.7_

- [x] 3. Domain entities — Repository interface extensions
  - [x] 3.1 Extend InvoicePaymentRepository interface with new methods
    - Append new methods to `InvoicePaymentRepository` interface in `services/billing-api/internal/domain/repository.go`: `GetByID(ctx context.Context, id string) (*InvoicePayment, error)`, `ListWithFilters(ctx context.Context, params PaymentListParams) (*PaymentListResult, error)`, `GetSummary(ctx context.Context, tenantID string, timezone string, periodMonth, periodYear *int) (*PaymentSummary, error)`, `FindDuplicate(ctx context.Context, customerID string, amount int64, method string, paymentDate time.Time) (bool, error)`
    - _Requirements: 1.1, 2.1, 9.1, 13.1_

  - [x] 3.2 Extend InvoiceRepository interface with new methods
    - Append new methods to `InvoiceRepository` interface in `services/billing-api/internal/domain/repository.go`: `FindOpenByCustomer(ctx context.Context, customerID string) ([]*Invoice, error)`, `FindOpenByCustomerForUpdate(ctx context.Context, customerID string) ([]*Invoice, error)`, `GetByIDsForUpdate(ctx context.Context, ids []string) ([]*Invoice, error)`
    - _Requirements: 4.1, 5.1, 5.4, 16.1_

  - [x] 3.3 Extend CustomerRepository interface with new method
    - Append new method to `CustomerRepository` interface in `services/billing-api/internal/domain/repository.go`: `SearchForPayment(ctx context.Context, tenantID, searchTerm string) ([]*Customer, error)`
    - _Requirements: 3.1, 3.2, 3.3_

  - [x] 3.4 Add ReceiptSequenceRepository interface
    - Append new interface to `services/billing-api/internal/domain/repository.go`: `ReceiptSequenceRepository` interface with `NextSequence(ctx context.Context, tenantID string, year, month int) (int, error)` — atomically increments and returns next receipt sequence, creates row if none exists, uses SELECT FOR UPDATE
    - _Requirements: 7.1, 7.2, 7.3_

- [x] 4. Property tests for pure domain functions
  - [x] 4.1 Write property test: FIFO Allocation Sum Invariant (Property 1)
    - **Property 1: FIFO Allocation Sum Invariant**
    - In `services/billing-api/internal/domain/payment_test.go`, use `rapid.Check` to verify that for any list of open invoices (each with total_amount > 0, paid_amount >= 0, paid_amount < total_amount) and for any positive payment amount, `AllocatePaymentFIFO(invoices, amount)` produces a result where `TotalAllocated + ExcessToCredit == amount` exactly.
    - **Validates: Requirements 5.8, 16.5**

  - [x] 4.2 Write property test: FIFO Allocation Status Determination (Property 2)
    - **Property 2: FIFO Allocation Status Determination**
    - In `services/billing-api/internal/domain/payment_test.go`, use `rapid.Check` to verify that for any invoice in the FIFO allocation result: if allocated_amount equals remaining (total_amount - paid_amount) then new_status == lunas; if allocated_amount > 0 but < remaining then new_status == bayar_sebagian; invoices with allocated_amount == 0 do not appear in allocations.
    - **Validates: Requirements 5.6, 5.7**

  - [x] 4.3 Write property test: FIFO Allocation Ordering (Property 3)
    - **Property 3: FIFO Allocation Ordering**
    - In `services/billing-api/internal/domain/payment_test.go`, use `rapid.Check` to verify that for any list of open invoices sorted by due_date ascending and any positive payment amount, if invoice at index i has allocated_amount < remaining_amount, then all invoices at index j > i have allocated_amount == 0 (full allocation before moving to next).
    - **Validates: Requirements 5.1, 5.5**

  - [x] 4.4 Write property test: Pay-All Clears All Invoices (Property 4)
    - **Property 4: Pay-All Clears All Invoices**
    - In `services/billing-api/internal/domain/payment_test.go`, use `rapid.Check` to verify that when payment amount equals the sum of all remaining amounts (total_arrears), every invoice in the result has new_status == lunas and excess_to_credit == 0.
    - **Validates: Requirements 6.1, 6.4**

  - [x] 4.5 Write property test: Receipt Number Format Round-Trip (Property 5)
    - **Property 5: Receipt Number Format Round-Trip**
    - In `services/billing-api/internal/domain/receipt_test.go`, use `rapid.Check` to verify that for any valid year (2000-2099), month (1-12), and sequence (1-99999), `FormatReceiptNumber(year, month, seq)` produces a string that when parsed with `ParseReceiptNumber` yields the original year, month, and sequence. SEQ is zero-padded to minimum 4 digits.
    - **Validates: Requirements 7.4, 14.1, 14.2, 14.3**

  - [x] 4.6 Write property test: Void Status Determination (Property 8)
    - **Property 8: Void Status Determination**
    - In `services/billing-api/internal/domain/payment_test.go`, use `rapid.Check` to verify that `DeterminePostVoidStatus(paidAmount, totalAmount, dueDate, now)` returns: belum_bayar if paidAmount == 0 and dueDate > now; terlambat if paidAmount == 0 and dueDate <= now; bayar_sebagian if 0 < paidAmount < totalAmount.
    - **Validates: Requirements 11.2, 11.3, 11.4**

  - [x] 4.7 Write property test: Remaining Amount and Total Arrears (Property 10)
    - **Property 10: Remaining Amount and Total Arrears Calculation**
    - In `services/billing-api/internal/domain/payment_test.go`, use `rapid.Check` to verify that for any set of open invoices, each invoice's remaining_amount == total_amount - paid_amount, and total_arrears == sum of all remaining_amount values.
    - **Validates: Requirements 4.2, 4.3**

- [x] 5. Checkpoint — Domain layer complete
  - Ensure all domain files compile (`go build ./...` in `services/billing-api`). Ensure property tests pass. Ask the user if questions arise.


- [x] 6. sqlc queries — receipt_sequences and payment extensions
  - [x] 6.1 Create queries/receipt_sequences.sql with receipt sequence queries
    - Create `services/billing-api/queries/receipt_sequences.sql` with sqlc queries for: `NextReceiptSequence` (:one, INSERT ON CONFLICT DO UPDATE SET last_seq = receipt_sequences.last_seq + 1, updated_at = NOW() RETURNING last_seq — same atomic pattern as invoice_sequences)
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 6.2 Extend queries/invoice_payments.sql with new payment queries
    - Append to `services/billing-api/queries/invoice_payments.sql`: `GetPaymentByID` (:one, SELECT with JOIN to invoices for invoice_number), `FindDuplicatePayment` (:one, SELECT EXISTS for same customer_id, amount, payment_method, payment_date within last 24 hours WHERE voided = false), `GetPaymentSummaryToday` (:one, COUNT + SUM for current date), `GetPaymentSummaryMonth` (:one, COUNT + SUM for specified month/year), `GetPaymentSummaryByMethod` (:many, GROUP BY payment_method for specified month/year)
    - Note: `ListWithFilters` is built dynamically in repository (same pattern as invoice_repo.go List) — not in sqlc
    - _Requirements: 1.1, 2.1, 2.2, 2.3, 2.4, 2.5, 9.1, 13.1_

  - [x] 6.3 Add queries for open invoices and customer search
    - Append to `services/billing-api/queries/invoices.sql`: `FindOpenInvoicesByCustomer` (:many, WHERE customer_id = $1 AND status IN ('belum_bayar', 'terlambat', 'bayar_sebagian') ORDER BY due_date ASC), `FindOpenInvoicesByCustomerForUpdate` (:many, same as above with FOR UPDATE), `GetInvoicesByIDsForUpdate` (:many, WHERE id = ANY($1) FOR UPDATE)
    - Append to `services/billing-api/queries/customers.sql`: `SearchCustomersForPayment` (:many, WHERE tenant_id = $1 AND (name ILIKE $2 OR customer_id_seq ILIKE $2 OR phone ILIKE $2) AND status IN ('aktif', 'isolir') AND deleted_at IS NULL LIMIT 10)
    - _Requirements: 3.1, 3.2, 3.3, 4.1, 5.1, 16.1_

  - [x] 6.4 Run sqlc generate to produce Go code
    - Run `sqlc generate` in `services/billing-api/` to regenerate repository files (adds `receipt_sequences.sql.go`, updates `invoice_payments.sql.go`, `invoices.sql.go`, `customers.sql.go`, updates `models.go`)
    - Verify generated code compiles
    - _Requirements: 1.1, 7.1, 13.1_

- [x] 7. Repository implementations — payment extensions
  - [x] 7.1 Create repository/receipt_sequence_repo.go
    - Create `services/billing-api/internal/repository/receipt_sequence_repo.go` implementing `domain.ReceiptSequenceRepository` — wraps sqlc-generated `NextReceiptSequence` query, same pattern as `invoice_sequence_repo.go`
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 7.2 Extend repository/invoice_payment_repo.go with new methods
    - Add methods to existing `services/billing-api/internal/repository/invoice_payment_repo.go`: `GetByID` (wraps sqlc query), `ListWithFilters` (dynamic query with filtering by payment_method, date_from/date_to, recorded_by, search on customer name/customer_id_seq/invoice_number, pagination, include_voided flag — same dynamic query pattern as invoice_repo.go List), `GetSummary` (calls sqlc queries for today/month/by_method and assembles PaymentSummary), `FindDuplicate` (wraps sqlc query)
    - Dynamic ListWithFilters query joins with invoices (for invoice_number) and customers (for name, customer_id_seq), supports ORDER BY payment_date DESC, created_at DESC
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.1, 2.2, 2.3, 2.4, 2.5, 9.1, 13.1_

  - [x] 7.3 Extend repository/invoice_repo.go with new methods
    - Add methods to existing `services/billing-api/internal/repository/invoice_repo.go`: `FindOpenByCustomer` (wraps sqlc query), `FindOpenByCustomerForUpdate` (wraps sqlc query, must be called within transaction), `GetByIDsForUpdate` (wraps sqlc query, must be called within transaction)
    - _Requirements: 4.1, 5.1, 5.4, 16.1_

  - [x] 7.4 Extend repository/customer_repo.go with SearchForPayment method
    - Add method to existing `services/billing-api/internal/repository/customer_repo.go`: `SearchForPayment` (wraps sqlc query, returns max 10 customers with status aktif/isolir matching search term by name, customer_id_seq, or phone)
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 8. Checkpoint — Data layer complete
  - Ensure all repository files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 9. Usecase layer — PaymentUsecase (list, summary, quick payment)
  - [x] 9.1 Create usecase/payment_usecase.go with PaymentUsecase struct and list/summary/search methods
    - Create `services/billing-api/internal/usecase/payment_usecase.go` with: `PaymentUsecase` struct (invoiceRepo, itemRepo, paymentRepo, auditRepo, receiptSeqRepo, settingsRepo, customerRepo, pool *pgxpool.Pool, queueClient *asynq.Client, logger zerolog.Logger), constructor `NewPaymentUsecase`, methods: `List(ctx, params) (*PaymentListResult, error)` — delegates to paymentRepo.ListWithFilters with defaults (page=1, page_size=25), `Summary(ctx, tenantID, periodMonth, periodYear) (*PaymentSummary, error)` — delegates to paymentRepo.GetSummary, `SearchCustomers(ctx, tenantID, searchTerm) ([]*Customer, error)` — validates searchTerm >= 2 chars else return ErrSearchTermTooShort, delegates to customerRepo.SearchForPayment, `GetOpenInvoices(ctx, customerID) (*OpenInvoicesResponse, error)` — fetches open invoices via invoiceRepo.FindOpenByCustomer, calculates remaining_amount and total_arrears, returns response
    - _Requirements: 1.1, 1.2, 1.7, 2.1, 2.5, 3.1, 3.5, 4.1, 4.2, 4.3, 4.4_

  - [x] 9.2 Create usecase/payment_multi.go with multi-invoice payment and pay-all methods
    - Create `services/billing-api/internal/usecase/payment_multi.go` with methods: `RecordMultiPayment(ctx, req MultiPaymentRequest, actor ActorInfo) (*MultiPaymentResponse, error)` and `PayAll(ctx, req PayAllRequest, actor ActorInfo) (*MultiPaymentResponse, error)`
    - `RecordMultiPayment`: BEGIN transaction → if invoice_ids provided: GetByIDsForUpdate (validate all belong to customer, not lunas/batal) → else: FindOpenByCustomerForUpdate → check penalty for terlambat invoices (add late fee item if needed, recalculate total) → call AllocatePaymentFIFO → for each allocation: INSERT invoice_payment (with receipt_group_id), UPDATE invoice paid_amount + status (optimistic locking) → if excess > 0: UPDATE customer credit_balance += excess (atomic via pgxpool.Pool) → generate receipt number via receiptSeqRepo.NextSequence → UPDATE all payment rows with receipt_number → INSERT audit logs → COMMIT → publish payment.recorded events via asynq → return response
    - `PayAll`: fetch open invoices → calculate total_arrears → if no invoices: return ErrNoOpenInvoices → delegate to RecordMultiPayment with amount = total_arrears
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10, 5.11, 5.12, 5.13, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 15.1, 15.2, 15.3, 15.4, 16.1, 16.2, 16.3, 16.4, 16.5_

  - [x] 9.3 Create usecase/payment_void.go with void payment method
    - Create `services/billing-api/internal/usecase/payment_void.go` with method: `VoidPayment(ctx, paymentID string, req VoidPaymentRequest, actor ActorInfo) (*VoidPaymentResponse, error)`
    - `VoidPayment`: BEGIN transaction → fetch payment by ID (check exists, check not voided → ErrPaymentAlreadyVoided, check within 24h → ErrVoidTimeLimitExceeded) → SELECT invoice FOR UPDATE → UPDATE invoice_payment SET voided=true, voided_at, voided_by, void_reason → UPDATE invoice paid_amount -= voided_amount → call DeterminePostVoidStatus for new status → UPDATE invoice status → if payment had excess to credit: UPDATE customer credit_balance -= excess (clamp to 0, log warning if would go negative) → INSERT audit log (invoice.payment_voided) → COMMIT → if invoice returned to terlambat: publish payment.voided.re_isolir event → return response
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7, 11.8_

  - [x] 9.4 Create usecase/payment_receipt.go with receipt retrieval method
    - Create `services/billing-api/internal/usecase/payment_receipt.go` with method: `GetReceipt(ctx, paymentID string) (*ReceiptData, error)`
    - `GetReceipt`: fetch payment by ID (check exists → ErrPaymentNotFound) → fetch invoice → fetch customer → fetch tenant name from billing settings → if receipt_group_id is set: fetch all payments with same receipt_group_id for multi-invoice receipt → assemble ReceiptData with all fields (receipt_number, tenant_name, payment_date, customer_name, customer_id_seq, invoices list, total_amount, payment_method, recorded_by_name, voided flag, void_reason) → return
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 9.1, 9.2, 9.3_

  - [x] 9.5 Create usecase/payment_bulk.go with bulk CSV import method
    - Create `services/billing-api/internal/usecase/payment_bulk.go` with method: `BulkImport(ctx, csvData []byte, actor ActorInfo) (*BulkImportResponse, error)`
    - `BulkImport`: parse CSV → validate row count <= 500 (else ErrCSVTooLarge) → validate all rows (customer_id_seq exists, amount > 0, valid method, valid date) → collect validation errors → if any validation errors: return 422 with per-row errors → for each valid row: check duplicate (FindDuplicate within 24h) → if duplicate: mark as skipped → else: process payment using FIFO allocation (same logic as RecordMultiPayment) → collect results → return BulkImportResponse with total_rows, success_count, failure_count, duplicates_skipped, per-row results
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7, 12.8, 12.9, 12.10, 13.1, 13.2, 13.3_

  - [x] 9.6 Create usecase/payment_proof.go with proof upload and retrieval methods
    - Create `services/billing-api/internal/usecase/payment_proof.go` with methods: `UploadProof(ctx, paymentID string, fileData []byte, filename string) (string, error)` and `GetProof(ctx, paymentID string) ([]byte, string, error)`
    - `UploadProof`: validate payment exists (GetByID) → validate file size <= 5MB (else ErrFileTooLarge) → validate file format is JPEG/PNG/WebP by checking magic bytes (else ErrInvalidFileFormat) → generate storage path: `uploads/payments/{tenant_id}/{payment_id}/{filename}` → write file to local filesystem → update invoice_payment.proof_image_url → return URL
    - `GetProof`: fetch payment by ID → check proof_image_url is not empty (else ErrProofNotFound) → read file from storage path → detect content type → return file bytes and content type
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8_

- [x] 10. Checkpoint — Usecase layer complete
  - Ensure all usecase files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 11. HTTP handler — PaymentHandler
  - [x] 11.1 Create handler/payment_handler.go with PaymentHandler struct and all endpoint methods
    - Create `services/billing-api/internal/handler/payment_handler.go` with: `PaymentHandler` struct (paymentUsecase *usecase.PaymentUsecase, validate *validator.Validate, logger zerolog.Logger), constructor `NewPaymentHandler`, and methods:
    - `List(c *fiber.Ctx) error` — parse query params (page, page_size, payment_method, date_from, date_to, recorded_by, search, include_voided) → validate → call usecase.List → return paginated response
    - `Summary(c *fiber.Ctx) error` — parse optional period_month/period_year → call usecase.Summary → return summary
    - `SearchCustomers(c *fiber.Ctx) error` — parse search query param → call usecase.SearchCustomers → return customer list
    - `GetOpenInvoices(c *fiber.Ctx) error` — parse customer_id from URL → call usecase.GetOpenInvoices → return open invoices with total_arrears
    - `RecordMultiPayment(c *fiber.Ctx) error` — parse body → validate (customer_id, amount, payment_method, payment_date required) → extract actor from JWT context → call usecase.RecordMultiPayment → return 200 with allocations + receipt
    - `PayAll(c *fiber.Ctx) error` — parse body → validate (customer_id, payment_method, payment_date required) → extract actor → call usecase.PayAll → return 200
    - `GetReceipt(c *fiber.Ctx) error` — parse payment_id from URL → call usecase.GetReceipt → return receipt data
    - `VoidPayment(c *fiber.Ctx) error` — parse payment_id from URL + body (reason) → validate → extract actor → call usecase.VoidPayment → return 200
    - `BulkImport(c *fiber.Ctx) error` — parse multipart file → read CSV bytes → extract actor → call usecase.BulkImport → return response
    - `UploadProof(c *fiber.Ctx) error` — parse payment_id from URL + multipart file → validate file size (max 5MB) and format (JPEG/PNG/WebP) → call usecase.UploadProof → return 200 with proof_image_url
    - `GetProof(c *fiber.Ctx) error` — parse payment_id from URL → call usecase.GetProof → return image file with appropriate Content-Type
    - Include `mapPaymentError` helper function mapping domain errors to HTTP responses (same pattern as InvoiceActionHandler)
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1, 5.2, 5.12, 5.13, 6.1, 6.5, 6.6, 8.3, 9.1, 9.2, 9.3, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 12.1, 12.5, 12.9, 17.1, 17.2, 17.3, 17.4, 17.6, 17.7_

- [x] 12. Router wiring and main.go dependency injection
  - [x] 12.1 Update handler/router.go with payment routes
    - Modify `services/billing-api/internal/handler/router.go`: add `PaymentHandler *PaymentHandler` to `RouterConfig` struct
    - Register payment read+write routes (admin + kasir): `GET /v1/payments` (List), `GET /v1/payments/summary` (Summary), `GET /v1/payments/quick/customers` (SearchCustomers), `GET /v1/payments/quick/customers/:customer_id/invoices` (GetOpenInvoices), `POST /v1/payments/multi` (RecordMultiPayment), `POST /v1/payments/pay-all` (PayAll), `GET /v1/payments/:payment_id/receipt` (GetReceipt), `POST /v1/payments/:payment_id/proof` (UploadProof), `GET /v1/payments/:payment_id/proof` (GetProof)
    - Register payment admin-only routes (tenant_admin): `POST /v1/payments/:payment_id/void` (VoidPayment), `POST /v1/payments/import` (BulkImport)
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1, 6.1, 8.3, 9.1, 10.1, 12.1, 17.6_

  - [x] 12.2 Update cmd/main.go to wire PaymentUsecase and PaymentHandler
    - Modify `services/billing-api/cmd/main.go`: instantiate `receiptSequenceRepo := repository.NewReceiptSequenceRepo(queries)`, instantiate `paymentUsecase := usecase.NewPaymentUsecase(invoiceRepo, invoiceItemRepo, invoicePaymentRepo, invoiceAuditLogRepo, receiptSequenceRepo, billingSettingsRepo, customerRepo, dbPool, queueClient, appLogger)`, instantiate `paymentHandler := handler.NewPaymentHandler(paymentUsecase, appLogger)`, add `PaymentHandler: paymentHandler` to `RouterConfig`
    - Follow the same dependency injection pattern as existing invoice wiring
    - _Requirements: 1.1, 5.1, 7.1, 8.1_

- [x] 13. Checkpoint — Full module compiles and routes registered
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 14. Unit tests for handlers and usecases
  - [x] 14.1 Write unit tests for PaymentHandler
    - In `services/billing-api/internal/handler/payment_handler_test.go`, test HTTP status codes, request parsing, response format for all endpoints: List (pagination, filters), Summary, SearchCustomers (400 for short term), GetOpenInvoices (404 for missing customer), RecordMultiPayment (400 validation, 422 invalid selection, 409 concurrent), PayAll (422 no open invoices), GetReceipt (404 not found), VoidPayment (403 forbidden, 422 already voided, 422 time limit), BulkImport (400 CSV too large, 422 validation errors)
    - _Requirements: 1.1, 2.1, 3.5, 4.4, 5.13, 6.6, 9.2, 9.3, 10.3, 10.4, 10.5, 10.6, 12.5, 12.9_

  - [x] 14.2 Write unit tests for PaymentUsecase — list, summary, search
    - In `services/billing-api/internal/usecase/payment_usecase_test.go`, test: List with default pagination, Summary aggregation, SearchCustomers with short term error, GetOpenInvoices with remaining_amount calculation and total_arrears
    - _Requirements: 1.2, 1.7, 2.5, 3.5, 4.2, 4.3_

  - [x] 14.3 Write unit tests for PaymentUsecase — multi-payment and pay-all
    - In `services/billing-api/internal/usecase/payment_multi_test.go`, test: RecordMultiPayment with FIFO allocation, RecordMultiPayment with invoice_ids override, RecordMultiPayment with excess to credit, PayAll with multiple invoices, PayAll with no open invoices error, late fee calculation during payment, optimistic locking conflict and retry
    - _Requirements: 5.5, 5.6, 5.7, 5.8, 5.10, 6.4, 6.6, 15.1, 15.3, 16.2, 16.3_

  - [x] 14.4 Write unit tests for PaymentUsecase — void
    - In `services/billing-api/internal/usecase/payment_void_test.go`, test: VoidPayment success within 24h, VoidPayment time limit exceeded, VoidPayment already voided, VoidPayment credit balance rollback (normal and clamped to 0), VoidPayment status determination (belum_bayar, terlambat, bayar_sebagian), VoidPayment re-isolir event publishing
    - _Requirements: 10.4, 10.5, 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7, 11.8_

  - [x] 14.5 Write unit tests for PaymentUsecase — bulk import
    - In `services/billing-api/internal/usecase/payment_bulk_test.go`, test: BulkImport valid CSV, BulkImport exceeds 500 rows, BulkImport with validation errors per row, BulkImport with duplicate detection (skip), BulkImport partial success/failure results
    - _Requirements: 12.4, 12.5, 12.6, 12.7, 12.8, 12.9, 13.1, 13.2, 13.3_

  - [x] 14.6 Write unit tests for PaymentUsecase — proof upload
    - In `services/billing-api/internal/usecase/payment_proof_test.go`, test: UploadProof success (JPEG), UploadProof file too large (>5MB → ErrFileTooLarge), UploadProof invalid format (PDF → ErrInvalidFileFormat), GetProof success, GetProof no proof (ErrProofNotFound), GetProof payment not found (ErrPaymentNotFound)
    - _Requirements: 17.2, 17.3, 17.4, 17.6, 17.7_

- [x] 15. Final checkpoint — All tests pass
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ensure all property tests and unit tests pass (`go test ./...`). Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after each layer
- Property tests validate universal correctness properties from the design document (7 properties for pure domain functions)
- Migration numbering starts at 000025 (after invoice-generation migrations 000016-000024)
- The module EXTENDS existing interfaces — `InvoiceRepository`, `InvoicePaymentRepository`, `CustomerRepository` gain new methods; no new interfaces for existing repos
- `ReceiptSequenceRepository` is the only NEW interface (new table)
- All code comments MUST be in Indonesian; variable/function names in English
- Max 200 lines per file — PaymentUsecase is split across 5 files (payment_usecase.go, payment_multi.go, payment_void.go, payment_receipt.go, payment_bulk.go)
- Dynamic `ListWithFilters` query in payment_repo follows the same pattern as `invoice_repo.go` List (manual SQL building)
- Credit balance operations MUST be atomic: use `pgxpool.Pool.Begin()` transactions with `SELECT FOR UPDATE`
- AllocatePaymentFIFO is a PURE function — no DB dependencies, fully testable with property-based tests
- DeterminePostVoidStatus is a PURE function — no DB dependencies
- The existing `InvoiceActionUsecase.RecordPayment` remains unchanged for backward compatibility
- Receipt number format `PAY-{YYYY}-{MM}-{SEQ}` uses 4-digit minimum (vs 3-digit for invoice numbers)
- Void is admin-only (RBAC enforced at router level), 24-hour time limit enforced in usecase
- Bulk CSV limit of 500 rows prevents memory issues and long-running transactions
- Event publishing (payment.recorded, payment.voided.re_isolir) is non-blocking — failures logged but don't fail the main operation
- `receipt_group_id` links multiple invoice_payment rows from a single multi-invoice payment for receipt retrieval
- Proof of transfer images stored in local filesystem at `uploads/payments/{tenant_id}/{payment_id}/` — max 5MB, JPEG/PNG/WebP only
- `proof_image_url` column on `invoice_payments` stores the relative path to the uploaded image
