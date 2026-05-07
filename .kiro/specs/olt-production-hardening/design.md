# Design Document

## Overview

OLT production hardening dilakukan secara incremental. Kita tidak memindahkan seluruh modul ke folder baru dalam satu langkah besar. Kita menambah struktur yang jelas di titik yang paling sering menjadi sumber bug: runtime guard, brand/model profile, ZTE index mapping, ZTE CLI/SNMP communication, monitoring store path, trap mapping, dan provisioning safety.

Desain ini mempertahankan layer aplikasi saat ini:

```text
apps/web/app/olt
  -> apps/web/app/api/network-service/[...path]/route.ts
  -> services/network-service/internal/handler
  -> services/network-service/internal/usecase
  -> services/network-service/internal/adapter
  -> services/network-service/internal/repository / metrics
```

Perubahan dilakukan dengan prinsip:

- usecase tetap brand-agnostic,
- brand logic tinggal di adapter/profile,
- transport SNMP/CLI tetap dapat dites tanpa OLT fisik,
- semua operasi live punya guard dan audit,
- UI menjadi alat debug, bukan hanya tabel data.

## Current Architecture

### UI

UI OLT sekarang berada di `apps/web/app/olt` dan komponen live di `apps/web/app/components/real-pages.tsx`. UI sudah memiliki halaman daftar, tambah, detail, ODP, dan provisioning. Namun detail OLT belum menampilkan semua data monitoring dan belum menyediakan tombol test SNMP/CLI.

### Backend Route

Route OLT berada di `services/network-service/internal/handler/router.go` di bawah guard modul `fiber_network`. Route sudah cukup luas untuk CRUD, monitoring, ODP, VLAN, service profile, provisioning, dan summary.

### Usecase

Usecase utama berada di:

- `olt_manager*.go`
- `olt_health_checker.go`
- `sync_engine.go`
- `alarm_manager*.go`
- `provisioning*.go`

Usecase sudah memiliki flow besar, tetapi beberapa path masih placeholder atau belum aman untuk live.

### Adapter

Adapter berada di `services/network-service/internal/adapter`. ZTE adalah adapter real pertama, sedangkan Huawei/FiberHome/VSOL/HSGQ masih stub.

## Proposed Architecture

### Runtime Guard

Tambahkan config baru:

```go
type Config struct {
    OLTHealthCheckEnabled bool
    OLTSyncEnabled bool
    OLTTrapEnabled bool
    OLTProvisioningWriteEnabled bool
    OLTSyncInterval int
    SNMPTrapPort int
}
```

Main wiring hanya start job sesuai guard:

```text
if OLTHealthCheckEnabled -> start health checker
if OLTTrapEnabled -> start trap receiver
if OLTSyncEnabled -> start sync engine with configured interval
if OLTProvisioningWriteEnabled == false -> provisioning usecase rejects write calls
```

Default production-safe:

- health check: false until explicitly enabled,
- sync: false until explicitly enabled,
- trap: false until explicitly enabled,
- provisioning write: false until explicitly enabled.

Untuk local/mock development, default dapat tetap nyaman lewat env compose, bukan hidden code default.

### Brand and Model Profile

Tambahkan profil di adapter/domain boundary:

```go
type OLTCapability string

type OLTBrandProfile struct {
    Brand domain.OLTBrand
    DisplayName string
    Models map[string]OLTModelProfile
}

type OLTModelProfile struct {
    Brand domain.OLTBrand
    Model string
    DisplayName string
    Capabilities map[OLTCapability]bool
    Addressing OLTAddressingProfile
    CLI OLTCLIProfile
}
```

Capability awal:

- `snmp_system_probe`
- `pon_monitoring`
- `ont_list`
- `ont_signal`
- `sfp_monitoring`
- `traffic_stats`
- `alarm_polling`
- `alarm_trap`
- `unregistered_ont`
- `ont_provisioning`
- `service_port`
- `ont_reboot`

Factory tetap interface yang sama, tetapi internalnya memakai registry:

```text
CreateAdapter(brand, model, snmpCfg, cliCfg)
  -> lookup profile
  -> if unsupported, return typed unsupported capability/model error
  -> instantiate adapter with profile
```

Jika interface existing belum siap menerima model, lakukan bridging incremental:

- tetap terima `brand` pada interface lama,
- adapter membaca `olt.Model` lewat constructor baru di usecase,
- setelah aman, baru update interface.

### Generic Probe

Auto-detect tidak boleh memakai brand-specific adapter sebelum brand diketahui.

Flow baru:

```text
Create OLT
  -> encrypt credentials
  -> insert offline record
  -> generic SNMP probe(sysDescr/sysName/sysUpTime)
  -> DetectBrand(sysDescr)
  -> DetectModel(brand, sysDescr)
  -> update OLT brand/model/firmware/status
  -> add health checker only if enabled
```

Generic probe memakai SNMPConnector langsung, bukan OLTAdapterFactory.

### ZTE Address Mapping

ZTE adapter perlu dedicated mapper:

