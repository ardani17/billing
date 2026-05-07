# Audit OLT Kode Aktif dan Rencana Kerja

Tanggal: 2026-05-07
Scope: modul OLT aktif di aplikasi ISPBoss, terutama `services/network-service`, UI `apps/web/app/olt`, dan referensi riset `snmp-zte`.

Dokumen ini tidak mengimplementasikan perubahan runtime. Tujuannya adalah mengunci pemahaman kode aktif, temuan risiko, rencana kerja, dan batasan desain sebelum OLT dibangun ulang secara lebih rapi mengikuti ketentuan aplikasi.

## Kesimpulan Singkat

OLT sudah menjadi modul besar yang terpisah dari billing dan MikroTik. Kerangka utama sudah ada: halaman web, route API, guard modul `fiber_network`, domain OLT/ODP/ONT, repository, migration, adapter pattern, ZTE adapter awal, health checker, sync engine, alarm manager, provisioning manager, audit log, Redis signal/traffic store, dan provisioning worker.

Namun kondisi saat ini belum layak dianggap integrasi OLT produksi yang lengkap. Banyak jalur sudah berbentuk, tetapi beberapa jalur penting masih placeholder, sebagian wiring belum memakai config, dan implementasi ZTE belum cukup mengikuti detail komunikasi dari folder riset `snmp-zte`.

Prioritas berikutnya bukan menambah banyak brand sekaligus. Prioritas yang lebih aman adalah menguatkan pondasi brand/model, ZTE C320 sebagai adapter produksi pertama, observability/debug trail, dan guard agar background job tidak diam-diam melakukan operasi live yang sulit dilacak.

## Struktur Kode Aktif Saat Ini

### UI Web

File utama:

- `apps/web/app/olt/layout.tsx`
- `apps/web/app/olt/page.tsx`
- `apps/web/app/olt/new/page.tsx`
- `apps/web/app/olt/[id]/page.tsx`
- `apps/web/app/olt/odp/page.tsx`
- `apps/web/app/olt/provisioning/page.tsx`
- `apps/web/app/settings/olt/page.tsx`
- `apps/web/app/components/real-pages.tsx`

Halaman OLT saat ini mengambil data dari proxy `/api/network-service/...`. Halaman yang sudah ada:

- Daftar OLT dan summary.
- Form tambah OLT.
- Detail OLT dasar.
- Alarm OLT.
- Daftar ODP.
- Daftar ONT provisioning.
- Settings OLT generik.

Catatan audit UI:

- UI detail belum menampilkan PON ports, ONT list per PON, SFP, traffic, capacity, unregistered ONT, VLAN, service profile, dan audit log sebagai workspace debug yang lengkap.
- Form tambah OLT tidak mengirim `brand`/`model`; auto-detect di backend seharusnya mengisi, tetapi auto-detect sekarang bermasalah.
- Pesan UI mengatakan test manual dari detail OLT, tetapi detail belum menyediakan tombol test SNMP/CLI.
- Beberapa field UI memakai fallback nama lama seperti `last_sync_at`, `location`, `pon_port`, `signal_dbm`, sementara DTO backend memakai `updated_at`, `pon_port_index`, dan field lain. Ini membuat data mudah terlihat kosong walaupun backend mengirim field yang benar.

### Proxy

File:

- `apps/web/app/api/network-service/[...path]/route.ts`

Proxy meneruskan request UI ke network-service. Modul OLT di web tidak memanggil backend langsung.

### Route Backend

File:

- `services/network-service/internal/handler/router.go`

Route OLT berada di bawah guard modul `fiber_network`:

- `/api/v1/olt/devices`
- `/api/v1/olt/devices/:id/test-snmp`
- `/api/v1/olt/devices/:id/test-cli`
- `/api/v1/olt/devices/:id/pon-ports`
- `/api/v1/olt/devices/:id/pon-ports/:port/onts`
- `/api/v1/olt/devices/:id/pon-ports/:port/traffic`
- `/api/v1/olt/devices/:id/alarms`
- `/api/v1/olt/devices/:id/sfp`
- `/api/v1/olt/devices/:id/capacity`
- `/api/v1/olt/odp`
- `/api/v1/olt/provisioning`
- `/api/v1/olt/vlans`
- `/api/v1/olt/service-profiles`
- `/api/v1/olt/summary`

Catatan audit route:

- Batas modul sudah benar: OLT dan peta fiber memakai `fiber_network`.
- Route monitoring dan provisioning sudah cukup luas.
- Belum ada endpoint debug terstruktur untuk melihat adapter profile, capability, OID/command sanitasi, atau hasil probe terakhir.

### Domain

File utama:

- `services/network-service/internal/domain/olt.go`
- `services/network-service/internal/domain/olt_adapter_types.go`
- `services/network-service/internal/domain/olt_dto.go`
- `services/network-service/internal/domain/olt_alarm.go`
- `services/network-service/internal/domain/olt_event.go`
- `services/network-service/internal/domain/odp.go`
- `services/network-service/internal/domain/ont.go`
- `services/network-service/internal/domain/provisioning_adapter_types.go`
- `services/network-service/internal/domain/provisioning_dto.go`
- `services/network-service/internal/domain/repository.go`
- `services/network-service/internal/domain/repository_provisioning.go`

Domain sudah memisahkan:

- OLT device registry.
- Brand constants: ZTE, Huawei, FiberHome, VSOL, HSGQ.
- SNMP/CLI config.
- Adapter response types.
- ODP.
- ONT lifecycle.
- Provisioning result dan params.
- Signal level.
- Alarm event.

Catatan audit domain:

- Brand masih hanya string enum, belum ada profile model/capability per brand.
- `AddONTParams` menyebut `ONTIndex` auto-assign jika 0, tetapi `ProvisioningResult` tidak membawa ONT index hasil assignment. Ini menjadi akar bug service-port yang memakai index 0.
- Belum ada tipe eksplisit untuk PON address seperti `shelf/slot/pon` atau `frame/slot/port`, sehingga kode ZTE hard-code ke bentuk `gpon-olt_1/{port}` dan `zteCalculateOLTIndex(0, portIndex)`.
- Belum ada capability map untuk membedakan fitur monitor/provisioning per brand dan per model.

### Adapter dan Connector

File utama:

- `services/network-service/internal/adapter/snmp_connector.go`
- `services/network-service/internal/adapter/cli_connector.go`
- `services/network-service/internal/adapter/cli_connector_telnet.go`
- `services/network-service/internal/adapter/olt_adapter_factory.go`
- `services/network-service/internal/adapter/olt_mock_adapter.go`
- `services/network-service/internal/adapter/olt_zte_adapter.go`
- `services/network-service/internal/adapter/olt_zte_oids.go`
- `services/network-service/internal/adapter/olt_zte_helpers.go`
- `services/network-service/internal/adapter/olt_zte_ont.go`
- `services/network-service/internal/adapter/olt_zte_provisioning.go`
- `services/network-service/internal/adapter/olt_huawei_adapter.go`
- `services/network-service/internal/adapter/olt_fiberhome_adapter.go`
- `services/network-service/internal/adapter/olt_vsol_adapter.go`
- `services/network-service/internal/adapter/olt_hsgq_adapter.go`

Yang sudah ada:

- SNMP v2c/v3 connector memakai `gosnmp`.
- CLI connector SSH/Telnet.
- Factory adapter berdasarkan brand.
- Mock adapter untuk `NETWORK_MODE=mock`.
- ZTE adapter awal untuk system info, ping, PON ports, ONT list/signal, alarm, SFP, traffic, provisioning command.
- Brand non-ZTE masih stub.

Catatan audit adapter:

- Factory live mode menolak brand kosong. Tetapi create OLT mencoba auto-detect dengan `CreateAdapter("", ...)`, sehingga auto-detect gagal sebelum bisa membaca `sysDescr`.
- `GetAllPONPorts` ZTE berjalan dari walk `ifAdminStatus`/`ifOperStatus` dan memberi `PortIndex` berdasarkan urutan array, bukan dari OID/interface index. Ini berisiko salah mapping.
- ZTE index memakai `zteCalculateOLTIndex(0, portIndex)`. Riset `snmp-zte` menunjukkan C320 lazim memakai board/pon yang tidak boleh diasumsikan 0-based. Contoh riset: board 1, PON 1 menghasilkan index berbeda dari board 0, PON 0.
- CLI ZTE command builder masih tertanam langsung di adapter. Belum ada builder/parser per model, sehingga debug command sulit.
- CLI connector Telnet membaca sampai prompt umum `#`, `>`, `$` dengan `bufio.Scanner`; belum ada penanganan prompt ZTE, pagination, echo cleanup, privilege/config mode, atau prompt per mode seperti riset `snmp-zte`.
- SSH `ExecuteMultiple` tidak benar-benar satu interactive session; ia loop memanggil `executeSSH` per command. Command yang butuh state seperti `interface ...` lalu `onu ...` bisa gagal pada OLT yang membutuhkan shell interaktif.
- Adapter non-ZTE dikembalikan factory tetapi method-nya stub. Secara UX ini terlihat "brand didukung", padahal belum produksi.

### Usecase OLT

File utama:

- `services/network-service/internal/usecase/olt_manager.go`
- `services/network-service/internal/usecase/olt_manager_crud.go`
- `services/network-service/internal/usecase/olt_manager_ops.go`
- `services/network-service/internal/usecase/olt_manager_update.go`
- `services/network-service/internal/usecase/olt_health_checker.go`
- `services/network-service/internal/usecase/olt_health_check.go`
- `services/network-service/internal/usecase/olt_capacity.go`
- `services/network-service/internal/usecase/sync_engine.go`
- `services/network-service/internal/usecase/sync_engine_helpers.go`
- `services/network-service/internal/usecase/alarm_manager.go`
- `services/network-service/internal/usecase/alarm_manager_trap.go`
- `services/network-service/internal/usecase/alarm_manager_parse.go`

Yang sudah ada:

- Create/list/get/update/delete OLT.
- Encrypt/decrypt SNMP dan CLI credential.
- Test SNMP dan CLI.
- Health checker per OLT.
- Status summary.
- PON ports, ONT list, SFP, capacity.
- Sync engine untuk PON/ONT/signal/traffic.
- Alarm polling dan trap receiver.

Catatan audit usecase:

- Auto-detect create OLT gagal di live mode karena adapter dibuat dengan brand kosong.
- Summary tidak menerima tenantID secara eksplisit, tetapi mengandalkan RLS context. Ini bisa diterima jika context selalu diset, tetapi perlu test multi-tenant.
- `GetTraffic` handler masih mengembalikan data kosong placeholder, padahal Redis `TrafficStore` sudah ada.
- Sync engine membandingkan ONT dari OLT dengan `dbONTs` kosong placeholder, sehingga semua ONT fisik terlihat unmanaged/unregistered secara konseptual.
- Sync engine tidak run immediate sync saat start, hanya ticker berikutnya.
- Sync interval config tersedia (`OLT_SYNC_INTERVAL`) tetapi wiring main memakai interval default constructor.
- Health checker/sync/trap dimulai otomatis saat service start. Belum ada flag enable/disable modul operasional OLT untuk environment yang belum siap OLT live.
- Trap receiver default port 162 berisiko butuh privilege di beberapa OS dan bisa gagal diam-diam jika port dipakai.
- Trap handling parse source IP tetapi tidak map source IP ke OLT/tenant sebelum insert alarm. Karena `olt_alarms.tenant_id` dan `olt_id` NOT NULL, jalur trap berisiko gagal insert atau tidak berguna.
- Capacity growth rate masih placeholder 0.

