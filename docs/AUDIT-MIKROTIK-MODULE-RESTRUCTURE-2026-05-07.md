# Audit dan Spec: Restruktur Folder Modul MikroTik

Tanggal: 2026-05-07
Repo: `C:\laragon\www\billing`
Scope: audit dampak dan spec pemindahan kode MikroTik backend ke folder modul khusus. Belum ada implementasi pemindahan file pada dokumen ini.

## Ringkasan Keputusan

Restruktur folder MikroTik bisa dilakukan dan memang disarankan sebelum modul bertambah besar. Namun ini harus dikerjakan bertahap karena Go package berbasis folder. Memindahkan file bukan sekadar merapikan direktori; import path, test package, constructor, route registration, worker, dan adapter factory ikut terdampak.

Keputusan aman:

1. Buat folder modul MikroTik di `services/network-service/internal/modules/mikrotik`.
2. Fase pertama pindahkan `handler`, `usecase`, `adapter`, dan `worker` MikroTik.
3. Jangan pindahkan `domain`, `repository`, `queries`, `migrations`, dan generated `*.sql.go` pada fase pertama.
4. Public route tetap sama: `/api/v1/mikrotik/...`.
5. Frontend path tetap sama: `apps/web/app/mikrotik` dan `apps/web/app/api/network/mikrotik`.
6. Refactor ini tidak boleh mengubah behavior router, PPPoE, DHCP, terminal, backup, hotspot, static IP, walled garden, bulk job, atau VPN.

## Kondisi Saat Ini

Backend `network-service` masih memakai struktur layer-based:

```text
services/network-service/internal/
  adapter/
  domain/
  handler/
  repository/
  usecase/
  worker/
```

MikroTik, OLT, provisioning, mapping, dan VPN bercampur dalam folder yang sama. Dari sisi debug, ini membuat developer harus memilah file berdasarkan prefix nama, bukan folder modul.

Baseline package saat audit:

```text
github.com/ispboss/ispboss/services/network-service/cmd
github.com/ispboss/ispboss/services/network-service/internal/adapter
github.com/ispboss/ispboss/services/network-service/internal/config
github.com/ispboss/ispboss/services/network-service/internal/crypto
github.com/ispboss/ispboss/services/network-service/internal/domain
github.com/ispboss/ispboss/services/network-service/internal/handler
github.com/ispboss/ispboss/services/network-service/internal/metrics
github.com/ispboss/ispboss/services/network-service/internal/middleware
github.com/ispboss/ispboss/services/network-service/internal/pool
github.com/ispboss/ispboss/services/network-service/internal/repository
github.com/ispboss/ispboss/services/network-service/internal/usecase
github.com/ispboss/ispboss/services/network-service/internal/worker
```

`go list ./...` di `services/network-service` berhasil pada baseline audit.

## Area MikroTik yang Terdampak

### Handler

File MikroTik yang terkait:

- `internal/handler/router_handler.go`
- `internal/handler/status_handler.go`
- `internal/handler/pppoe_handler.go`
- `internal/handler/session_handler.go`
- `internal/handler/mikrotik_operational_handler.go`
- `internal/handler/dhcp_handler.go`
- `internal/handler/static_ip_handler.go`
- `internal/handler/walled_garden_handler.go`
- `internal/handler/hotspot_handler.go`
- `internal/handler/terminal_handler.go`
- `internal/handler/backup_handler.go`
- `internal/handler/bulk_job_handler.go`
- `internal/handler/router.go` untuk route registration
- test terkait: `pppoe_handler_test.go`, `router_handler_test.go`, `status_handler_test.go`

Catatan penting:

- `canUseMikroTikTerminal` saat ini ada di `terminal_handler.go` dan dipakai oleh backup/bulk/terminal.
- `fiberLocalsString` saat ini ada di `dhcp_handler.go` dan dipakai banyak handler MikroTik.
- Jika handler dipecah per subfolder terlalu cepat, helper ini harus dibuat shared dalam package handler MikroTik.

### Usecase

File MikroTik yang terkait:

- `router_usecase.go`
- `health_checker.go`
- `event_publisher.go`
- `sync_scheduler.go`
- `mikrotik_operational.go`
- `pppoe_manager.go`
- `pppoe_crud.go`
- `pppoe_manager_handlers.go`
- `pppoe_manager_helpers.go`
- `pppoe_manager_isolir.go`
- `pppoe_manager_unisolir.go`
- `pppoe_manager_suspend.go`
- `pppoe_manager_package.go`
- `pppoe_profile_resolver.go`
- `pppoe_profile_sync.go`
- `pppoe_sessions.go`
- `pppoe_sync.go`
- `pppoe_event_publisher.go`
- `dhcp_manager.go`
- `static_ip_manager.go`
- `walled_garden_manager.go`
- `hotspot_manager.go`
- `terminal_manager.go`
- `backup_manager.go`
- `bulk_job_manager.go`

MikroTik-adjacent VPN yang perlu diperlakukan khusus:

- `vpn_manager.go`
- `vpn_manager_create.go`
- `vpn_manager_autoconfig.go`
- `vpn_manager_setup.go`
- `vpn_manager_helpers.go`
- `vpn_health_monitor.go`
- `vpn_key_generator.go`
- `vpn_script_generator.go`
- `vpn_script_templates.go`
- `vpn_script_templates_ext.go`
- `vpn_bandwidth_store.go`
- `vpn_event_publisher.go`

Catatan penting:

- Semua file `pppoe_*` harus pindah satu paket karena menggunakan receiver `*pppoeManager` dan helper unexported.
- `WithMikroTikAuditActor` didefinisikan di area DHCP, tetapi dipakai backup, bulk, DHCP, hotspot, static IP, terminal, walled garden. Helper ini harus dipindah ke file khusus seperti `audit_context.go` dalam package usecase MikroTik.
- `sync_scheduler.go` adalah scheduler PPPoE, jadi ikut MikroTik.
- `event_publisher.go` adalah event router online/offline, jadi ikut Router/MikroTik.

### Adapter

File RouterOS yang terkait:

- `internal/adapter/adapter.go`
- `internal/adapter/factory.go`
- `internal/adapter/live_adapter.go`
- `internal/adapter/mock_adapter.go`
- `internal/adapter/command_builder_factory.go`
- `internal/adapter/command_builder_v6.go`
- `internal/adapter/command_builder_v7.go`
- test terkait command builder dan live adapter

MikroTik-adjacent VPN adapter:

- `vpn_command_builder.go`
- `vpn_command_builder_common.go`
- `vpn_command_builder_test.go`

Catatan penting:

- OLT adapter berada di folder yang sama saat ini, tetapi tidak boleh ikut tersentuh pada fase MikroTik.
- `pool.NewPoolManager` menerima factory `func() domain.RouterOSAdapter`, jadi aman selama factory import path diperbarui.
- `domain.RouterOSAdapter`, `domain.CommandBuilder`, dan DTO command tetap di `internal/domain` pada fase pertama.

### Worker

File worker yang terkait:

- `internal/worker/pppoe_worker.go`
- `internal/worker/pppoe_handlers.go`
- `internal/worker/pppoe_helpers.go`
- `internal/worker/pppoe_worker_test.go`

Catatan penting:

- Worker provisioning OLT juga memakai event `customer.terminated`, tetapi saat ini tidak diregister aktif di main.
- Jangan mengubah event name dan retry behavior saat memindahkan package.

### Repository, Queries, dan Migrations

Jangan dipindahkan pada fase pertama:

- `internal/repository/routers.sql.go`
- `internal/repository/pppoe_users.sql.go`
- `internal/repository/pppoe_profiles.sql.go`
- `internal/repository/dhcp_bindings.sql.go`
- `internal/repository/mikrotik_command_audit_logs.sql.go`
- `internal/repository/static_ip_assignments.sql.go`
- `internal/repository/router_backups.sql.go`
- `internal/repository/mikrotik_bulk_job_repo.go`
- `queries/*.sql`
- `migrations/*.sql`

