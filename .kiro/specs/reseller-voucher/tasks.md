# Implementation Plan: Reseller & Voucher Management Module

## Overview

Bottom-up implementation of the Reseller & Voucher Management module for ISPBoss billing-api. Starts with database migrations (4 tables), then domain entities (reseller + voucher state machines, code generation, errors), sqlc queries, repositories, usecases (reseller CRUD, reseller actions, reseller auth, voucher generate, voucher actions, voucher purchase, voucher expiry, voucher print), handlers (admin reseller, admin voucher, reseller auth, reseller dashboard), router wiring, and finally the asynq worker. Each task builds on the previous and is independently testable. All code is Go, using existing patterns from the customer/package modules (Fiber, sqlc, pgx, asynq, go-playground/validator, rapid). Reseller auth is SEPARATE from admin auth — resellers use phone+password with their own JWT flow. Balance operations are atomic via `SELECT ... FOR UPDATE` + DB transactions.

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Create migration 000012: create resellers table
    - Create `services/billing-api/migrations/000012_create_resellers.up.sql` — `resellers` table with 13 columns (`id`, `tenant_id`, `name`, `phone`, `email`, `address`, `password_hash`, `balance`, `daily_purchase_limit`, `status`, `last_login`, `created_at`, `updated_at`), CHECK constraints (`status` IN aktif/suspended/nonaktif, `balance` >= 0, `daily_purchase_limit` >= 0), RLS policies (tenant_isolation + tenant_insert), unique constraint `(tenant_id, phone)`, composite indexes on `(tenant_id, status)` and `(tenant_id, phone)`
    - Create `services/billing-api/migrations/000012_create_resellers.down.sql` — drop policies, constraints, indexes, and table
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7_

  - [x] 1.2 Create migration 000013: create vouchers table
    - Create `services/billing-api/migrations/000013_create_vouchers.up.sql` — `vouchers` table with 14 columns (`id`, `tenant_id`, `code`, `package_id`, `reseller_id`, `status`, `sell_price_snapshot`, `reseller_price_snapshot`, `purchased_at`, `activated_at`, `expires_at`, `voided_at`, `created_at`, `updated_at`), FK to `packages(id)` and `resellers(id)`, CHECK constraint (`status` IN tersedia/terjual/aktif/selesai/expired/void), RLS policies, unique constraint `(tenant_id, code)`, composite indexes on `(tenant_id, status)`, `(tenant_id, package_id)`, `(tenant_id, reseller_id)`, `(tenant_id, status, expires_at)`
    - Create `services/billing-api/migrations/000013_create_vouchers.down.sql` — drop policies, constraints, indexes, and table
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

  - [x] 1.3 Create migration 000014: create voucher_audit_logs table
    - Create `services/billing-api/migrations/000014_create_voucher_audit_logs.up.sql` — `voucher_audit_logs` table with 8 columns (`id`, `tenant_id`, `voucher_id`, `action`, `actor_id`, `actor_name`, `metadata`, `created_at`), FK to `vouchers(id)`, RLS policies (SELECT + INSERT only — append-only table), composite indexes on `(tenant_id, voucher_id)` and `(tenant_id, created_at)`
    - Create `services/billing-api/migrations/000014_create_voucher_audit_logs.down.sql` — drop policies, indexes, and table
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 1.4 Create migration 000015: create reseller_transactions table
    - Create `services/billing-api/migrations/000015_create_reseller_transactions.up.sql` — `reseller_transactions` table with 10 columns (`id`, `tenant_id`, `reseller_id`, `type`, `amount`, `balance_before`, `balance_after`, `reference_id`, `notes`, `created_at`), FK to `resellers(id)`, CHECK constraint (`type` IN deposit/purchase/refund), RLS policies, composite indexes on `(tenant_id, reseller_id)` and `(tenant_id, reseller_id, created_at)`
    - Create `services/billing-api/migrations/000015_create_reseller_transactions.down.sql` — drop policies, indexes, and table
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

- [x] 2. Domain entities — Reseller
  - [x] 2.1 Create domain/reseller.go with Reseller entity, state machine, and errors
    - Create `services/billing-api/internal/domain/reseller.go` with: `ResellerStatus` type and constants (`aktif`, `suspended`, `nonaktif`), `Reseller` struct (13 fields + computed `TotalVouchersSold`), `ValidResellerTransitions` map, `CanResellerTransition`, `ResellerTransition`, `AllowedResellerTargets` functions, domain error variables (`ErrResellerNotFound`, `ErrResellerPhoneDuplicate`, `ErrResellerAccountDisabled`, `ErrInvalidResellerTransition`, `ErrResellerInvalidCredentials`, `ErrResellerAccountLocked`, `ErrInsufficientBalance`, `ErrDailyLimitExceeded`, `ErrVoucherForbidden`, `ErrPackageNotActive`)
    - Reuse `ErrConfirmationMismatch` from `domain/customer.go` (do not redefine)
    - _Requirements: 1.1, 1.5, 8.1, 8.2, 8.5, 8.6, 32.1, 32.2, 32.3_

  - [ ]* 2.2 Write property test: New Reseller Default Active Status (Property 1)
    - **Property 1: New Reseller Default Active Status**
    - In `services/billing-api/internal/domain/reseller_test.go`, use `rapid.Check` to verify that for any valid reseller creation, the resulting reseller always has `status` set to `aktif` and `balance` set to the requested value (or 0 if not provided), regardless of any other field values
    - **Validates: Requirements 4.1**

  - [ ]* 2.3 Write property test: Reseller State Machine Determinism (Property 2)
    - **Property 2: Reseller State Machine Determinism**
    - In `services/billing-api/internal/domain/reseller_test.go`, use `rapid.Check` to verify that for any valid `ResellerStatus` and any target status, `ResellerTransition` is deterministic: valid transitions yield the target status, invalid transitions return error and status remains unchanged. Specifically: `aktif` → [`suspended`, `nonaktif`], `suspended` → [`aktif`, `nonaktif`], `nonaktif` → [] (terminal)
    - **Validates: Requirements 8.1, 8.2, 8.5, 8.6, 32.1, 32.2, 32.3**

  - [ ]* 2.4 Write property test: Password Hash Round-Trip (Property 9)
    - **Property 9: Password Hash Round-Trip**
    - In `services/billing-api/internal/domain/reseller_test.go`, use `rapid.Check` to verify that for any plaintext password P, hashing P with bcrypt and then verifying P against the hash always succeeds, and verifying any string Q ≠ P against the hash always fails
    - **Validates: Requirements 4.5, 9.1**

