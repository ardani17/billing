# Implementation Plan: Customer CRUD Module

## Overview

Bottom-up implementation of the Customer CRUD module for ISPBoss billing-api. Starts with database migrations, then domain entities, sqlc queries, repositories, usecases, handlers, and finally router wiring. Each task builds on the previous and is independently testable. All code is Go, using existing patterns from the auth-rbac spec (Fiber, sqlc, pgx, asynq, go-playground/validator, rapid).

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Create migration 000006: drop old customers table
    - Create `services/billing-api/migrations/000006_drop_old_customers.up.sql` — drop RLS policies, indexes, and the old sample `customers` table from migration 000002
    - Create `services/billing-api/migrations/000006_drop_old_customers.down.sql` — recreate the old sample customers table with its original schema, RLS policies, and indexes
    - _Requirements: 1.1_

  - [x] 1.2 Create migration 000007: create areas table
    - Create `services/billing-api/migrations/000007_create_areas.up.sql` — `areas` table with columns (`id`, `tenant_id`, `name`, `description`, `odp_id`, `center_lat`, `center_lng`, `created_at`, `updated_at`), RLS policies (tenant_isolation + tenant_insert), unique constraint `(tenant_id, name)`, index on `tenant_id`
    - Create `services/billing-api/migrations/000007_create_areas.down.sql` — drop policies, constraints, indexes, and table
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 1.3 Create migration 000008: create new customers table
    - Create `services/billing-api/migrations/000008_create_customers.up.sql` — full `customers` table with 24 columns, 3 CHECK constraints (`due_date` 1-28, `connection_method` enum, `status` enum), RLS policies, unique constraints `(tenant_id, phone)` and `(tenant_id, customer_id_seq)`, 7 composite indexes including partial index for active customers
    - Create `services/billing-api/migrations/000008_create_customers.down.sql` — drop policies, constraints, indexes, and table
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8_

  - [x] 1.4 Create migration 000009: create audit_logs table
    - Create `services/billing-api/migrations/000009_create_audit_logs.up.sql` — `audit_logs` table with columns (`id`, `tenant_id`, `entity_type`, `entity_id`, `action`, `actor_id`, `actor_name`, `changes` JSONB, `metadata` JSONB, `created_at`), RLS policies, composite indexes on `(tenant_id, entity_type, entity_id)` and `(tenant_id, created_at)`
    - Create `services/billing-api/migrations/000009_create_audit_logs.down.sql` — drop policies, indexes, and table
    - _Requirements: 3.1, 3.2, 3.3_

