# Tasks — OLT Management Layer

## Overview

Implementasi OLT Management Layer di `services/network-service/`. Layer ini menangani management plane OLT: registrasi perangkat dengan auto-detect, adapter pattern multi-brand (ZTE, Huawei, FiberHome, VSOL, HSGQ), koneksi SNMP dan CLI, health check, ODP/splitter management, PON port monitoring, ONT status monitoring, alarm management, SFP monitoring, periodic sync, traffic monitoring, capacity planning, HTTP API, dan event publishing.

## Tasks

- [x] 1. Domain Entities, Constants, dan Errors
  - [x] 1.1 Buat file `internal/domain/olt.go` — OLT entity struct (termasuk SNMPPort field), OLTStatus constants (online, offline, maintenance), ValidOLTTransitions map, CanTransitionOLT helper, OLTBrand constants (zte, huawei, fiberhome, vsol, hsgq), DetectBrand helper, SNMPVersion constants (v2c, v3), CLIProtocol constants (ssh, telnet), OLTHealthCheckUpdate struct
    - Maksimal 200 baris per file
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 6.6_

  - [x] 1.2 Buat file `internal/domain/olt_signal.go` — SignalLevel constants (normal, warning, weak, critical), ClassifySignal helper function, threshold constants (-25, -27, -30 dBm)
    - Maksimal 200 baris per file
    - _Requirements: 2.8, 10.2_

  - [x] 1.3 Buat file `internal/domain/odp.go` — ODP entity struct, SplitterCapacity helper function, splitter type constants ("1:4", "1:8", "1:16", "1:32")
    - Maksimal 200 baris per file
    - _Requirements: 8.1, 8.4, 8.5_

  - [x] 1.4 Buat file `internal/domain/olt_alarm.go` — OLTAlarm struct, OLTAlarmRecord struct, alarm_type constants (ont_los, ont_dying_gasp, pon_port_down, power_failure, high_temperature, ont_signal_degraded), severity constants (critical, major, minor, warning, clear), alarm source constants (trap, polling)
    - Maksimal 200 baris per file
    - _Requirements: 11.3, 11.4, 11.5_

  - [x] 1.5 Update file `internal/domain/errors.go` — Tambahkan OLT-specific domain errors: ErrOLTNotFound, ErrOLTNameExists, ErrOLTInvalidStatusTransition, ErrOLTOffline, ErrOLTDeleted, ErrSNMPConnectionFailed, ErrSNMPTimeout, ErrSNMPAuthFailed, ErrCLIConnectionFailed, ErrCLITimeout, ErrCLIAuthFailed, ErrUnsupportedBrand, ErrBrandDetectionFailed, ErrODPNotFound, ErrODPNameExists, ErrODPFull, ErrInvalidSplitterType, ErrAlarmNotFound, ErrTrapReceiverFailed
    - _Requirements: 2.4, 4.6, 5.6, 6.3_

- [x] 2. Domain DTOs dan Adapter Response Types
  - [x] 2.1 Buat file `internal/domain/olt_dto.go` — Request DTOs (CreateOLTRequest, UpdateOLTRequest, OLTListParams), Response DTOs (OLTResponse, OLTDetailResponse, OLTListResult, OLTStatusSummary, CLITestResult), OLTCapacity dan PortCapacity structs
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 15.1, 15.4, 16.1, 16.2, 16.3, 16.4, 19.1, 20.1, 20.2, 20.3_

  - [x] 2.2 Buat file `internal/domain/odp_dto.go` — Request DTOs (CreateODPRequest, UpdateODPRequest, ODPListParams), Response DTOs (ODPResponse, ODPDetailResponse, ODPListResult)
    - Maksimal 200 baris per file
    - _Requirements: 16.15, 16.16, 16.17, 16.18, 16.19_

  - [x] 2.3 Buat file `internal/domain/olt_adapter_types.go` — Adapter response types: OLTSystemInfo, PONPortStatus, ONTStatus, ONTSignalInfo, SFPInfo, PONTrafficStats, ONTSignalPoint, PONTrafficPoint, SNMPConfig, CLIConfig, SNMPResult, SNMPValueType
    - Maksimal 200 baris per file
    - _Requirements: 3.1, 4.1, 4.2, 9.1, 9.4, 10.1, 12.1, 14.1_

  - [x] 2.4 Buat file `internal/domain/olt_event.go` — Event type constants (olt.device_offline, olt.device_online, olt.alarm), Event payloads (OLTDeviceOfflinePayload, OLTDeviceOnlinePayload, OLTAlarmPayload), AlarmListParams, AlarmListResult
    - Maksimal 200 baris per file
    - _Requirements: 11.6, 17.1, 17.2, 17.3, 17.4_

