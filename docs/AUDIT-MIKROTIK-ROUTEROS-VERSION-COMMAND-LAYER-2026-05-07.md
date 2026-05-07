# Audit MikroTik RouterOS Version Command Layer - 2026-05-07

## Tujuan

Audit ini mengecek cara aplikasi membedakan RouterOS v6 dan v7 setelah kode MikroTik dipindahkan ke `services/network-service/internal/modules/mikrotik`.

Konteks penting: router MikroTik yang saat ini dipakai testing adalah RouterOS v6, sehingga fallback default harus tetap aman untuk v6.

## Kesimpulan

Fondasi version-aware sudah ada, tetapi belum cukup luas.

Yang sudah ada:

- Versi RouterOS disimpan di `routers.router_os_version`.
- `Create` dengan `TestOnCreate` dan `TestConnection` membaca versi dari `/system/resource/print`.
- PPPoE memakai `NewCommandBuilder(router.RouterOSVersion)`.
- Sudah ada file builder terpisah untuk `command_builder_v6.go` dan `command_builder_v7.go`.
- WireGuard sudah ditolak untuk router non-v7.

Yang belum aman untuk pengembangan jangka panjang:

- Parser versi masih `strings.HasPrefix(version, "7")`.
- `commandBuilderV7` masih embed v6 dan belum punya perbedaan eksplisit kecuali komentar/override minimal.
- Banyak command MikroTik masih hardcoded langsung di usecase, bukan lewat version-aware command layer.
- VPN command builder belum version-aware. Ia langsung membuat command WireGuard/L2TP/PPTP/SSTP/OpenVPN tanpa factory berdasarkan versi.
- Health checker membaca system resource tetapi belum menyegarkan `RouterOSVersion`, `BoardName`, dan metadata resource lain ke tabel router.
- Belum ada capability object yang menjawab fitur seperti `supports_wireguard`, `supports_routing_table`, atau fallback v6.

## Temuan Detail

### 1. Deteksi versi sudah ada, tetapi masih rapuh

File: `services/network-service/internal/domain/constants.go`

Saat ini:

```go
func IsRouterOSv7(version string) bool {
    return strings.HasPrefix(version, "7")
}
```

Risiko:

- String seperti `" 7.14.3"` akan dianggap bukan v7.
- String `"RouterOS 7.14.3"` akan dianggap bukan v7.
- String kosong langsung fallback ke v6 tanpa sinyal status "unknown".
- Tidak ada helper `MajorVersion`, `IsRouterOSv6`, atau `NormalizeRouterOSVersion`.

Rekomendasi:

- Tambah parser versi di modul MikroTik adapter/domain yang mengambil angka mayor pertama dari string versi.
- Unknown tetap fallback ke v6 untuk command destructive, tetapi statusnya harus bisa dibaca sebagai `unknown`.

### 2. PPPoE sudah paling siap untuk v6/v7

File:

- `services/network-service/internal/modules/mikrotik/adapter/command_builder_factory.go`
- `services/network-service/internal/modules/mikrotik/adapter/command_builder_v6.go`
- `services/network-service/internal/modules/mikrotik/adapter/command_builder_v7.go`
- `services/network-service/internal/modules/mikrotik/usecase/pppoe_manager.go`

Alur saat ini:

- `pppoeManager` menerima `cmdBuilderFactory`.
- `buildCommandBuilder(router)` memakai `router.RouterOSVersion`.
- PPPoE CRUD, sync, session, isolir, unisolir, suspend, package change, dan profile sync memakai builder untuk banyak command inti.

Risiko:

- `commandBuilderV7` masih menumpang ke v6.
- Tidak ada capability gate untuk command yang benar-benar v7-only.
- Test v6/v7 saat ini mostly memverifikasi command sama, bukan membuktikan command yang berbeda.

Rekomendasi:

- Pertahankan PPPoE sebagai prioritas pertama karena router testing v6.
- Tambahkan test fixture v6 `6.49.18 (long-term)` sesuai router testing.
- Tambahkan case versi v7 yang memang beda jika ditemukan.

### 3. Raw command masih luas di manager operasional

File:

- `mikrotik_operational.go`
- `dhcp_manager.go`
- `static_ip_manager.go`
- `hotspot_manager.go`
- `walled_garden_manager.go`
- `backup_manager.go`
- `terminal_manager.go`
- `router_usecase.go`

