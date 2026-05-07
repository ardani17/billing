# Tasks - MikroTik RouterOS Version Command Layer

## Overview

Checklist ini dibuat dari audit `docs/AUDIT-MIKROTIK-ROUTEROS-VERSION-COMMAND-LAYER-2026-05-07.md`.

Router testing saat ini adalah RouterOS v6, jadi semua implementasi harus menjaga v6 sebagai baseline utama.

## Phase 0 - Audit and Spec

- [x] 0.1 Audit alur RouterOS version dan command builder existing
  - Output: `docs/AUDIT-MIKROTIK-ROUTEROS-VERSION-COMMAND-LAYER-2026-05-07.md`
  - _Requirements: all_

- [x] 0.2 Buat spec requirements, design, dan tasks
  - Output: `.kiro/specs/mikrotik-routeros-version-command-layer`
  - _Requirements: all_

## Phase 1 - Version Parser and Capability

- [x] 1.1 Tambah `adapter/version.go`
  - Implement `RouterOSMajor`, `ParseRouterOSMajor`, dan `NormalizeRouterOSVersion`
  - Result: implemented with domain-backed wrappers
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 1.2 Update `domain.IsRouterOSv7`
  - Gunakan parser baru tanpa menghapus API lama
  - Result: `IsRouterOSv7` now uses `ParseRouterOSMajor`
  - _Requirements: 1.4_

- [x] 1.3 Tambah `adapter/capabilities.go`
  - Implement `RouterOSCapabilities` dan `CapabilitiesFor`
  - Result: implemented with conservative WireGuard capability
  - _Requirements: 2.1, 2.2, 2.3_

- [x] 1.4 Tambah unit test parser versi
  - Case: `6.49.18 (long-term)`, `7.14.3`, ` RouterOS 7.14.3`, empty, malformed
  - Result: covered in adapter tests
  - _Requirements: 7.2_

## Phase 2 - PPPoE Builder Hardening

- [x] 2.1 Update `NewCommandBuilder` agar memakai parser versi baru
  - v6/unknown fallback ke `commandBuilderV6`
  - Result: implemented
  - _Requirements: 3.1, 3.2, 7.3_

- [x] 2.2 Tambah test command builder untuk router testing v6
  - Version fixture: `6.49.18 (long-term)`
  - Result: covered in `TestProperty_VersionAwareCommandPathSelection`
  - _Requirements: 3.4_

- [x] 2.3 Dokumentasikan v7 inherited behavior
  - Jika command v7 masih sama dengan v6, test harus menyatakan itu intentional
  - Result: v7 builder remains explicit wrapper over v6 until a real command difference is introduced
  - _Requirements: 3.3, 7.4_

## Phase 3 - VPN Version-Aware Builder

- [x] 3.1 Buat VPN command builder factory
  - Pilih v6/v7 berdasarkan router version
  - Result: `NewVPNCommandBuilderForVersion`
  - _Requirements: 4.1_

- [x] 3.2 Pisahkan file VPN builder v6 dan v7
  - Mulai dengan v7 embed v6 jika belum ada command berbeda
  - Result: `vpn_command_builder_v6.go` and `vpn_command_builder_v7.go`
  - _Requirements: 4.1, 4.3_

- [x] 3.3 Update WireGuard validation memakai capability
  - Unknown dan v6 harus ditolak
  - Result: validation uses `CapabilitiesForRouterOS`
  - _Requirements: 2.2, 2.3, 4.2_

- [x] 3.4 Update VPN auto-configure agar memilih builder per-router
  - Ambil `router.RouterOSVersion` sebelum execute command
  - Result: `AutoConfigure` builds VPN commands with router-specific builder
  - _Requirements: 4.1_

- [x] 3.5 Tambah test WireGuard v6/unknown/v7
  - Result: covered in usecase integration tests
  - _Requirements: 4.2, 7.1_

## Phase 4 - High-Risk Raw Write Commands

- [x] 4.1 Migrasikan command write DHCP manager ke command layer
  - Target: lease add/set/remove
  - Result: add/set/remove use `CommandBuilder`
  - _Requirements: 5.1, 5.2_

- [x] 4.2 Migrasikan command write static IP manager ke command layer
  - Target: address-list add/set/remove dan queue add/set/remove
  - Result: address-list and queue writes use `CommandBuilder`
  - _Requirements: 5.1, 5.2_

- [x] 4.3 Migrasikan command write hotspot manager ke command layer
  - Target: user add/set/remove
  - Result: hotspot user writes use `CommandBuilder`
  - _Requirements: 5.1, 5.2_

- [x] 4.4 Migrasikan command write walled garden manager ke command layer
  - Target: firewall nat/filter/address-list add/set/remove
  - Result: firewall and address-list writes use `CommandBuilder`
  - _Requirements: 5.1, 5.2_

- [x] 4.5 Tentukan apakah backup/export lifecycle perlu builder
  - Target: `/export`, `/file/print`, `/file/get`, `/file/remove`
  - Result: kept as raw lifecycle commands for this phase; no v6/v7 difference identified, lower risk than customer-affecting write commands
  - _Requirements: 5.1, 5.2_

## Phase 5 - Metadata Freshness

- [x] 5.1 Update health checker success path
  - Refresh `RouterOSVersion`, `BoardName`, `CPUCount`, `TotalRAMMB`, dan `Identity` secara best-effort
  - Result: implemented via `refreshRouterMetadata`
  - _Requirements: 6.2, 6.3_

- [x] 5.2 Tambah test health checker untuk version refresh
  - Result: covered in health checker tests
  - _Requirements: 6.2, 7.1_

## Phase 6 - Verification

- [x] 6.1 Jalankan targeted tests package adapter MikroTik
  - Command: `go test ./internal/modules/mikrotik/adapter`
  - Result: passed on 2026-05-07
  - _Requirements: 7.1_

- [x] 6.2 Jalankan targeted tests package usecase MikroTik
  - Command: `go test ./internal/modules/mikrotik/usecase`
  - Result: passed on 2026-05-07
  - _Requirements: 7.1_

- [x] 6.3 Jalankan full network-service tests
  - Command: `go test ./...`
  - Result: passed on 2026-05-07
  - _Requirements: 7.1_

- [x] 6.4 Jalankan build web jika ada perubahan contract/API route
  - Command: `npm.cmd --workspace @ispboss/web run build`
  - Result: passed on 2026-05-07
  - _Requirements: 5.4_

## Phase 7 - Final Acceptance Audit

- [x] 7.1 Verifikasi semua task tertutup
  - Result: tidak ada checkbox kosong tersisa
  - _Requirements: all_

- [x] 7.2 Verifikasi write command target tidak lagi memakai raw literal execute
  - Result: DHCP lease, static IP address-list/queue, hotspot user, dan walled garden firewall/address-list write command memakai command builder
  - _Requirements: 5.1, 5.2_

- [x] 7.3 Verifikasi route publik tidak berubah
  - Result: tidak ada perubahan route `/api/v1/mikrotik/...` atau Next.js proxy path
  - _Requirements: 5.4_