- [x] 3. Repository Interfaces
  - [x] 3.1 Update file `internal/domain/repository.go` — Tambahkan OLTRepository, ODPRepository, AlarmRepository, SignalStore, TrafficStore, OLTEventPublisher, OLTAdapter, OLTAdapterFactory, SNMPConnector, CLIConnector, OLTManager, ODPManager, OLTHealthChecker, AlarmManager, SyncEngine interfaces
    - _Requirements: 1.1, 3.1, 3.3, 4.1, 5.1, 7.1, 8.1, 11.1, 13.1_

- [x] 4. Property Tests untuk Domain Logic
  - [x] 4.1 Buat file `internal/domain/olt_status_test.go` — Property test untuk OLT status transition validation
    - **Property 1: OLT Status Transition Validation**
    - **Validates: Requirements 2.3, 2.4**

  - [x] 4.2 Buat file `internal/domain/signal_level_test.go` — Property test untuk signal level classification (exhaustive dan mutually exclusive)
    - **Property 2: Signal Level Classification**
    - **Validates: Requirements 2.8, 10.2**

  - [x] 4.3 Buat file `internal/domain/brand_detect_test.go` — Property test untuk brand detection dari sysDescr string
    - **Property 3: Brand Detection from sysDescr**
    - **Validates: Requirements 6.2, 6.6**

  - [x] 4.4 Buat file `internal/crypto/encryptor_olt_test.go` — Property test untuk credential encryption round-trip (reuse existing encryptor)
    - **Property 4: Credential Encryption Round-Trip**
    - **Validates: Requirements 18.1, 18.2, 18.5**

  - [x] 4.5 Buat file `internal/domain/odp_test.go` — Property test untuk splitter capacity mapping
    - **Property 5: Splitter Capacity Mapping**
    - **Validates: Requirements 8.5**

- [x] 5. Checkpoint — Domain Layer
  - Ensure all tests pass, ask the user if questions arise.


- [x] 6. Database Migrations
  - [x] 6.1 Buat SQL migration file `migrations/000008_create_olts.up.sql` — CREATE TABLE olts dengan semua kolom, UNIQUE constraint (tenant_id, name WHERE deleted_at IS NULL), indexes (tenant_id, status), RLS policy
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 6.2 Buat SQL migration file `migrations/000009_create_odps.up.sql` — CREATE TABLE odps dengan semua kolom, UNIQUE constraint (tenant_id, name WHERE deleted_at IS NULL), index (olt_id, pon_port_index), RLS policy
    - _Requirements: 8.1, 8.2, 8.3_

  - [x] 6.3 Buat SQL migration file `migrations/000010_create_olt_alarms.up.sql` — CREATE TABLE olt_alarms dengan semua kolom, indexes (olt_id+status, tenant_id+created_at, created_at untuk purge), RLS policy
    - _Requirements: 11.3_

- [x] 7. sqlc Queries dan Repository Wrappers
  - [x] 7.1 Buat sqlc queries file `queries/olts.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, List, CountByStatus, GetActiveOLTs, GetOnlineOLTs, NameExists, UpdateHealthCheck, UpdateONTCounts)
    - _Requirements: 1.1, 7.2, 7.4, 7.5, 13.6, 19.1, 20.1_

  - [x] 7.2 Buat sqlc queries file `queries/odps.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, List, NameExists, GetByOLTAndPort)
    - _Requirements: 8.1, 16.15, 16.16, 16.17, 16.18, 16.19_

  - [x] 7.3 Buat sqlc queries file `queries/olt_alarms.sql` — CRUD queries (Create, List, CountActive, CountActiveByTenant, ClearAlarm, PurgeOlderThan)
    - _Requirements: 11.3, 11.7, 16.11_

  - [x] 7.4 Jalankan `sqlc generate` dan buat repository wrapper `internal/repository/olt_repo.go`
    - _Requirements: 1.1_

  - [x] 7.5 Buat repository wrapper `internal/repository/odp_repo.go`
    - _Requirements: 8.1_

  - [x] 7.6 Buat repository wrapper `internal/repository/alarm_repo.go`
    - _Requirements: 11.3_

