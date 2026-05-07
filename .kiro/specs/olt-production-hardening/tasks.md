# Tasks - OLT Production Hardening

## Overview

Checklist ini adalah rencana kerja setelah audit kode OLT aktif pada 2026-05-07. Kerjakan berurutan. Jangan langsung menambah brand lain sebelum ZTE C320 stabil dan debug path jelas.

## Phase 0 - Audit and Spec

- [x] 0.1 Audit folder riset `snmp-zte` dan status integrasinya ke aplikasi
  - Output: `docs/AUDIT-SNMP-ZTE-OLT-INTEGRATION-2026-05-07.md`
  - _Requirements: 5.7_

- [x] 0.2 Audit keseluruhan kode OLT aktif
  - Output: `docs/AUDIT-OLT-KODE-DAN-RENCANA-2026-05-07.md`
  - _Requirements: all_

- [x] 0.3 Buat spec production hardening
  - Output: `.kiro/specs/olt-production-hardening`
  - _Requirements: all_

## Phase 1 - Runtime Safety Guard

- [x] 1.1 Tambahkan config OLT guard di `services/network-service/internal/config/config.go`
  - Fields: `OLT_HEALTH_CHECK_ENABLED`, `OLT_SYNC_ENABLED`, `OLT_TRAP_ENABLED`, `OLT_PROVISIONING_WRITE_ENABLED`
  - Include defaults and env binding
  - _Requirements: 1.1, 1.7_

- [x] 1.2 Update `services/network-service/cmd/main.go` agar health checker, trap receiver, dan sync engine hanya start jika enabled
  - Log enabled/disabled state
  - Do not log credentials
  - _Requirements: 1.2, 1.3, 1.4, 1.7_

- [x] 1.3 Wire `OLT_SYNC_INTERVAL` into sync engine constructor
  - Remove hidden default from main path
  - _Requirements: 1.6_

- [x] 1.4 Tambahkan provisioning write guard ke provisioning usecase
  - Reject write operations when disabled
  - Cover provision, decommission, reboot, bulk execute, auto-provisioning
  - _Requirements: 1.5, 8.6_

- [x] 1.5 Tambahkan tests untuk config guard dan startup behavior
  - Covered: config defaults/env override, provisioning write guard, and compile-verified guarded startup wiring.
  - _Requirements: 10.7_

## Phase 2 - Brand and Model Profile Registry

- [x] 2.1 Tambahkan brand/model profile types
  - Candidate file: `services/network-service/internal/adapter/olt_brand_profile.go`
  - _Requirements: 2.1, 2.2_

- [x] 2.2 Tambahkan capability constants dan helper
  - Candidate file: `services/network-service/internal/adapter/olt_capabilities.go`
  - _Requirements: 2.3_

- [x] 2.3 Tambahkan profile ZTE C320
  - Include supported capabilities and initial addressing profile
  - _Requirements: 2.2, 2.3, 4.2_

- [x] 2.4 Update adapter factory agar memakai profile/capability registry
  - Unsupported brand/model returns clear typed error
  - _Requirements: 2.4, 2.5_

- [x] 2.5 Update usecase agar tetap brand-agnostic
  - No normal operation `if brand == zte` in usecase
  - _Requirements: 2.6_

- [x] 2.6 Tambahkan unit tests registry/factory/capability
  - _Requirements: 10.1_

## Phase 3 - Generic Probe and Auto-Detect

- [x] 3.1 Buat generic SNMP probe service/helper
  - Reads sysDescr, sysName, sysUpTime using standard OIDs
  - _Requirements: 3.1, 3.2_

- [x] 3.2 Tambahkan brand/model detection from sysDescr
  - Reuse `domain.DetectBrand`
  - Add ZTE model matcher
  - _Requirements: 3.3, 3.4_

- [x] 3.3 Fix create OLT auto-detect
  - Replace `CreateAdapter("", ...)` with generic probe then profile lookup
  - _Requirements: 3.5, 3.6, 3.7_

- [x] 3.4 Update API response to expose probe warning/result when auto-detect fails
  - _Requirements: 3.6, 9.6_

- [x] 3.5 Add tests for create OLT auto-detect success/failure
  - _Requirements: 3.1, 3.7_

## Phase 4 - ZTE C320 Index and Communication

- [x] 4.1 Buat `olt_zte_index.go`
  - Implement ZTE address/index mapper
  - Do not hard-code board 0
  - _Requirements: 4.1, 4.2, 4.4_

- [x] 4.2 Tambahkan table tests untuk ZTE index dari riset `snmp-zte`
  - Include board/PON examples
  - _Requirements: 4.3, 10.2_

- [x] 4.3 Update ZTE monitoring code agar memakai mapper
  - PON ports, ONT list, signal, SFP, traffic
  - _Requirements: 4.4, 4.5, 4.6_

- [x] 4.4 Pisahkan ZTE OID builders dari OID constants
  - Candidate file: `olt_zte_oids.go` plus builders
  - _Requirements: 5.1_

- [x] 4.5 Buat ZTE CLI command builder
  - Candidate file: `olt_zte_commands.go`
  - Cover add ONT, remove ONT, add/remove service-port, reboot, show unconfigured
  - _Requirements: 5.2, 10.4_

