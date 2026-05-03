# Implementation Plan: Package CRUD Module

## Overview

Bottom-up implementation of the Package CRUD module for ISPBoss billing-api. Starts with database migrations, then domain entities, sqlc queries, repositories, usecases, handlers, and finally router wiring. Each task builds on the previous and is independently testable. All code is Go, using existing patterns from the customer-crud module (Fiber, sqlc, pgx, asynq, go-playground/validator, rapid). Both PPPoE and Voucher package types are stored in a single `packages` table with a `type` discriminator column.

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Create migration 000010: create packages table
    - Create `services/billing-api/migrations/000010_create_packages.up.sql` — `packages` table with 29 columns, CHECK constraints (`type` IN pppoe/voucher, `quota_type` IN unlimited/monthly_quota/fup/quota, `download_mbps` > 0, `upload_mbps` > 0), RLS policies (tenant_isolation + tenant_insert), unique constraint `(tenant_id, name)`, composite indexes on `(tenant_id, type)`, `(tenant_id, is_active)`, `(tenant_id, type, is_active)`
    - Create `services/billing-api/migrations/000010_create_packages.down.sql` — drop policies, constraints, indexes, and table
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7_

  - [x] 1.2 Create migration 000011: add FK from customers.package_id to packages.id
    - Create `services/billing-api/migrations/000011_add_customers_package_fk.up.sql` — add foreign key constraint `fk_customers_package_id` from `customers.package_id` to `packages.id`
    - Create `services/billing-api/migrations/000011_add_customers_package_fk.down.sql` — drop the foreign key constraint
    - _Requirements: 1.8_

- [x] 2. Domain entities and validation helpers
  - [x] 2.1 Create domain/package.go with Package entity
    - Create `services/billing-api/internal/domain/package.go` with the full `Package` struct (29 fields + computed `CustomerCount`), type constants (`PackageType`, `BandwidthType`, `QuotaType`, `QuotaAction`, `DurationUnit`), domain error variables (`ErrPackageNotFound`, `ErrPackageNameDuplicate`, `ErrPackageHasCustomers`, `ErrPackageAlreadyActive`, `ErrPackageAlreadyInactive`, `ErrInsufficientMargin`, `ErrTypeChangeNotAllowed`, `ErrBurstFieldsIncomplete`), and helper functions: `ValidateResellerMargin(sellPrice, resellerPrice int64) error`, `GenerateProfileName(name string) string`, `GenerateDuplicateName(originalName string, existingNames []string) string`, `ValidateBurstFields(burstDown, burstUp, burstThreshold, burstTime *int) error`
    - Reuse `ErrConfirmationMismatch` from `domain/customer.go` (do not redefine)
    - _Requirements: 1.1, 2.8, 3.7, 9.1, 9.2, 14.10, 15.1_

  - [x] 2.2 Write property test: Reseller Margin Integrity (Property 2)
    - **Property 2: Reseller Margin Integrity**
    - In `services/billing-api/internal/domain/package_test.go`, use `rapid.Check` to verify that `ValidateResellerMargin` returns nil iff `resellerPrice < sellPrice` and `sellPrice - resellerPrice >= 500`; returns `ErrInsufficientMargin` for all other combinations
    - **Validates: Requirements 3.4, 14.8, 15.1, 15.3**

  - [x] 2.3 Write property test: Profile Name Auto-Generation (Property 5)
    - **Property 5: Profile Name Auto-Generation**
    - In `services/billing-api/internal/domain/package_test.go`, use `rapid.Check` to verify that for any package name, `GenerateProfileName` produces a lowercase string with spaces replaced by hyphens, and the result contains no uppercase letters or spaces
    - **Validates: Requirements 2.8, 3.7**

  - [x] 2.4 Write property test: Burst Fields All-or-Nothing (Property 6)
    - **Property 6: Burst Fields All-or-Nothing**
    - In `services/billing-api/internal/domain/package_test.go`, use `rapid.Check` to verify that `ValidateBurstFields` returns nil when all four burst fields are provided or all four are nil; returns `ErrBurstFieldsIncomplete` for any partial combination (1, 2, or 3 fields provided)
    - **Validates: Requirements 2.5, 14.10**

  - [x] 2.5 Write property test: Duplicate Name Generation (Property 11)
    - **Property 11: Duplicate Name Generation**
    - In `services/billing-api/internal/domain/package_test.go`, use `rapid.Check` to verify that `GenerateDuplicateName` produces `"{name} (Copy)"` when no collision exists, and `"{name} (Copy N)"` when previous copies exist, and the result is never in the `existingNames` list
    - **Validates: Requirements 9.1, 9.2**

  - [x] 2.6 Create domain/package_event.go with event payload types
    - Create `services/billing-api/internal/domain/package_event.go` with `PackagePriceChangedPayload` struct containing `PackageID`, `PackageName`, `PackageType`, `OldPrice`, `NewPrice`
    - _Requirements: 4.6, 13.1, 13.2_

  - [x] 2.7 Append PackageRepository interface and DTOs to domain/repository.go
    - Append `PackageRepository` interface (10 methods: `Create`, `GetByID`, `Update`, `Delete`, `List`, `UpdateIsActive`, `NameExists`, `CustomerCount`, `ListNamesByPrefix`) to `services/billing-api/internal/domain/repository.go`
    - Add DTOs: `CreatePackageRequest`, `UpdatePackageRequest`, `DeletePackageRequest`, `PackageListParams`, `PackageListResult`, `PackageDetail`
    - _Requirements: 2.1, 3.1, 4.1, 5.1, 6.1, 7.1, 8.1, 9.1_

