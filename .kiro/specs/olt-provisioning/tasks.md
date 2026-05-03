# Tasks — OLT Provisioning Layer

## Overview

Implementasi OLT Provisioning Layer di `services/network-service/`. Layer ini dibangun di atas OLT Management Layer yang sudah diimplementasikan. Provisioning layer menangani: manajemen entitas ONT, deteksi ONT unregistered, provisioning ONT (single dan bulk), decommission ONT, reboot ONT, VLAN management, service profile management, deteksi port migration, command builder per brand, audit trail, event integration, provisioning settings per tenant, dan HTTP API.

## Tasks

- [x] 1. Domain Entities, Constants, dan Errors
  - [x] 1.1 Buat file `internal/domain/ont.go` — ONT entity struct, ONTStatus constants (registered, provisioned, unregistered, missing, decommissioned), ProvisioningState constants (pending, in_progress, completed, failed), ValidONTTransitions map, CanTransitionONT helper
    - Maksimal 200 baris per file
    - _Requirements: 1.1, 1.5, 1.6_

  - [x] 1.2 Buat file `internal/domain/vlan.go` — VLAN entity struct, VLANStrategy constants (single, per_paket, per_odp, per_pelanggan), VLANType constants (data, voice, management), VLANResolveContext struct
    - Maksimal 200 baris per file
    - _Requirements: 9.1, 9.4_

  - [x] 1.3 Buat file `internal/domain/service_profile.go` — ServiceProfile entity struct
    - Maksimal 200 baris per file
    - _Requirements: 10.1_

  - [x] 1.4 Buat file `internal/domain/audit_log_provisioning.go` — ProvisioningAuditLog entity struct, AuditAction constants (ont_provision, ont_decommission, ont_reboot, service_port_add, service_port_remove, bulk_provision, auto_provision)
    - Maksimal 200 baris per file
    - _Requirements: 12.1, 12.4, 12.5_

  - [x] 1.5 Buat file `internal/domain/provisioning_settings.go` — ProvisioningSettings entity struct, default values helper
    - Maksimal 200 baris per file
    - _Requirements: 15.1, 15.5_

  - [x] 1.6 Update file `internal/domain/errors.go` — Tambahkan provisioning-specific domain errors: ErrONTNotFound, ErrONTSerialNumberExists, ErrONTPositionExists, ErrONTAlreadyProvisioned, ErrONTNotProvisioned, ErrCustomerHasActiveONT, ErrProvisioningInProgress, ErrProvisioningFailed, ErrDecommissionFailed, ErrRebootFailed, ErrVLANNotFound, ErrVLANIDExists, ErrVLANInUse, ErrVLANResolutionFailed, ErrServiceProfileNotFound, ErrServiceProfileExists, ErrServiceProfileInUse, ErrNoProfileMapping, ErrBulkNotFound, ErrInvalidCSVFormat, ErrBulkAlreadyExecuted, ErrInvalidVLANStrategy
    - _Requirements: 3.4, 3.6, 3.7, 7.4, 8.4, 9.7, 10.4, 10.6_

