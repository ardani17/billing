# Implementation Plan: MikroTik Router Foundation Layer

## Overview

Implementasi bottom-up untuk MikroTik Router Foundation Layer di `services/network-service/`. Dimulai dari database migration, domain entities, crypto, adapter, pool, repository, metrics, usecase, handler, health checker, dan wiring. Mengikuti pattern yang sama dengan notification-service dan billing-api.

## Tasks

- [x] 1. Database migration dan sqlc setup
  - [x] 1.1 Buat file migration SQL `services/network-service/migrations/001_create_routers_table.sql`
    - Buat tabel `routers` dengan semua kolom sesuai design (id, tenant_id, name, host, port, username, password_encrypted, use_ssl, service_types, router_os_version, board_name, cpu_count, total_ram_mb, identity, status, health_check_interval_sec, last_online_at, last_checked_at, last_uptime_sec, failure_count, notes, deleted_at, created_at, updated_at)
    - Kolom `service_types` bertipe JSONB NOT NULL DEFAULT '["pppoe"]' — menyimpan array tipe layanan (pppoe, hotspot, dhcp_binding, static)
    - Buat unique index `idx_routers_tenant_name` pada (tenant_id, name) WHERE deleted_at IS NULL
    - Buat index `idx_routers_tenant_id` dan `idx_routers_status` WHERE deleted_at IS NULL
    - Enable Row-Level Security dan buat policy `routers_tenant_isolation`
    - Set default values: port=8728, use_ssl=false, status='offline', health_check_interval_sec=60, failure_count=0
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 1.2 Buat file `services/network-service/sqlc.yaml` dan SQL query files
    - Buat `sqlc.yaml` dengan konfigurasi engine postgresql, package repository, sql_package pgx/v5
    - Buat `services/network-service/queries/routers.sql` dengan query: CreateRouter, GetRouterByID, UpdateRouter, SoftDeleteRouter, ListRouters, CountByStatus, GetActiveRouters, NameExists, UpdateHealthCheck
    - _Requirements: 1.1, 5.1, 5.3, 5.5, 5.6, 6.2, 6.3_