Contoh raw command:

- `/interface/print`
- `/interface/monitor-traffic`
- `/ip/pool/print`
- `/ip/pool/used/print`
- `/ip/firewall/nat/print`
- `/ip/firewall/filter/print`
- `/ip/firewall/address-list/print`
- `/ip/dhcp-server/lease/add`
- `/ip/hotspot/user/add`
- `/queue/simple/add`
- `/export`
- `/system/package/print`
- `/system/reboot`

Risiko:

- Jika command/parameter berbeda di v7, perubahan akan tersebar di banyak file.
- Sulit menulis test v6/v7 karena path command tidak punya satu pusat.
- Sebagian manager hanya mengambil adapter dari `connect`, tidak membawa `router.RouterOSVersion`.

Rekomendasi:

- Buat command catalog/versioned builder tambahan, bukan langsung memecah modul.
- Ubah helper `connect` bertahap agar mengembalikan `*domain.Router` juga untuk manager yang butuh memilih command berdasarkan versi.

### 4. VPN perlu dipisahkan capability-nya

File:

- `vpn_command_builder.go`
- `vpn_command_builder_common.go`
- `vpn_manager_create.go`

Yang sudah aman:

- `validateRouterVersion` menolak WireGuard jika `!IsRouterOSv7`.

Yang belum:

- `NewVPNCommandBuilder()` tidak menerima versi router.
- L2TP/PPTP/SSTP/OpenVPN command tidak punya builder v6/v7.
- Script generator tidak jelas memakai template version-aware atau capability-aware.

Rekomendasi:

- Tambah `NewVPNCommandBuilder(routerOSVersion string)`.
- Tambah capability check: WireGuard v7-only.
- Untuk v6 testing, pastikan seluruh protokol non-WireGuard tetap tidak rusak.

### 5. Metadata version belum selalu fresh

File:

- `router_usecase.go`
- `health_checker.go`
- `live_adapter.go`

Saat ini:

- `Create` dengan `TestOnCreate` menyimpan version.
- `TestConnection` menyimpan version.
- `GetSystemResource` membaca version lengkap dari RouterOS.
- Health checker membaca system resource untuk metrics, tetapi tidak menyimpan version/board/CPU/RAM ke router.

Risiko:

- Jika router di-upgrade dari v6 ke v7, command builder tetap memakai versi lama sampai user menjalankan test connection manual.

Rekomendasi:

- Health checker sukses harus menyegarkan `RouterOSVersion`, `BoardName`, `CPUCount`, `TotalRAMMB`, dan `Identity` secara best-effort.
- Tambahkan `router_os_major` hanya jika benar-benar dibutuhkan; fase awal cukup parser dari string.

## Rekomendasi Arsitektur

Jangan buat dua modul besar `mikrotik_v6` dan `mikrotik_v7`.

Gunakan satu modul:

```text
internal/modules/mikrotik/
  adapter/
    version.go
    capabilities.go
    command_builder_factory.go
    command_builder_v6.go
    command_builder_v7.go
    vpn_command_builder_factory.go
    vpn_command_builder_v6.go
    vpn_command_builder_v7.go
```

Business usecase tetap satu:

```text
internal/modules/mikrotik/usecase/
```

Command/action yang berbeda antar versi dipusatkan di adapter command layer.

## Prioritas Pengerjaan

1. Parser versi dan capability object.
2. Perkuat PPPoE builder dan test v6 karena router testing saat ini v6.
3. Refactor VPN builder agar version-aware.
4. Migrasikan command raw berisiko tinggi: DHCP lease, static IP queue/address-list, walled garden firewall, hotspot user.
5. Operational read-only command boleh belakangan karena risikonya lebih rendah.
6. Health checker refresh metadata version agar upgrade v6 ke v7 otomatis terbaca.

## Acceptance Criteria

- Router v6 testing tetap menjalankan PPPoE sync/CRUD dengan command v6.
- Versi `"6.49.18 (long-term)"` diparse sebagai major 6.
- Versi `"7.14.3"` diparse sebagai major 7.
- Versi kosong/unknown fallback ke v6 untuk command execution.
- WireGuard tetap ditolak untuk v6.
- `go test ./...` di `services/network-service` lulus.
- Build web tidak wajib untuk command-layer backend, tetapi boleh dijalankan sebagai sanity check.
