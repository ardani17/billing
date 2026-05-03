# Implementation Plan: Isolir System

## Overview

Bottom-up implementation of the Isolir System module for the billing-api service. The plan starts with the database migration, builds up domain entities and pure functions with property tests, then layers repository, usecase, handler, and worker components. Each task builds on the previous one, ensuring no orphaned code.

## Tasks

- [x] 1. Database migration for pending_syncs table
  - [x] 1.1 Create `migrations/000031_create_pending_syncs.up.sql`
    - Create `pending_syncs` table with columns: id (UUID PK), tenant_id (UUID FK NOT NULL), customer_id (UUID FK NOT NULL REFERENCES customers(id)), operation_type (VARCHAR NOT NULL), status (VARCHAR NOT NULL DEFAULT 'pending'), retry_count (INTEGER NOT NULL DEFAULT 0), max_retries (INTEGER NOT NULL DEFAULT 5), last_retry_at (TIMESTAMPTZ), next_retry_at (TIMESTAMPTZ), error_message (TEXT), metadata (JSONB), created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - Enable RLS with tenant isolation policies (SELECT, INSERT, UPDATE, DELETE)
    - Add CHECK constraints: operation_type IN ('isolir', 'un_isolir', 'suspend'), status IN ('pending', 'in_progress', 'completed', 'failed'), retry_count BETWEEN 0 AND max_retries
    - Create indexes: (tenant_id, customer_id), (tenant_id, status), (status, next_retry_at) WHERE status = 'pending'
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_

  - [x] 1.2 Create `migrations/000031_create_pending_syncs.down.sql`
    - Drop pending_syncs table
    - _Requirements: 1.1_

- [x] 2. Domain entities and pure functions
  - [x] 2.1 Create `domain/isolir.go`
    - Define SyncOperationType enum (isolir, un_isolir, suspend)
    - Define SyncStatus enum (pending, in_progress, completed, failed)
    - Define PendingSync struct with all fields including joined fields (CustomerName, CustomerIDSeq)
    - Define PendingSyncListResult struct with pagination
    - Define IsolirSummary struct (TotalIsolir, TotalSuspend, TotalPendingSync, RevenueAtRisk)
    - Implement `CalculateNextRetryAt(retryCount int, now time.Time) time.Time` with backoff schedule
    - Implement `currentDateInTimezone(tz string) time.Time` helper
    - Implement `daysOverdue(dueDate time.Time, currentDate time.Time) int` helper
    - Define domain errors: ErrNoPendingSync, ErrNoPenaltyToWaive, ErrOutstandingInvoicesExist
    - All comments in Indonesian, max 200 lines
    - _Requirements: 1.1, 5.4, 12.1, 12.4_

  - [x] 2.2 Create `domain/isolir_event.go`
    - Define CustomerIsolirPayload struct
    - Define CustomerUnIsolirPayload struct
    - Define CustomerSuspendPayload struct
    - Define PenaltyAddedPayload struct
    - Define task type constants: TaskAutoIsolirCron, TaskSuspendCron, TaskPeriodicSync, TaskPaymentOnlineReceived, TaskPaymentRecorded, TaskPaymentVoidedReIsolir
    - All comments in Indonesian, max 80 lines
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 2.3 Write property tests for CalculateNextRetryAt (domain/isolir_test.go)
    - **Property 1: Backoff delay calculation is deterministic and monotonically increasing**
    - Use `pgregory.net/rapid` to generate random retryCount in [0,4] and random time
    - Verify result equals now + backoffDelays[retryCount]
    - Verify delay sequence is monotonically non-decreasing
    - **Validates: Requirements 5.4**

  - [x] 2.4 Write property tests for CalculateLateFee (domain/invoice_test.go — extend)
    - **Property 2: Late fee calculation correctness across penalty types**
    - Generate random BillingSettings with penalty_enabled=true, random subtotal, random daysOverdue
    - Verify fixed returns penalty_amount, percentage returns subtotal*percentage/100, daily returns daily_amount*daysOverdue
    - Verify penalty_enabled=false always returns 0
    - **Validates: Requirements 8.2, 8.3, 8.4, 8.5**

  - [x] 2.5 Write property test for late fee cap invariant (domain/invoice_test.go — extend)
    - **Property 3: Late fee cap invariant**
    - Generate random BillingSettings with penalty_max_amount > 0, random subtotal, random daysOverdue
    - Verify result never exceeds penalty_max_amount
    - **Validates: Requirements 8.6**

  - [x] 2.6 Write property test for daysOverdue (domain/isolir_test.go)
    - **Property 5: Overdue eligibility detection with timezone awareness**
    - Generate random dueDate and currentDate pairs
    - Verify daysOverdue returns correct non-negative integer
    - Verify daysOverdue(dueDate, currentDate) > threshold iff currentDate is more than threshold days past dueDate
    - **Validates: Requirements 2.3, 4.2, 12.1, 12.2**