- [x] 8. SNMP Connector Implementation
  - [x] 8.1 Tambahkan `github.com/gosnmp/gosnmp` ke go.mod
    - _Requirements: 4.3_

  - [x] 8.2 Buat file `internal/adapter/snmp_connector.go` — SNMPConnector implementation: Get, Walk, GetBulk, Ping. Support v2c (community) dan v3 (username/auth/priv). Timeout 5s connect, 10s request. Port 161 default.
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_

  - [x] 8.3 Buat file `internal/adapter/snmp_connector_test.go` — Unit tests: v2c/v3 config building, timeout handling, error classification
    - _Requirements: 4.4, 4.6_

- [x] 9. CLI Connector Implementation (SSH/Telnet)
  - [x] 9.1 Buat file `internal/adapter/cli_connector.go` — CLIConnector implementation: Execute, ExecuteMultiple, TestConnection. SSH via golang.org/x/crypto/ssh, Telnet support. Connect-on-demand (no pool). Timeout 10s connect, 30s command. Enable password support.
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_

  - [x] 9.2 Buat file `internal/adapter/cli_connector_test.go` — Unit tests: SSH/Telnet config, connect-on-demand, enable password, error classification
    - _Requirements: 5.1, 5.6_

- [x] 10. OLT Adapter Interface + Mock Adapter + Factory
  - [x] 10.1 Buat file `internal/adapter/olt_adapter_factory.go` — OLTAdapterFactory implementation: CreateAdapter berdasarkan brand, return MockOLTAdapter jika NETWORK_MODE=mock
    - Maksimal 200 baris per file
    - _Requirements: 3.3, 3.4, 3.5_

  - [x] 10.2 Buat file `internal/adapter/olt_mock_adapter.go` — MockOLTAdapter implementation: semua method OLTAdapter interface return data simulasi realistis (brand "zte", model "C320", firmware "V2.1.0", pon_ports 8, total_ont 245)
    - Maksimal 200 baris per file
    - _Requirements: 3.4, 3.6_

  - [x] 10.3 Buat file `internal/adapter/olt_factory_test.go` — Property test untuk adapter factory brand mapping (exhaustive mapping)
    - **Property 10: Adapter Factory Brand Mapping**
    - **Validates: Requirements 3.3**

- [x] 11. Checkpoint — Infrastructure Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 12. ZTE Adapter (First Real Adapter)
  - [x] 12.1 Buat file `internal/adapter/olt_zte_adapter.go` — ZTEAdapter implementation: GetSystemInfo (sysDescr, sysUpTime OIDs), Ping (sysUpTime GET), GetAllPONPorts, GetONTList, GetONTSignal, GetAlarms, GetSFPInfo, GetTrafficStats. Referensi OID dari diskusi snmp-zte repo.
    - Maksimal 200 baris per file, split jika perlu ke `olt_zte_pon.go` dan `olt_zte_ont.go`
    - _Requirements: 3.1, 3.2, 9.1, 9.4, 10.1, 12.1, 14.1_

  - [x] 12.2 Buat file `internal/adapter/olt_zte_oids.go` — ZTE-specific SNMP OID constants: sysDescr, sysUpTime, ifAdminStatus, ifOperStatus, PON port OIDs, ONT status OIDs, signal OIDs, SFP OIDs, traffic counter OIDs
    - Maksimal 200 baris per file
    - _Requirements: 3.2_

  - [x] 12.3 Buat file `internal/adapter/olt_zte_test.go` — Unit tests: OID mapping, response parsing, error handling
    - _Requirements: 3.2_

- [x] 13. Stub Adapters (Huawei, FiberHome, VSOL, HSGQ)
  - [x] 13.1 Buat file `internal/adapter/olt_huawei_adapter.go` — HuaweiAdapter stub: implement OLTAdapter interface, semua method return ErrUnsupportedBrand atau mock data minimal. Placeholder untuk implementasi masa depan.
    - Maksimal 200 baris per file
    - _Requirements: 3.2_

  - [x] 13.2 Buat file `internal/adapter/olt_fiberhome_adapter.go` — FiberHomeAdapter stub: implement OLTAdapter interface, semua method return ErrUnsupportedBrand atau mock data minimal.
    - Maksimal 200 baris per file
    - _Requirements: 3.2_

  - [x] 13.3 Buat file `internal/adapter/olt_vsol_adapter.go` — VSOLAdapter stub: implement OLTAdapter interface, semua method return ErrUnsupportedBrand atau mock data minimal.
    - Maksimal 200 baris per file
    - _Requirements: 3.2_

  - [x] 13.4 Buat file `internal/adapter/olt_hsgq_adapter.go` — HSGQAdapter stub: implement OLTAdapter interface, semua method return ErrUnsupportedBrand atau mock data minimal.
    - Maksimal 200 baris per file
    - _Requirements: 3.2_