### Provisioning

File utama:

- `services/network-service/internal/usecase/provisioning_manager.go`
- `services/network-service/internal/usecase/provisioning_provision.go`
- `services/network-service/internal/usecase/provisioning_decommission.go`
- `services/network-service/internal/usecase/provisioning_reboot.go`
- `services/network-service/internal/usecase/provisioning_auto.go`
- `services/network-service/internal/usecase/provisioning_bulk.go`
- `services/network-service/internal/usecase/provisioning_queries.go`
- `services/network-service/internal/handler/provisioning_handler.go`
- `services/network-service/internal/handler/provisioning_handler_bulk.go`
- `services/network-service/internal/handler/provisioning_handler_settings.go`
- `services/network-service/internal/worker/provisioning_worker.go`

Yang sudah ada:

- Single ONT provisioning.
- Bulk validation/execution.
- Decommission.
- Reboot.
- Auto-provisioning skeleton.
- Port migration skeleton.
- Customer terminated worker untuk auto-decommission.
- Provisioning audit log.

Catatan audit provisioning:

- `ProvisionONT` membuat ONT DB dengan `ONTIndex: 0`, lalu `AddServicePort` memakai `ont.ONTIndex`, bukan hasil ONT index dari OLT. Ini berisiko membuat service-port ke ONT index 0.
- `AddONT` di ZTE command juga memakai `params.ONTIndex`, padahal request tidak mengisi index. Command bisa menjadi `onu 0 ...`, yang belum tentu valid.
- Tidak ada rollback/compensation jika `AddONT` sukses tetapi `AddServicePort` gagal.
- `DecommissionONT` mengosongkan `CustomerID` sebelum event payload mengambil customer id, sehingga event bisa kehilangan customer id.
- Audit log sudah ada, tetapi belum ada sanitasi command/response yang eksplisit untuk credential atau data sensitif.
- `HandlePortMigration` dan `ConfirmMigration` masih placeholder; belum ada tabel pending migration.
- Auto-provisioning mencari `existingONT` setelah membuat record baru dengan serial yang sama; karena serial unik per tenant, logika ini perlu ditata ulang agar tidak bertabrakan dengan record baru.

### Repository, Migration, dan Storage

File utama:

- `services/network-service/migrations/000008_create_olts.up.sql`
- `services/network-service/migrations/000009_create_odps.up.sql`
- `services/network-service/migrations/000010_create_olt_alarms.up.sql`
- `services/network-service/migrations/000011_create_vlans.up.sql`
- `services/network-service/migrations/000012_create_service_profiles.up.sql`
- `services/network-service/migrations/000013_create_onts.up.sql`
- `services/network-service/migrations/000014_create_provisioning_audit_logs.up.sql`
- `services/network-service/migrations/000015_create_provisioning_settings.up.sql`
- `services/network-service/queries/olts.sql`
- `services/network-service/queries/odps.sql`
- `services/network-service/queries/olt_alarms.sql`
- `services/network-service/queries/onts.sql`
- `services/network-service/queries/provisioning_audit_logs.sql`
- `services/network-service/internal/metrics/signal_store.go`
- `services/network-service/internal/metrics/traffic_store.go`

Yang sudah ada:

- Tabel OLT, ODP, alarms, VLAN, service profile, ONT, audit logs, settings.
- RLS untuk tabel-tabel utama.
- Redis sorted-set store untuk signal dan traffic.

Catatan audit data:

- `onts` unik pada `(olt_id, pon_port_index, ont_index)`, sehingga ONT index 0 dari provisioning dapat menjadi bottleneck dan gagal untuk ONT kedua pada PON sama.
- Tidak ada tabel untuk device model profile/capability.
- Tidak ada tabel/log untuk probe history, last SNMP/CLI test result, sanitized command trace, atau adapter diagnostics.
- Redis traffic/signal store sudah ada, tetapi endpoint bacanya belum lengkap di handler/UI.