- [x] 3. Checkpoint — Domain layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. sqlc queries for pending_syncs
  - [x] 4.1 Create sqlc query file `queries/pending_syncs.sql`
    - Write queries: CreatePendingSync, GetPendingSyncByID, UpdatePendingSyncStatus, UpdatePendingSyncRetry, MarkPendingSyncCompleted, MarkPendingSyncFailed, FindPendingSyncsForRetry (status=pending, next_retry_at <= now, LIMIT batch), FindPendingSyncsByCustomer, FindPendingSyncsByTenantAndStatus (paginated), CountPendingSyncsByTenantAndStatuses, ResetRetryForCustomer, ResetRetryAll
    - _Requirements: 1.1, 5.2, 5.7, 6.1, 7.1_

  - [x] 4.2 Create sqlc query file `queries/invoice_isolir.sql`
    - Write queries: FindOverdueForIsolir (invoices with status belum_bayar/terlambat where current_date > due_date + grace_period_days, JOIN customers WHERE status=aktif), FindOverdueForSuspend (similar but customers WHERE status=isolir, using suspend_days), HasOutstandingInvoices (any invoice with status NOT IN lunas/batal for customer), SumOutstandingAmount, CountOutstandingInvoices
    - _Requirements: 2.3, 3.2, 4.2, 10.2, 13.1_

  - [x] 4.3 Run `sqlc generate` to produce Go code
    - Execute sqlc generate in the billing-api service directory
    - _Requirements: 1.1_

- [x] 5. Repository implementation
  - [x] 5.1 Create `repository/pending_sync_repo.go`
    - Implement PendingSyncRepository interface using generated sqlc code
    - Map between domain.PendingSync and sqlc-generated types
    - Implement all methods: Create, GetByID, UpdateStatus, UpdateRetry, MarkCompleted, MarkFailed, FindPendingForRetry, FindByCustomer, FindByTenantAndStatus, ResetRetryForCustomer, ResetRetryAll, CountByTenantAndStatuses
    - Max 180 lines, comments in Indonesian
    - _Requirements: 1.1, 5.2, 5.7, 6.1, 7.1_

  - [x] 5.2 Extend `domain/repository.go` with PendingSyncRepository interface and extended InvoiceRepository methods
    - Add PendingSyncRepository interface definition
    - Add FindOverdueForIsolir, FindOverdueForSuspend, HasOutstandingInvoices, SumOutstandingAmount, CountOutstandingInvoices to InvoiceRepository interface
    - _Requirements: 1.1, 2.3, 3.2, 4.2, 10.2, 13.1_

  - [x] 5.3 Extend existing InvoiceRepo with isolir-specific query methods
    - Implement FindOverdueForIsolir, FindOverdueForSuspend, HasOutstandingInvoices, SumOutstandingAmount, CountOutstandingInvoices in the existing invoice_repo.go (or a new invoice_repo_isolir.go file if needed for 200-line limit)
    - _Requirements: 2.3, 3.2, 4.2, 10.2, 13.1_