- [x] 14. OLT Manager Usecase (CRUD, Auto-Detect, Test Connection)
  - [x] 14.1 Buat file `internal/usecase/olt_manager.go` — Struct oltManager dengan dependencies (OLTRepo, ODPRepo, AlarmRepo, OLTAdapterFactory, SNMPConnector, CLIConnector, CredentialEncryptor, OLTEventPublisher, SignalStore, TrafficStore), constructor NewOLTManager
    - Maksimal 200 baris per file
    - _Requirements: 6.1, 7.1_

  - [x] 14.2 Implementasi `Create` — Validate input, encrypt credentials, save to DB, test SNMP (best-effort), auto-detect brand/model/firmware via sysDescr, update OLT record, return OLTResponse
    - _Requirements: 6.1, 6.2, 6.3, 18.1_

  - [x] 14.3 Implementasi `GetByID` — Retrieve OLT by ID, include active alarm count, mask credentials, return OLTDetailResponse
    - _Requirements: 16.3, 18.4_

  - [x] 14.4 Implementasi `Update` — Validate allowed fields (name, host, credentials, interval, notes, status), encrypt new credentials if provided, update DB
    - _Requirements: 20.2, 20.3_

  - [x] 14.5 Implementasi `Delete` — Soft-delete OLT, stop health check monitoring
    - _Requirements: 20.4_

  - [x] 14.6 Implementasi `List` — Query DB with pagination, filter by status/brand/search, return OLTListResult
    - _Requirements: 16.2, 20.1_

  - [x] 14.7 Implementasi `TestSNMP` — Decrypt credentials, create adapter, call GetSystemInfo, return OLTSystemInfo
    - _Requirements: 6.4_

  - [x] 14.8 Implementasi `TestCLI` — Decrypt credentials, call CLIConnector.TestConnection, return CLITestResult
    - _Requirements: 6.5_

  - [x] 14.9 Implementasi `GetStatusSummary` — CountByStatus, count active alarms, return OLTStatusSummary
    - _Requirements: 16.14, 19.1, 19.2_

  - [x] 14.10 Implementasi `GetPONPorts`, `GetONTList`, `GetSFPStatus` — Delegate to adapter via factory, decrypt credentials, return adapter response
    - _Requirements: 9.1, 9.2, 9.3, 12.1, 16.8, 16.9, 16.12_

  - [x] 14.11 Buat file `internal/usecase/olt_manager_test.go` — Unit tests: Create, Update, Delete, List, TestSNMP, TestCLI, GetStatusSummary
    - _Requirements: 6.1, 6.4, 6.5, 20.1_


- [x] 15. ODP Manager Usecase (CRUD)
  - [x] 15.1 Buat file `internal/usecase/odp_manager.go` — Struct odpManager dengan dependencies (ODPRepo, OLTRepo), constructor NewODPManager. Implementasi Create (auto-set capacity dari splitter_type), GetByID, Update, Delete (soft-delete), List (filter by olt_id, pon_port)
    - Maksimal 200 baris per file
    - _Requirements: 8.4, 8.5, 8.6, 16.15, 16.16, 16.17, 16.18, 16.19_

  - [x] 15.2 Buat file `internal/usecase/odp_manager_test.go` — Unit tests: Create with auto-capacity, Update, Delete, List, ODP full warning
    - _Requirements: 8.5, 8.6_

- [x] 16. OLT Health Checker (Background Goroutine)
  - [x] 16.1 Buat file `internal/usecase/olt_health_checker.go` — OLTHealthChecker implementation: Start (load active OLTs, start ticker per OLT), Stop, AddOLT, RemoveOLT, UpdateInterval. Health check logic: SNMP Ping via adapter, success → reset failure_count + update last_checked_at, failure → increment failure_count, 3x failure → status offline + publish olt.device_offline, recovery → status online + publish olt.device_online. Skip OLT dengan status maintenance.
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 20.5_

  - [x] 16.2 Buat file `internal/usecase/olt_health_checker_test.go` — Unit tests: success/failure handling, threshold 3x, maintenance skip, recovery detection
    - _Requirements: 7.2, 7.3, 7.4, 7.5, 7.6_

