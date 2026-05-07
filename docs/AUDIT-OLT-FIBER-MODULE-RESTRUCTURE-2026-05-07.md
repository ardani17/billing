# Audit OLT Fiber Module Restructure - 2026-05-07

## Tujuan

Memisahkan kode OLT dari package root `internal/adapter`, `internal/handler`, `internal/usecase`, dan `internal/worker` ke folder modul khusus agar pengembangan OLT/ZTE dan brand berikutnya tidak bercampur dengan MikroTik atau FTTH mapping.

## Temuan Audit

Sebelum restruktur:

- MikroTik sudah terstruktur di `services/network-service/internal/modules/mikrotik`.
- OLT masih tersebar di root package:
  - `internal/adapter/olt_*`, `snmp_connector*`, `cli_connector*`
  - `internal/handler/olt_*`, `odp_*`, `provisioning_*`, `vlan_*`, `service_profile_*`
  - `internal/usecase/olt_*`, `alarm_*`, `sync_engine*`, `provisioning_*`, `odp_*`, `vlan_*`, `service_profile_*`
  - `internal/worker/provisioning_*`
- Root router masih mendaftarkan route OLT langsung, sehingga makin sulit dibaca ketika OLT, MikroTik, dan mapping bertambah kompleks.

## Struktur Baru

Kode OLT/Fiber sekarang berada di:

```text
services/network-service/internal/modules/fiber/
  adapter/   # adapter OLT, SNMP/CLI connector, brand profile, ZTE mapper/parser/commands
  handler/   # OLT, ODP, provisioning, VLAN, service profile handlers dan route module
  usecase/   # OLT manager, alarm, sync engine, provisioning, ODP, VLAN, service profile
  worker/    # provisioning event worker
```

Root `internal/handler/router.go` sekarang mendelegasikan route OLT ke:

```go
fiberhandler.RegisterRoutes(api, fiberhandler.RouterConfig{...})
```

Pola ini sengaja dibuat sejajar dengan:

```text
services/network-service/internal/modules/mikrotik/
```

## Batas Modul

Tetap di root/shared:

- `internal/domain`: kontrak domain dan interface lintas modul.
- `internal/repository`: akses database/sqlc yang masih dipakai lintas modul.
- `internal/metrics`: Redis metrics store untuk router dan OLT.
- `internal/handler`: root router, health, dan FTTH mapping handler yang belum menjadi modul terpisah.

Masuk modul fiber:

- Komunikasi OLT: adapter, SNMP, CLI, ZTE-specific code.
- HTTP surface OLT: devices, ODP, provisioning, VLAN, service profile.
- Business logic OLT: health checker, alarm, sync, provisioning, capacity, profile/VLAN.
- Worker OLT provisioning.

## Prinsip Debugging

- Masalah komunikasi OLT/ZTE dimulai dari `modules/fiber/adapter`.
- Masalah API response atau validasi dimulai dari `modules/fiber/handler`.
- Masalah flow bisnis, sync, alarm, dan provisioning dimulai dari `modules/fiber/usecase`.
- Masalah event customer termination untuk ONT dimulai dari `modules/fiber/worker`.
- Masalah schema/query tetap dicek di `internal/repository` dan `services/network-service/queries`.

## Verifikasi

- `go test ./...` di `services/network-service`: lulus.