- [x] 6. Checkpoint — Repository layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Usecase layer
  - [x] 7.1 Create `usecase/isolir_usecase.go`
    - Define IsolirUsecase struct with all dependencies (customerRepo, invoiceRepo, invoiceItemRepo, pendingSyncRepo, settingsRepo, auditRepo, pool, queueClient, logger)
    - Implement NewIsolirUsecase constructor
    - Implement ProcessAutoIsolir(ctx): iterate tenants with auto_isolir enabled, find eligible customers, transition aktif→isolir, create pending_sync, publish events, write audit log
    - Implement ProcessSuspend(ctx): iterate tenants, find isolir customers past suspend_days, transition isolir→suspend, create pending_sync, publish events, write audit log
    - Max 200 lines, comments in Indonesian
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 4.8_

  - [x] 7.2 Create `usecase/isolir_unisolir.go`
    - Implement ProcessUnIsolir(ctx, tenantID, customerID, trigger): check customer status is isolir, check all invoices settled, transition isolir→aktif, create pending_sync, publish events, write audit log
    - Implement ProcessReactivate(ctx, customerID, actor): check customer status is suspend, check all invoices settled, transition suspend→aktif, create pending_sync, publish events, write audit log
    - Implement ProcessReIsolir(ctx, tenantID, customerID): check customer status is aktif, check has outstanding invoices past grace period, transition aktif→isolir, create pending_sync, publish events, write audit log (triggered by payment.voided.re_isolir)
    - Max 200 lines, comments in Indonesian
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7_

  - [x] 7.3 Create `usecase/isolir_sync.go`
    - Implement ProcessPeriodicSync(ctx): query pending_syncs ready for retry, re-publish events, increment retry_count, calculate next_retry_at, mark failed if max_retries reached
    - Implement ManualSync(ctx, customerID, actor): reset retry_count, re-publish event for customer
    - Implement ManualSyncAll(ctx, tenantID, actor): reset all pending/failed, re-publish events, return count
    - Max 150 lines, comments in Indonesian
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 7.4 Create `usecase/isolir_penalty.go`
    - Implement ProcessLateFee(ctx, tenantID, invoice, settings, daysOverdue): calculate fee using CalculateLateFee, add penalty line item, update invoice totals, publish event, write audit log
    - Implement WaivePenalty(ctx, invoiceID, actor): find penalty item, remove it, recalculate totals, publish event, write audit log
    - Max 150 lines, comments in Indonesian
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8, 8.9, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

  - [x] 7.5 Create `usecase/isolir_summary.go`
    - Implement GetDashboardSummary(ctx, tenantID): count isolir customers, suspend customers, pending syncs, sum revenue at risk
    - Implement GetPendingSyncs(ctx, tenantID, status, page, pageSize): paginated pending sync list
    - Max 100 lines, comments in Indonesian
    - _Requirements: 13.1, 13.2, 13.3, 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 8. Handler layer
  - [x] 8.1 Create `handler/isolir_handler.go`
    - Define IsolirHandler struct with isolirUsecase dependency
    - Implement NewIsolirHandler constructor
    - Implement ManualSync (POST /v1/isolir/sync/:customer_id) — validate params, call usecase, return count
    - Implement ManualSyncAll (POST /v1/isolir/sync-all) — call usecase, return count
    - Implement ListPendingSyncs (GET /v1/isolir/pending-syncs) — parse query params (status, page, page_size), call usecase, return paginated list
    - Implement Summary (GET /v1/isolir/summary) — call usecase, return summary
    - Implement WaivePenalty (POST /v1/invoices/:id/waive-penalty) — validate, call usecase, handle errors (404, 422)
    - Implement Reactivate (POST /v1/customers/:id/reactivate) — validate, call usecase, handle errors (404, 422)
    - Max 200 lines, comments in Indonesian
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.3, 7.4, 7.5, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 13.1, 13.2, 13.3_

