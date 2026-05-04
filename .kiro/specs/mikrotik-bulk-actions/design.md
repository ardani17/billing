# Design: MikroTik Bulk Actions

## Backend

Add `mikrotik_bulk_jobs` in `network-service`.

Fields:

- `id`
- `tenant_id`
- `action`
- `status`
- `router_ids`
- `total_count`
- `success_count`
- `failed_count`
- `results`
- `error_message`
- `requested_by`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

Actions:

- `backup`
- `firmware_check`
- `pppoe_sync`

Statuses:

- `queued`
- `running`
- `succeeded`
- `partial_failed`
- `failed`

Execution is sequential and on-demand. The request returns after the job has been processed for now, which keeps the first implementation deterministic and avoids adding a scheduler/worker loop.

## API

- `POST /api/v1/mikrotik/bulk-jobs`
- `GET /api/v1/mikrotik/bulk-jobs`
- `GET /api/v1/mikrotik/bulk-jobs/:id`

Payload:

```json
{
  "action": "backup",
  "router_ids": ["uuid"],
  "scope": "selected"
}
```

If `scope = "all_active"` or `router_ids` is empty, the manager resolves all active tenant routers.

## UI

Add `/mikrotik/bulk` page and sidebar entry.

The page shows:

- router selector
- action selector
- run button
- current result summary
- recent job history

## Safety

- Uses the same role guard as backup/terminal actions.
- Does not start polling.
- Does not run reboot or reset actions.
- Persists every per-router failure message.