- [x] 2. Domain entities, constants, errors, dan DTOs
  - [x] 2.1 Buat `services/network-service/internal/domain/constants.go`
    - Definisikan RouterStatus type dan constants: StatusOnline, StatusOffline, StatusMaintenance
    - Definisikan ValidRouterTransitions map dan fungsi CanTransitionRouter
    - Definisikan ServiceType constants: ServicePPPoE, ServiceHotspot, ServiceDHCP, ServiceStatic
    - Definisikan CommandPriority constants: PriorityHigh (3), PriorityMedium (2), PriorityLow (1)
    - Definisikan fungsi IsRouterOSv7(version string) bool — cek apakah versi dimulai dengan "7"
    - _Requirements: 2.2, 2.3, 2.4, 2.6, 3.7, 4.10_

  - [x] 2.2 Buat `services/network-service/internal/domain/router.go`
    - Definisikan struct Router dengan semua field sesuai design (termasuk ServiceTypes []string)
    - Definisikan struct ConnectionConfig, SystemResource, RouterMetrics, RouterMetricsPoint, PoolStats, StatusSummary, HealthCheckUpdate
    - _Requirements: 2.1, 2.5, 2.6, 6.2, 7.1, 9.4_

  - [x] 2.3 Buat `services/network-service/internal/domain/errors.go`
    - Definisikan semua domain errors: ErrRouterNotFound, ErrRouterNameExists, ErrInvalidStatusTransition, ErrConfirmationMismatch, ErrRouterOffline, ErrConnectionFailed, ErrConnectionTimeout, ErrPoolExhausted, ErrRateLimited, ErrEncryptionFailed, ErrDecryptionFailed, ErrInvalidEncryptionKey, ErrRouterDeleted
    - _Requirements: 2.4, 4.6, 4.7, 4.8, 5.9, 8.4, 8.7_

  - [x] 2.4 Buat `services/network-service/internal/domain/dto.go`
    - Definisikan CreateRouterRequest (termasuk ServiceTypes []string), UpdateRouterRequest, RebootRequest DTOs dengan validation tags
    - Definisikan RouterResponse, RouterDetailResponse, RouterListParams, RouterListResult
    - Definisikan event payloads: RouterOfflinePayload, RouterOnlinePayload, RouterRebootPayload
    - Implementasikan helper functions: SuccessResponse, ErrorResponse, PaginatedResponse (reuse pattern dari notification-service)
    - _Requirements: 5.1, 5.4, 5.5, 5.8, 10.1, 10.2, 10.3_

  - [x] 2.5 Buat `services/network-service/internal/domain/repository.go`
    - Definisikan interface RouterRepository dengan semua method sesuai design
    - Definisikan interface MetricsStore dengan method Store, Query, GetLatest
    - Definisikan interface EventPublisher dengan method PublishRouterOffline, PublishRouterOnline, PublishUnexpectedReboot
    - Definisikan interface CredentialEncryptor dengan method Encrypt, Decrypt
    - Definisikan interface RouterOSAdapter, ConnPool, PoolManager, HealthChecker, RouterUsecase
    - _Requirements: 1.1, 3.1, 4.1, 5.1, 6.1, 8.1, 9.1, 10.1_

  - [x] 2.6 Write property tests untuk domain (status transition, reboot confirmation, status summary)
    - **Property 1: Status transition correctness** — test CanTransitionRouter untuk semua kombinasi status
    - **Property 4: Reboot confirmation validation** — test bahwa hanya nama yang cocok yang diterima
    - **Property 8: Status summary invariant** — test total == online + offline + maintenance
    - **Validates: Requirements 2.3, 2.4, 5.8, 5.9, 7.1**

- [x] 3. Credential encryption (AES-256-GCM)
  - [x] 3.1 Buat `services/network-service/internal/crypto/crypto.go`
    - Implementasikan struct AESEncryptor yang mengimplementasikan CredentialEncryptor interface
    - Fungsi NewAESEncryptor(key []byte) yang memvalidasi key harus 32 bytes
    - Method Encrypt: generate random nonce 12 bytes, encrypt dengan AES-256-GCM, return base64-encoded (nonce + ciphertext)
    - Method Decrypt: decode base64, extract nonce, decrypt dengan AES-256-GCM
    - Error messages TIDAK boleh mengekspos key atau plaintext
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7_

  - [x] 3.2 Write property tests untuk crypto
    - **Property 9: Encryption nonce uniqueness** — dua kali encrypt plaintext yang sama menghasilkan ciphertext berbeda
    - **Property 10: Encryption round-trip** — Decrypt(Encrypt(p)) == p untuk semua string valid
    - **Property 11: Wrong key decryption error safety** — decrypt dengan key berbeda harus error, error tidak mengandung key/plaintext
    - **Validates: Requirements 8.5, 8.6, 8.7**

- [x] 4. RouterOS adapter (interface, mock, live, factory)
  - [x] 4.1 Buat `services/network-service/internal/adapter/adapter.go`
    - Re-export RouterOSAdapter interface dari domain (atau definisikan di sini jika lebih clean)
    - Definisikan ConnectionConfig re-export
    - _Requirements: 3.1_

  - [x] 4.2 Buat `services/network-service/internal/adapter/mock_adapter.go`
    - Implementasikan MockAdapter struct yang mengimplementasikan RouterOSAdapter
    - Connect: selalu sukses (no-op)
    - Close: no-op
    - Execute: return predefined responses berdasarkan command
    - GetSystemResource: return SystemResource dengan values realistis (version "6.49.10", board "RB750Gr3", CPU 2, RAM 256MB, uptime 3888000)
    - Ping: selalu sukses
    - _Requirements: 3.3, 3.5_

  - [x] 4.3 Buat `services/network-service/internal/adapter/live_adapter.go`
    - Implementasikan LiveAdapter struct menggunakan go-routeros library
    - Connect: buka koneksi TCP ke router dengan timeout dari ConnectionConfig
    - Close: tutup koneksi
    - Execute: jalankan command via routeros.Client dan parse response
    - GetSystemResource: execute "/system/resource/print" dan parse ke SystemResource. Detect versi v6/v7 dari field Version
    - Ping: execute "/system/identity/print" sebagai health check
    - Return ErrConnectionFailed/ErrConnectionTimeout jika gagal
    - _Requirements: 3.4, 3.6, 3.7_

  - [x] 4.4 Buat `services/network-service/internal/adapter/factory.go`
    - Fungsi NewAdapter(mode string) yang return MockAdapter jika mode=="mock", LiveAdapter jika mode=="live"
    - _Requirements: 3.3, 3.4_