Alasan:

- `sqlc.yaml` saat ini satu generator: `queries/` dan `migrations/` ke package `internal/repository`.
- Memecah repository berarti perlu desain SQLC baru dan import package generated baru.
- Fase pertama cukup memperbaiki debug boundary tanpa risiko generated code churn.

## Risiko Utama

1. **Import path pecah**
   - Pindah folder mengubah package path.
   - Mitigasi: pindahkan per klaster dan update import secara mekanis.

2. **Unexported helper hilang**
   - Contoh: `pppoeManager`, `parseDisabledField`, `WithMikroTikAuditActor`, `canUseMikroTikTerminal`.
   - Mitigasi: pindahkan file terkait dalam satu package, bukan subpackage kecil di awal.

3. **Route berubah tanpa sengaja**
   - Public API `/api/v1/mikrotik/...` dipakai frontend.
   - Mitigasi: route snapshot sebelum dan sesudah refactor harus sama.

4. **Worker tidak terdaftar**
   - `pppoeWorker.RegisterHandlers(mux)` harus tetap jalan.
   - Mitigasi: main import worker baru dan test event worker.

5. **RouterOS adapter factory tidak dipakai**
   - `adapter.NewAdapter`, `adapter.NewCommandBuilder`, dan VPN command builder akan berubah import path.
   - Mitigasi: buat alias sementara atau update main/pool wiring secara eksplisit.

6. **Test internal package gagal**
   - Banyak test berada di package yang sama agar bisa akses helper unexported.
   - Mitigasi: pindahkan test bersama source file.

7. **OLT ikut terdampak**
   - OLT masih import `internal/adapter` dan `internal/usecase`.
   - Mitigasi: jangan rename package lama secara total dalam satu langkah; hanya move file MikroTik dan biarkan package OLT tetap berjalan.

8. **VPN terlupakan**
   - Route VPN berada di `/api/v1/mikrotik/vpn`.
   - Mitigasi: masukkan VPN ke fase MikroTik, tetapi setelah core RouterOS/PPPoE stabil.

## Struktur Target

Fase pertama:

```text
services/network-service/internal/modules/mikrotik/
  adapter/
    adapter.go
    factory.go
    live_adapter.go
    mock_adapter.go
    command_builder_factory.go
    command_builder_v6.go
    command_builder_v7.go
    vpn_command_builder.go
    vpn_command_builder_common.go
  handler/
    routes.go
    router_handler.go
    status_handler.go
    pppoe_handler.go
    session_handler.go
    operational_handler.go
    dhcp_handler.go
    static_ip_handler.go
    walled_garden_handler.go
    hotspot_handler.go
    terminal_handler.go
    backup_handler.go
    bulk_job_handler.go
    vpn_handler.go
    helpers.go
  usecase/
    router_usecase.go
    health_checker.go
    event_publisher.go
    sync_scheduler.go
    audit_context.go
    pppoe_*.go
    dhcp_manager.go
    static_ip_manager.go
    walled_garden_manager.go
    hotspot_manager.go
    terminal_manager.go
    backup_manager.go
    bulk_job_manager.go
    vpn_*.go
  worker/
    pppoe_worker.go
    pppoe_handlers.go
    pppoe_helpers.go
```

Tetap di lokasi lama pada fase pertama:

```text
services/network-service/internal/domain/
services/network-service/internal/repository/
services/network-service/queries/
services/network-service/migrations/
services/network-service/internal/handler/health.go
services/network-service/internal/handler/router.go
```

`internal/handler/router.go` boleh tetap menjadi root route composer sementara, tetapi route MikroTik harus didelegasikan ke `modules/mikrotik/handler.RegisterRoutes`.

## Spec Eksekusi Aman

### Phase 0 - Baseline

1. Jalankan `go list ./...` di `services/network-service`.
2. Catat route MikroTik di `internal/handler/router.go`.
3. Catat constructor MikroTik di `cmd/main.go`.
4. Jangan ubah frontend, DB, migration, SQLC.

### Phase 1 - Buat Package MikroTik Adapter

