# Design - MikroTik RouterOS Version Command Layer

## Current State

Modul MikroTik sudah berada di:

```text
services/network-service/internal/modules/mikrotik
```

Di dalam adapter sudah ada:

```text
adapter/command_builder_factory.go
adapter/command_builder_v6.go
adapter/command_builder_v7.go
```

PPPoE sudah memakai factory berdasarkan `router.RouterOSVersion`. Namun banyak manager lain masih menjalankan command literal langsung melalui `adapter.Execute`.

## Target Design

Tetap satu modul MikroTik:

```text
internal/modules/mikrotik/
  adapter/
  handler/
  usecase/
  worker/
```

Tambahkan version/capability layer di adapter:

```text
internal/modules/mikrotik/adapter/
  version.go
  capabilities.go
  command_builder_factory.go
  command_builder_v6.go
  command_builder_v7.go
  vpn_command_builder_factory.go
  vpn_command_builder_v6.go
  vpn_command_builder_v7.go
```

## Version Model

Tambahkan tipe internal adapter:

```go
type RouterOSMajor int

const (
    RouterOSUnknown RouterOSMajor = 0
    RouterOSv6      RouterOSMajor = 6
    RouterOSv7      RouterOSMajor = 7
)
```

Parser:

```go
func ParseRouterOSMajor(version string) RouterOSMajor
func NormalizeRouterOSVersion(version string) string
```

Aturan:

- Ambil angka mayor pertama dari string versi.
- `6.49.18 (long-term)` => v6.
- `7.14.3` => v7.
- `RouterOS 7.14.3` => v7.
- Empty/malformed => unknown.
- Unknown fallback ke v6 saat memilih command builder.

## Capability Model

Tambahkan:

```go
type RouterOSCapabilities struct {
    Major             RouterOSMajor
    SupportsWireGuard bool
}

func CapabilitiesFor(version string) RouterOSCapabilities
```

Aturan awal:

- v7: `SupportsWireGuard=true`
- v6: `SupportsWireGuard=false`
- unknown: `SupportsWireGuard=false`

Capability baru ditambah hanya saat ada fitur yang butuh keputusan versi.

## Command Builder Model

PPPoE tetap memakai interface `domain.CommandBuilder`.

Factory:

```go
func NewCommandBuilder(routerOSVersion string) domain.CommandBuilder
```

Aturan:

- v7 => `commandBuilderV7`
- v6/unknown => `commandBuilderV6`

`commandBuilderV7` boleh embed v6, tetapi setiap behavior yang memang berbeda harus override eksplisit dan punya test.

## VPN Builder Model

Ubah constructor dari:

```go
NewVPNCommandBuilder()
```

menjadi salah satu pola:

```go
NewVPNCommandBuilder(routerOSVersion string)
```

atau:

```go
NewVPNCommandBuilderFactory()
builder := factory.ForVersion(router.RouterOSVersion)
```

Pilihan yang lebih aman untuk dependency injection adalah factory, karena `vpnManager` bekerja per-router.

## Usecase Integration

PPPoE:

- Sudah punya `cmdBuilderFactory`.
- Perlu test tambahan untuk parser/capability v6 real.

VPN:

- Saat auto-configure, ambil router dan pilih builder berdasarkan `router.RouterOSVersion`.
- WireGuard check memakai capability, bukan langsung `IsRouterOSv7`.

DHCP/static/hotspot/walled garden/backup:

- Jangan ubah semua sekaligus.
- Mulai dari command write.
- Jika manager membutuhkan versi, ubah helper `connect` agar mengembalikan router juga:

```go
func connect(...) (*domain.Router, domain.RouterOSAdapter, func(), error)
```

Operational read-only:

- Boleh tetap raw pada fase pertama.
- Nanti bisa dimasukkan ke `read_command_catalog.go` jika perlu.

## Metadata Refresh

`router_usecase.go` sudah menyimpan version pada create/test connection.

Health checker perlu update metadata best-effort:

- `RouterOSVersion`
- `BoardName`
- `CPUCount`
- `TotalRAMMB`
- `Identity`

Ini penting agar router yang upgrade dari v6 ke v7 otomatis memakai builder v7 pada operasi berikutnya.

## Testing Strategy

Unit tests:

- `ParseRouterOSMajor`
- `CapabilitiesFor`
- `NewCommandBuilder` fallback
- WireGuard capability gate
- PPPoE builder untuk `6.49.18 (long-term)`

Integration-style tests:

- PPPoE manager dengan router version v6 harus memakai builder v6.
- VPN WireGuard pada v6 harus ditolak.
- Unknown version harus fallback ke v6.

Regression:

- `go test ./...` di `services/network-service`.

## Non-Goals

- Tidak memecah modul menjadi `mikrotik_v6` dan `mikrotik_v7`.
- Tidak mengubah route web/API.
- Tidak mengubah schema database pada fase awal.
- Tidak mengubah command yang belum terbukti berbeda hanya demi variasi.
