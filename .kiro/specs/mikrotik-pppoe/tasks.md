# Tasks — PPPoE Management Layer

## Task 1: Domain Entities, Constants, dan Errors

- [x] 1.1 Buat file `internal/domain/pppoe.go` — PPPoE User entity, SyncStatus constants, PPPoESession struct, BuildComment/ParseComment/IsISPBossComment functions, GenerateProfileName function
- [x] 1.2 Buat file `internal/domain/pppoe_profile.go` — PPPoE Profile entity
- [x] 1.3 Buat file `internal/domain/pppoe_dto.go` — Request/Response DTOs (CreatePPPoEUserRequest, PPPoEUserListParams, PPPoEUserListResult, SyncResult, SyncStatusSummary), incoming event payloads (CustomerActivatedPayload, CustomerIsolirPayload, CustomerUnIsolirPayload, CustomerSuspendPayload, CustomerTerminatedPayload alias, PackageChangedPayload), outgoing event payloads (CommandResultPayload, SyncFailedPayload)
- [x] 1.4 Buat file `internal/domain/pppoe_command.go` — RouterOS command parameter structs (PPPoESecretParams, PPPoEProfileParams, NATRuleParams, SimpleQueueParams)
- [x] 1.5 Update file `internal/domain/errors.go` — Tambahkan PPPoE-specific domain errors (ErrPPPoEUserNotFound, ErrPPPoEUsernameExists, ErrPPPoEProfileNotFound, ErrProfileNameExists, ErrInvalidConnectionMethod, ErrInvalidIsolirMethod, ErrInvalidCommentFormat, ErrSyncInProgress, ErrMaxRetriesExhausted, ErrSessionNotFound)
- [x] 1.6 Buat file `internal/domain/pppoe_test.go` — Property tests untuk BuildComment/ParseComment round-trip (Property 1), GenerateProfileName idempotence (Property 2), IsISPBossComment detection (Property 1)

## Task 2: PPPoE Repository Interfaces dan Database Schema

- [x] 2.1 Update file `internal/domain/repository.go` — Tambahkan PPPoEUserRepository dan PPPoEProfileRepository interfaces
- [x] 2.2 Buat SQL migration file `pppoe_users` — CREATE TABLE pppoe_users dengan semua kolom, unique index, RLS policy
- [x] 2.3 Buat SQL migration file `pppoe_profiles` — CREATE TABLE pppoe_profiles dengan semua kolom, unique index, RLS policy
- [x] 2.4 Buat sqlc queries file untuk pppoe_users — CRUD queries (Create, GetByID, GetByUsername, GetByCustomerID, Update, SoftDelete, List, GetByRouterID, GetSyncStatusSummary, UpdateSyncStatus, BulkUpdateSyncStatus)
- [x] 2.5 Buat sqlc queries file untuk pppoe_profiles — CRUD queries (Create, GetByID, GetByPackageID, GetByProfileName, Update, ListByTenant)
- [x] 2.6 Jalankan `sqlc generate` dan buat repository wrapper `internal/repository/pppoe_user_repo.go`
- [x] 2.7 Buat repository wrapper `internal/repository/pppoe_profile_repo.go`

## Task 3: RouterOS Command Builder

- [x] 3.1 Buat file `internal/domain/command_builder.go` — CommandBuilder interface definition
- [x] 3.2 Buat file `internal/adapter/command_builder_v6.go` — Implementasi CommandBuilder untuk RouterOS v6 (semua methods: CreateSecret, SetSecret, RemoveSecret, PrintSecrets, RemoveActiveSession, PrintActiveSessions, CreateProfile, SetProfile, CreateNATRule, RemoveNATRuleByComment, CreateSimpleQueue, SetSimpleQueue, RemoveSimpleQueue, ResetSimpleQueueCounters)
- [x] 3.3 Buat file `internal/adapter/command_builder_v7.go` — Implementasi CommandBuilder untuk RouterOS v7 (handle parameter differences, termasuk ResetSimpleQueueCounters)
- [x] 3.4 Buat file `internal/adapter/command_builder_factory.go` — NewCommandBuilder factory function yang memilih v6/v7 berdasarkan IsRouterOSv7()
- [x] 3.5 Buat file `internal/adapter/command_builder_test.go` — Property tests: secret params completeness (Property 3), profile params with conditional burst (Property 4), version-aware paths (Property 10), isolir NAT rule builder (Property 11)

## Task 4: PPPoE Manager Usecase — Core Operations

- [x] 4.1 Buat file `internal/usecase/pppoe_manager.go` — Struct pppoeManager dengan dependencies (PPPoEUserRepo, PPPoEProfileRepo, RouterRepo, PoolManager, Crypto, EventPublisher, CommandBuilder factory), constructor NewPPPoEManager
- [x] 4.2 Implementasi `HandleCustomerActivated` — Resolve router, build PPPoE secret params, execute via pool, save to DB, publish command_result
- [x] 4.3 Implementasi `HandleIsolir` — Sequence: disable user → disconnect session → add firewall rule (NAT atau DNS berdasarkan isolir_method), publish command_result, gunakan PriorityHigh
- [x] 4.4 Implementasi `HandleUnIsolir` — Sequence: enable user → remove firewall rules by comment → reset simple queue counters (if use_simple_queue enabled), publish command_result, gunakan PriorityHigh
- [x] 4.5 Implementasi `HandleSuspend` — Sequence: disconnect session → remove secret → remove simple queue (if exists) → remove firewall rules, soft-delete DB record, publish command_result
- [x] 4.6 Implementasi `HandlePackageChanged` — Resolve new profile (create if not exists on router) → update secret profile → update simple queue (if enabled) → disconnect session for reconnect, publish command_result