- [x] 3. Domain entities — Voucher
  - [x] 3.1 Create domain/voucher.go with Voucher entity, state machine, code generation, and errors
    - Create `services/billing-api/internal/domain/voucher.go` with: `VoucherStatus` type and constants (`tersedia`, `terjual`, `aktif`, `selesai`, `expired`, `void`), `CodeFormat` type and constants (`digits`, `letters`, `mixed`), `TransactionType` type and constants (`deposit`, `purchase`, `refund`), `Voucher` struct (16 fields + joined `PackageName`, `ResellerName`), `VoucherAuditLog` struct, `ResellerTransaction` struct, `ValidVoucherTransitions` map, `CanVoucherTransition`, `VoucherTransition`, `AllowedVoucherTargets` functions, `GenerateVoucherCode` (crypto/rand, single code), `GenerateVoucherCodes` (batch with collision avoidance, maxRetries=3), domain error variables (`ErrVoucherNotFound`, `ErrInvalidVoucherTransition`, `ErrInvalidPackageType`)
    - _Requirements: 2.1, 2.5, 16.1, 16.2, 16.3, 16.4, 16.5, 16.6, 16.7, 18.1, 18.2, 18.3_

  - [ ]* 3.2 Write property test: Voucher State Machine Determinism (Property 3)
    - **Property 3: Voucher State Machine Determinism**
    - In `services/billing-api/internal/domain/voucher_test.go`, use `rapid.Check` to verify that for any valid `VoucherStatus` and any target status, `VoucherTransition` is deterministic: valid transitions yield the target, invalid transitions return error. Specifically: `tersedia` → [`terjual`, `void`], `terjual` → [`aktif`, `expired`, `void`], `aktif` → [`selesai`], and `selesai`/`expired`/`void` are terminal
    - **Validates: Requirements 18.1, 18.2, 18.3**

  - [ ]* 3.3 Write property test: Voucher Code Format Correctness (Property 5)
    - **Property 5: Voucher Code Format Correctness**
    - In `services/billing-api/internal/domain/voucher_test.go`, use `rapid.Check` to verify that for any `code_format` (digits/letters/mixed), `code_length` (6-16), and optional `prefix`, `GenerateVoucherCode` produces a code where: the random part has exactly `code_length` characters, all characters match the format charset (`[0-9]` for digits, `[A-Z]` for letters, `[A-Z0-9]` for mixed), and the full code equals `prefix + random_part`
    - **Validates: Requirements 16.1, 16.2, 16.3, 16.4, 16.5**

  - [ ]* 3.4 Write property test: Voucher Code Uniqueness (Property 6)
    - **Property 6: Voucher Code Uniqueness Within Tenant**
    - In `services/billing-api/internal/domain/voucher_test.go`, use `rapid.Check` to verify that for any batch of generated voucher codes, all full codes are unique, collision avoidance retries up to 3 times per code, and codes that fail all retries are reported as `total_failed`
    - **Validates: Requirements 15.6, 15.7, 16.7**

  - [x] 3.5 Create domain/voucher_event.go with event payload types
    - Create `services/billing-api/internal/domain/voucher_event.go` with event payload structs: `ResellerCreatedPayload`, `ResellerStatusChangedPayload`, `VoucherBatchGeneratedPayload`, `VoucherPurchasedPayload`
    - _Requirements: 34.1, 34.2, 34.3, 34.4, 34.5_

  - [x] 3.6 Append repository interfaces and DTOs to domain/repository.go
    - Append to `services/billing-api/internal/domain/repository.go`: `ResellerRepository` interface (12 methods: `Create`, `GetByID`, `GetByPhone`, `Update`, `UpdateStatus`, `UpdatePasswordHash`, `UpdateLastLogin`, `List`, `PhoneExists`, `GetForUpdate`, `UpdateBalance`, `CountTodayPurchases`), `VoucherRepository` interface (15 methods: `BulkCreate`, `GetByID`, `GetByCode`, `UpdateStatus`, `List`, `ListByReseller`, `GetAvailableByPackage`, `BulkUpdateStatus`, `BulkAssign`, `AssignToReseller`, `GetExpiredVouchers`, `CodeExists`, `GetByIDs`, `CountByResellerAndStatus`, `CountSoldToday`), `VoucherAuditLogRepository` interface (3 methods: `Create`, `BulkCreate`, `ListByVoucher`), `ResellerTransactionRepository` interface (3 methods: `Create`, `ListByReseller`, `ListDepositsByReseller`)
    - Add DTOs: `CreateResellerRequest`, `UpdateResellerRequest`, `DepositRequest`, `DeactivateResellerRequest`, `ResellerListParams`, `ResellerListResult`, `ResellerDetail`, `ResellerLoginRequest`, `ResellerLoginResponse`, `GenerateVoucherRequest`, `GenerateVoucherResult`, `VoucherListParams`, `VoucherListResult`, `VoucherDetail`, `BulkVoucherIDsRequest`, `BulkAssignRequest`, `DashboardSummary`, `BuyVoucherRequest`, `BuyVoucherResult`, `ResellerVoucherListParams`, `ResellerTxListParams`, `ResellerTxListResult`
    - _Requirements: 4.1, 5.1, 6.1, 7.1, 8.1, 9.1, 10.1, 11.1, 12.1, 15.1, 17.1, 19.1, 23.1, 24.1, 25.1, 27.1, 28.1_

