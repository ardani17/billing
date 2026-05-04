# Implementation Plan: MikroTik Completion Layer

## Overview

Rencana ini mengubah daftar fitur besar di `diskusi/08-mikrotik.md` menjadi urutan implementasi yang aman. Urutan dibuat supaya fitur read-only live data selesai lebih dulu, lalu fitur write/action yang lebih berisiko dikerjakan setelah fondasi audit dan konfirmasi siap.

## Tasks

- [ ] 1. Phase 1 - Operational read-only domain and adapter support
  - [x] 1.1 Add domain DTOs for router interfaces, traffic samples, IP pool usage, managed firewall rules, and router logs
  - [x] 1.2 Add on-demand RouterOS execution support for interface print, monitor traffic, IP pool print/used, firewall managed print, and log print
  - [x] 1.3 Add parser helpers that normalize RouterOS v6/v7 response maps into typed DTOs
  - [ ] 1.4 Add unit tests for parser helpers using RouterOS v6.49-style sample responses
  - [x] 1.5 Ensure read-only operations use on-demand adapter execution and do not trigger scheduler/polling
  - _Requirements: 1, 2, 3, 4, 5, 14_

- [ ] 2. Phase 1 - Operational read-only usecase and handlers
  - [x] 2.1 Add MikroTik operational usecase with methods: ListInterfaces, GetTraffic, ListIPPools, ListManagedFirewall, ListLogs
  - [x] 2.2 Add Fiber handlers and network-service routes for the five read-only endpoints
  - [x] 2.3 Add web proxy routes under `apps/web/app/api/network/mikrotik/routers/[id]/...`
  - [ ] 2.4 Add handler tests for validation, router not found, and successful response mapping
  - [x] 2.5 Smoke test against local network-service and CHR where safe
  - _Requirements: 1, 2, 3, 4, 5_

- [ ] 3. Phase 1 - Web UI submenu panels
  - [x] 3.1 Extend sidebar MikroTik detail submenu with Traffic, Interfaces, IP Pool, Firewall, and Logs
  - [x] 3.2 Add Next route pages: `/mikrotik/[id]/traffic`, `/interfaces`, `/ip-pool`, `/firewall`, `/logs`
  - [x] 3.3 Add focused panel components under `components/router-detail/`
  - [x] 3.4 Add inline loading, empty, and error states for each panel
  - [ ] 3.5 Verify mobile layout does not require horizontal page scrolling
  - _Requirements: 1, 2, 3, 4, 5_

- [ ] 4. Checkpoint - Phase 1 verification
  - [x] 4.1 Run `go test ./...` in `services/network-service`
  - [x] 4.2 Run `npm.cmd --workspace @ispboss/web run build`
  - [x] 4.3 Restart localhost dev server cleanly
  - [x] 4.4 Verify all new pages return HTTP 200
  - [x] 4.5 Commit and push Phase 1

- [ ] 5. Phase 2 - DHCP read model
  - [ ] 5.1 Add domain DTOs for DHCP servers, leases, static bindings, and networks
  - [ ] 5.2 Add read-only command builders and parsers for `/ip/dhcp-server/print`, `/ip/dhcp-server/lease/print`, and networks
  - [ ] 5.3 Add read endpoints and web proxy routes
  - [ ] 5.4 Add DHCP submenu UI with Servers, Leases, Static Bindings, Networks sections
  - _Requirements: 6_

- [ ] 6. Phase 2 - DHCP static binding writes
  - [ ] 6.1 Add `dhcp_bindings` migration, queries, repository, and domain model
  - [ ] 6.2 Add create/update/delete binding usecases with idempotent RouterOS writes
  - [ ] 6.3 Add explicit confirmation for delete/disable actions
  - [ ] 6.4 Add audit rows for DHCP binding write operations
  - [ ] 6.5 Add UI forms for managed static bindings
  - _Requirements: 6, 13, 14_

- [ ] 7. Phase 3 - Static IP management
  - [ ] 7.1 Add `static_ip_assignments` migration, queries, repository, and domain model
  - [ ] 7.2 Add provisioning usecase for address-list and optional simple queue
  - [ ] 7.3 Add isolate/unisolate for static IP customers
  - [ ] 7.4 Add Static IP submenu UI
  - [ ] 7.5 Add tests for idempotency and confirmation safety
  - _Requirements: 7, 9, 13, 14_

- [ ] 8. Phase 4 - Walled garden completion
  - [ ] 8.1 Complete RouterOS command builders for DNS redirect, HTTP redirect, and block-all whitelist
  - [ ] 8.2 Add tenant settings lookup for isolir method
  - [ ] 8.3 Ensure firewall rules and address lists are prefixed with `ISPBoss:`
  - [ ] 8.4 Add UI status showing current managed walled garden rules per router
  - [ ] 8.5 Add integration tests for isolir/unisolir idempotency using mock adapter
  - _Requirements: 9, 13, 14_

- [ ] 9. Phase 5 - Hotspot and voucher integration
  - [ ] 9.1 Add Hotspot domain DTOs and RouterOS command builders
  - [ ] 9.2 Add hotspot user/profile/active endpoints
  - [ ] 9.3 Connect voucher activation event to hotspot user creation
  - [ ] 9.4 Generate custom hotspot login page from tenant branding
  - [ ] 9.5 Add Hotspot submenu UI
  - _Requirements: 8, 13, 14_

- [ ] 10. Phase 6 - Terminal safety and audit
  - [ ] 10.1 Add `mikrotik_command_audit_logs` migration and repository
  - [ ] 10.2 Add terminal command validator with denylist and optional allowlist mode
  - [ ] 10.3 Add terminal execute endpoint with RBAC guard
  - [ ] 10.4 Write audit rows for every terminal attempt
  - [ ] 10.5 Add Terminal submenu UI with warning and command history
  - _Requirements: 10, 13, 14_

- [ ] 11. Phase 7 - Backup and firmware
  - [ ] 11.1 Add `router_backups` migration, repository, and storage abstraction
  - [ ] 11.2 Add manual backup endpoint and download endpoint
  - [ ] 11.3 Add restore endpoint with confirmation
  - [ ] 11.4 Add scheduled backup worker disabled by default
  - [ ] 11.5 Add firmware tracking read endpoint and outdated warning logic
  - [ ] 11.6 Add Backup and Firmware submenu UI
  - _Requirements: 11, 13, 14_

- [ ] 12. Phase 8 - Bulk actions
  - [ ] 12.1 Add `mikrotik_bulk_jobs` migration, repository, and job status model
  - [ ] 12.2 Add async job handlers for bulk sync, backup, firmware check, and export status
  - [ ] 12.3 Add bulk action UI with confirmation and progress states
  - [ ] 12.4 Add tests for tenant isolation and job status transitions
  - _Requirements: 12, 13, 14_

- [ ] 13. Final hardening
  - [ ] 13.1 Review all MikroTik write paths for audit coverage
  - [ ] 13.2 Review all scheduler defaults to avoid repeated router login in local/dev
  - [ ] 13.3 Run full service tests and web build
  - [ ] 13.4 Run opt-in CHR smoke tests for safe read/write flows
  - [ ] 13.5 Update project report/checklist for completed MikroTik features