- [x] 17. Alarm Manager (Trap Receiver + Polling)
  - [x] 17.1 Buat file `internal/usecase/alarm_manager.go` — AlarmManager implementation: StartTrapReceiver (port 162, goroutine listener), StopTrapReceiver, PollAlarms (via adapter GetAlarms), GetAlarms (query DB with filter), PurgeOldAlarms (90 hari). Parse trap PDU per brand, save alarm to DB, publish olt.alarm event.
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7_

  - [x] 17.2 Buat file `internal/usecase/alarm_manager_test.go` — Unit tests: trap parsing, polling, alarm save, purge, event publish
    - _Requirements: 11.1, 11.2, 11.7_

- [x] 18. Checkpoint — Business Logic Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 19. Sync Engine (Periodic 30-min Sync)
  - [x] 19.1 Buat file `internal/usecase/sync_engine.go` — SyncEngine implementation: Start (30 min ticker), Stop, SyncOLT (manual trigger). Per OLT: GetAllPONPorts → per port GetONTList + GetTrafficStats + per ONT GetONTSignal. Compare OLT data vs DB: classify unmanaged/missing/updated/synced. Update total_ont_count. Store signal dan traffic data ke Redis.
    - Maksimal 200 baris per file, split jika perlu
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6_

  - [x] 19.2 Buat file `internal/usecase/sync_engine_test.go` — Property test untuk sync comparison correctness + unit tests
    - **Property 7: Sync Comparison Correctness**
    - **Validates: Requirements 13.2, 13.3, 13.4, 13.5, 13.6**

- [x] 20. Signal Store + Traffic Store (Redis)
  - [x] 20.1 Buat file `internal/metrics/signal_store.go` — SignalStore implementation: Redis sorted sets (key: olt:signal:{olt_id}:{port}:{ont}), Store data point dengan unix timestamp as score, Query by time range, GetLatest, 30-day TTL
    - Maksimal 200 baris per file
    - _Requirements: 10.3, 10.4_

  - [x] 20.2 Buat file `internal/metrics/traffic_store.go` — TrafficStore implementation: Redis sorted sets (key: olt:traffic:{olt_id}:{port}), Store data point dengan unix timestamp as score, Query by time range, GetLatest, 7-day TTL
    - Maksimal 200 baris per file
    - _Requirements: 14.2, 14.3_

  - [x] 20.3 Buat file `internal/metrics/signal_store_test.go` — Unit tests: Store, Query, GetLatest, TTL verification (menggunakan miniredis)
    - _Requirements: 10.3_

  - [x] 20.4 Buat file `internal/metrics/traffic_store_test.go` — Unit tests: Store, Query, GetLatest, TTL verification (menggunakan miniredis)
    - _Requirements: 14.2_

- [x] 21. OLT Event Publisher
  - [x] 21.1 Buat file `internal/usecase/olt_event_publisher.go` — OLTEventPublisher implementation: PublishDeviceOffline (event type "olt.device_offline"), PublishDeviceOnline (event type "olt.device_online"), PublishAlarm (event type "olt.alarm"). Best-effort publish with error logging, correlation_id generation via UUID.
    - Maksimal 200 baris per file
    - _Requirements: 17.1, 17.2, 17.3, 17.4_

  - [x] 21.2 Buat file `internal/usecase/olt_event_publisher_test.go` — Property test untuk event payload completeness + unit tests
    - **Property 8: Event Payload Completeness**
    - **Validates: Requirements 17.1, 17.2, 17.3, 17.4**

- [x] 22. Capacity Planning Logic
  - [x] 22.1 Buat file `internal/usecase/olt_capacity.go` — Implementasi GetCapacity pada oltManager: hitung total_pon_ports, active_pon_ports, total_ont_slots (ports × 64), used_ont_slots, available_ont_slots, utilization_percent, growth_rate_per_month (rata-rata 3 bulan terakhir), estimated_months_remaining. Per-port breakdown dengan warning jika > 90%.
    - Maksimal 200 baris per file
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

  - [x] 22.2 Buat file `internal/usecase/olt_capacity_test.go` — Property test untuk capacity calculation correctness
    - **Property 6: Capacity Calculation Correctness**
    - **Validates: Requirements 15.1, 15.2, 15.3, 15.4, 15.5**

  - [x] 22.3 Buat file `internal/usecase/olt_summary_test.go` — Property test untuk OLT status summary invariant
    - **Property 9: OLT Status Summary Invariant**
    - **Validates: Requirements 19.1**