- [x] 5. Connection pool
  - [x] 5.1 Buat `services/network-service/internal/pool/pool.go`
    - Implementasikan struct connPool yang mengimplementasikan ConnPool interface
    - Max 5 koneksi per pool, lazy connect
    - Idle timeout 5 menit, max lifetime 1 jam
    - Health ping setiap 30 detik pada idle connections
    - Queue commands saat pool penuh dengan priority queue (PriorityHigh dequeue duluan)
    - Rate limiter: max 10 commands/detik per router (menggunakan token bucket atau sliding window)
    - WarmUp method: saat antrian > 10 perintah, buka koneksi paralel hingga max capacity
    - Get method menerima CommandPriority parameter untuk priority-based dequeue
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 4.8, 4.9, 4.10_

  - [x] 5.2 Buat `services/network-service/internal/pool/manager.go`
    - Implementasikan struct poolManager yang mengimplementasikan PoolManager interface
    - GetPool: return existing pool atau buat baru (thread-safe dengan sync.RWMutex)
    - ClosePool: tutup pool untuk router tertentu
    - CloseAll: tutup semua pool
    - _Requirements: 4.1_

  - [x] 5.3 Write property tests untuk connection pool
    - **Property 2: Pool capacity invariant** — pool tidak pernah melebihi 5 koneksi aktif secara bersamaan
    - **Property 3: Rate limiting enforcement** — rate tidak melebihi 10 commands/detik
    - **Property 14: Priority queue ordering** — High priority commands dequeue sebelum Medium dan Low saat pool penuh
    - **Validates: Requirements 4.1, 4.6, 4.8, 4.10**

- [x] 6. Checkpoint — Pastikan semua test pass
  - Pastikan semua test pass, ask the user if questions arise.

- [x] 7. sqlc query generation dan repository
  - [x] 7.1 Generate sqlc code
    - Jalankan `sqlc generate` di `services/network-service/`
    - Verifikasi generated code di `internal/repository/`
    - _Requirements: 1.1_

  - [x] 7.2 Buat `services/network-service/internal/repository/router_repo.go`
    - Implementasikan struct RouterRepo yang mengimplementasikan RouterRepository interface
    - Bungkus sqlc-generated Queries dengan mapping pgtype ↔ domain types (pattern dari notification-service)
    - Implementasikan semua method: Create, GetByID, Update, SoftDelete, List, CountByStatus, GetActiveRouters, NameExists, UpdateHealthCheck
    - Map pgx.ErrNoRows ke ErrRouterNotFound
    - _Requirements: 1.1, 5.1, 5.3, 5.5, 5.6, 6.2, 6.3_