- [x] 3. Checkpoint — Domain layer complete
  - Ensure all domain files compile (`go build ./...` in `services/billing-api`). Ensure property tests pass. Ask the user if questions arise.

- [x] 4. sqlc queries
  - [x] 4.1 Create queries/packages.sql with package queries
    - Create `services/billing-api/queries/packages.sql` with sqlc queries for: `CreatePackage` (:one, INSERT RETURNING *), `GetPackageByID` (:one, SELECT with customer_count subquery), `UpdatePackage` (:one, UPDATE RETURNING *), `DeletePackage` (:exec), `UpdatePackageIsActive` (:one, UPDATE is_active RETURNING *), `PackageNameExists` (:one, SELECT EXISTS), `PackageCustomerCount` (:one, COUNT from customers WHERE deleted_at IS NULL), `ListPackageNamesByPrefix` (:many, SELECT name WHERE name LIKE prefix)
    - Note: `List` query is built dynamically in repository (same pattern as customer) — not in sqlc
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1, 5.8, 6.1, 7.1, 8.1, 9.1, 17.1, 17.2_

  - [x] 4.2 Run sqlc generate to produce Go code
    - Run `sqlc generate` in `services/billing-api/` to regenerate `internal/repository/` files (adds `packages.sql.go`, updates `models.go`)
    - Verify generated code compiles
    - _Requirements: 1.1_