## Task 5: PPPoE Manager Usecase — Sync dan Sessions

- [x] 5.1 Implementasi `SyncRouter` di `internal/usecase/pppoe_sync.go` — Retrieve secrets from router, compare with DB, categorize (synced/orphan/missing/out_of_sync), auto-fix missing dan out_of_sync, return SyncResult
- [x] 5.2 Implementasi `GetActiveSessions` — Execute /ppp/active/print via pool, parse response ke []PPPoESession
- [x] 5.3 Implementasi `DisconnectSession` — Execute /ppp/active/remove via pool
- [x] 5.4 Implementasi `GetSessionCount` — Execute /ppp/active/print count-only via pool
- [x] 5.5 Implementasi `SyncProfile` — Create/update profile di semua router dengan service_type pppoe, parallel goroutines, log error per router
- [x] 5.6 Buat file `internal/usecase/pppoe_sync_test.go` — Property tests: sync diff algorithm (Property 5), sync result count invariant (Property 6)

## Task 6: PPPoE Manager Usecase — Manual CRUD

- [x] 6.1 Implementasi `CreateUser` — Validate input, encrypt password, build command, execute on router, save to DB
- [x] 6.2 Implementasi `DeleteUser` — Disconnect session, remove from router, soft-delete from DB
- [x] 6.3 Implementasi `ListUsers` — Query DB with pagination
- [x] 6.4 Implementasi `GetSyncStatus` — Query DB for sync status summary per router

## Task 7: Event Worker

- [x] 7.1 Buat file `internal/worker/pppoe_worker.go` — PPPoEEventWorker struct, RegisterHandlers (customer.activated, customer.isolir, customer.un_isolir, customer.suspend, customer.terminated, package.changed). customer.terminated menggunakan handler yang sama dengan customer.suspend
- [x] 7.2 Implementasi handler functions — Decode TaskEnvelope, validate payload, filter connection_method=pppoe, delegate ke PPPoEManager
- [x] 7.3 Implementasi retry logic — Exponential backoff (30s, 1m, 2m, 5m, 10m), max 5 retries, mark failed_permanent setelah exhausted, publish mikrotik.sync_failed event saat permanent failure
- [x] 7.4 Buat file `internal/worker/pppoe_worker_test.go` — Property test: retry backoff schedule (Property 7), unit tests: handler routing, payload validation

## Task 8: PPPoE Event Publisher

- [x] 8.1 Buat file `internal/usecase/pppoe_event_publisher.go` — PPPoEEventPublisher implementation, PublishCommandResult method, PublishSyncFailed method (event type "mikrotik.sync_failed")
- [x] 8.2 Buat file `internal/usecase/pppoe_event_test.go` — Property tests: command result payload completeness (Property 8), error message safety (Property 9)

## Task 9: HTTP Handlers

- [x] 9.1 Buat file `internal/handler/pppoe_handler.go` — PPPoEHandler struct dengan methods: ListUsers, CreateUser, DeleteUser, DisconnectUser, GetSyncStatus, TriggerSync
- [x] 9.2 Buat file `internal/handler/session_handler.go` — SessionHandler struct dengan methods: GetSessions, DisconnectSession, GetSessionCount
- [x] 9.3 Update file `internal/handler/router.go` — Tambahkan route registration untuk PPPoE endpoints (/api/v1/mikrotik/routers/:id/pppoe/users, /sessions, /sync, /sync-status)
- [x] 9.4 Buat file `internal/handler/pppoe_handler_test.go` — Unit tests: request validation, response format, error mapping

## Task 10: Sync Job (Periodic Background)

- [x] 10.1 Buat file `internal/usecase/sync_scheduler.go` — SyncScheduler struct yang menjalankan periodic sync setiap 15 menit (configurable) untuk semua online routers dengan service_type pppoe
- [x] 10.2 Implementasi scheduling logic — Iterate online routers, skip offline/maintenance, call SyncRouter per router, log results
- [x] 10.3 Buat file `internal/usecase/sync_scheduler_test.go` — Unit tests: scheduling logic, skip offline routers

## Task 11: Wiring dan Integration

- [x] 11.1 Update `internal/config/config.go` — Tambahkan config fields: SyncIntervalMinutes, DefaultIsolirMethod, WalledGardenIP, DNSServerIP
- [x] 11.2 Update `cmd/main.go` — Wire PPPoE dependencies: PPPoEUserRepo, PPPoEProfileRepo, PPPoEManager, PPPoEEventPublisher, PPPoEEventWorker, SyncScheduler, PPPoEHandler, SessionHandler
- [x] 11.3 Register asynq worker handlers — Tambahkan PPPoEEventWorker.RegisterHandlers ke asynq ServeMux
- [x] 11.4 Start SyncScheduler di goroutine — Mulai periodic sync saat service start, stop saat shutdown
- [x] 11.5 Integration test end-to-end — Test full flow: event masuk → worker → manager → mock adapter → DB → event keluar
