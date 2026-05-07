# Tasks - MikroTik Module Restructure

## Overview

Checklist ini dibuat setelah audit struktur kode MikroTik pada 2026-05-07. Gunakan checklist ini saat implementasi agar perpindahan folder bisa dilacak dan tidak ada bagian yang tertinggal.

## Phase 0 - Audit and Spec

- [x] 0.1 Audit struktur backend `network-service`
  - Output: `docs/AUDIT-MIKROTIK-MODULE-RESTRUCTURE-2026-05-07.md`
  - _Requirements: all_

- [x] 0.2 Buat spec requirements, design, dan tasks
  - Output: `.kiro/specs/mikrotik-module-restructure`
  - _Requirements: all_

## Phase 1 - MikroTik Adapter Package

- [x] 1.1 Buat folder `services/network-service/internal/modules/mikrotik/adapter`
  - _Requirements: 1.1, 1.2_

- [x] 1.2 Pindahkan RouterOS adapter live/mock dan factory
  - Files: `adapter.go`, `factory.go`, `live_adapter.go`, `mock_adapter.go`
  - _Requirements: 3.1, 3.3_

- [x] 1.3 Pindahkan RouterOS command builder v6/v7
  - Files: `command_builder_factory.go`, `command_builder_v6.go`, `command_builder_v7.go`
  - _Requirements: 3.2_

- [x] 1.4 Pindahkan tests adapter terkait
  - Files: `command_builder_test.go`, `live_adapter_test.go`
  - _Requirements: 6.1_

- [x] 1.5 Update import path di main/usecase/tests
  - _Requirements: 2.3, 6.2_

## Phase 2 - MikroTik Usecase Package

- [x] 2.1 Buat folder `services/network-service/internal/modules/mikrotik/usecase`
  - _Requirements: 1.1, 1.2_

- [x] 2.2 Pindahkan router lifecycle usecase
  - Files: `router_usecase.go`, `health_checker.go`, `event_publisher.go`
  - _Requirements: 2.3_

- [x] 2.3 Pindahkan PPPoE usecase sebagai satu klaster
  - Files: `pppoe_*.go`, `sync_scheduler.go`
  - _Requirements: 4.1, 4.4_

- [x] 2.4 Pindahkan operational manager MikroTik
  - Files: `mikrotik_operational.go`, `dhcp_manager.go`, `static_ip_manager.go`, `walled_garden_manager.go`, `hotspot_manager.go`, `terminal_manager.go`, `backup_manager.go`, `bulk_job_manager.go`
  - _Requirements: 2.3, 3.1_

- [x] 2.5 Buat `audit_context.go`
  - Pindahkan `WithMikroTikAuditActor` dan type context audit dari `dhcp_manager.go`
  - _Requirements: 4.1_

- [x] 2.6 Pindahkan tests usecase MikroTik terkait
  - Files: `health_test.go`, `pppoe_*_test.go`, dan tests MikroTik lain yang relevan
  - _Requirements: 6.1_

- [x] 2.7 Update import handler/worker/main ke package usecase MikroTik baru
  - _Requirements: 2.3, 6.2_

## Phase 3 - PPPoE Worker Package

- [x] 3.1 Buat folder `services/network-service/internal/modules/mikrotik/worker`
  - _Requirements: 1.1, 1.2_

- [x] 3.2 Pindahkan PPPoE worker dan helper
  - Files: `pppoe_worker.go`, `pppoe_handlers.go`, `pppoe_helpers.go`
  - _Requirements: 4.2, 4.3_

- [x] 3.3 Pindahkan worker test
  - File: `pppoe_worker_test.go`
  - _Requirements: 6.1_

- [x] 3.4 Update `cmd/main.go` agar memakai worker package baru
  - _Requirements: 2.3, 4.2_

## Phase 4 - MikroTik Handler and Routes Package

- [x] 4.1 Buat folder `services/network-service/internal/modules/mikrotik/handler`
  - _Requirements: 1.1, 1.2_