- [x] 5. Checkpoint — sqlc layer complete
  - Ensure all generated files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 6. Repository implementation
  - [x] 6.1 Create repository/package_repo.go
    - Create `services/billing-api/internal/repository/package_repo.go` implementing `domain.PackageRepository` — wraps sqlc-generated queries, handles `List` with dynamic filtering/search/sorting/pagination (build query manually since sqlc doesn't support dynamic WHERE), implements `CustomerCount` via sqlc query, implements `ListNamesByPrefix` for duplicate name generation
    - All queries must include `tenant_id` filter at application level (RLS is safety net)
    - Dynamic list query supports: filter by `type`, `is_active`, `search` (ILIKE on name), sorting by `name`/`monthly_price`/`sell_price`/`download_mbps`/`created_at`, pagination with `customer_count` subquery
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 6.1, 7.1, 8.1, 9.1, 11.1, 11.2, 17.1, 17.3_

- [x] 7. Checkpoint — Data layer complete
  - Ensure all repository files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [ ] 8. Usecase layer — core CRUD
  - [x] 8.1 Create usecase/package_usecase.go with PackageUsecase
    - Create `services/billing-api/internal/usecase/package_usecase.go` implementing `Create`, `GetByID`, `Update`, `Delete`, `List` methods
    - `Create`: type-conditional validation (margin for voucher, burst all-or-nothing, quota conditional) → check name duplicate → auto-generate profile name if not provided → create package with `is_active=true` → write audit log (`package.created`)
    - `GetByID`: fetch package → optionally fetch audit logs → return `PackageDetail`
    - `Update`: fetch existing → reject type change → type-conditional validation on merged fields → check name duplicate → update → compute changed fields → write audit log (`package.updated`) with old/new values → publish `package.price_changed` event if price changed
    - `Delete`: fetch package → verify `confirmation_name` matches package name → check `CustomerCount` → if > 0 return `ErrPackageHasCustomers` → hard delete → write audit log (`package.deleted`)
    - `List`: delegate to repository with params, apply defaults (page=1, page_size=25)
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 5.1, 5.2, 5.7, 6.1, 6.2, 6.3, 8.1, 8.2, 8.3, 8.4, 8.5, 12.1, 12.2, 12.3, 13.1, 13.2, 13.3, 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 14.8, 14.9, 14.10, 14.11, 14.12, 15.1, 15.2, 15.3, 16.1, 16.2, 16.3, 17.1, 17.2, 17.3_

  - [ ] 8.2 Write property test: New Package Default Active Status (Property 1)
    - **Property 1: New Package Default Active Status**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that for any valid creation request (PPPoE or Voucher), the resulting package always has `is_active == true`, regardless of any `is_active` value in the request
    - **Validates: Requirements 2.1, 3.1**

  - [ ] 8.3 Write property test: Type-Field Consistency (Property 3)
    - **Property 3: Type-Field Consistency**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that for any PPPoE package, `monthly_price` and `bandwidth_type` are non-null and voucher-specific fields are null; for any Voucher package, `sell_price`, `reseller_price`, `duration_value`, `duration_unit` are non-null and PPPoE-specific fields are null
    - **Validates: Requirements 16.1, 16.2, 16.4**

  - [ ] 8.4 Write property test: Type Immutability After Creation (Property 4)
    - **Property 4: Type Immutability After Creation**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that for any existing package with type T, an update request with a different type value is rejected with `ErrTypeChangeNotAllowed`, and the package type remains T
    - **Validates: Requirements 16.3**

  - [ ] 8.5 Write property test: Pagination Metadata Correctness (Property 12)
    - **Property 12: Pagination Metadata Correctness**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that for any total count and page_size, `total_pages == ceil(total / page_size)`, page is within [1, max(1, total_pages)], and items on current page equals `min(page_size, total - (page-1)*page_size)`
    - **Validates: Requirements 5.7**

- [ ] 9. Usecase layer — actions (activate, deactivate, duplicate)
  - [x] 9.1 Create usecase/package_action.go
    - Create `services/billing-api/internal/usecase/package_action.go` implementing `Activate`, `Deactivate`, `Duplicate` methods on `PackageUsecase`
    - `Activate`: fetch package → if already active return `ErrPackageAlreadyActive` → update `is_active=true` → write audit log (`package.activated`)
    - `Deactivate`: fetch package → if already inactive return `ErrPackageAlreadyInactive` → update `is_active=false` → write audit log (`package.deactivated`)
    - `Duplicate`: fetch source package → list names by prefix for collision check → `GenerateDuplicateName` → create new package with copied fields, new UUID, generated name, `is_active=true`, fresh timestamps → write audit log (`package.duplicated`, metadata: source_id)
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 9.1, 9.2, 9.3, 9.4, 12.1, 12.3_

  - [ ] 9.2 Write property test: Audit Trail Completeness (Property 8)
    - **Property 8: Audit Trail Completeness**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that for any package mutation (create, update, delete, activate, deactivate, duplicate), exactly one audit log is inserted with correct `entity_type` = `"package"`, `entity_id`, `action`, `actor_id`, `actor_name`, and for updates the `changes` column contains old/new values of changed fields
    - **Validates: Requirements 2.7, 3.6, 4.5, 7.5, 8.5, 9.4, 12.1, 12.2, 12.3**

  - [ ] 9.3 Write property test: Event Publishing on Price Change (Property 9)
    - **Property 9: Event Publishing on Price Change**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that a `package.price_changed` event is published if and only if `monthly_price` (PPPoE) or `sell_price` (Voucher) changes; the event contains `tenant_id`, `timestamp`, `correlation_id` (UUID v4), `package_id`, `package_name`, `package_type`, `old_price`, `new_price`; no event is published when price does not change
    - **Validates: Requirements 4.6, 13.1, 13.2, 13.3**

  - [ ] 9.4 Write property test: Delete Confirmation Matching (Property 10)
    - **Property 10: Delete Confirmation Matching**
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, use `rapid.Check` to verify that delete succeeds iff `confirmation_name` matches the package name exactly (case-sensitive); mismatched names return `ErrConfirmationMismatch` and the package remains unchanged
    - **Validates: Requirements 8.1, 8.2**

- [x] 10. Checkpoint — Usecase layer complete
  - Ensure all usecase files compile (`go build ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

- [ ] 11. HTTP handlers — package CRUD
  - [x] 11.1 Create handler/package_handler.go
    - Create `services/billing-api/internal/handler/package_handler.go` with `PackageHandler` struct (depends on `PackageUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewPackageHandler`, and methods: `List`, `Get`, `Create`, `Update`, `Delete`
    - Register struct-level custom validators (`validatePackageCreate`, `validatePackageUpdate`) for type-conditional validation (PPPoE requires monthly_price + bandwidth_type; Voucher requires sell_price + reseller_price + duration_value + duration_unit; quota conditional; burst all-or-nothing; margin check)
    - `List`: parse query params → validate → call usecase → return paginated response
    - `Get`: parse ID + `include` query param → call usecase → return detail
    - `Create`: parse body → validate (struct + type-conditional) → call usecase → return 201
    - `Update`: parse ID + body → validate (struct + type-conditional) → call usecase → return 200
    - `Delete`: parse ID + body (`confirmation_name`) → validate → call usecase → return 200
    - Map domain errors to HTTP responses using the error mapping table from design
    - Return validation errors as aggregated array (HTTP 400, `VALIDATION_ERROR`)
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 6.1, 6.2, 6.3, 8.1, 8.2, 8.3, 8.4, 14.13_

  - [ ] 11.2 Write property test: Validation Error Aggregation (Property 7)
    - **Property 7: Validation Error Aggregation**
    - In `services/billing-api/internal/handler/package_handler_test.go`, use `rapid.Check` to verify that for any request body with multiple invalid fields, the response returns HTTP 400 with `VALIDATION_ERROR` code and an array of field-level errors covering ALL invalid fields in a single response
    - **Validates: Requirements 14.13**

- [x] 12. HTTP handlers — package actions
  - [x] 12.1 Create handler/package_action.go
    - Create `services/billing-api/internal/handler/package_action.go` with methods: `Activate`, `Deactivate`, `Duplicate` on `PackageHandler`
    - `Activate`: parse ID → call usecase → map errors → return 200
    - `Deactivate`: parse ID → call usecase → map errors → return 200
    - `Duplicate`: parse ID → call usecase → map errors → return 201
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 9.1, 9.3_

- [x] 13. Router wiring and RBAC configuration
  - [x] 13.1 Update handler/router.go with package routes
    - Modify `services/billing-api/internal/handler/router.go`: add `PackageHandler` to `RouterConfig` struct, register 8 package endpoints under the `api` group (which already has auth + tenant middleware)
    - Configure two RBAC groups for packages: read (admin + operator + kasir GET-only) for `GET /packages` and `GET /packages/:id`; admin-only (tenant_admin) for `POST /packages`, `PUT /packages/:id`, `DELETE /packages/:id`, `POST /packages/:id/activate`, `POST /packages/:id/deactivate`, `POST /packages/:id/duplicate`
    - _Requirements: 10.1, 10.2, 10.3, 10.4_

  - [x] 13.2 Update cmd/main.go to wire package dependencies
    - Modify `services/billing-api/cmd/main.go`: instantiate `PackageRepo` (using sqlc queries + dbPool), `PackageUsecase` (using packageRepo, auditLogRepo, queueClient, logger), `PackageHandler` (using packageUsecase, logger), and pass `PackageHandler` to `RouterConfig`
    - Follow the same dependency injection pattern as existing customer/area wiring
    - _Requirements: 11.1, 11.2, 11.4_

- [x] 14. Checkpoint — Full module compiles and routes registered
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ensure all tests pass (`go test ./...`). Ask the user if questions arise.

- [ ] 15. Write unit tests for handlers and usecases
  - [ ] 15.1 Write unit tests for PackageHandler
    - In `services/billing-api/internal/handler/package_handler_test.go`, test HTTP status codes, request parsing, response format for all CRUD + action endpoints, including error cases (404 PACKAGE_NOT_FOUND, 409 PACKAGE_NAME_DUPLICATE, 409 PACKAGE_HAS_CUSTOMERS, 400 CONFIRMATION_MISMATCH, 400 INSUFFICIENT_MARGIN, 400 TYPE_CHANGE_NOT_ALLOWED, 400 BURST_FIELDS_INCOMPLETE, 400 PACKAGE_ALREADY_ACTIVE, 400 PACKAGE_ALREADY_INACTIVE)
    - _Requirements: 2.6, 3.5, 4.3, 4.4, 6.3, 7.3, 7.4, 8.2, 8.3, 8.4, 14.13_

  - [ ] 15.2 Write unit tests for PackageUsecase
    - In `services/billing-api/internal/usecase/package_usecase_test.go`, test business logic: name duplicate check, type change rejection, customer count check for delete, confirmation mismatch, margin validation, burst validation, profile name auto-generation, price change event publishing
    - _Requirements: 2.6, 2.7, 2.8, 3.5, 3.6, 3.7, 4.3, 4.4, 4.5, 4.6, 8.2, 8.3, 8.5, 15.2, 16.3_

  - [ ] 15.3 Write unit tests for PackageAction (activate, deactivate, duplicate)
    - In `services/billing-api/internal/usecase/package_action_test.go`, test activate already-active error, deactivate already-inactive error, duplicate with name collisions, duplicate field copying
    - _Requirements: 7.3, 7.4, 7.5, 9.1, 9.2, 9.3, 9.4_

- [x] 16. Final checkpoint — All tests pass
  - Ensure all tests pass (`go test ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after each layer
- Property tests validate universal correctness properties from the design document (12 properties total)
- The `packages` table does NOT exist yet — migration 000010 creates it
- The `customers.package_id` FK is added in migration 000011 AFTER the packages table exists
- Both PPPoE and Voucher packages go in the same table with `type` discriminator
- No soft-delete — use `is_active` boolean + hard delete (only when 0 customers)
- `customer_count` is computed at query time via subquery, not stored
- Type-conditional validation uses struct-level validators registered on `go-playground/validator`
- The existing `ErrConfirmationMismatch` from `domain/customer.go` is reused (not redefined)
- The existing `AuditLogRepository` is reused — no new audit infrastructure needed
- `domain/repository.go` is appended (not replaced) — existing auth/customer/area interfaces remain
- The dynamic `List` query in repository follows the same pattern as `customer_repo.go`
- All code comments MUST be in Indonesian; variable/function names in English
- Max 200 lines per file — split into `package_handler.go` + `package_action.go` and `package_usecase.go` + `package_action.go`