- [x] 4.6 Buat ZTE parser
  - Candidate file: `olt_zte_parser.go`
  - Cover sysDescr, unregistered ONT, command success/failure, interface parse
  - _Requirements: 5.3, 10.3_

- [x] 4.7 Buat atau perbaiki ZTE-aware session layer
  - Prompt matching, pagination, echo cleanup, config/interface mode, timeout classification
  - _Requirements: 5.4, 5.5_

- [x] 4.8 Add sanitized transport metadata for ZTE adapter operations
  - Implemented: ZTE provisioning results now include brand/transport/operation metadata and audit storage sanitizes commands/responses.
  - _Requirements: 5.6, 8.7_

## Phase 5 - Monitoring Data Path

- [x] 5.1 Update traffic handler to query `TrafficStore`
  - Replace placeholder empty response
  - _Requirements: 6.1, 6.7_

- [x] 5.2 Add signal latest/history read path
  - Either endpoint or ONT detail enrichment
  - _Requirements: 6.2_

- [x] 5.3 Add repository query to list DB ONTs by OLT
  - Needed by sync reconciliation
  - _Requirements: 6.3, 6.4_

- [x] 5.4 Update sync engine reconciliation
  - Compare OLT ONTs and DB ONTs by serial number
  - Detect unregistered and port migration
  - _Requirements: 6.3, 6.4, 6.5_

- [x] 5.5 Add optional immediate sync behavior
  - Config-driven
  - _Requirements: 6.6_

- [x] 5.6 Add handler/usecase tests for no data, disabled sync, store unavailable, and OLT failure states
  - Covered: traffic/signal empty store, OLT-not-found failure, write-disabled dry-run, and handler preview path.
  - _Requirements: 6.7, 10.6_

## Phase 6 - Alarm Trap Reliability

- [x] 6.1 Add OLT lookup by host/source IP
  - Implemented: repository-level `GetByHost` support plus fallback active-OLT scan.
  - Query/repository support
  - _Requirements: 7.1_

- [x] 6.2 Update trap handler to set tenant_id and olt_id
  - Do not create DB alarm without mapped OLT
  - _Requirements: 7.1, 7.2, 7.3_

- [x] 6.3 Add alarm dedupe for repeated active alarms
  - _Requirements: 7.4_

- [x] 6.4 Add clear event handling
  - _Requirements: 7.5_

- [x] 6.5 Update alarm API/UI DTO if needed
  - Include source, severity, type, PON, ONT, created_at, cleared_at
  - _Requirements: 7.6_

- [x] 6.6 Add tests for mapped trap, unknown trap source, dedupe, and clear event
  - _Requirements: 10.6_

## Phase 7 - Provisioning Safety

- [x] 7.1 Add assigned ONT index result type
  - Candidate: `AddONTResult` or capability interface
  - _Requirements: 8.1, 8.2_

- [x] 7.2 Update ZTE AddONT flow to resolve assigned ONT index
  - Use CLI/SNMP confirmation from OLT
  - _Requirements: 8.1, 8.2_

- [x] 7.3 Persist resolved ONT index before adding service-port
  - _Requirements: 8.3_

- [x] 7.4 Add compensation/manual recovery state for partial failure
  - Partial: service-port failure now triggers best-effort `RemoveONT`; explicit manual recovery state remains.
  - _Requirements: 8.4_

- [x] 7.5 Add dry-run command preview
  - _Requirements: 8.5_

- [x] 7.6 Enrich provisioning audit log
  - Brand, model, transport, operation, sanitized commands/responses, correlation id
  - _Requirements: 8.7_

- [x] 7.7 Fix decommission event customer_id preservation
  - _Requirements: 8.8_

- [x] 7.8 Add usecase tests for index resolution, service-port using resolved index, partial failure, write guard, and event payload
  - _Requirements: 10.5_

## Phase 8 - UI Debug Workspace

- [x] 8.1 Refactor OLT detail UI into debug workspace
  - Sections/tabs: overview, connection, PON/ONT, signal/SFP/traffic, alarms, provisioning, VLAN/profile, audit/debug
  - _Requirements: 9.1_

- [x] 8.2 Add manual test SNMP and test CLI actions
  - _Requirements: 9.2_

- [x] 8.3 Show brand/model capabilities and unsupported states
  - _Requirements: 9.3, 9.4_

- [x] 8.4 Align UI field names with backend DTO
  - Avoid stale fallbacks hiding real values
  - _Requirements: 9.5_

- [x] 8.5 Show last check, last sync, last probe, and last error state
  - _Requirements: 9.6_

- [x] 8.6 Verify UI build and smoke OLT pages in mock mode
  - _Requirements: 9.1, 9.6_

## Phase 9 - Brand Expansion Gate

- [x] 9.1 Mark non-ZTE brands as unsupported/partial in capability registry until real adapters exist
  - _Requirements: 2.5_

- [x] 9.2 Define checklist for adding a new brand
  - Profile, mapper, OID builders, parsers, command builders, adapter tests, UI capability
  - _Requirements: 2.1, 2.3, 10.1_

- [x] 9.3 Add next brand only after ZTE C320 passes production-readiness tests
  - Gate enforced by decision: no new real brand adapter was added in this implementation pass.
  - _Requirements: 10.1, 10.8_