- [x] 2. Domain entities and state machine
  - [x] 2.1 Replace domain/customer.go with expanded Customer entity
    - Replace `services/billing-api/internal/domain/customer.go` with the full Customer struct (24 fields), `CustomerStatus` type with constants (`pending`, `aktif`, `isolir`, `suspend`, `berhenti`), `ConnectionMethod` type with constants (`pppoe`, `hotspot`, `dhcp_binding`, `static`), `ValidTransitions` map, `CanTransition()`, `Transition()`, `AllowedTargets()` functions, `GenerateCustomerID()` function, `GeneratePPPoEUsername()` and `GeneratePPPoEPassword()` functions
    - Include all domain error variables: `ErrCustomerNotFound`, `ErrPhoneDuplicate`, `ErrInvalidStatusTransition`, `ErrConfirmationMismatch`, `ErrSamePackage`, `ErrPackageNotFound`, `ErrCustomerDeleted`
    - _Requirements: 4.1, 4.2, 5.1, 5.2, 11.4, 23.1_

  - [x] 2.2 Write property test: Customer ID Generation Format (Property 1)
    - **Property 1: Customer ID Generation Format**
    - In `services/billing-api/internal/domain/customer_test.go`, use `rapid.Check` to verify that for any positive integer seq, `GenerateCustomerID(seq)` produces `PLG-{zero-padded-seq}` (min 3 digits), and parsing the numeric suffix back yields the original seq
    - **Validates: Requirements 4.1, 4.2**

  - [x] 2.3 Write property test: State Machine Determinism (Property 5)
    - **Property 5: State Machine Determinism and Completeness**
    - In `services/billing-api/internal/domain/customer_test.go`, use `rapid.Check` to verify that for any pair `(current, target)`, `CanTransition` returns true iff target is in `ValidTransitions[current]`, `Transition` returns target on valid transitions, and returns error with allowed targets on invalid transitions
    - **Validates: Requirements 11.3, 11.4, 23.1, 23.2, 23.3**

  - [x] 2.4 Create domain/customer_event.go with event payload types
    - Create `services/billing-api/internal/domain/customer_event.go` with structs: `CustomerCreatedPayload`, `CustomerActivatedPayload`, `CustomerIsolatedPayload`, `CustomerUnblockedPayload`, `CustomerTerminatedPayload`, `PackageChangedPayload` — each with the fields specified in the design
    - _Requirements: 21.1, 21.2, 21.3, 21.4, 21.5, 21.6_

  - [x] 2.5 Create domain/area.go with Area entity
    - Create `services/billing-api/internal/domain/area.go` with `Area` struct, `CreateAreaRequest`, `UpdateAreaRequest` DTOs, and error variables `ErrAreaNotFound`, `ErrAreaNameDuplicate`, `ErrAreaHasCustomers`
    - _Requirements: 2.1, 13.3, 13.4, 13.7_

  - [x] 2.6 Create domain/audit_log.go with AuditLog entity
    - Create `services/billing-api/internal/domain/audit_log.go` with `AuditLog` struct
    - _Requirements: 3.1, 20.1_

  - [x] 2.7 Append repository interfaces to domain/repository.go
    - Append `CustomerRepository` (14 methods), `AreaRepository` (7 methods), and `AuditLogRepository` (2 methods) interfaces to `services/billing-api/internal/domain/repository.go` as defined in the design
    - Add all request/response DTOs: `CustomerListParams`, `CustomerListResult`, `PaginationMeta`, `CustomerDetail`, `BulkActionResult`, `BulkFailure`, `BulkEditFields`, `BulkResult`, `CreateCustomerRequest`, `UpdateCustomerRequest`, `DeleteCustomerRequest`, `ChangePackageRequest`, `BulkIDsRequest`, `BulkNotifyRequest`, `BulkChangePackageRequest`, `BulkEditRequest`
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 7.1, 8.1, 9.1, 10.1, 14.7_

- [x] 3. Checkpoint — Domain layer complete
  - Ensure all domain files compile (`go build ./...` in `services/billing-api`). Ensure property tests pass. Ask the user if questions arise.

- [x] 4. sqlc queries
  - [x] 4.1 Replace queries/customers.sql with full customer queries
    - Replace `services/billing-api/queries/customers.sql` with sqlc queries for: `CreateCustomer`, `GetCustomerByID`, `UpdateCustomer`, `SoftDeleteCustomer`, `UpdateCustomerStatus`, `UpdateCustomerPackage`, `GetMaxCustomerSeq`, `PhoneExists`, `CountCustomersByStatus` — all queries must include `WHERE deleted_at IS NULL` where appropriate and use the full 24-column schema
    - _Requirements: 1.1, 6.7, 7.1, 7.4, 8.1, 9.1, 10.1, 11.1, 12.1, 17.1, 17.2_

  - [x] 4.2 Create queries/areas.sql with area queries
    - Create `services/billing-api/queries/areas.sql` with sqlc queries for: `CreateArea`, `GetAreaByID`, `UpdateArea`, `DeleteArea`, `ListAreas` (with customer count via LEFT JOIN), `AreaNameExists`, `AreaCustomerCount`
    - _Requirements: 2.1, 13.1, 13.2, 13.5, 13.6, 13.7_

  - [x] 4.3 Create queries/audit_logs.sql with audit log queries
    - Create `services/billing-api/queries/audit_logs.sql` with sqlc queries for: `CreateAuditLog`, `ListAuditLogsByEntity` (ordered by `created_at DESC`)
    - _Requirements: 3.1, 20.1, 20.3_

  - [x] 4.4 Run sqlc generate to produce Go code
    - Run `sqlc generate` in `services/billing-api/` to regenerate `internal/repository/` files (`customers.sql.go`, `areas.sql.go`, `audit_logs.sql.go`, `models.go`, `db.go`)
    - Verify generated code compiles
    - _Requirements: 1.1, 2.1, 3.1_

