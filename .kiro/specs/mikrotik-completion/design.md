# Design Document - MikroTik Completion Layer

## Overview

MikroTik Completion Layer melengkapi modul MikroTik agar seluruh submenu dari `diskusi/08-mikrotik.md` punya backend dan UI yang rapi. Desain mengikuti struktur yang sudah berjalan:

```text
domain -> repository -> usecase -> handler -> web proxy -> UI page/panel
```

Untuk fitur yang hanya membaca data live dari RouterOS, kita tidak membuat tabel permanen lebih dulu kecuali data tersebut perlu histori, audit, atau status job. Ini menjaga implementasi ringan dan mengurangi risiko data stale.

## Source Specs

- `diskusi/08-mikrotik.md`
- `.kiro/specs/mikrotik-router`
- `.kiro/specs/mikrotik-pppoe`
- `.kiro/specs/mikrotik-vpn`
- `.kiro/specs/isolir-system`
- `.kiro/specs/package-crud`
- `.kiro/specs/reseller-voucher`

## Current Gap Summary

| Area | Current Status | Completion Strategy |
|---|---|---|
| Router CRUD/test | Mostly done | Keep, harden, reuse |
| PPPoE | Mostly done | Keep, add UI polish and CHR tests |
| VPN | Backend exists | Connect UI progressively |
| Traffic/interfaces/IP pool/firewall/log | Missing as exposed API | Build read-only live endpoints first |
| DHCP binding | Missing as module | Add schema/usecase/handler after read-only tabs |
| Static IP | Missing as module | Add after DHCP foundation |
| Hotspot/voucher | Missing as MikroTik module | Add after static/DHCP |
| Backup/firmware | Missing | Add metadata schema and async jobs |
| Terminal | Missing | Add late, with blacklist and audit |
| Bulk actions | Missing | Add async job model after backup/firmware |
| Audit trail | Partial in provisioning only | Add MikroTik command audit table |

## Backend Route Plan

### Phase 1 - Operational Read-Only Routes

```text
GET /api/v1/mikrotik/routers/:id/interfaces
GET /api/v1/mikrotik/routers/:id/traffic
GET /api/v1/mikrotik/routers/:id/ip-pools
GET /api/v1/mikrotik/routers/:id/firewall/managed
GET /api/v1/mikrotik/routers/:id/logs
```

These routes execute RouterOS read commands only. They do not alter router configuration.

### Phase 2 - DHCP Binding Routes

```text
GET    /api/v1/mikrotik/routers/:id/dhcp/servers
GET    /api/v1/mikrotik/routers/:id/dhcp/leases
GET    /api/v1/mikrotik/routers/:id/dhcp/bindings
POST   /api/v1/mikrotik/routers/:id/dhcp/bindings
PUT    /api/v1/mikrotik/routers/:id/dhcp/bindings/:bindingId
DELETE /api/v1/mikrotik/routers/:id/dhcp/bindings/:bindingId
GET    /api/v1/mikrotik/routers/:id/dhcp/networks
```

`networks` is read-only. Bindings are the only DHCP writes managed by ISPBoss.

### Phase 3 - Static IP Routes

```text
GET    /api/v1/mikrotik/routers/:id/static-ip/assignments
POST   /api/v1/mikrotik/routers/:id/static-ip/assignments
PUT    /api/v1/mikrotik/routers/:id/static-ip/assignments/:assignmentId
DELETE /api/v1/mikrotik/routers/:id/static-ip/assignments/:assignmentId
POST   /api/v1/mikrotik/routers/:id/static-ip/assignments/:assignmentId/isolate
POST   /api/v1/mikrotik/routers/:id/static-ip/assignments/:assignmentId/unisolate
```

### Phase 4 - Hotspot Routes