### Wiring Runtime

File utama:

- `services/network-service/cmd/main.go`
- `services/network-service/internal/config/config.go`

Yang sudah ada:

- OLT repository, adapter factory, signal/traffic store, event publisher, OLT manager, ODP manager, health checker, alarm manager, sync engine, provisioning manager, handlers, dan worker di-wire di main.
- Config punya `NETWORK_MODE`, `OLT_HEALTH_CHECK_INTERVAL`, `OLT_SYNC_INTERVAL`, `SNMP_TRAP_PORT`.

Catatan audit wiring:

- `OLT_SYNC_INTERVAL` belum jelas dipakai oleh `NewSyncEngine`; main masih terlihat memakai default sync interval.
- Health checker, trap receiver, dan sync engine dimulai otomatis. Untuk fase riset/live OLT, perlu guard eksplisit agar tidak melakukan polling/trap binding tanpa persetujuan environment.
- Tidak ada mode "read-only OLT live" vs "provisioning write enabled". Padahal provisioning CLI adalah operasi berisiko.

## Status Integrasi `snmp-zte`

Folder `snmp-zte` belum terintegrasi sebagai dependency runtime. Ia tidak masuk `go.work`, tidak menjadi import di `services/network-service`, dan tidak distart oleh docker/app. Yang sudah terjadi adalah sebagian ide OID dan command dari riset tampaknya disalin manual ke ZTE adapter.

Nilai penting dari `snmp-zte`:

- Riset komunikasi SNMP ZTE C320 lebih detail.
- Ada pola CLI Telnet ZTE yang lebih realistis.
- Ada fitur read yang lebih banyak: board info, empty slots, fan/temp, VLAN/profile list, ONU detail, traffic, errors.
- Ada command write: create/delete/rename ONU, VLAN/profile/service operations.

Prinsip porting:

- Jangan import folder riset mentah-mentah ke aplikasi.
- Ambil konsep komunikasi, OID map, parser, dan command builder yang valid.
- Bentuk ulang ke style aplikasi: domain/usecase/adapter/repository, tenant-aware, config-aware, testable, sanitized audit, dan module guard.

## Temuan Risiko Prioritas

### P0 - Harus dibereskan sebelum live provisioning

1. Auto-detect live mode gagal karena brand kosong masuk factory.
   - Lokasi: `services/network-service/internal/usecase/olt_manager_crud.go`
   - Dampak: OLT baru tetap offline/brand kosong walaupun SNMP benar.

2. ONT index provisioning tidak pernah di-resolve.
   - Lokasi: `services/network-service/internal/usecase/provisioning_provision.go`
   - Dampak: `AddONT`/`AddServicePort` dapat memakai ONT index 0 dan menulis konfigurasi salah.

3. ZTE index dan address model masih hard-code.
   - Lokasi: `services/network-service/internal/adapter/olt_zte_oids.go`, `olt_zte_ont.go`, `olt_zte_adapter.go`
   - Dampak: SNMP query bisa membaca port/ONT yang salah pada C320/C300/C600.

4. CLI multi-command belum benar-benar interactive shell untuk SSH dan Telnet belum ZTE-aware.
   - Lokasi: `services/network-service/internal/adapter/cli_connector.go`, `cli_connector_telnet.go`, `olt_zte_provisioning.go`
   - Dampak: command mode seperti `interface ...` lalu `onu ...` bisa gagal atau output tidak bisa diparse.

5. Tidak ada guard write operation.
   - Dampak: jika app masuk live, provisioning/decommission/reboot dapat dipanggil tanpa mode operasional yang cukup eksplisit.

### P1 - Harus dibereskan sebelum monitoring dipercaya