- [x] 2. Domain DTOs dan Adapter Types
  - [x] 2.1 Buat file `internal/domain/ont_dto.go` — Request DTOs (ProvisionONTRequest, ONTListParams), Response DTOs (ONTResponse, ONTDetailResponse, ONTListResult)
    - Maksimal 200 baris per file
    - _Requirements: 14.1, 14.9, 14.10_

  - [x] 2.2 Buat file `internal/domain/vlan_dto.go` — Request DTOs (CreateVLANRequest, UpdateVLANRequest, VLANListParams), Response DTOs (VLANResponse, VLANListResult)
    - Maksimal 200 baris per file
    - _Requirements: 9.6_

  - [x] 2.3 Buat file `internal/domain/service_profile_dto.go` — Request DTOs (CreateServiceProfileRequest, UpdateServiceProfileRequest, ServiceProfileListParams), Response DTOs (ServiceProfileResponse, ServiceProfileListResult)
    - Maksimal 200 baris per file
    - _Requirements: 10.5_

  - [x] 2.4 Buat file `internal/domain/provisioning_dto.go` — BulkPreview, BulkRowPreview, BulkResult, BulkRowResult, UpdateSettingsRequest, AuditLogListParams, AuditLogListResult structs
    - Maksimal 200 baris per file
    - _Requirements: 5.3, 5.6, 12.6, 15.4_

  - [x] 2.5 Buat file `internal/domain/provisioning_adapter_types.go` — AddONTParams, RemoveONTParams, AddServicePortParams, RemoveServicePortParams, RebootONTParams, ProvisioningResult, UnregisteredONT structs
    - Maksimal 200 baris per file
    - _Requirements: 4.6_

  - [x] 2.6 Buat file `internal/domain/provisioning_event.go` — Event type constants (ont.provisioned, ont.decommissioned, ont.auto_provisioned, ont.auto_provision_failed, ont.port_migrated), Event payloads (ONTProvisionedPayload, ONTDecommissionedPayload, ONTAutoProvisionedPayload, ONTAutoProvisionFailedPayload, ONTPortMigratedPayload)
    - Maksimal 200 baris per file
    - _Requirements: 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [x] 3. Repository Interfaces (Extend Existing)
  - [x] 3.1 Update file `internal/domain/repository.go` — Tambahkan ONTRepository, VLANRepository, ServiceProfileRepository, AuditLogRepository, ProvisioningSettingsRepository interfaces. Extend OLTAdapter interface dengan provisioning methods (AddONT, RemoveONT, AddServicePort, RemoveServicePort, RebootONT, GetUnregisteredONTs). Extend OLTEventPublisher interface dengan provisioning event methods (PublishONTProvisioned, PublishONTDecommissioned, PublishONTAutoProvisioned, PublishONTAutoProvisionFailed, PublishONTPortMigrated). Tambahkan ProvisioningManager, VLANManager, ServiceProfileManager interfaces.
    - _Requirements: 1.1, 4.6, 9.1, 10.1, 12.1, 13.1, 15.1_

- [x] 4. Property Tests untuk Domain Logic
  - [x] 4.1 Buat file `internal/adapter/olt_provisioning_cmd_test.go` — Property test untuk command builder produces valid commands per brand
    - **Property 1: Command Builder Produces Valid Commands per Brand**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5, 4.7**

  - [x] 4.2 Buat file `internal/usecase/provisioning_reboot_test.go` — Property test untuk reboot status guard (hanya ONT provisioned yang boleh reboot)
    - **Property 3: Reboot Status Guard**
    - **Validates: Requirements 8.4**

  - [x] 4.3 Buat file `internal/usecase/provisioning_settings_test.go` — Property test untuk default provisioning settings (tenant tanpa settings record mendapat default values)
    - **Property 8: Default Provisioning Settings**
    - **Validates: Requirements 15.5**

- [x] 5. Checkpoint — Domain Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Database Migrations (5 New Tables)
  - [x] 6.1 Buat SQL migration file `migrations/000011_create_vlans.up.sql` — CREATE TABLE vlans dengan semua kolom, UNIQUE constraint (olt_id, vlan_id WHERE deleted_at IS NULL), index (olt_id), RLS policy
    - _Requirements: 9.1, 9.2, 9.3_

  - [x] 6.2 Buat SQL migration file `migrations/000012_create_service_profiles.up.sql` — CREATE TABLE service_profiles dengan semua kolom, UNIQUE constraint (olt_id, line_profile_id, service_profile_id WHERE deleted_at IS NULL), indexes (olt_id, package_id), RLS policy
    - _Requirements: 10.1, 10.2, 10.3_

  - [x] 6.3 Buat SQL migration file `migrations/000013_create_onts.up.sql` — CREATE TABLE onts dengan semua kolom, UNIQUE constraints (tenant_id+serial_number, olt_id+pon_port+ont_index WHERE deleted_at IS NULL), indexes (olt_id+status, customer_id, tenant_id), RLS policy
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 6.4 Buat SQL migration file `migrations/000014_create_provisioning_audit_logs.up.sql` — CREATE TABLE provisioning_audit_logs dengan semua kolom, indexes (olt_id+created_at, ont_id+created_at, tenant_id+created_at, action+created_at), RLS policy. Append-only: no update/delete.
    - _Requirements: 12.1, 12.2, 12.7_

  - [x] 6.5 Buat SQL migration file `migrations/000015_create_provisioning_settings.up.sql` — CREATE TABLE provisioning_settings dengan semua kolom, UNIQUE constraint (tenant_id), RLS policy
    - _Requirements: 15.1, 15.2_