- [x] 9. Worker layer
  - [x] 9.1 Create `worker/isolir_worker.go`
    - Define IsolirWorker struct with isolirUsecase dependency
    - Implement NewIsolirWorker constructor
    - Implement RegisterHandlers to register: TaskAutoIsolirCron, TaskSuspendCron, TaskPeriodicSync, TaskPaymentOnlineReceived, TaskPaymentRecorded, TaskPaymentVoidedReIsolir
    - Implement handleAutoIsolirCron — call ProcessAutoIsolir
    - Implement handleSuspendCron — call ProcessSuspend
    - Implement handlePeriodicSync — call ProcessPeriodicSync
    - Implement handlePaymentOnlineReceived — deserialize payload, call ProcessUnIsolir with trigger "payment_received"
    - Implement handlePaymentRecorded — deserialize payload, call ProcessUnIsolir with trigger "payment_received"
    - Implement handlePaymentVoidedReIsolir — deserialize PaymentVoidedReIsolirPayload, call ProcessReIsolir to transition aktif→isolir if customer has outstanding invoices past grace period
    - Note: payment.online.received is ALSO handled by GatewayWorker — both workers process the same event for different purposes
    - Max 200 lines, comments in Indonesian
    - _Requirements: 2.1, 3.1, 4.1, 5.1_

- [x] 10. Router wiring and main.go DI
  - [x] 10.1 Update `handler/router.go`
    - Add IsolirHandler field to RouterConfig struct
    - Register isolir route group under `api` (auth + tenant middleware): POST /isolir/sync/:customer_id, POST /isolir/sync-all, GET /isolir/pending-syncs, GET /isolir/summary
    - Register waive-penalty route: POST /invoices/:id/waive-penalty under invoicesAdmin group
    - Register reactivate route: POST /customers/:id/reactivate under customersWrite group
    - Apply RBAC (tenant_admin + operator for sync/pending-syncs/summary, tenant_admin only for waive-penalty and reactivate)
    - _Requirements: 6.1, 6.2, 7.1, 9.1, 10.1, 13.1_

  - [x] 10.2 Update `cmd/main.go`
    - Instantiate PendingSyncRepo
    - Instantiate IsolirUsecase with all dependencies
    - Instantiate IsolirHandler
    - Instantiate IsolirWorker and call RegisterHandlers(mux)
    - Add IsolirHandler to RouterConfig
    - Register cron jobs in scheduler: isolir.auto_isolir_cron at "0 1 * * *", isolir.suspend_cron at "0 2 * * *", isolir.periodic_sync at "*/15 * * * *"
    - _Requirements: 2.1, 4.1, 5.1_

- [x] 11. Unit tests
  - [x] 11.1 Write unit tests for IsolirHandler (handler/isolir_handler_test.go)
    - Test ManualSync: 200 success, 404 no pending sync
    - Test ManualSyncAll: 200 success with count
    - Test ListPendingSyncs: 200 with pagination, filter by status
    - Test Summary: 200 with correct structure
    - Test WaivePenalty: 200 success, 404 not found, 422 no penalty, 422 not editable
    - Test Reactivate: 200 success, 404 not found, 422 outstanding invoices, 422 invalid status
    - _Requirements: 6.1, 6.3, 7.1, 9.1, 9.2, 9.3, 10.1, 10.3, 10.4, 13.1_

  - [x] 11.2 Write property test for event payload completeness (domain/isolir_event_test.go)
    - **Property 4: Event payload completeness**
    - Generate random valid customer/tenant data
    - Construct all payload types (CustomerIsolirPayload, CustomerUnIsolirPayload, CustomerSuspendPayload, PenaltyAddedPayload)
    - Verify tenant_id and customer_id are always non-empty
    - **Validates: Requirements 11.5**

  - [x] 11.3 Write unit tests for IsolirWorker (worker/isolir_worker_test.go)
    - Test task type registration
    - Test handler dispatch for each task type
    - _Requirements: 2.1, 3.1, 4.1, 5.1_

- [x] 12. Final checkpoint
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- Migration numbering starts at 000031 (confirmed: last existing is 000030)
- Max 200 lines per file constraint applies to all new files
- All code comments must be in Indonesian
- Monetary values stored as BIGINT (Rupiah)
- The existing GatewayWorker already handles `payment.online.received` — the IsolirWorker ALSO registers for this event (both process it for different purposes)
- The existing InvoiceCronUsecase handles `invoice.overdue_cron` — late fee processing hooks into this flow via the IsolirUsecase