```go
type ZTEAddress struct {
    Shelf int
    Slot int
    PON int
    ONT int
}

type ZTEIndexMapper interface {
    OLTIndex(addr ZTEAddress) (int, error)
    CLIInterface(addr ZTEAddress) (string, error)
    ParseCLIInterface(raw string) (ZTEAddress, error)
}
```

Mapper diuji dengan contoh dari `snmp-zte` dan data lapangan user.

Prinsip:

- UI boleh menampilkan PON sederhana, tetapi backend harus tahu mapping slot/board.
- Jangan asumsikan board 0.
- Jangan asumsikan hasil walk berurutan sama dengan port fisik.
- Untuk model yang belum punya mapping valid, capability write harus disabled.

### ZTE Communication Split

Struktur file target dalam folder adapter saat ini:

```text
internal/adapter/
  olt_zte_adapter.go
  olt_zte_models.go
  olt_zte_oids.go
  olt_zte_index.go
  olt_zte_parser.go
  olt_zte_commands.go
  olt_zte_monitoring.go
  olt_zte_provisioning.go
  olt_zte_session.go
```

Pembagian tanggung jawab:

- `olt_zte_oids.go`: constant OID base.
- `olt_zte_index.go`: address/index conversion.
- `olt_zte_parser.go`: parse sysDescr, CLI output, unregistered ONT, command result.
- `olt_zte_commands.go`: build command only.
- `olt_zte_session.go`: interactive session/prompt/pagination/cleanup.
- `olt_zte_monitoring.go`: SNMP monitoring orchestration.
- `olt_zte_provisioning.go`: write orchestration with guard/audit metadata.

### Transport Audit

Setiap operasi live perlu metadata:

```go
type OLTTransportAudit struct {
    OLTID string
    Brand string
    Model string
    Operation string
    Transport string
    SanitizedRequests []string
    SanitizedResponses []string
    StartedAt time.Time
    DurationMS int64
    Status string
    ErrorMessage string
    CorrelationID string
}
```

Ini dapat disimpan awalnya ke provisioning audit untuk write operation, lalu diperluas ke diagnostic/probe history jika diperlukan.

### Monitoring Data Path

Traffic handler seharusnya:

```text
validate OLT id and port
parse range
trafficStore.Query(oltID, port, from, to)
return points
```

Signal data sebaiknya disediakan:

- latest signal saat ONT list/detail,
- optional endpoint history signal per ONT.

Sync engine seharusnya:

```text
Get active OLTs
for each OLT:
  create adapter
  fetch PON ports
  fetch OLT ONTs per PON
  fetch DB ONTs for OLT
  reconcile by serial_number
  store signal
  store traffic
  detect unregistered
  detect port migration
  update total counts
```

DB repository butuh query list ONT by OLT tanpa status kosong.

### Alarm Trap

Trap handling flow:

```text
receive trap
extract source IP
lookup OLT by host/source IP
if not found:
  log unknown trap source and return
parse trap
create alarm with tenant_id and olt_id
dedupe active alarm
publish event
```

Perlu query repo:

- `GetOLTByHost`
- optional normalized host/IP matching.

### Provisioning Safety

Flow baru:

```text
validate request
check write guard
get OLT/profile/VLAN
create pending ONT or provisioning job
adapter.AddONT returns assigned ONT index
persist resolved ONT index
adapter.AddServicePort uses resolved ONT index
on partial failure:
  compensation remove ONT/service-port when safe
  or mark manual_recovery_required
audit all commands with correlation id
publish event
```

Adapter result perlu diperkaya:

```go
type AddONTResult struct {
    ProvisioningResult
    AssignedONTIndex int
}
```

Untuk menjaga interface lama, bisa ditambahkan method baru di capability interface:

```go
type ONTProvisioner interface {
    AddONT(ctx context.Context, params AddONTParams) (*AddONTResult, error)
}
```

### UI Debug Workspace

Detail OLT menjadi workspace:

```text
Overview
Connection
PON and ONT
Signal / SFP / Traffic
Alarms
Provisioning
VLAN / Service Profile
Audit / Debug
```

UI harus memperlihatkan:

- brand/model/capabilities,
- last check/sync,
- SNMP/CLI test actions,
- unsupported capability,
- error reason,
- last probe/operation status.

## Migration Strategy

1. Tambahkan config guard tanpa mengubah behavior data.
2. Tambahkan profile/capability registry.
3. Perbaiki auto-detect generic probe.
4. Kerjakan ZTE mapper/parser/command builder dengan test.
5. Ganti path monitoring placeholder.
6. Perbaiki provisioning safety.
7. Baru perkuat UI.

## Non-Goals

- Tidak mengimpor `snmp-zte` sebagai package runtime.
- Tidak menambah semua brand sekaligus.
- Tidak menjalankan destructive command ke OLT live dalam automated test.
- Tidak mengganti seluruh struktur folder network-service secara besar-besaran dalam satu commit.

## Verification

Minimum verification tiap fase:

- `go test ./...` di `services/network-service` untuk perubahan backend.
- Unit tests spesifik ZTE mapper/parser/command builder.
- Handler tests untuk traffic/signal/alarm/provisioning guard.
- UI build untuk perubahan web.
- Manual smoke hanya dengan mock mode kecuali user mengizinkan OLT live.
