# Requirements Document

## Introduction

Spec ini mendefinisikan pekerjaan hardening dan rebuild bertahap untuk modul OLT di ISPBoss. Spec ini dibuat setelah audit kode aktif pada 2026-05-07 dan pembelajaran dari folder riset `snmp-zte`.

Scope spec ini bukan membuat modul OLT dari nol. Scope-nya adalah memperkuat modul yang sudah ada agar siap dipakai secara live dengan prinsip:

- aman untuk environment produksi,
- mudah di-debug,
- multi-brand tetapi tidak penuh percabangan liar,
- ZTE C320 menjadi adapter produksi pertama,
- komunikasi SNMP/CLI mengikuti hasil riset tetapi ditulis ulang sesuai arsitektur aplikasi.

Spec awal `.kiro/specs/olt-management` tetap menjadi referensi fitur besar. Spec ini menjadi lapisan hardening dan production readiness.

## Glossary

- **OLT Runtime Guard**: config yang menentukan apakah health check, sync, trap, dan write provisioning boleh aktif.
- **Brand Profile**: metadata brand OLT seperti ZTE, Huawei, FiberHome, VSOL, HSGQ.
- **Model Profile**: metadata model spesifik di bawah brand, misalnya ZTE C320, termasuk indexing, prompt CLI, dan capability.
- **Capability**: fitur yang didukung adapter/model, misalnya monitoring SNMP, CLI provisioning, traffic counter, trap, unregistered ONT discovery.
- **Transport Audit**: metadata operasi SNMP/CLI yang aman disimpan untuk debug, termasuk command/OID yang sudah disanitasi.
- **Dry Run**: mode membangun command provisioning tanpa mengirim ke OLT.
- **Resolved ONT Index**: ONT index aktual yang akan dipakai service-port setelah add/lookup berhasil.

## Requirements

### Requirement 1: Runtime Safety Guard

**User Story:** Sebagai operator sistem, saya ingin proses OLT live bisa dinyalakan per bagian, agar aplikasi aman dijalankan walaupun perangkat OLT belum siap.

#### Acceptance Criteria

1. THE Network_Service SHALL expose config flags for OLT health check, sync engine, SNMP trap receiver, and provisioning write operations.
2. WHEN `OLT_HEALTH_CHECK_ENABLED` is false, THE Network_Service SHALL NOT start OLT health checker workers.
3. WHEN `OLT_SYNC_ENABLED` is false, THE Network_Service SHALL NOT start OLT sync engine.
4. WHEN `OLT_TRAP_ENABLED` is false, THE Network_Service SHALL NOT bind SNMP trap port.
5. WHEN `OLT_PROVISIONING_WRITE_ENABLED` is false, THE Network_Service SHALL reject add/remove/reboot/write provisioning operations with a clear domain error.
6. THE Network_Service SHALL use `OLT_SYNC_INTERVAL` from config instead of a hidden default when sync is enabled.
7. THE Network_Service SHALL log OLT runtime guard state at startup without logging credentials.

### Requirement 2: Brand and Model Profile Registry

**User Story:** Sebagai developer, saya ingin dukungan brand dan model OLT dikelola dalam registry, agar penambahan brand baru tidak membuat usecase sulit di-debug.

#### Acceptance Criteria

1. THE Network_Service SHALL define brand profiles for supported OLT brands.
2. THE Network_Service SHALL define model profiles under each brand, starting with ZTE C320.
3. EACH model profile SHALL define capability flags for monitoring, provisioning, traffic, alarm, unregistered ONT, and service-port operations.
4. THE adapter factory SHALL select adapter using brand and model profile when available.
5. WHEN a brand or model is unsupported, THE API SHALL return a clear unsupported capability response instead of pretending the feature is implemented.
6. THE usecase layer SHALL NOT branch on concrete brand names for normal OLT operations.

### Requirement 3: Generic SNMP Probe and Auto-Detect

**User Story:** Sebagai admin ISP, saya ingin registrasi OLT bisa mendeteksi brand/model dari SNMP sysDescr, agar saya tidak perlu mengisi informasi teknis secara manual.

#### Acceptance Criteria

1. THE Network_Service SHALL perform generic SNMP system probe before creating a brand-specific adapter.
2. THE generic probe SHALL read sysDescr, sysName, and sysUpTime using standard MIB OIDs.
3. THE generic probe SHALL detect brand from sysDescr using domain brand detection.
4. THE generic probe SHALL detect model using brand-specific model matchers after brand detection.
5. WHEN probe succeeds, THE OLT record SHALL be updated with brand, model, firmware, status, last_online_at, and last_checked_at.
6. WHEN probe fails, THE OLT record SHALL remain created as offline with a warning/result that can be inspected.
7. THE current live-mode bug where auto-detect calls adapter factory with an empty brand SHALL be removed.

### Requirement 4: ZTE C320 Index and Address Mapping

**User Story:** Sebagai engineer jaringan, saya ingin mapping board/PON/ONT ZTE benar dan dites, agar SNMP dan CLI tidak membaca atau menulis port yang salah.

#### Acceptance Criteria

1. THE ZTE adapter SHALL use a dedicated index/address mapper instead of hard-coded `board=0`.
2. THE mapper SHALL represent ZTE address components such as shelf/frame, slot/board, PON, and ONT where needed.
3. THE mapper SHALL include test cases based on the verified examples from `snmp-zte`.
4. THE mapper SHALL support converting UI/API PON selection into SNMP OLT index and CLI interface name.
5. THE adapter SHALL avoid using array walk order as authoritative PON port index unless validated by OID suffix parsing.
6. WHEN mapping cannot be resolved, THE adapter SHALL fail with a clear mapping error before any write command is sent.