```text
GET    /api/v1/mikrotik/routers/:id/hotspot/users
POST   /api/v1/mikrotik/routers/:id/hotspot/users
PUT    /api/v1/mikrotik/routers/:id/hotspot/users/:userId
DELETE /api/v1/mikrotik/routers/:id/hotspot/users/:userId
GET    /api/v1/mikrotik/routers/:id/hotspot/profiles
GET    /api/v1/mikrotik/routers/:id/hotspot/active
POST   /api/v1/mikrotik/routers/:id/hotspot/login-template/generate
```

### Phase 5 - Backup, Firmware, Terminal, Bulk

```text
POST /api/v1/mikrotik/routers/:id/backups
GET  /api/v1/mikrotik/routers/:id/backups
GET  /api/v1/mikrotik/routers/:id/backups/:backupId/download
POST /api/v1/mikrotik/routers/:id/backups/:backupId/restore
GET  /api/v1/mikrotik/routers/:id/firmware
POST /api/v1/mikrotik/routers/:id/terminal/execute
POST /api/v1/mikrotik/bulk/jobs
GET  /api/v1/mikrotik/bulk/jobs/:jobId
```

## RouterOS Command Plan

| Feature | RouterOS Commands | Write? |
|---|---|---|
| Interfaces | `/interface/print` | No |
| Traffic | `/interface/monitor-traffic` | No |
| IP pools | `/ip/pool/print`, `/ip/pool/used/print` | No |
| Firewall managed | `/ip/firewall/nat/print`, `/ip/firewall/filter/print`, `/ip/firewall/address-list/print` | No |
| Logs | `/log/print` | No |
| DHCP servers | `/ip/dhcp-server/print` | No |
| DHCP leases | `/ip/dhcp-server/lease/print` | No |
| DHCP bindings | `/ip/dhcp-server/lease/add,set,remove` | Yes |
| Static IP | `/ip/firewall/address-list/add,set,remove`, `/queue/simple/add,set,remove` | Yes |
| Hotspot | `/ip/hotspot/user/*`, `/ip/hotspot/active/print` | Yes |
| Backup export | `/export` or `/system/backup` strategy | Yes |
| Terminal | arbitrary allowlisted commands | Yes/No |

## Data Model Additions

Phase 1 read-only does not need new database tables.

Later phases add:

- `mikrotik_command_audit_logs`
- `dhcp_bindings`
- `static_ip_assignments`
- `hotspot_users` or mapping to voucher records
- `router_backups`
- `mikrotik_bulk_jobs`

Each new table must include `tenant_id`, timestamps, and soft-delete where applicable.

## Web UI Plan

Sidebar submenu under MikroTik will grow in this order:

1. Overview
2. PPPoE users
3. Session live
4. Sinkronisasi
5. Traffic
6. Interfaces
7. IP Pool
8. Firewall
9. Logs
10. DHCP
11. Static IP
12. Hotspot
13. Backup
14. Firmware
15. Terminal

The UI must use small focused panels under `apps/web/app/mikrotik/components/router-detail/` and avoid returning to a large single file.

## Safety Rules

1. Read-only routes may load on page open.
2. Write routes must be explicit user actions.
3. Any repeated polling must be opt-in in the UI and bounded.
4. Any scheduler must be disabled by default in local/dev.
5. All destructive actions require confirmation text or a specific confirmation field.
6. Do not store plain RouterOS credentials.
7. Do not expose credentials in logs, UI, or test fixtures.

## Testing Strategy

- Unit tests for parser functions and command builders.
- Handler tests for route status and validation.
- Usecase tests for idempotency and safety checks.
- Opt-in CHR integration tests behind environment variables:
  - `MIKROTIK_TEST_HOST`
  - `MIKROTIK_TEST_PORT`
  - `MIKROTIK_TEST_USERNAME`
  - `MIKROTIK_TEST_PASSWORD`
  - `MIKROTIK_TEST_SSL`
- Smoke tests through web proxy endpoints after each phase.

## Implementation Order

The first implementation phase after this spec should be **Phase 1: Operational Read-Only Routes**, because it provides visible value in the new MikroTik sidebar without changing router configuration.