- [x] 8. Metrics store (Redis sorted sets)
  - [x] 8.1 Buat `services/network-service/internal/metrics/store.go`
    - Implementasikan struct redisMetricsStore yang mengimplementasikan MetricsStore interface
    - Store: ZADD ke sorted set "router:{id}:metrics" dengan score=unix timestamp, member=JSON metrics. ZREMRANGEBYSCORE untuk enforce 7-day TTL
    - Query: ZRANGEBYSCORE dengan from/to timestamps, unmarshal JSON, return sorted ascending
    - GetLatest: ZREVRANGEBYSCORE limit 1
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [x] 8.2 Write property tests untuk metrics store
    - **Property 12: Metrics store round-trip with ordering** — store lalu query mengembalikan data dalam range yang benar, sorted ascending
    - **Validates: Requirements 9.3, 9.4**

- [x] 9. Event publisher
  - [x] 9.1 Buat `services/network-service/internal/usecase/event_publisher.go`
    - Implementasikan struct eventPublisher yang mengimplementasikan EventPublisher interface
    - Gunakan pkg/queue.EnqueueTask untuk publish events
    - PublishRouterOffline: buat TaskEnvelope dengan event_type "mikrotik.router_offline" dan RouterOfflinePayload
    - PublishRouterOnline: buat TaskEnvelope dengan event_type "mikrotik.router_online" dan RouterOnlinePayload
    - PublishUnexpectedReboot: buat TaskEnvelope dengan event_type "mikrotik.router_unexpected_reboot" dan RouterRebootPayload
    - Setiap event HARUS memiliki correlation_id (auto-generated oleh pkg/queue jika kosong)
    - Best-effort: log error jika publish gagal, jangan return error ke caller
    - _Requirements: 10.1, 10.2, 10.3, 10.4_

  - [x] 9.2 Write property tests untuk event publisher
    - **Property 13: Event payload completeness with correlation ID** — setiap event memiliki non-empty correlation_id, correct event_type, dan semua required payload fields
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4**

- [x] 10. Router usecase dan health checker
  - [x] 10.1 Buat `services/network-service/internal/usecase/router_usecase.go`
    - Implementasikan struct routerUsecase yang mengimplementasikan RouterUsecase interface
    - Dependencies: RouterRepository, CredentialEncryptor, PoolManager, MetricsStore, EventPublisher
    - Create: encrypt password → repo.Create → test connection (best-effort) → auto-detect info → update router → publish event jika online
    - GetByID: repo.GetByID → jika online, ambil live metrics via adapter
    - Update: validasi → encrypt password jika berubah → repo.Update → update pool jika host/port/credentials berubah
    - Delete: repo.SoftDelete → poolManager.ClosePool
    - List: repo.List dengan pagination
    - TestConnection: decrypt password → connect → GetSystemResource
    - Reboot: validasi confirmation_name == router.Name → execute "/system/reboot"
    - GetStatusSummary: repo.CountByStatus → map ke StatusSummary
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10, 7.1, 7.2_

  - [x] 10.2 Buat `services/network-service/internal/usecase/health_checker.go`
    - Implementasikan struct healthChecker yang mengimplementasikan HealthChecker interface
    - Dependencies: RouterRepository, PoolManager, MetricsStore, EventPublisher, CredentialEncryptor
    - Start: ambil semua active routers, jalankan goroutine ticker per router
    - Per-tick: skip jika status==maintenance → decrypt password → connect/ping → GetSystemResource
    - Success: reset failure_count, update last_checked_at, store metrics, detect reboot (uptime < previous), publish events
    - Failure: increment failure_count, jika >= 3 → set offline + publish router_offline event
    - AddRouter/RemoveRouter/UpdateInterval: manage goroutine lifecycle
    - Stop: cancel semua goroutines
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8_

  - [x] 10.3 Write property tests untuk health checker
    - **Property 5: Successful health check resets failure state** — setelah success, failure_count==0 dan last_checked_at updated
    - **Property 6: Failed health check increments failure count** — failure_count naik 1, saat reach 3 → status offline
    - **Property 7: Reboot detection via uptime comparison** — current uptime < previous uptime → publish reboot event
    - **Validates: Requirements 6.2, 6.3, 6.4, 6.5, 6.6**