### Requirement 5: ZTE Communication Layer

**User Story:** Sebagai developer, saya ingin komunikasi ZTE dipisah menjadi OID map, parser, command builder, dan session handling, agar error mudah dilacak.

#### Acceptance Criteria

1. THE ZTE adapter SHALL separate OID constants from OID builders.
2. THE ZTE adapter SHALL separate CLI command builders from adapter orchestration.
3. THE ZTE adapter SHALL separate CLI output parsers from transport execution.
4. THE CLI connector or ZTE session layer SHALL handle interactive command sequences that depend on config/interface mode.
5. THE ZTE session layer SHALL handle prompt matching, pagination, command echo cleanup, and timeout classification.
6. THE ZTE adapter SHALL expose sanitized operation metadata for debug and audit.
7. THE implementation SHALL use `snmp-zte` as research reference but SHALL NOT import it as a runtime dependency.

### Requirement 6: Monitoring Data Path

**User Story:** Sebagai operator, saya ingin data monitoring OLT yang tampil di API/UI berasal dari store live yang benar, agar tidak tertipu data kosong placeholder.

#### Acceptance Criteria

1. THE traffic endpoint SHALL query `TrafficStore` for requested OLT, PON port, and time range.
2. THE signal endpoint or latest signal response SHALL query `SignalStore` for ONT signal data.
3. THE sync engine SHALL reconcile ONTs detected from OLT with ONTs stored in database.
4. THE sync engine SHALL detect unmanaged/unregistered ONTs without assuming DB ONT list is empty.
5. THE sync engine SHALL detect port migration using DB state and OLT state.
6. THE sync engine SHALL run an optional immediate sync on startup when enabled by config.
7. THE API SHALL distinguish no data, sync not enabled, store unavailable, and OLT communication failure.

### Requirement 7: Alarm Trap Reliability

**User Story:** Sebagai operator NOC, saya ingin alarm trap selalu terkait tenant dan OLT yang benar, agar alarm bisa ditindaklanjuti.

#### Acceptance Criteria

1. THE trap handler SHALL map source IP/host to a registered OLT before creating an alarm record.
2. THE trap handler SHALL set tenant_id and olt_id for every persisted trap alarm.
3. WHEN source IP cannot be mapped, THE handler SHALL log and publish diagnostic information without violating RLS constraints.
4. THE alarm manager SHALL support dedupe for repeated active alarms.
5. THE alarm manager SHALL support clear events where the device reports recovery.
6. THE alarm API SHALL expose enough fields for UI troubleshooting: source, severity, type, PON, ONT, created_at, cleared_at.

### Requirement 8: Provisioning Safety

**User Story:** Sebagai operator provisioning, saya ingin operasi ONT tidak menulis service-port sebelum ONT index benar, agar konfigurasi OLT tidak rusak.

#### Acceptance Criteria

1. THE provisioning flow SHALL resolve ONT index before adding service-port.
2. THE `AddONT` adapter result SHALL include assigned or resolved ONT index.
3. THE provisioning flow SHALL persist the resolved ONT index before subsequent service-port operations.
4. WHEN add ONT succeeds but add service-port fails, THE flow SHALL execute compensation or mark state clearly for manual recovery.
5. THE flow SHALL support dry-run command preview for write operations.
6. THE flow SHALL reject write operations when write guard is disabled.
7. THE audit log SHALL store sanitized commands, responses, status, operation, brand, model, transport, and correlation_id.
8. THE decommission event SHALL preserve customer_id in emitted payload before clearing DB relation.

### Requirement 9: UI Debug Workspace

**User Story:** Sebagai admin ISP, saya ingin detail OLT menampilkan alat debug lengkap, agar saya bisa tahu masalah ada di credential, SNMP, CLI, mapping, atau data store.

#### Acceptance Criteria

1. THE OLT detail UI SHALL include tabs or sections for overview, connection test, PON/ONT, signal/SFP/traffic, alarms, provisioning, VLAN/profile, and audit/debug.
2. THE OLT detail UI SHALL provide manual test SNMP and test CLI actions.
3. THE UI SHALL show brand/model capability status.
4. THE UI SHALL show unsupported feature states clearly instead of blank data.
5. THE UI SHALL use backend DTO field names consistently.
6. THE UI SHALL show last sync/check/probe state and errors where available.

### Requirement 10: Testing and Verification

**User Story:** Sebagai maintainer, saya ingin setiap bagian OLT live punya test yang menahan regresi, agar refactor multi-brand tetap aman.

#### Acceptance Criteria

1. THE implementation SHALL include unit tests for brand/model registry.
2. THE implementation SHALL include table tests for ZTE index/address mapping.
3. THE implementation SHALL include parser tests for ZTE CLI output.
4. THE implementation SHALL include command builder tests for ZTE write operations.
5. THE implementation SHALL include usecase tests for provisioning ONT index resolution and rollback paths.
6. THE implementation SHALL include handler tests for traffic/signal/alarm error states.
7. THE implementation SHALL include config/wiring tests or startup assertions for OLT runtime guards.
8. THE implementation SHALL NOT require destructive live OLT operations in automated tests.