- [x] 23. Checkpoint — Usecase Layer Complete
  - Ensure all tests pass, ask the user if questions arise.

- [x] 24. HTTP Handlers (OLT + ODP)
  - [x] 24.1 Buat file `internal/handler/olt_handler.go` — OLTHandler struct dengan methods: CreateOLT (POST /devices), ListOLTs (GET /devices), GetOLT (GET /devices/:id), UpdateOLT (PUT /devices/:id), DeleteOLT (DELETE /devices/:id), TestSNMP (POST /devices/:id/test-snmp), TestCLI (POST /devices/:id/test-cli), GetPONPorts (GET /devices/:id/pon-ports), GetONTList (GET /devices/:id/pon-ports/:port/onts), GetTraffic (GET /devices/:id/pon-ports/:port/traffic), GetAlarms (GET /devices/:id/alarms), GetSFP (GET /devices/:id/sfp), GetCapacity (GET /devices/:id/capacity), GetSummary (GET /summary)
    - Maksimal 200 baris per file, split jika perlu ke `olt_handler_monitoring.go`
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.6, 16.7, 16.8, 16.9, 16.10, 16.11, 16.12, 16.13, 16.14, 16.20_

  - [x] 24.2 Buat file `internal/handler/odp_handler.go` — ODPHandler struct dengan methods: CreateODP (POST /odp), ListODPs (GET /odp), GetODP (GET /odp/:id), UpdateODP (PUT /odp/:id), DeleteODP (DELETE /odp/:id)
    - Maksimal 200 baris per file
    - _Requirements: 16.15, 16.16, 16.17, 16.18, 16.19_

  - [x] 24.3 Buat file `internal/handler/olt_handler_test.go` — Unit tests: request validation, response format, error mapping (400/404/409/422/502/504)
    - _Requirements: 16.1, 16.20_

- [x] 25. Route Registration + Config + Wiring
  - [x] 25.1 Update `internal/config/config.go` — Tambahkan OLT config fields: OLTHealthCheckInterval (default 300), OLTSyncInterval (default 1800), SNMPTrapPort (default 162), MaxONTPerPort (default 64)
    - _Requirements: 7.1, 13.1, 11.1, 15.4_

  - [x] 25.2 Update `internal/handler/router.go` — Tambahkan OLTHandler dan ODPHandler ke RouterConfig struct, register OLT route group (/api/v1/olt/devices, /api/v1/olt/odp, /api/v1/olt/summary)
    - _Requirements: 16.1, 16.14, 16.15_

  - [x] 25.3 Update `cmd/main.go` — Wire OLT dependencies: OLTRepo, ODPRepo, AlarmRepo, SNMPConnector, CLIConnector, OLTAdapterFactory, SignalStore, TrafficStore, OLTEventPublisher, OLTManager, ODPManager, OLTHealthChecker, AlarmManager, SyncEngine, OLTHandler, ODPHandler. Start OLTHealthChecker, AlarmManager (trap receiver), SyncEngine goroutines. Stop on shutdown.
    - _Requirements: 7.1, 11.1, 13.1_

- [x] 26. Integration Tests
  - [x] 26.1 Integration test end-to-end — Test full flow: create OLT → test SNMP → auto-detect brand → health check cycle (online → offline → recovery). Test ODP CRUD. Test alarm flow (trap + polling). Test sync engine cycle. Test cross-tenant isolation (RLS). Test capacity planning calculation.
    - _Requirements: 1.2, 6.1, 7.4, 7.5, 8.1, 11.1, 13.1, 16.20_

- [x] 27. Final Checkpoint
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- Maksimal 200 baris per file Go — split ke file terpisah jika melebihi
- Gunakan `pgregory.net/rapid` untuk property-based testing (sudah ada di go.mod)
- Gunakan `github.com/gosnmp/gosnmp` untuk SNMP (perlu ditambahkan ke go.mod)
- `golang.org/x/crypto/ssh` sudah tersedia di go.mod
- Semua komentar dalam bahasa Indonesia
- Stub adapters (Huawei, FiberHome, VSOL, HSGQ) return ErrUnsupportedBrand atau mock data — implementasi penuh di masa depan
- ZTE adapter referensi OID dari diskusi snmp-zte repo