- [x] 7. sqlc Queries dan Repository Wrappers
  - [x] 7.1 Buat sqlc queries file `queries/onts.sql` — CRUD queries (Create, GetByID, GetBySerialNumber, Update, SoftDelete, List, ListByOLTAndStatus, GetByCustomerID, SerialNumberExists, PositionExists, UpdateStatus, UpdatePortMigration, DeleteUnregisteredByOLT)
    - _Requirements: 1.1, 2.1, 2.4, 14.9_

  - [x] 7.2 Buat sqlc queries file `queries/vlans.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, List, GetByOLTAndVLANID, GetDefaultVLAN, VLANIDExists, CountActiveONTs)
    - _Requirements: 9.1, 9.6, 9.7_

  - [x] 7.3 Buat sqlc queries file `queries/service_profiles.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, List, GetByPackageAndOLT, ProfileExists, CountActiveONTs)
    - _Requirements: 10.1, 10.5, 10.6_

  - [x] 7.4 Buat sqlc queries file `queries/provisioning_audit_logs.sql` — Create, List (with filters: olt_id, ont_id, action, date range, pagination). No update/delete queries.
    - _Requirements: 12.1, 12.6, 12.7_

  - [x] 7.5 Buat sqlc queries file `queries/provisioning_settings.sql` — GetByTenantID, Upsert
    - _Requirements: 15.1, 15.3, 15.4_

  - [x] 7.6 Jalankan `sqlc generate` dan buat repository wrapper `internal/repository/ont_repo.go`
    - _Requirements: 1.1_

  - [x] 7.7 Buat repository wrapper `internal/repository/vlan_repo.go`
    - _Requirements: 9.1_

  - [x] 7.8 Buat repository wrapper `internal/repository/service_profile_repo.go`
    - _Requirements: 10.1_

  - [x] 7.9 Buat repository wrapper `internal/repository/audit_log_repo.go`
    - _Requirements: 12.1_

  - [x] 7.10 Buat repository wrapper `internal/repository/provisioning_settings_repo.go`
    - _Requirements: 15.1_

- [x] 8. Extend OLT Adapter dengan Provisioning Methods
  - [x] 8.1 Update file `internal/adapter/olt_zte_adapter.go` (atau buat `olt_zte_provisioning.go` jika melebihi 200 baris) — Implementasi AddONT, RemoveONT, AddServicePort, RemoveServicePort, RebootONT, GetUnregisteredONTs untuk ZTE. CLI commands: `onu add sn`, `service-port add vlan`, `no onu`, `no service-port`, `onu reset`. Gunakan CLIConnector existing.
    - Maksimal 200 baris per file
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.8_

  - [x] 8.2 Update stub adapters (`olt_huawei_adapter.go`, `olt_fiberhome_adapter.go`, `olt_vsol_adapter.go`, `olt_hsgq_adapter.go`) — Tambahkan provisioning method stubs yang return ErrUnsupportedBrand atau placeholder response
    - Maksimal 200 baris per file
    - _Requirements: 4.8_

  - [x] 8.3 Update file `internal/adapter/olt_mock_adapter.go` — Tambahkan provisioning method mocks yang return simulated success responses tanpa network connection
    - Maksimal 200 baris per file
    - _Requirements: 4.9_

