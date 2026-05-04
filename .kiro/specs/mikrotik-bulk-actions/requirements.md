# Requirements: MikroTik Bulk Actions

## Goal

Tenant admin can run operational MikroTik actions across selected routers without enabling periodic API login.

## Scope

- Manual bulk backup.
- Manual bulk firmware check.
- Manual bulk PPPoE sync.
- Job history and per-router results.
- UI for choosing routers and reviewing outcomes.

## Non-Goals

- No scheduler.
- No automatic polling.
- No OLT validation until hardware is available.
- No destructive bulk reboot/reset.

## Requirements

1. Bulk actions SHALL run only when a tenant admin explicitly presses an action button.
2. Bulk actions SHALL support either selected router IDs or all active tenant routers.
3. Each bulk action SHALL create a persistent `mikrotik_bulk_jobs` record.
4. Each job SHALL store status, counts, per-router result payload, requester, and timestamps.
5. Bulk backup SHALL reuse the existing backup manager and its permission-safe inventory fallback.
6. Bulk firmware check SHALL reuse the existing firmware read path.
7. Bulk PPPoE sync SHALL reuse the existing PPPoE sync path.
8. A failed router SHALL not stop the remaining routers from being processed.
9. The UI SHALL show recent jobs and per-router success/failure summary.
10. The implementation SHALL keep RouterOS API access on-demand only.