1. Traffic endpoint masih placeholder.
   - Lokasi: `services/network-service/internal/handler/olt_handler_monitoring.go`
   - Dampak: UI/API mengembalikan data kosong walaupun sync menyimpan ke Redis.

2. Sync engine belum reconcile dengan DB ONT sebenarnya.
   - Lokasi: `services/network-service/internal/usecase/sync_engine.go`
   - Dampak: semua ONT fisik dapat dianggap unmanaged, port migration tidak akurat.

3. Trap alarm tidak map source IP ke OLT/tenant.
   - Lokasi: `services/network-service/internal/usecase/alarm_manager_trap.go`
   - Dampak: insert alarm bisa gagal atau alarm tidak bisa dikaitkan tenant/OLT.

4. `OLT_SYNC_INTERVAL` dan runtime guard belum dipakai secara konsisten.
   - Lokasi: `services/network-service/cmd/main.go`, `internal/config/config.go`
   - Dampak: behavior background job sulit diprediksi di deploy.

5. PON port mapping dari `ifAdminStatus`/`ifOperStatus` belum filter interface GPON.
   - Dampak: daftar port bisa berisi interface non-PON.

### P2 - Penting untuk debuggability dan ekspansi brand

1. Belum ada brand/model profile dan capability registry.
2. Brand non-ZTE terlihat ada, tetapi masih stub.
3. Command builder/parser belum dipisah dari adapter.
4. UI belum menyediakan workspace debug operator.
5. Audit log provisioning belum menyimpan transport/OID/command metadata yang cukup untuk trace.
6. Capacity planning growth masih placeholder.
7. Settings OLT masih generic, belum mengatur behavior OLT sebenarnya.

## Struktur Target yang Disarankan

Untuk menghindari kode sulit di-debug saat banyak brand/model masuk, struktur target sebaiknya tetap mengikuti layer aplikasi, tetapi menambah pemisahan di adapter:

```text
services/network-service/internal/adapter/
  olt_adapter_factory.go
  olt_brand_profile.go
  olt_capabilities.go
  olt_probe.go
  olt_transport_audit.go
  olt_zte_adapter.go
  olt_zte_models.go
  olt_zte_oids.go
  olt_zte_index.go
  olt_zte_parser.go
  olt_zte_commands.go
  olt_zte_monitoring.go
  olt_zte_provisioning.go
  olt_huawei_adapter.go
  olt_huawei_models.go
  ...
```

Aturan desain:

- Usecase tidak boleh berisi `if brand == zte` untuk operasi normal.
- Factory memilih adapter berdasarkan brand/model profile.
- Adapter menerima transport SNMP/CLI, tetapi tidak menyimpan data DB.
- Command builder hanya membangun command, parser hanya membaca output, adapter mengorkestrasi.
- Semua operasi live punya metadata: brand, model, transport, operation, sanitized command atau OID, latency, status.
- Model profile menyimpan perbedaan seperti slot/board/PON indexing, prompt CLI, pagination, max ONT per port, dan capability.

## Planning Implementasi Bertahap

### Fase 0 - Freeze dan dokumentasi

Output:

- Dokumen audit ini.
- Spec pekerjaan `olt-production-hardening`.
- Tidak ada perubahan runtime.

Tujuan:

- Menyamakan pemahaman struktur.
- Mengunci urutan kerja agar tidak implementasi acak.

### Fase 1 - Runtime safety dan config guard

Output:

- Config OLT operasional:
  - `OLT_HEALTH_CHECK_ENABLED`
  - `OLT_SYNC_ENABLED`
  - `OLT_TRAP_ENABLED`
  - `OLT_PROVISIONING_WRITE_ENABLED`
  - `OLT_SYNC_INTERVAL`
  - `SNMP_TRAP_PORT`
- Main wiring memakai config tersebut.
- Startup log menyatakan OLT background job aktif/nonaktif.

Tujuan:

- App aman dijalankan pada environment yang belum siap OLT live.
- Provisioning write tidak aktif tanpa keputusan eksplisit.