- [x] 4. Checkpoint — Domain layer complete
  - Ensure all domain files compile (`go build ./...` in `services/billing-api`). Ensure property tests pass. Ask the user if questions arise.

- [x] 5. sqlc queries
  - [x] 5.1 Create queries/resellers.sql with reseller queries
    - Create `services/billing-api/queries/resellers.sql` with sqlc queries for: `CreateReseller` (:one), `GetResellerByID` (:one, with `total_vouchers_sold` subquery), `GetResellerByPhone` (:one, for login), `UpdateReseller` (:one), `UpdateResellerStatus` (:one), `UpdateResellerPasswordHash` (:exec), `UpdateResellerLastLogin` (:exec), `UpdateResellerBalance` (:exec), `GetResellerForUpdate` (:one, SELECT ... FOR UPDATE), `ResellerPhoneExists` (:one, SELECT EXISTS)
    - Note: `List` query is built dynamically in repository (same pattern as customer/package) — not in sqlc
    - _Requirements: 1.1, 4.1, 5.1, 6.1, 7.1, 8.1, 9.1, 10.1, 12.1, 30.2_

  - [x] 5.2 Create queries/vouchers.sql with voucher queries
    - Create `services/billing-api/queries/vouchers.sql` with sqlc queries for: `BulkCreateVouchers` (:copyfrom), `GetVoucherByID` (:one, with joined package_name + reseller_name), `GetVoucherByCode` (:one), `UpdateVoucherStatus` (:one), `UpdateVoucherVoid` (:one), `AssignVoucherToReseller` (:one, set snapshot + purchased_at + expires_at), `AdminAssignVoucher` (:one, no snapshot), `GetExpiredVouchers` (:many, status=terjual AND expires_at < NOW()), `VoucherCodeExists` (:one), `GetVouchersByIDs` (:many), `CountVouchersByResellerAndStatus` (:one), `CountVouchersSoldToday` (:one), `UpdateVoucherExpired` (:one)
    - Note: `List` and `ListByReseller` queries are built dynamically in repository
    - _Requirements: 2.1, 15.1, 17.1, 18.1, 19.1, 21.1, 24.1, 25.1_

  - [x] 5.3 Create queries/voucher_audit_logs.sql with audit log queries
    - Create `services/billing-api/queries/voucher_audit_logs.sql` with sqlc queries for: `CreateVoucherAuditLog` (:one), `ListVoucherAuditLogsByVoucher` (:many, ORDER BY created_at ASC)
    - _Requirements: 3.1, 3.4_

  - [x] 5.4 Create queries/reseller_transactions.sql with transaction queries
    - Create `services/billing-api/queries/reseller_transactions.sql` with sqlc queries for: `CreateResellerTransaction` (:one), `ListResellerTransactions` (:many, ORDER BY created_at DESC with LIMIT/OFFSET), `CountResellerTransactions` (:one), `ListResellerDeposits` (:many, type=deposit), `CountResellerDeposits` (:one)
    - _Requirements: 11.1, 27.1, 28.1_

  - [x] 5.5 Run sqlc generate to produce Go code
    - Run `sqlc generate` in `services/billing-api/` to regenerate `internal/repository/` files (adds `resellers.sql.go`, `vouchers.sql.go`, `voucher_audit_logs.sql.go`, `reseller_transactions.sql.go`, updates `models.go`)
    - Verify generated code compiles
    - _Requirements: 1.1, 2.1, 3.1, 11.1_

