# Design - MikroTik Module Restructure

## Current Design

`network-service` saat ini memakai struktur horizontal:

```text
internal/adapter
internal/domain
internal/handler
internal/repository
internal/usecase
internal/worker
```

Struktur ini membuat MikroTik dan OLT bercampur di folder yang sama. Route sudah terpisah secara konsep, tetapi kode backend belum.

## Target Design

Target fase pertama:

```text
internal/modules/mikrotik/
  adapter/
  handler/
  usecase/
  worker/
```

Package lama tetap ada untuk OLT, provisioning, map, shared health, middleware, metrics, pool, repository, dan domain.

## Package Boundary

### Adapter

`internal/modules/mikrotik/adapter` berisi RouterOS live/mock adapter, factory, dan command builder. Package ini tetap mengimplementasikan interface dari `internal/domain`.

### Usecase

`internal/modules/mikrotik/usecase` berisi router usecase, health checker router, PPPoE, DHCP, static IP, walled garden, hotspot, terminal, backup, bulk job, sync scheduler, event publisher, dan VPN usecase.

### Handler

`internal/modules/mikrotik/handler` berisi semua HTTP handler MikroTik dan route registrar khusus MikroTik. Root `internal/handler/router.go` tetap menjadi composer utama lintas modul, tetapi blok MikroTik didelegasikan.

### Worker

`internal/modules/mikrotik/worker` berisi PPPoE worker dan helper event lifecycle.

## Files That Stay Shared

Fase pertama sengaja tidak memindahkan:

- `internal/domain`
- `internal/repository`
- `internal/pool`
- `internal/middleware`
- `internal/metrics`
- `queries`
- `migrations`

Alasan utama: domain dan repository masih menjadi kontrak lintas package, sementara repository generated SQLC masih satu output package.

## Route Registration Design

Root route composer tetap melakukan:

```go
api := cfg.App.Group("/api/v1")
api.Use(middleware.Auth(cfg.JWTSecret))
api.Use(middleware.TenantContext(cfg.JWTSecret))
```

Lalu MikroTik didelegasikan:

```go
mikrotikhandler.RegisterRoutes(api, mikrotikhandler.RouterConfig{...})
```

Package MikroTik handler tetap membuat group:

```go
mikrotik := api.Group("/mikrotik", mikrotikGuard)
```

Dengan ini route publik tidak berubah.

## Bootstrap Design

`cmd/main.go` tetap menjadi composition root, tetapi import berubah:

- `mikrotikadapter`
- `mikrotikusecase`
- `mikrotikhandler`
- `mikrotikworker`

Repository tetap dibuat dari `internal/repository`.

## Migration Strategy

Pengerjaan dilakukan berfase:

1. Adapter
2. Usecase
3. Worker
4. Handler/routes
5. VPN MikroTik
6. Full verification

Setiap fase harus compile sebelum lanjut. Jika satu fase gagal karena import cycle, rollback fase itu saja, bukan seluruh repo.

## Risk Controls

- Jangan ubah route string.
- Jangan ubah DTO JSON.
- Jangan ubah SQL atau migration.
- Jangan rename exported constructor kecuali import path saja yang berubah.
- Jangan pecah PPPoE menjadi subpackage kecil pada fase pertama.
- Jangan memindahkan OLT adapter atau OLT usecase saat pekerjaan MikroTik.