- [x] 9. Checkpoint — Infrastructure Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Provisioning Manager Usecase
  - [x] 10.1 Buat file `internal/usecase/provisioning_manager.go` — Struct provisioningManager dengan dependencies (ONTRepo, VLANRepo, ServiceProfileRepo, AuditLogRepo, SettingsRepo, OLTRepo, OLTAdapterFactory, OLTEventPublisher, CLIConnector, CredentialEncryptor), constructor NewProvisioningManager
    - Maksimal 200 baris per file
    - _Requirements: 3.1_

  - [x] 10.2 Implementasi `ProvisionONT` — Validate input (serial_number unik, customer belum punya ONT aktif, posisi tersedia), resolve service profile via ServiceProfileManager, resolve VLAN via VLANManager, set provisioning_state="in_progress", create adapter via factory, call AddONT + AddServicePort, update status="provisioned" + state="completed", create audit log, publish "ont.provisioned" event
    - Split ke file terpisah jika melebihi 200 baris
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

  - [x] 10.3 Implementasi `DecommissionONT` — Get ONT by ID, set provisioning_state="in_progress", create adapter, call RemoveServicePort + RemoveONT, update status="decommissioned" + clear customer_id + set last_decommissioned_at, create audit log, publish "ont.decommissioned" event
    - _Requirements: 7.1, 7.3, 7.4, 7.6_

  - [x] 10.4 Implementasi `RebootONT` — Validate ONT status == "provisioned", create adapter, call RebootONT, create audit log, return ProvisioningResult
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [x] 10.5 Implementasi `ValidateBulk` dan `ExecuteBulk` — Parse CSV (columns: sn_ont, pelanggan_id, pon_port, vlan, odp, deskripsi), validate per row, return BulkPreview. ExecuteBulk: provision valid rows sequentially, per-row error handling (continue on failure), return BulkResult
    - Split ke file `internal/usecase/provisioning_bulk.go` jika perlu
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_

  - [x] 10.6 Implementasi `GetBulkTemplate` — Return CSV template bytes dengan header columns
    - _Requirements: 5.8_

  - [x] 10.7 Implementasi `HandleUnregisteredONT` — Create ONT record dengan status="unregistered", check auto_provisioning_enabled di settings, jika enabled dan SN match customer → auto-provision, publish event sesuai hasil
    - _Requirements: 2.1, 2.3, 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 10.8 Implementasi `HandlePortMigration` dan `ConfirmMigration` — Publish "ont.port_migrated" event, check auto_port_migration_enabled, jika enabled → auto-update DB, jika disabled → flag sebagai "port_migrated" dan tunggu konfirmasi admin
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 10.9 Implementasi `HandleCustomerTerminated` — Lookup ONT by customer_id, execute decommission sequence, retry pattern exponential backoff (30s, 1m, 2m, 5m, 10m max 5 retries)
    - _Requirements: 7.2, 7.5_

  - [x] 10.10 Implementasi `GetONTByID`, `ListONTs`, `GetUnregisteredONTs` — Query ONT repo, include relasi (OLT name, ODP name, VLAN name, service profile name), paginasi dan filter
    - _Requirements: 2.2, 14.2, 14.9, 14.10_

  - [x] 10.11 Implementasi `GetAuditLogs` — Query audit log repo dengan filter (olt_id, ont_id, action, date range) dan paginasi
    - _Requirements: 12.6, 14.11_

  - [x] 10.12 Implementasi `GetSettings` dan `UpdateSettings` — Get settings by tenant_id (return defaults jika tidak ada), upsert settings
    - _Requirements: 15.3, 15.4, 15.5_

  - [x] 10.13 Buat file `internal/usecase/provisioning_manager_test.go` — Unit tests: ProvisionONT happy path, validation errors (SN exists, customer has ONT, position taken), CLI failure handling
    - _Requirements: 3.1, 3.4, 3.6, 3.7_

  - [x] 10.14 Buat file `internal/usecase/provisioning_decommission_test.go` — Unit tests: manual decommission, event-driven decommission, CLI failure, retry logic
    - _Requirements: 7.1, 7.2, 7.4, 7.5_

  - [x] 10.15 Buat file `internal/usecase/provisioning_bulk_test.go` — Property test untuk bulk provisioning count invariant + unit tests: CSV parsing, validation, execution, per-row error handling
    - **Property 2: Bulk Provisioning Count Invariant**
    - **Validates: Requirements 5.2, 5.6**

  - [x] 10.16 Buat file `internal/usecase/provisioning_auto_test.go` — Unit tests: auto-provisioning enabled/disabled, SN match/no-match, success/failure events
    - _Requirements: 6.1, 6.3, 6.4, 6.5_