- [x] 6. Checkpoint — sqlc layer complete
  - Ensure all generated files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 7. Repository implementations
  - [x] 7.1 Create repository/reseller_repo.go
    - Create `services/billing-api/internal/repository/reseller_repo.go` implementing `domain.ResellerRepository` — wraps sqlc-generated queries, handles `List` with dynamic filtering/search/sorting/pagination (build query manually since sqlc doesn't support dynamic WHERE), implements `GetForUpdate` for row-level locking, `UpdateBalance` for atomic balance updates, `CountTodayPurchases` via sqlc query
    - All queries must include `tenant_id` filter at application level (RLS is safety net)
    - Dynamic list query supports: filter by `status`, `search` (ILIKE on name or phone), sorting by `name`/`balance`/`created_at`, pagination with `total_vouchers_sold` subquery
    - _Requirements: 4.1, 5.1, 5.2, 5.3, 5.4, 5.5, 6.1, 7.1, 8.1, 10.1, 12.1, 30.1, 30.2_

  - [x] 7.2 Create repository/voucher_repo.go
    - Create `services/billing-api/internal/repository/voucher_repo.go` implementing `domain.VoucherRepository` — wraps sqlc-generated queries, handles `List` and `ListByReseller` with dynamic filtering/search/sorting/pagination (build query manually), implements `BulkCreate` using sqlc copyfrom, `BulkUpdateStatus` and `BulkAssign` with per-item error handling, `AssignToReseller` for purchase flow, `GetExpiredVouchers` for cron
    - Dynamic list query supports: filter by `package_id`, `status`, `reseller_id`, `search` (ILIKE on code), sorting by `code`/`status`/`created_at`/`purchased_at`, pagination with joined `package_name` and `reseller_name`
    - _Requirements: 15.1, 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8, 19.2, 19.3, 21.1, 24.1, 25.1_

  - [x] 7.3 Create repository/voucher_audit_repo.go
    - Create `services/billing-api/internal/repository/voucher_audit_repo.go` implementing `domain.VoucherAuditLogRepository` — wraps sqlc-generated queries, implements `BulkCreate` by iterating and calling `CreateVoucherAuditLog` for each entry
    - _Requirements: 3.1, 3.4_

  - [x] 7.4 Create repository/reseller_tx_repo.go
    - Create `services/billing-api/internal/repository/reseller_tx_repo.go` implementing `domain.ResellerTransactionRepository` — wraps sqlc-generated queries, implements `ListByReseller` and `ListDepositsByReseller` with pagination metadata calculation
    - _Requirements: 11.1, 27.1, 28.1_

- [x] 8. Checkpoint — Data layer complete
  - Ensure all repository files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 9. Usecase layer — Reseller CRUD
  - [x] 9.1 Create usecase/reseller_usecase.go with ResellerUsecase
    - Create `services/billing-api/internal/usecase/reseller_usecase.go` implementing `Create`, `GetByID`, `Update`, `List` methods
    - `Create`: validate phone uniqueness → hash password with bcrypt → create reseller with `status=aktif`, `balance=0` (or requested) → write audit log (`reseller.created`) → publish `reseller.created` event
    - `GetByID`: fetch reseller → optionally fetch audit logs → return `ResellerDetail`
    - `Update`: fetch existing → validate phone uniqueness (exclude self) → update → compute changed fields → write audit log (`reseller.updated`) with old/new values
    - `List`: delegate to repository with params, apply defaults (page=1, page_size=25)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 6.1, 6.2, 6.3, 7.1, 7.2, 7.3, 7.4, 7.5, 30.2, 34.1, 35.1, 35.2, 35.3, 35.4_

  - [ ]* 9.2 Write property test: Phone Uniqueness Within Tenant (Property 8)
    - **Property 8: Phone Uniqueness Within Tenant**
    - In `services/billing-api/internal/usecase/reseller_usecase_test.go`, use `rapid.Check` to verify that for any two resellers within the same tenant, their `phone` values are distinct; any creation or update that would result in a duplicate phone is rejected with `PHONE_DUPLICATE`
    - **Validates: Requirements 4.4, 7.3**

  - [ ]* 9.3 Write property test: Pagination Metadata Correctness (Property 12)
    - **Property 12: Pagination Metadata Correctness**
    - In `services/billing-api/internal/usecase/reseller_usecase_test.go`, use `rapid.Check` to verify that for any total count and page_size, `total_pages == ceil(total / page_size)`, page is within [1, max(1, total_pages)], and items on current page equals `min(page_size, total - (page-1)*page_size)`
    - **Validates: Requirements 5.5, 17.7, 25.5, 27.3, 28.4**

- [x] 10. Usecase layer — Reseller Actions (suspend, activate, deactivate, reset-password, deposit, withdraw)
  - [x] 10.1 Create usecase/reseller_action.go
    - Create `services/billing-api/internal/usecase/reseller_action.go` implementing `Suspend`, `Activate`, `Deactivate`, `ResetPassword`, `Deposit`, `Withdraw` methods
    - `Suspend`: fetch reseller → `ResellerTransition(current, suspended)` → update status → write audit log (`reseller.status_changed`) → publish `reseller.status_changed` event
    - `Activate`: fetch reseller → `ResellerTransition(current, aktif)` → update status → write audit log → publish event
    - `Deactivate`: fetch reseller → verify `confirmation_name` matches reseller name → `ResellerTransition(current, nonaktif)` → update status → void all `tersedia` vouchers owned by reseller → write voucher audit logs (`voucher.voided`, reason=reseller_deactivated) → write audit log (`reseller.status_changed`) → publish event → invalidate all reseller sessions
    - `ResetPassword`: fetch reseller → generate random 8-char alphanumeric password → hash with bcrypt → update password_hash → invalidate all sessions → write audit log (`reseller.password_reset`) → return plaintext password
    - `Deposit`: BEGIN TX → `GetForUpdate(id)` (row lock) → update balance (balance + amount) → create reseller_transaction (type=deposit, balance_before, balance_after) → write audit log (`reseller.deposit`, amount + notes) → COMMIT → return updated reseller
    - `Withdraw`: BEGIN TX → `GetForUpdate(id)` (row lock) → verify balance ≥ amount → update balance (balance - amount) → create reseller_transaction (type=withdraw, balance_before, balance_after) → write audit log (`reseller.withdraw`, amount + notes) → COMMIT → return updated reseller. Return `ErrInsufficientBalance` if balance < amount
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8, 9.1, 9.2, 9.3, 9.4, 9.5, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7, 34.1, 34.2, 35.2_

  - [ ]* 10.2 Write property test: Transaction Balance Consistency (Property 14)
    - **Property 14: Transaction Balance Consistency**
    - In `services/billing-api/internal/usecase/reseller_action_test.go`, use `rapid.Check` to verify that for any reseller transaction record, `balance_after == balance_before + amount` for deposits and refunds, and `balance_after == balance_before - amount` for purchases
    - **Validates: Requirements 11.5, 10.1, 24.6**

- [x] 11. Usecase layer — Reseller Auth
  - [x] 11.1 Create usecase/reseller_auth.go
    - Create `services/billing-api/internal/usecase/reseller_auth.go` implementing `Login`, `Logout`, `RefreshToken` methods
    - `Login`: check rate limiter (phone-based) → `GetByPhone(tenantID, phone)` → verify status is `aktif` (return `ErrResellerAccountDisabled` if suspended/nonaktif) → compare password with bcrypt → create session in `sessions` table (user_id = reseller UUID, 24h expiry) → generate JWT with claims (`reseller_id`, `tenant_id`, `name`, `role=reseller`) → update `last_login` → reset rate limiter → return tokens + reseller
    - `Logout`: delete session by token hash
    - `RefreshToken`: get session by refresh token hash → verify not expired → generate new token pair → update session
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 13.1, 13.2, 13.3, 13.4, 14.1, 14.2, 14.3, 14.4, 14.5_

- [x] 12. Checkpoint — Reseller usecases complete
  - Ensure all reseller usecase files compile (`go build ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

- [x] 13. Usecase layer — Voucher Generate & List
  - [x] 13.1 Create usecase/voucher_usecase.go with VoucherUsecase
    - Create `services/billing-api/internal/usecase/voucher_usecase.go` implementing `Generate`, `List`, `GetByID` methods
    - `Generate`: validate package exists and `type=voucher` → if quantity ≤ 500: generate codes via `GenerateVoucherCodes` → `BulkCreate` vouchers with status `tersedia` → write voucher audit logs (`voucher.generated`) → publish `voucher.batch_generated` event → return result with vouchers. If quantity > 500: enqueue `voucher.async_generate` asynq job → return result with `job_id` and HTTP 202
    - `List`: delegate to repository with params, apply defaults (page=1, page_size=25)
    - `GetByID`: fetch voucher → fetch voucher audit logs → return `VoucherDetail`
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7, 15.8, 15.9, 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8, 34.3_

- [x] 14. Usecase layer — Voucher Actions (bulk void, bulk assign, export CSV)
  - [x] 14.1 Create usecase/voucher_action.go
    - Create `services/billing-api/internal/usecase/voucher_action.go` implementing `BulkVoid`, `BulkAssign`, `ExportCSV` methods
    - `BulkVoid`: fetch vouchers by IDs → for each, check status == `tersedia` → transition to `void` → write voucher audit log (`voucher.voided`) → return `BulkActionResult` with success/failure counts and failure details
    - `BulkAssign`: validate reseller exists → fetch vouchers by IDs → for each, check status == `tersedia` → assign to reseller (admin assignment, no balance deduction, no snapshot) → write voucher audit log (`voucher.assigned`) → return `BulkActionResult`
    - `ExportCSV`: fetch vouchers with filters → format as CSV bytes → return
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5, 19.6_

  - [ ]* 14.2 Write property test: Bulk Void Eligibility (Property 10)
    - **Property 10: Bulk Void Eligibility**
    - In `services/billing-api/internal/usecase/voucher_action_test.go`, use `rapid.Check` to verify that for any set of voucher IDs, only vouchers with status `tersedia` are transitioned to `void`; vouchers with other statuses fail with reason; `success_count + failure_count == len(ids)`
    - **Validates: Requirements 19.2, 19.5**

- [x] 15. Usecase layer — Voucher Purchase (reseller buy)
  - [x] 15.1 Create usecase/voucher_purchase.go
    - Create `services/billing-api/internal/usecase/voucher_purchase.go` implementing `Buy` method
    - `Buy`: BEGIN TX → `GetForUpdate(resellerID)` (row lock) → verify status == `aktif` → check daily purchase limit (`CountTodayPurchases` + quantity ≤ limit, 0 = unlimited) → fetch package (verify type=voucher, is_active=true) → calculate totalCost = quantity × reseller_price → verify balance ≥ totalCost → generate voucher codes → `BulkCreate` vouchers → for each voucher: `AssignToReseller` (set sell_price_snapshot, reseller_price_snapshot, purchased_at, expires_at = now + voucher_expiry_days) → write voucher audit log (`voucher.sold`, actor=reseller) → `UpdateBalance(balance - totalCost)` → create reseller_transaction (type=purchase) → COMMIT → publish `voucher.purchased` event → return `BuyVoucherResult`
    - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.5, 24.6, 24.7, 24.8, 24.9, 31.1, 33.1, 33.2, 33.3, 34.4_

  - [ ]* 15.2 Write property test: Balance Conservation (Property 4)
    - **Property 4: Balance Conservation**
    - In `services/billing-api/internal/usecase/voucher_purchase_test.go`, use `rapid.Check` to verify that for any reseller with initial balance B and any sequence of deposits, purchases, and refunds, the final balance equals B + Σ(deposits) + Σ(refunds) - Σ(purchases), and at no point does the balance go below zero
    - **Validates: Requirements 1.6, 10.1, 11.5, 21.2, 21.6, 24.1, 24.9, 33.3, 33.4**

  - [ ]* 15.3 Write property test: Price Snapshot Integrity (Property 7)
    - **Property 7: Price Snapshot Integrity**
    - In `services/billing-api/internal/usecase/voucher_purchase_test.go`, use `rapid.Check` to verify that for any purchased voucher, `sell_price_snapshot` and `reseller_price_snapshot` are non-null, `reseller_price_snapshot < sell_price_snapshot`, and subsequent package price changes do not modify the snapshot values
    - **Validates: Requirements 31.1, 31.2, 31.3, 31.4**

  - [ ]* 15.4 Write property test: Expiry Date Calculation (Property 13)
    - **Property 13: Expiry Date Calculation**
    - In `services/billing-api/internal/usecase/voucher_purchase_test.go`, use `rapid.Check` to verify that for any purchased voucher, `expires_at == purchased_at + voucher_expiry_days` (default 90), and changes to the tenant's `voucher_expiry_days` do not affect already-purchased vouchers
    - **Validates: Requirements 22.1, 22.2, 22.3**

- [x] 16. Usecase layer — Voucher Expiry (cron job logic)
  - [x] 16.1 Create usecase/voucher_expiry.go
    - Create `services/billing-api/internal/usecase/voucher_expiry.go` implementing `ProcessExpiredVouchers` method
    - `ProcessExpiredVouchers`: loop in batches (batchSize=100) → `GetExpiredVouchers(batchSize)` → for each expired voucher: BEGIN TX → `GetForUpdate(resellerID)` (row lock) → transition voucher to `expired` → refund `reseller_price_snapshot` to reseller balance → create reseller_transaction (type=refund, reference_id=voucher_id) → write voucher audit log (`voucher.expired`, actor=System) → COMMIT → continue until no more expired vouchers
    - _Requirements: 21.1, 21.2, 21.3, 21.4, 21.5, 21.6_

- [x] 17. Usecase layer — Voucher Print (PDF generation)
  - [x] 17.1 Create usecase/voucher_print.go
    - Create `services/billing-api/internal/usecase/voucher_print.go` implementing `GeneratePDF` method
    - `GeneratePDF`: fetch vouchers by IDs → for each voucher, resolve package info (name, bandwidth, duration) → generate PDF with grid layout (8-12 voucher cards per A4 page), each card showing: tenant name, voucher code, package name, bandwidth, duration, sell price (from `sell_price_snapshot` if purchased, else current package price), expiry date (if purchased), tenant contact info
    - Use `maroto` or `gofpdf` library — can start with a placeholder interface that returns a simple PDF
    - _Requirements: 19.1, 20.1, 20.2, 20.3, 20.4, 20.5, 26.1, 26.4_

- [x] 18. Checkpoint — Usecase layer complete
  - Ensure all usecase files compile (`go build ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

- [x] 19. HTTP handlers — Admin Reseller CRUD
  - [x] 19.1 Create handler/reseller_handler.go
    - Create `services/billing-api/internal/handler/reseller_handler.go` with `ResellerHandler` struct (depends on `ResellerUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewResellerHandler`, and methods: `List`, `Get`, `Create`, `Update`
    - Register custom validator `phone_id` for Indonesian phone format (`+62` or `08`, 10-15 digits)
    - `List`: parse query params → validate → call usecase → return paginated response
    - `Get`: parse ID + `include` query param → call usecase → return detail
    - `Create`: parse body → validate → call usecase → return 201
    - `Update`: parse ID + body → validate → call usecase → return 200
    - Map domain errors to HTTP responses using the error mapping table from design
    - Return validation errors as aggregated array (HTTP 400, `VALIDATION_ERROR`)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 6.1, 6.2, 6.3, 7.1, 7.2, 7.3, 7.4, 7.5, 35.1, 35.2, 35.3, 35.4, 35.9_

  - [ ]* 19.2 Write property test: Validation Error Aggregation (Property 11)
    - **Property 11: Validation Error Aggregation**
    - In `services/billing-api/internal/handler/reseller_handler_test.go`, use `rapid.Check` to verify that for any reseller creation request with multiple invalid fields, the response returns HTTP 400 with `VALIDATION_ERROR` code and an array of field-level errors covering ALL invalid fields in a single response
    - **Validates: Requirements 35.9**

- [x] 20. HTTP handlers — Admin Reseller Actions
  - [x] 20.1 Create handler/reseller_action.go
    - Create `services/billing-api/internal/handler/reseller_action.go` with `ResellerActionHandler` struct (depends on `ResellerActionUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewResellerActionHandler`, and methods: `Suspend`, `Activate`, `Deactivate`, `ResetPassword`, `Deposit`, `Withdraw`
    - `Suspend`: parse ID → call usecase → map errors → return 200
    - `Activate`: parse ID → call usecase → map errors → return 200
    - `Deactivate`: parse ID + body (`confirmation_name`) → validate → call usecase → map errors → return 200
    - `ResetPassword`: parse ID → call usecase → return 200 with plaintext password
    - `Deposit`: parse ID + body (`amount`, `notes`) → validate → call usecase → return 200
    - `Withdraw`: parse ID + body (`amount`, `notes`) → validate → call usecase → map errors (ErrInsufficientBalance) → return 200
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 9.1, 9.2, 9.4, 10.1, 10.2, 10.3, 11.1, 11.2, 11.3, 11.4, 11.5_

- [x] 21. HTTP handlers — Admin Voucher
  - [x] 21.1 Create handler/voucher_handler.go
    - Create `services/billing-api/internal/handler/voucher_handler.go` with `VoucherHandler` struct (depends on `VoucherUsecase`, `VoucherActionUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewVoucherHandler`, and methods: `Generate`, `List`, `Get`, `BulkVoid`, `BulkAssign`, `Export`
    - `Generate`: parse body → validate (package_id, quantity, code_format, code_length, prefix) → register custom validator `alphanum_hyphen` for prefix → call usecase → return 201 (sync) or 202 (async)
    - `List`: parse query params → validate → call usecase → return paginated response
    - `Get`: parse ID → call usecase → return detail with audit logs
    - `BulkVoid`: parse body (voucher_ids) → validate → call usecase → return 200 with `BulkActionResult`
    - `BulkAssign`: parse body (voucher_ids, reseller_id) → validate → call usecase → return 200 with `BulkActionResult`
    - `Export`: parse query params → call usecase → return CSV file with `Content-Disposition` header
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.9, 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8, 19.2, 19.3, 19.4, 19.5, 35.5, 35.6, 35.7, 35.9_

  - [x] 21.2 Create handler/voucher_print.go
    - Create `services/billing-api/internal/handler/voucher_print.go` with `VoucherPrintHandler` struct (depends on `VoucherPrintUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewVoucherPrintHandler`, and method: `BulkPrint`
    - `BulkPrint`: parse body (voucher_ids) → validate → extract tenant info from JWT → call usecase `GeneratePDF` → return PDF with `Content-Type: application/pdf`
    - _Requirements: 19.1, 20.1, 20.2, 20.3, 20.4, 20.5_

- [x] 22. HTTP handlers — Reseller Auth
  - [x] 22.1 Create handler/reseller_auth_handler.go
    - Create `services/billing-api/internal/handler/reseller_auth_handler.go` with `ResellerAuthHandler` struct (depends on `ResellerAuthUsecase`, `LoginRateLimiter`, `*validator.Validate`, `zerolog.Logger`), constructor `NewResellerAuthHandler`, and methods: `Login`, `Logout`, `Refresh`
    - `Login`: parse body (phone, password) → validate → call usecase → return 200 with tokens + reseller
    - `Logout`: extract token hash from context → call usecase → return 200
    - `Refresh`: parse body (refresh_token) → call usecase → return 200 with new tokens
    - Map domain errors: `ErrResellerInvalidCredentials` → 401, `ErrResellerAccountDisabled` → 403, `ErrResellerAccountLocked` → 429
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 13.1, 13.2, 13.3, 14.1, 14.2, 14.3_

- [x] 23. HTTP handlers — Reseller Dashboard
  - [x] 23.1 Create handler/reseller_dashboard.go
    - Create `services/billing-api/internal/handler/reseller_dashboard.go` with `ResellerDashboardHandler` struct (depends on `ResellerUsecase`, `VoucherPurchaseUsecase`, `VoucherUsecase`, `VoucherPrintUsecase`, `ResellerTransactionRepository`, `*validator.Validate`, `zerolog.Logger`), constructor `NewResellerDashboardHandler`, and methods: `Summary`, `Buy`, `MyVouchers`, `Print`, `DepositHistory`, `TransactionHistory`
    - `Summary`: extract reseller_id from JWT → fetch balance, sold_today count, available_vouchers count → return `DashboardSummary`
    - `Buy`: parse body (package_id, quantity) → validate → call `VoucherPurchaseUsecase.Buy` → return 200 with `BuyVoucherResult`
    - `MyVouchers`: parse query params → call `VoucherUsecase.ListByReseller` → return paginated response
    - `Print`: parse body (voucher_ids) → verify all vouchers belong to authenticated reseller → call `VoucherPrintUsecase.GeneratePDF` → return PDF
    - `DepositHistory`: parse query params → call `ResellerTransactionRepository.ListDepositsByReseller` → return paginated response
    - `TransactionHistory`: parse query params → call `ResellerTransactionRepository.ListByReseller` → return paginated response
    - _Requirements: 23.1, 23.2, 23.3, 24.1, 24.2, 24.3, 24.4, 24.5, 25.1, 25.2, 25.3, 25.4, 25.5, 25.6, 26.1, 26.2, 26.3, 26.4, 27.1, 27.2, 27.3, 28.1, 28.2, 28.3, 28.4_

- [x] 24. Checkpoint — Handler layer complete
  - Ensure all handler files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 25. Reseller Auth Middleware
  - [x] 25.1 Create middleware/reseller_auth.go
    - Create `services/billing-api/internal/middleware/reseller_auth.go` with `ResellerAuth` middleware function that validates reseller JWT tokens, extracts `reseller_id`, `tenant_id`, `name`, `role` from claims, sets them in Fiber locals, and rejects admin tokens (role != `reseller`). Also create `resellerLoginRateLimiterMiddleware` for phone-based rate limiting on the reseller login endpoint
    - _Requirements: 12.2, 13.1, 13.2, 13.3, 13.4, 14.4, 29.5, 29.6_

- [x] 26. Router wiring and RBAC configuration
  - [x] 26.1 Update handler/router.go with reseller and voucher routes
    - Modify `services/billing-api/internal/handler/router.go`: add `ResellerHandler`, `ResellerActionHandler`, `VoucherHandler`, `VoucherPrintHandler`, `ResellerAuthHandler`, `ResellerDashboardHandler`, `ResellerRateLimiter` to `RouterConfig` struct
    - Register reseller auth routes (public, rate limited): `POST /v1/reseller/auth/login`, `POST /v1/reseller/auth/refresh`
    - Register reseller auth protected routes (reseller JWT): `POST /v1/reseller/auth/logout`
    - Register reseller dashboard routes (reseller JWT + tenant context): `GET /v1/reseller/dashboard`, `POST /v1/reseller/vouchers/buy`, `GET /v1/reseller/vouchers`, `POST /v1/reseller/vouchers/print`, `GET /v1/reseller/deposit`, `GET /v1/reseller/history`
    - Register admin reseller routes with RBAC: read (admin + operator GET-only) for `GET /v1/resellers` and `GET /v1/resellers/:id`; admin-only (tenant_admin) for `POST /v1/resellers`, `PUT /v1/resellers/:id`, `POST /v1/resellers/:id/suspend`, `POST /v1/resellers/:id/activate`, `POST /v1/resellers/:id/deactivate`, `POST /v1/resellers/:id/reset-password`, `POST /v1/resellers/:id/deposit`, `POST /v1/resellers/:id/withdraw`
    - Register admin voucher routes with RBAC: read (admin + operator GET-only) for `GET /v1/vouchers` and `GET /v1/vouchers/:id`; admin-only (tenant_admin) for `POST /v1/vouchers/generate`, `POST /v1/vouchers/bulk/print`, `POST /v1/vouchers/bulk/void`, `POST /v1/vouchers/bulk/assign`, `GET /v1/vouchers/export`
    - _Requirements: 29.1, 29.2, 29.3, 29.4, 29.5, 29.6_

  - [x] 26.2 Update cmd/main.go to wire all new dependencies
    - Modify `services/billing-api/cmd/main.go`: instantiate all new repositories (`ResellerRepo`, `VoucherRepo`, `VoucherAuditLogRepo`, `ResellerTxRepo`), all new usecases (`ResellerUsecase`, `ResellerActionUsecase`, `ResellerAuthUsecase`, `VoucherUsecase`, `VoucherActionUsecase`, `VoucherPurchaseUsecase`, `VoucherExpiryUsecase`, `VoucherPrintUsecase`), all new handlers (`ResellerHandler`, `ResellerActionHandler`, `VoucherHandler`, `VoucherPrintHandler`, `ResellerAuthHandler`, `ResellerDashboardHandler`), create a second `LoginRateLimiter` for reseller login, and pass all to `RouterConfig`
    - Follow the same dependency injection pattern as existing customer/package wiring
    - _Requirements: 29.1, 30.1_

- [x] 27. Checkpoint — Full module compiles and routes registered
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 28. Worker — Async voucher generate + expiry cron
  - [x] 28.1 Create worker/voucher_worker.go
    - Create `services/billing-api/internal/worker/voucher_worker.go` with `VoucherWorker` struct that registers two asynq handlers: (1) `voucher.async_generate` task handler — deserializes `GenerateVoucherRequest` from payload, calls `VoucherUsecase.Generate` with the request, (2) `voucher.expiry_cron` periodic task — calls `VoucherExpiryUsecase.ProcessExpiredVouchers`
    - Register the worker in `cmd/main.go` — start asynq server in a goroutine alongside the HTTP server, register the periodic expiry cron (daily at midnight)
    - _Requirements: 15.5, 21.1, 21.5_

- [x] 29. Checkpoint — Worker compiles and integrates
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ensure all tests pass (`go test ./...`). Ask the user if questions arise.

- [ ] 30. Write unit tests for handlers and usecases
  - [ ]* 30.1 Write unit tests for ResellerHandler
    - In `services/billing-api/internal/handler/reseller_handler_test.go`, test HTTP status codes, request parsing, response format for all CRUD endpoints, including error cases (404 RESELLER_NOT_FOUND, 409 PHONE_DUPLICATE, 400 VALIDATION_ERROR)
    - _Requirements: 4.2, 4.4, 6.3, 7.3, 35.9_

  - [ ]* 30.2 Write unit tests for ResellerActionHandler
    - In `services/billing-api/internal/handler/reseller_action_test.go`, test HTTP status codes for suspend, activate, deactivate, reset-password, deposit endpoints, including error cases (422 INVALID_STATUS_TRANSITION, 400 CONFIRMATION_MISMATCH, 404 RESELLER_NOT_FOUND)
    - _Requirements: 8.4, 8.5, 9.4, 10.4_

  - [ ]* 30.3 Write unit tests for VoucherHandler
    - In `services/billing-api/internal/handler/voucher_handler_test.go`, test HTTP status codes for generate (201 sync, 202 async), list, bulk void, bulk assign, export endpoints, including error cases (400 INVALID_PACKAGE_TYPE, 400 VALIDATION_ERROR)
    - _Requirements: 15.4, 15.5, 15.9, 19.5, 35.5, 35.6, 35.7_

  - [ ]* 30.4 Write unit tests for ResellerAuthHandler
    - In `services/billing-api/internal/handler/reseller_auth_handler_test.go`, test login success, invalid credentials (401), account disabled (403), account locked (429), logout, refresh
    - _Requirements: 12.3, 12.4, 12.5, 13.2_

  - [ ]* 30.5 Write unit tests for ResellerDashboardHandler
    - In `services/billing-api/internal/handler/reseller_dashboard_test.go`, test dashboard summary, buy voucher (success, insufficient balance, daily limit exceeded), my vouchers, print (forbidden if not owner), deposit history, transaction history
    - _Requirements: 23.1, 23.3, 24.3, 24.4, 24.5, 26.2, 26.3_

  - [ ]* 30.6 Write unit tests for ResellerUsecase
    - In `services/billing-api/internal/usecase/reseller_usecase_test.go`, test business logic: phone duplicate check, create with defaults, update with phone conflict, list with pagination
    - _Requirements: 4.4, 4.5, 4.6, 7.3, 7.5_

  - [ ]* 30.7 Write unit tests for ResellerActionUsecase
    - In `services/billing-api/internal/usecase/reseller_action_test.go`, test status transitions (valid + invalid), confirmation mismatch for deactivate, password reset session invalidation, deposit with transaction logging
    - _Requirements: 8.3, 8.4, 8.5, 8.8, 9.1, 9.3, 10.1, 10.5_

  - [ ]* 30.8 Write unit tests for ResellerAuthUsecase
    - In `services/billing-api/internal/usecase/reseller_auth_test.go`, test login success, wrong password, disabled account, rate limiting, session creation, logout, refresh
    - _Requirements: 12.1, 12.3, 12.4, 12.5, 12.6, 13.1, 13.2, 13.3_

  - [ ]* 30.9 Write unit tests for VoucherUsecase
    - In `services/billing-api/internal/usecase/voucher_usecase_test.go`, test generate sync/async threshold (≤500 vs >500), invalid package type, code generation with collision handling
    - _Requirements: 15.4, 15.5, 15.6, 15.7, 15.9_

  - [ ]* 30.10 Write unit tests for VoucherPurchaseUsecase
    - In `services/billing-api/internal/usecase/voucher_purchase_test.go`, test buy flow: insufficient balance, daily limit exceeded, disabled account, atomic rollback, price snapshot correctness
    - _Requirements: 24.3, 24.4, 24.5, 24.8, 31.1_

  - [ ]* 30.11 Write unit tests for VoucherExpiryUsecase
    - In `services/billing-api/internal/usecase/voucher_expiry_test.go`, test expiry batch processing, refund calculation, transaction logging, batch loop termination
    - _Requirements: 21.1, 21.2, 21.3, 21.4, 21.5, 21.6_

- [x] 31. Final checkpoint — All tests pass
  - Ensure all tests pass (`go test ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after each layer
- Property tests validate universal correctness properties from the design document (14 properties total)
- The `resellers` table does NOT exist yet — migration 000012 creates it
- The `vouchers` table depends on `packages` (FK) and `resellers` (FK) — migration order matters
- The `voucher_audit_logs` table is SEPARATE from the shared `audit_logs` table — different schema, append-only
- Reseller data changes use the shared `audit_logs` table with `entity_type='reseller'`
- Reseller auth is SEPARATE from admin auth — different JWT claims, different middleware, different login endpoint
- Balance operations MUST be atomic: `SELECT ... FOR UPDATE` + DB transaction + row lock
- Voucher generate has two paths: sync (≤500) returns vouchers, async (>500) returns job_id
- Voucher expiry is a daily cron job via asynq scheduler, processes in batches of 100
- PDF generation can start as a placeholder interface — full implementation with `maroto`/`gofpdf` later
- All code comments MUST be in Indonesian; variable/function names in English
- Max 200 lines per file — split handlers and usecases into multiple files as shown in the file structure
- The existing `ErrConfirmationMismatch` from `domain/customer.go` is reused (not redefined)
- The existing `AuditLogRepository` is reused for reseller data audit — no new audit infrastructure needed
- The existing `SessionRepository` is reused for reseller sessions — `user_id` = reseller UUID
- The existing `LoginRateLimiter` is adapted for phone-based identification for reseller login
- `domain/repository.go` is appended (not replaced) — existing auth/customer/area/package interfaces remain
- Dynamic `List` queries in repositories follow the same pattern as `customer_repo.go` and `package_repo.go`