### Fase 2 - Brand/model profile dan capability registry

Output:

- `OLTBrandProfile` dan `OLTModelProfile`.
- Capability map per brand/model:
  - SNMP monitoring.
  - CLI provisioning.
  - unregistered ONT discovery.
  - service-port operations.
  - traffic counters.
  - alarm polling/trap.
- Factory memakai brand/model profile.
- UI/API dapat menampilkan fitur supported/unsupported.

Tujuan:

- Multi-brand tetap mudah di-debug.
- Stub brand tidak terlihat seperti full support.

### Fase 3 - ZTE C320 production adapter

Output:

- ZTE index/address abstraction.
- Test table untuk board/pon/onu index dari riset `snmp-zte`.
- SNMP OID map dirapikan.
- ZTE parser untuk system info, PON, ONT list, signal, SFP, traffic, unregistered ONT.
- ZTE command builder untuk add/remove/reboot/service-port.
- CLI session ZTE-aware: prompt, pagination, config mode, cleanup output.

Tujuan:

- ZTE menjadi adapter produksi pertama, bukan sekadar subset copy.

### Fase 4 - Monitoring data path

Output:

- Traffic endpoint membaca `TrafficStore`.
- Signal endpoint atau latest signal path yang jelas.
- Sync engine reconcile DB ONT.
- Port migration pending state.
- Alarm trap map source IP ke OLT/tenant.
- Alarm dedupe dan clear behavior.

Tujuan:

- Monitoring yang tampil di UI benar-benar berasal dari data live/store.

### Fase 5 - Provisioning safety

Output:

- Resolve ONT index sebelum service-port.
- `AddONT` result membawa assigned ONT index.
- Rollback/compensation saat sebagian langkah gagal.
- Dry-run/preview command.
- Write guard.
- Sanitized audit log.
- Better event payload.

Tujuan:

- Operasi write ke OLT bisa dipertanggungjawabkan dan bisa diaudit.

### Fase 6 - UI debug workspace

Output:

- Detail OLT dengan tab:
  - Overview.
  - Connection test.
  - PON/ONT.
  - Signal/SFP/traffic.
  - Alarm.
  - Provisioning.
  - VLAN/profile.
  - Audit/debug.
- Tombol test SNMP/CLI.
- Tampilan capability per brand/model.
- Error state jelas, bukan data kosong diam-diam.

Tujuan:

- Operator bisa tahu masalah ada di credential, SNMP OID, CLI prompt, brand profile, atau data store.

### Fase 7 - Brand berikutnya

Output:

- Baru setelah ZTE stabil, tambah Huawei/FiberHome/VSOL/HSGQ satu per satu.
- Setiap brand wajib punya profile, command builder/parser, adapter tests, dan UI capability.

Tujuan:

- Ekspansi brand tidak merusak adapter ZTE dan tidak membuat usecase penuh percabangan.

## Spec Pekerjaan

Spec formal dibuat di:

- `.kiro/specs/olt-production-hardening/requirements.md`
- `.kiro/specs/olt-production-hardening/design.md`
- `.kiro/specs/olt-production-hardening/tasks.md`

Spec lama `.kiro/specs/olt-management` tetap dianggap sebagai spec awal. Spec baru ini adalah hardening/rebuild plan berdasarkan audit kode aktif dan riset `snmp-zte`.

## Acceptance Criteria Audit

Pekerjaan OLT dianggap siap masuk implementasi jika:

- Semua temuan P0 masuk task spec.
- Semua operasi live punya guard config.
- ZTE C320 punya mapping index yang dites.
- Provisioning tidak memakai ONT index 0 tanpa resolve.
- Traffic/signal/alarm punya data path baca/tulis yang jelas.
- UI detail memberi alat debug yang cukup.
- Brand non-ZTE tidak ditampilkan sebagai full support sebelum adapter nyata selesai.