- [x] 5. Repository implementations
  - [x] 5.1 Create repository/customer_repo.go
    - Create `services/billing-api/internal/repository/customer_repo.go` implementing `domain.CustomerRepository` — wraps sqlc-generated queries, handles list with dynamic filtering/search/sorting/pagination (build query manually for List since sqlc doesn't support dynamic WHERE), implements `BulkUpdateStatus`, `BulkUpdateFields`, `BulkSoftDelete` by iterating individual operations
    - All queries must include `tenant_id` filter at application level (RLS is safety net)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 7.1, 7.4, 8.1, 9.1, 10.1, 14.1, 14.2, 14.4, 14.5, 14.6, 17.1, 17.2, 19.2_

  - [x] 5.2 Create repository/area_repo.go
    - Create `services/billing-api/internal/repository/area_repo.go` implementing `domain.AreaRepository` — wraps sqlc-generated queries, includes tenant_id filter
    - _Requirements: 13.1, 13.2, 13.5, 13.6, 13.7, 19.2_

  - [x] 5.3 Create repository/audit_log_repo.go
    - Create `services/billing-api/internal/repository/audit_log_repo.go` implementing `domain.AuditLogRepository` — wraps sqlc-generated queries
    - _Requirements: 20.1, 20.2, 20.3_

- [x] 6. Checkpoint — Data layer complete
  - Ensure all repository files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 7. Custom validators
  - [x] 7.1 Register custom validators (phone_id, mac_addr)
    - Create a `RegisterCustomValidators` function (in `services/billing-api/internal/handler/` or a shared validation file) that registers `phone_id` validator (starts with `+62`, followed by 9-13 digits) and `mac_addr` validator (six groups of two hex digits separated by colons, e.g., `AA:BB:CC:DD:EE:FF`) on a `*validator.Validate` instance
    - _Requirements: 22.1, 22.4_

  - [x] 7.2 Write property test: Field Validation Rules (Property 6)
    - **Property 6: Field Validation Rules**
    - In `services/billing-api/internal/handler/customer_handler_test.go` (or a dedicated validation test file), use `rapid.Check` to verify: phone accepted iff starts with `+62` + 9-13 digits; latitude in [-90,90]; longitude in [-180,180]; mac_address matches `XX:XX:XX:XX:XX:XX`; due_date in [1,28]; name length in [3,255]; address non-empty and max 1000
    - **Validates: Requirements 22.1, 22.3, 22.4, 22.5, 22.6, 22.7**

- [x] 8. Usecase layer — core CRUD
  - [x] 8.1 Create usecase/customer_usecase.go with CustomerUsecase
    - Create `services/billing-api/internal/usecase/customer_usecase.go` implementing `Create`, `GetByID`, `Update`, `SoftDelete`, `List`, `Stats` methods
    - `Create`: validate → check phone duplicate → get max seq → generate customer ID → auto-generate PPPoE if needed → create customer with status `pending` → write audit log → publish `customer.created` event
    - `GetByID`: fetch customer → optionally fetch audit logs → return `CustomerDetail`
    - `Update`: validate → check phone duplicate → update → compute changed fields → write audit log with old/new values
    - `SoftDelete`: fetch customer → verify confirmation_name matches → soft delete → write audit log → publish `customer.terminated` event
    - `List`: delegate to repository with params, apply defaults (page=1, page_size=25)
    - `Stats`: delegate to repository `CountByStatus`
    - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 6.1, 6.2, 7.1, 7.2, 7.3, 7.4, 8.1, 8.2, 8.4, 8.5, 8.6, 9.1, 9.2, 9.3, 9.4, 9.5, 10.1, 10.2, 10.3, 10.4, 10.5, 17.1, 17.2, 20.1, 20.2, 21.1_

  - [x] 8.2 Write property test: New Customer Default Status (Property 8)
    - **Property 8: New Customer Default Status**
    - In `services/billing-api/internal/usecase/customer_usecase_test.go`, use `rapid.Check` to verify that for any valid creation request, the resulting customer always has `status == "pending"`
    - **Validates: Requirements 8.1**

  - [x] 8.3 Write property test: PPPoE Auto-Generation Completeness (Property 2)
    - **Property 2: PPPoE Auto-Generation Completeness**
    - In `services/billing-api/internal/usecase/customer_usecase_test.go`, use `rapid.Check` to verify that for any customer with `connection_method == "pppoe"`, both `pppoe_username` and `pppoe_password` are populated; auto-generated username follows `{first-name-lowercase}-{id-lowercase-no-dash}` format; auto-generated password is exactly 8 alphanumeric characters
    - **Validates: Requirements 5.1, 5.2, 5.3**

  - [x] 8.4 Write property test: Pagination Metadata Correctness (Property 13)
    - **Property 13: Pagination Metadata Correctness**
    - In `services/billing-api/internal/usecase/customer_usecase_test.go`, use `rapid.Check` to verify that for any total count and page_size, `total_pages == ceil(total / page_size)`, page is within [1, total_pages], and items on current page equals `min(page_size, total - (page-1)*page_size)`
    - **Validates: Requirements 6.6**

- [x] 9. Usecase layer — status transitions and package change
  - [x] 9.1 Create usecase/customer_status.go
    - Create `services/billing-api/internal/usecase/customer_status.go` implementing `Isolir`, `Activate`, `ChangePackage` methods on `CustomerUsecase`
    - `Isolir`: fetch customer → validate transition (aktif → isolir) via `domain.CanTransition` → update status → write audit log → publish `customer.isolated` event
    - `Activate`: fetch customer → validate transition (pending/isolir/suspend → aktif) → update status → write audit log → publish `customer.activated` or `customer.unblocked` event (unblocked if from isolir)
    - `ChangePackage`: fetch customer → validate package_id differs → update package → write audit log → publish `package.changed` event
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 12.1, 12.2, 12.3, 12.4, 20.4, 21.2, 21.3, 21.4, 21.6_

  - [x] 9.2 Write property test: Audit Trail Completeness (Property 9)
    - **Property 9: Audit Trail Completeness**
    - In `services/billing-api/internal/usecase/customer_usecase_test.go`, use `rapid.Check` to verify that for any customer mutation (create, update, delete, status change, package change), exactly one audit log is inserted with correct `entity_type`, `entity_id`, `action`, `actor_id`, `actor_name`, and for updates the `changes` column contains old/new values
    - **Validates: Requirements 8.6, 9.5, 10.4, 11.5, 12.4, 20.1, 20.2**

  - [x] 9.3 Write property test: Event Publishing on Lifecycle Changes (Property 10)
    - **Property 10: Event Publishing on Lifecycle Changes**
    - In `services/billing-api/internal/usecase/customer_usecase_test.go`, use `rapid.Check` to verify that for any lifecycle operation (create, activate, isolir, unblock, terminate, package change), exactly one event is published with correct `event_type`, `tenant_id`, `timestamp`, `correlation_id` (UUID v4), and payload contains all required fields
    - **Validates: Requirements 8.5, 10.5, 21.1, 21.2, 21.3, 21.4, 21.5, 21.6, 21.7**

- [x] 10. Usecase layer — bulk actions
  - [x] 10.1 Create usecase/customer_bulk.go
    - Create `services/billing-api/internal/usecase/customer_bulk.go` implementing `BulkIsolir`, `BulkActivate`, `BulkNotify`, `BulkChangePackage`, `BulkEdit`, `BulkDelete` methods on `CustomerUsecase`
    - Each method: iterate customer IDs → apply individual operation → collect successes/failures → write audit log per customer → publish events per customer → return `BulkActionResult` with `total`, `success_count`, `failure_count`, `failures`
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 14.8_

  - [x] 10.2 Write property test: Bulk Action Result Invariant (Property 11)
    - **Property 11: Bulk Action Result Invariant**
    - In `services/billing-api/internal/usecase/customer_bulk_test.go`, use `rapid.Check` to verify that for any bulk action result, `total == success_count + failure_count`, `total` equals input IDs count, and `failure_count` equals length of `failures` array
    - **Validates: Requirements 14.7**

- [x] 11. Usecase layer — import/export and area
  - [x] 11.1 Create usecase/customer_import.go
    - Create `services/billing-api/internal/usecase/customer_import.go` implementing `ImportCSV` and `GetImportTemplate` methods
    - `ImportCSV`: validate file type → store file temporarily → enqueue asynq job (`customer.import`) → return job_id
    - `GetImportTemplate`: return CSV bytes with correct column headers and one example row
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

  - [x] 11.2 Create usecase/customer_export.go
    - Create `services/billing-api/internal/usecase/customer_export.go` implementing `ExportCSV` method
    - `ExportCSV`: enqueue asynq job (`customer.export`) with filter params, format, and optional columns list → return job_id
    - If `columns` is provided, only include specified columns in export; if empty, include all columns
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

  - [x] 11.3 Create usecase/area_usecase.go
    - Create `services/billing-api/internal/usecase/area_usecase.go` implementing `AreaUsecase` with `Create`, `GetByID`, `Update`, `Delete`, `List` methods
    - `Create`: validate → check name duplicate → create area
    - `Delete`: check customer count → if > 0 return `ErrAreaHasCustomers` with count → else delete
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [x] 12. Checkpoint — Usecase layer complete
  - Ensure all usecase files compile (`go build ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

- [x] 13. HTTP handlers — customer CRUD
  - [x] 13.1 Create handler/customer_handler.go
    - Create `services/billing-api/internal/handler/customer_handler.go` with `CustomerHandler` struct (depends on `CustomerUsecase`, `*validator.Validate`, `zerolog.Logger`), constructor `NewCustomerHandler`, and methods: `List`, `Get`, `Create`, `Update`, `Delete`, `Stats`
    - Register custom validators (`phone_id`, `mac_addr`) in the constructor
    - `List`: parse query params → validate → call usecase → return paginated response
    - `Get`: parse ID + `include` query param → call usecase → return detail
    - `Create`: parse body → validate → call usecase → return 201
    - `Update`: parse ID + body → validate → call usecase → return 200
    - `Delete`: parse ID + body (confirmation_name) → validate → call usecase → return 200
    - `Stats`: call usecase → return stats map
    - Map domain errors to HTTP responses using the error mapping table from design
    - Return validation errors as aggregated array (HTTP 400, `VALIDATION_ERROR`)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 7.1, 7.2, 7.3, 7.4, 8.1, 8.2, 8.3, 8.4, 9.1, 9.2, 9.3, 9.4, 10.1, 10.2, 10.3, 17.1, 17.2, 22.8_

  - [x] 13.2 Write property test: Validation Error Aggregation (Property 7)
    - **Property 7: Validation Error Aggregation**
    - In `services/billing-api/internal/handler/customer_handler_test.go`, use `rapid.Check` to verify that for any request body with multiple invalid fields, the response returns HTTP 400 with `VALIDATION_ERROR` code and an array of field-level errors covering ALL invalid fields
    - **Validates: Requirements 22.8**

- [x] 14. HTTP handlers — customer actions, bulk, and I/O
  - [x] 14.1 Create handler/customer_action.go
    - Create `services/billing-api/internal/handler/customer_action.go` with methods: `Isolir`, `Activate`, `ChangePackage` on `CustomerHandler`
    - Each method: parse ID (+ body for change-package) → validate → call usecase → map errors → return response
    - _Requirements: 11.1, 11.2, 11.3, 12.1, 12.2, 12.3_

  - [x] 14.2 Create handler/customer_bulk.go
    - Create `services/billing-api/internal/handler/customer_bulk.go` with methods: `BulkIsolir`, `BulkActivate`, `BulkNotify`, `BulkChangePackage`, `BulkEdit`, `BulkDelete` on `CustomerHandler`
    - Each method: parse body → validate → call usecase → return `BulkActionResult`
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7_

  - [x] 14.3 Create handler/customer_io.go
    - Create `services/billing-api/internal/handler/customer_io.go` with methods: `Import`, `Export`, `ImportTemplate` on `CustomerHandler`
    - `Import`: parse multipart file → call usecase → return 202 with job_id
    - `Export`: parse query params (format + filters) → call usecase → return 202 with job_id
    - `ImportTemplate`: call usecase → return CSV file download
    - _Requirements: 15.1, 15.5, 16.1_

- [x] 15. HTTP handlers — area
  - [x] 15.1 Create handler/area_handler.go
    - Create `services/billing-api/internal/handler/area_handler.go` with `AreaHandler` struct, constructor `NewAreaHandler`, and methods: `List`, `Get`, `Create`, `Update`, `Delete`
    - Map domain errors (`ErrAreaNotFound`, `ErrAreaNameDuplicate`, `ErrAreaHasCustomers`) to HTTP responses
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [x] 16. Router wiring and RBAC configuration
  - [x] 16.1 Update handler/router.go with customer and area routes
    - Modify `services/billing-api/internal/handler/router.go`: add `CustomerHandler` and `AreaHandler` to `RouterConfig` struct, register all 18 customer endpoints and 5 area endpoints under the `api` group (which already has auth + tenant middleware)
    - Configure three RBAC groups for customers: read (admin + operator + kasir GET-only), write (admin + operator), admin-only (import, export, bulk delete)
    - Configure one RBAC group for areas: admin + operator
    - Replace the `// TODO: Tambahkan route bisnis` comment with actual route registrations
    - _Requirements: 18.1, 18.2, 18.3, 18.4_

  - [x] 16.2 Update cmd/main.go to wire customer and area dependencies
    - Modify `services/billing-api/cmd/main.go`: instantiate customer/area/audit repositories, customer/area usecases, customer/area handlers, and pass them to `RouterConfig`
    - Follow the same dependency injection pattern as existing auth/user wiring
    - _Requirements: 19.1, 19.4_

- [x] 17. Checkpoint — Full module compiles and routes registered
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ensure all tests pass (`go test ./...`). Ask the user if questions arise.

- [x] 18. Write property test: Soft-Delete Exclusion (Property 3)
  - **Property 3: Soft-Delete Exclusion**
  - In `services/billing-api/internal/usecase/customer_usecase_test.go`, use `rapid.Check` to verify that for any dataset with active and soft-deleted customers, list/stats/detail operations never return soft-deleted customers
  - **Validates: Requirements 6.7, 7.4, 17.2**

- [x] 19. Write unit tests for handlers and usecases
  - [x] 19.1 Write unit tests for CustomerHandler
    - In `services/billing-api/internal/handler/customer_handler_test.go`, test HTTP status codes, request parsing, response format for all CRUD + action + bulk + I/O endpoints, including error cases (404, 409, 422, 400)
    - _Requirements: 7.3, 8.4, 9.3, 10.2, 10.3, 11.3, 12.2, 12.3, 22.8_

  - [x] 19.2 Write unit tests for AreaHandler
    - In `services/billing-api/internal/handler/area_handler_test.go`, test HTTP status codes and error responses for all area endpoints
    - _Requirements: 13.4, 13.7_

  - [x] 19.3 Write unit tests for CustomerUsecase
    - In `services/billing-api/internal/usecase/customer_usecase_test.go`, test business logic: phone duplicate check, confirmation mismatch, same package error, invalid status transitions, PPPoE auto-generation
    - _Requirements: 5.1, 5.2, 8.4, 9.3, 10.2, 11.3, 12.2, 12.3_

  - [x] 19.4 Write unit tests for AreaUsecase
    - In `services/billing-api/internal/usecase/area_usecase_test.go`, test area name duplicate, area has customers error, CRUD operations
    - _Requirements: 13.4, 13.6, 13.7_

- [x] 20. Final checkpoint — All tests pass
  - Ensure all tests pass (`go test ./...` in `services/billing-api`). Ensure all property tests pass. Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after each layer
- Property tests validate universal correctness properties from the design document (13 properties total; Properties 4 and 12 are integration-level and deferred to integration test setup)
- The existing `customers.sql.go` and `customers.sql` will be replaced since the old table is dropped in migration 000006
- The existing `domain/customer.go` will be fully replaced with the expanded entity
- `domain/repository.go` is appended (not replaced) — existing auth interfaces remain
- Import/Export usecases enqueue asynq jobs; the actual worker processing is a separate concern
- All queries include application-level `tenant_id` filter; RLS is the safety net per Requirement 19