1. Buat folder `internal/modules/mikrotik/adapter`.
2. Pindahkan RouterOS adapter dan command builder.
3. Update package name menjadi `adapter` atau `mikrotikadapter`.
4. Update import di `cmd/main.go`, tests, dan usecase yang memakai adapter factory.
5. Jalankan `go test ./internal/modules/mikrotik/adapter`.

### Phase 2 - Buat Package MikroTik Usecase

1. Buat folder `internal/modules/mikrotik/usecase`.
2. Pindahkan router usecase, health checker, event publisher, PPPoE, DHCP, static IP, walled garden, hotspot, terminal, backup, bulk manager.
3. Pindahkan tests terkait.
4. Buat `audit_context.go` untuk `WithMikroTikAuditActor`.
5. Update import handler/worker/main dari `internal/usecase` ke package baru.
6. Jangan pindah OLT/provisioning/map usecase.
7. Jalankan `go test ./internal/modules/mikrotik/usecase`.

### Phase 3 - Buat Package MikroTik Worker

1. Buat folder `internal/modules/mikrotik/worker`.
2. Pindahkan PPPoE worker dan tests.
3. Update worker import ke usecase MikroTik baru.
4. Update `cmd/main.go` agar `mikrotikworker.NewPPPoEEventWorker` dan `RegisterHandlers` tetap terpakai.
5. Jalankan `go test ./internal/modules/mikrotik/worker`.

### Phase 4 - Buat Package MikroTik Handler dan Routes

1. Buat folder `internal/modules/mikrotik/handler`.
2. Pindahkan handler MikroTik dan tests.
3. Buat helper `helpers.go` untuk `fiberLocalsString` dan `canUseMikroTikTerminal`.
4. Buat `routes.go` dengan fungsi `RegisterRoutes(api fiber.Router, cfg RouterConfig)`.
5. Di `internal/handler/router.go`, ganti blok MikroTik menjadi delegasi ke package baru.
6. Pastikan public route tidak berubah.
7. Jalankan `go test ./internal/modules/mikrotik/handler`.

### Phase 5 - VPN MikroTik

1. Pindahkan handler/usecase/adapter VPN ke modul MikroTik karena route publik ada di `/api/v1/mikrotik/vpn`.
2. Jaga domain VPN tetap di `internal/domain`.
3. Update `cmd/main.go`.
4. Jalankan test VPN terkait.

### Phase 6 - Full Verification

1. Jalankan `go test ./...` di `services/network-service`.
2. Jalankan `npm.cmd --workspace @ispboss/web run build` dari root repo bila perlu validasi frontend proxy.
3. Jika stack live tersedia, smoke endpoint:
   - `GET /api/v1/mikrotik/routers`
   - `GET /api/v1/mikrotik/status/summary`
   - `POST /api/v1/mikrotik/routers/:id/test`
   - `GET /api/v1/mikrotik/routers/:id/pppoe/users`
   - `GET /api/v1/mikrotik/routers/:id/terminal/audit`
4. Pastikan endpoint OLT tetap compile dan route guard fiber network tidak berubah.

## Non-Goals

- Tidak mengganti route publik.
- Tidak mengganti schema DB.
- Tidak memecah SQLC generated repository.
- Tidak memindahkan OLT pada pekerjaan MikroTik pertama.
- Tidak mengubah business logic PPPoE, router health, terminal policy, atau bulk job.
- Tidak membersihkan semua folder lama sekaligus.

## Acceptance Criteria

- Ada folder khusus MikroTik di `internal/modules/mikrotik`.
- `cmd/main.go` tetap bisa bootstrap MikroTik, OLT, provisioning, map, dan worker.
- Semua endpoint `/api/v1/mikrotik/...` tetap sama.
- Worker PPPoE tetap register event yang sama.
- OLT/provisioning/map tidak ikut berubah behavior.
- `go test ./...` di `services/network-service` lulus setelah refactor.
- Spec tasks di `.kiro/specs/mikrotik-module-restructure/tasks.md` dipakai untuk menandai progres.