- [x] 11. VLAN Manager Usecase
  - [x] 11.1 Buat file `internal/usecase/vlan_manager.go` — Struct vlanManager dengan dependencies (VLANRepo, OLTRepo), constructor NewVLANManager. Implementasi Create, GetByID, Update, Delete (cek active ONTs sebelum delete), List, ResolveVLAN (strategy resolution: single → default VLAN, per_paket → VLAN by package, per_odp → VLAN by ODP, per_pelanggan → unique VLAN)
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 9.4, 9.5, 9.6, 9.7_

  - [x] 11.2 Buat file `internal/usecase/vlan_manager_test.go` — Unit tests: CRUD, delete guard (VLAN in use), ResolveVLAN per strategy
    - _Requirements: 9.5, 9.7_

  - [x] 11.3 Buat file `internal/usecase/vlan_strategy_test.go` — Property test untuk VLAN strategy resolution correctness
    - **Property 4: VLAN Strategy Resolution**
    - **Validates: Requirements 9.5**

- [x] 12. Service Profile Manager Usecase
  - [x] 12.1 Buat file `internal/usecase/service_profile_manager.go` — Struct serviceProfileManager dengan dependencies (ServiceProfileRepo, OLTRepo), constructor NewServiceProfileManager. Implementasi Create, GetByID, Update, Delete (cek active ONTs sebelum delete), List, ResolveProfile (lookup by package_id + olt_id)
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 10.4, 10.5, 10.6_

  - [x] 12.2 Buat file `internal/usecase/service_profile_manager_test.go` — Unit tests: CRUD, delete guard (profile in use), ResolveProfile happy path + no mapping error
    - _Requirements: 10.4, 10.6_

- [x] 13. Provisioning Event Publisher (Extend Existing)
  - [x] 13.1 Update file `internal/usecase/olt_event_publisher.go` — Tambahkan method PublishONTProvisioned, PublishONTDecommissioned, PublishONTAutoProvisioned, PublishONTAutoProvisionFailed, PublishONTPortMigrated. Best-effort publish with error logging, correlation_id generation via UUID.
    - Maksimal 200 baris per file, split ke `provisioning_event_publisher.go` jika perlu
    - _Requirements: 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

  - [x] 13.2 Buat file `internal/usecase/provisioning_event_test.go` — Property test untuk provisioning event payload completeness
    - **Property 7: Provisioning Event Payload Completeness**
    - **Validates: Requirements 13.2, 13.3, 13.4, 13.5, 13.6, 13.7**

- [x] 14. Provisioning Event Worker
  - [x] 14.1 Buat file `internal/worker/provisioning_worker.go` — ProvisioningEventWorker struct, RegisterHandlers (register "customer.terminated" handler ke asynq ServeMux), HandleCustomerTerminated (parse payload, call ProvisioningManager.HandleCustomerTerminated). Retry pattern: exponential backoff 30s, 1m, 2m, 5m, 10m (max 5 retries). Pattern sama dengan PPPoEEventWorker.
    - Maksimal 200 baris per file
    - _Requirements: 7.2, 7.5, 13.1_

  - [x] 14.2 Buat file `internal/worker/provisioning_worker_test.go` — Unit tests: event handling, retry delays, error scenarios
    - _Requirements: 7.2, 7.5, 13.1_

- [x] 15. Checkpoint — Business Logic Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 16. HTTP Handlers
  - [x] 16.1 Buat file `internal/handler/provisioning_handler.go` — ProvisioningHandler struct dengan methods: ProvisionONT (POST /provisioning/ont), ListONTs (GET /provisioning/onts), GetONT (GET /provisioning/onts/:id), DecommissionONT (POST /provisioning/ont/:id/decommission), RebootONT (POST /provisioning/ont/:id/reboot), ConfirmMigration (POST /provisioning/ont/:id/confirm-migration), GetUnregisteredONTs (GET /devices/:id/unregistered-onts), BulkUpload (POST /provisioning/bulk), BulkExecute (POST /provisioning/bulk/execute), BulkTemplate (GET /provisioning/bulk/template), GetAuditLogs (GET /provisioning/audit-logs), GetSettings (GET /provisioning/settings), UpdateSettings (PUT /provisioning/settings)
    - Maksimal 200 baris per file, split ke `provisioning_handler_bulk.go` dan `provisioning_handler_settings.go` jika perlu
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 14.8, 14.9, 14.10, 14.11, 14.12, 15.3, 15.4_

  - [x] 16.2 Buat file `internal/handler/vlan_handler.go` — VLANHandler struct dengan methods: CreateVLAN (POST /devices/:id/vlans), ListVLANs (GET /devices/:id/vlans), UpdateVLAN (PUT /vlans/:id), DeleteVLAN (DELETE /vlans/:id)
    - Maksimal 200 baris per file
    - _Requirements: 9.6_

  - [x] 16.3 Buat file `internal/handler/service_profile_handler.go` — ServiceProfileHandler struct dengan methods: CreateServiceProfile (POST /devices/:id/service-profiles), ListServiceProfiles (GET /devices/:id/service-profiles), UpdateServiceProfile (PUT /service-profiles/:id), DeleteServiceProfile (DELETE /service-profiles/:id)
    - Maksimal 200 baris per file
    - _Requirements: 10.5_

  - [x] 16.4 Buat file `internal/handler/provisioning_handler_test.go` — Unit tests: request validation, response format, error mapping (400/404/409/422/502/504)
    - _Requirements: 14.1, 14.12_

  - [x] 16.5 Buat file `internal/handler/vlan_handler_test.go` — Unit tests: CRUD endpoints, validation, error mapping
    - _Requirements: 9.6_

  - [x] 16.6 Buat file `internal/handler/service_profile_handler_test.go` — Unit tests: CRUD endpoints, validation, error mapping
    - _Requirements: 10.5_