- [x] 11. Handler layer
  - [x] 11.1 Buat `services/network-service/internal/handler/router_handler.go`
    - Implementasikan struct RouterHandler dengan dependency RouterUsecase
    - Method Create: parse body → validate → extract tenant_id → usecase.Create → return 201
    - Method GetByID: extract id param → usecase.GetByID → return 200
    - Method Update: parse body → validate → usecase.Update → return 200
    - Method Delete: extract id param → usecase.Delete → return 204
    - Method List: parse query params (page, page_size, status, search) → usecase.List → return 200 paginated
    - Method TestConnection: extract id param → usecase.TestConnection → return 200
    - Method Reboot: parse body → validate → usecase.Reboot → return 200
    - Error mapping: domain errors → HTTP status codes sesuai tabel di design
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10_

  - [x] 11.2 Buat `services/network-service/internal/handler/status_handler.go`
    - Implementasikan struct StatusHandler dengan dependency RouterUsecase
    - Method GetSummary: extract tenant_id → usecase.GetStatusSummary → return 200
    - _Requirements: 7.1, 7.2_

- [x] 12. Router wiring, config update, dan main.go DI
  - [x] 12.1 Update `services/network-service/internal/config/config.go`
    - Tambahkan field EncryptionKey string `mapstructure:"ENCRYPTION_KEY"`
    - Tambahkan validasi: EncryptionKey wajib dan harus 32 bytes
    - _Requirements: 8.3, 8.4_

  - [x] 12.2 Update `services/network-service/internal/handler/router.go` (route registration)
    - Tambahkan RouterHandler dan StatusHandler ke RouterConfig struct
    - Daftarkan routes: POST/GET /api/v1/mikrotik/routers, GET/PUT/DELETE /api/v1/mikrotik/routers/:id, POST /api/v1/mikrotik/routers/:id/test, POST /api/v1/mikrotik/routers/:id/reboot, GET /api/v1/mikrotik/status/summary
    - _Requirements: 5.1, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 7.1_

  - [x] 12.3 Update `services/network-service/cmd/main.go`
    - Tambahkan dependency injection: crypto → adapter factory → pool manager → repository → metrics store → event publisher → usecase → health checker → handlers
    - Tambahkan go-routeros ke go.mod
    - Tambahkan pgregory.net/rapid ke go.mod (dev dependency)
    - Start health checker di goroutine terpisah
    - Graceful shutdown: stop health checker → close pool manager → close connections
    - _Requirements: 3.3, 3.4, 4.1, 6.1, 8.3_

  - [x] 12.4 Update `services/network-service/go.mod`
    - Tambahkan dependency: `github.com/go-routeros/routeros/v3`
    - Tambahkan dev dependency: `pgregory.net/rapid`
    - Jalankan `go mod tidy`
    - _Requirements: 3.4_

- [x] 13. Handler unit tests
  - [x] 13.1 Write unit tests untuk RouterHandler
    - Test Create: valid request → 201, invalid body → 422, duplicate name → 409
    - Test GetByID: found → 200, not found → 404
    - Test Update: valid → 200, invalid transition → 422
    - Test Delete: success → 204, not found → 404
    - Test List: pagination, filter by status
    - Test TestConnection: online → 200 with system info, offline → 502
    - Test Reboot: valid confirmation → 200, mismatch → 400
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10_

  - [x] 13.2 Write unit tests untuk StatusHandler
    - Test GetSummary: return correct counts
    - _Requirements: 7.1, 7.2_

- [x] 14. Final checkpoint — Pastikan semua test pass
  - Pastikan semua test pass, ask the user if questions arise.
  - Verifikasi `go build ./...` berhasil di `services/network-service/`
  - Verifikasi `go vet ./...` tidak ada warning

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (14 properties dari design)
- Unit tests validate specific examples and edge cases
- Semua komentar dalam bahasa Indonesia, max 200 baris per file
- Pattern mengikuti notification-service: sqlc + manual mock + domain-driven
- NETWORK_MODE=mock sebagai default untuk development