- [x] 4.2 Pindahkan handler MikroTik
  - Files: `router_handler.go`, `status_handler.go`, `pppoe_handler.go`, `session_handler.go`, `mikrotik_operational_handler.go`, `dhcp_handler.go`, `static_ip_handler.go`, `walled_garden_handler.go`, `hotspot_handler.go`, `terminal_handler.go`, `backup_handler.go`, `bulk_job_handler.go`
  - _Requirements: 2.1, 2.4_

- [x] 4.3 Buat helper handler MikroTik
  - Pindahkan `fiberLocalsString` dan `canUseMikroTikTerminal` ke helper package handler MikroTik
  - _Requirements: 2.1_

- [x] 4.4 Buat `routes.go` untuk register route MikroTik
  - Public route harus tetap `/api/v1/mikrotik/...`
  - _Requirements: 2.1, 2.2, 2.4_

- [x] 4.5 Update root `internal/handler/router.go`
  - Delegasikan blok MikroTik ke package handler MikroTik baru
  - Jangan ubah route OLT/provisioning/map
  - _Requirements: 1.3, 2.1_

- [x] 4.6 Pindahkan handler tests MikroTik
  - Files: `pppoe_handler_test.go`, `router_handler_test.go`, `status_handler_test.go`
  - _Requirements: 6.1_

## Phase 5 - VPN Under MikroTik

- [x] 5.1 Pindahkan VPN handler ke folder MikroTik handler
  - Reason: public route berada di `/api/v1/mikrotik/vpn`
  - _Requirements: 2.1_

- [x] 5.2 Pindahkan VPN usecase ke folder MikroTik usecase
  - Files: `vpn_*.go`
  - _Requirements: 2.3_

- [x] 5.3 Pindahkan VPN command builder ke folder MikroTik adapter
  - Files: `vpn_command_builder.go`, `vpn_command_builder_common.go`, tests
  - _Requirements: 3.1_

## Phase 6 - Verification

- [x] 6.1 Jalankan `go list ./...` di `services/network-service`
  - Result: passed on 2026-05-07
  - _Requirements: 6.2_

- [x] 6.2 Jalankan `go test ./...` di `services/network-service`
  - Result: passed on 2026-05-07
  - _Requirements: 6.3_

- [x] 6.3 Verifikasi route MikroTik tidak berubah
  - Compare route block before/after
  - Result: delegated to `internal/modules/mikrotik/handler/routes.go`; public path tetap `/api/v1/mikrotik/...`
  - _Requirements: 2.1, 2.2_

- [x] 6.4 Verifikasi OLT/provisioning/map tetap compile
  - Result: covered by `go test ./...` in `services/network-service`
  - _Requirements: 1.3_

- [x] 6.5 Opsional: jalankan build web
  - Command: `npm.cmd --workspace @ispboss/web run build`
  - Result: passed on 2026-05-07
  - _Requirements: 6.4_

## Phase 7 - Final Acceptance Audit

- [x] 7.1 Verifikasi tidak ada checkbox task yang masih kosong
  - Result: pemeriksaan checkbox kosong tidak menemukan task terbuka
  - _Requirements: all_

- [x] 7.2 Verifikasi folder lama tidak lagi memuat handler/usecase/adapter/worker MikroTik utama
  - Result: source MikroTik utama berada di `internal/modules/mikrotik`; folder lama hanya memuat OLT/provisioning/map/shared shim yang masih diperlukan
  - _Requirements: 1.1, 1.2, 1.3, 5.1_

- [x] 7.3 Verifikasi helper handler MikroTik menjadi file helper package
  - Result: `fiberLocalsString` dan `canUseMikroTikTerminal` berada di `internal/modules/mikrotik/handler/helpers.go`
  - _Requirements: 2.1_

- [x] 7.4 Verifikasi audit context MikroTik menjadi file usecase package
  - Result: `WithMikroTikAuditActor` berada di `internal/modules/mikrotik/usecase/audit_context.go`
  - _Requirements: 4.1_