- [x] 17. Route Registration + Wiring
  - [x] 17.1 Update `internal/handler/router.go` — Tambahkan ProvisioningHandler, VLANHandler, ServiceProfileHandler ke RouterConfig struct. Register route groups: /api/v1/olt/provisioning/* (ONT, bulk, audit, settings), /api/v1/olt/devices/:id/vlans, /api/v1/olt/devices/:id/service-profiles, /api/v1/olt/devices/:id/unregistered-onts, /api/v1/olt/vlans/:id, /api/v1/olt/service-profiles/:id
    - _Requirements: 14.1, 9.6, 10.5_

  - [x] 17.2 Update `cmd/main.go` — Wire provisioning dependencies: ONTRepo, VLANRepo, ServiceProfileRepo, AuditLogRepo, ProvisioningSettingsRepo, ProvisioningManager, VLANManager, ServiceProfileManager, ProvisioningHandler, VLANHandler, ServiceProfileHandler, ProvisioningEventWorker. Register ProvisioningEventWorker handlers ke asynq ServeMux. Start worker on startup, stop on shutdown.
    - _Requirements: 3.1, 7.2, 13.1_

- [x] 18. Integration Tests
  - [x] 18.1 Buat file `internal/usecase/provisioning_integration_test.go` — Integration test end-to-end: create VLAN → create service profile → provision ONT → verify status → decommission ONT → verify status. Test bulk provisioning flow: CSV upload → validate → execute. Test cross-tenant isolation (RLS). Test port migration detection + confirmation.
    - _Requirements: 1.2, 3.1, 5.1, 7.1, 9.1, 10.1, 11.5, 14.12_

  - [x] 18.2 Buat file `internal/usecase/provisioning_audit_test.go` — Property test untuk audit log completeness
    - **Property 6: Audit Log Completeness**
    - **Validates: Requirements 12.3, 12.4, 12.5**

  - [x] 18.3 Buat file `internal/usecase/port_migration_test.go` — Property test untuk port migration detection
    - **Property 5: Port Migration Detection**
    - **Validates: Requirements 11.1**

- [x] 19. Final Checkpoint
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- Maksimal 200 baris per file Go — split ke file terpisah jika melebihi
- Gunakan `pgregory.net/rapid` untuk property-based testing (sudah ada di go.mod)
- Gunakan sqlc untuk query generation (sudah ada di codebase)
- Fiber v2 untuk HTTP handlers (sudah ada di codebase)
- asynq untuk event worker (sudah ada di codebase)
- Semua komentar dalam bahasa Indonesia
- Stub adapters (Huawei, FiberHome, VSOL, HSGQ) return ErrUnsupportedBrand — implementasi penuh di masa depan
- ZTE adapter implementasi penuh untuk provisioning CLI commands
- Mock adapter return simulated success untuk development tanpa OLT fisik
- Migration numbering mulai dari 000011 (setelah 000010_create_olt_alarms dari olt-management)
- Layer ini extend interface dan file yang sudah ada dari olt-management, bukan membuat ulang
