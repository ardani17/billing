# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **OLT Management Layer** di `services/network-service/`. Layer ini dibangun di atas **MikroTik Router Foundation Layer** (spec `mikrotik-router`) dan **VPN Tunnel Management Layer** (spec `mikrotik-vpn`) yang sudah diimplementasikan.

OLT (Optical Line Terminal) mengelola layer fisik jaringan fiber (FTTH). ISPBoss mendukung multi-brand OLT (ZTE, Huawei, FiberHome, VSOL, HSGQ) dengan adapter pattern per brand. Komunikasi ke OLT menggunakan **SNMP** untuk monitoring dan **SSH/Telnet** untuk provisioning command.

Scope spec ini mencakup **management plane** saja: OLT entity CRUD, registrasi dengan auto-detect, adapter pattern, koneksi SNMP dan CLI, health check, ODP/splitter management, PON port monitoring, ONT status monitoring, alarm management, SFP monitoring, periodic sync, traffic monitoring, capacity planning, HTTP API, event publishing, dan security. **Provisioning ONT** (add/remove ONT, unregistered detection, VLAN management, bulk provisioning, decommission) akan dicover di spec terpisah (`olt-provisioning`).

## Glossary

- **Network_Service**: Go microservice (`services/network-service/`) yang menangani semua integrasi perangkat jaringan (MikroTik, OLT)
- **OLT**: Optical Line Terminal — perangkat utama jaringan fiber yang mengelola koneksi ke semua ONT pelanggan
- **OLT_Manager**: Komponen usecase yang mengelola lifecycle OLT: registrasi, monitoring, sync, alarm
- **OLT_Adapter**: Interface adapter per brand OLT yang mengabstraksi perbedaan SNMP OID dan CLI command antar brand
- **ZTE_Adapter**: Implementasi adapter untuk OLT brand ZTE (C300, C320, C600) menggunakan SNMP + Telnet/SSH
- **Huawei_Adapter**: Implementasi adapter untuk OLT brand Huawei (MA56xx) menggunakan SNMP + SSH
- **FiberHome_Adapter**: Implementasi adapter untuk OLT brand FiberHome (AN5516) menggunakan SNMP + Telnet
- **VSOL_Adapter**: Implementasi adapter untuk OLT brand VSOL (V1600G, V1600D) menggunakan SNMP + Telnet/SSH
- **HSGQ_Adapter**: Implementasi adapter untuk OLT brand HSGQ menggunakan SNMP + Telnet
- **OLT_Status**: Status konektivitas OLT: online, offline, atau maintenance
- **ONT**: Optical Network Terminal — perangkat di sisi pelanggan yang terhubung ke OLT via fiber
- **ODP**: Optical Distribution Point — splitter yang membagi sinyal fiber dari OLT ke beberapa ONT (tipe: 1:4, 1:8, 1:16, 1:32)
- **PON_Port**: Port di OLT yang terhubung ke jaringan fiber downstream (ke ODP dan ONT)
- **SFP_Module**: Small Form-factor Pluggable — modul transceiver optik yang terpasang di PON port
- **SNMP_Connector**: Komponen yang mengelola koneksi SNMP ke OLT untuk monitoring (polling dan trap)
- **CLI_Connector**: Komponen yang mengelola koneksi SSH/Telnet ke OLT untuk provisioning command (connect-on-demand)
- **Health_Checker**: Background job yang memonitor status konektivitas OLT via SNMP ping secara periodik
- **Alarm_Manager**: Komponen yang mengelola alarm dari OLT (trap receiver + polling), menyimpan history, dan publish event
- **Sync_Engine**: Komponen yang melakukan periodic sync data OLT dengan database (OLT = source of truth untuk data fisik)
- **Traffic_Store**: Penyimpanan traffic data PON port di Redis time-series dengan retensi 7 hari
- **Signal_Store**: Penyimpanan signal data ONT di Redis time-series dengan retensi 30 hari
- **Credential_Encryptor**: Komponen yang mengenkripsi/mendekripsi credential OLT menggunakan AES-256-GCM (reuse dari MikroTik layer)
- **Tenant**: Organisasi ISP yang menggunakan platform ISPBoss (multi-tenant SaaS)
- **RLS**: Row-Level Security di PostgreSQL untuk isolasi data antar tenant
- **TaskEnvelope**: Format standar event antar service via Redis queue (pkg/queue)
- **Mock_Adapter**: Implementasi OLT adapter yang mengembalikan response simulasi tanpa koneksi ke OLT fisik

## Requirements

### Requirement 1: Database Schema untuk OLT Registry

**User Story:** As a platform engineer, I want a database schema for storing OLT device information per tenant, so that each ISP tenant can manage their own set of OLT devices in isolation.

#### Acceptance Criteria

1. THE Network_Service SHALL store OLT entities in an `olts` table with columns: id (UUID), tenant_id (UUID), name (VARCHAR 100), host (VARCHAR 255), snmp_version (VARCHAR 5), snmp_port (INTEGER DEFAULT 161), snmp_community_encrypted (TEXT nullable), snmp_username (VARCHAR 100 nullable), snmp_auth_protocol (VARCHAR 10 nullable), snmp_auth_password_encrypted (TEXT nullable), snmp_priv_protocol (VARCHAR 10 nullable), snmp_priv_password_encrypted (TEXT nullable), cli_protocol (VARCHAR 10), cli_port (INTEGER), cli_username (VARCHAR 100), cli_password_encrypted (TEXT), cli_enable_password_encrypted (TEXT nullable), brand (VARCHAR 50), model (VARCHAR 100), firmware_version (VARCHAR 100), pon_port_count (INTEGER), total_ont_count (INTEGER), status (VARCHAR 20), health_check_interval_sec (INTEGER), last_online_at (TIMESTAMPTZ), last_checked_at (TIMESTAMPTZ), failure_count (INTEGER), notes (TEXT), deleted_at (TIMESTAMPTZ), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `olts` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (tenant_id, name) WHERE deleted_at IS NULL to prevent duplicate OLT names within a tenant
4. THE Network_Service SHALL set default values: snmp_version='v2c', snmp_port=161, cli_protocol='ssh', cli_port=22, status='offline', health_check_interval_sec=300, failure_count=0, total_ont_count=0

### Requirement 2: OLT Domain Entities dan Status Constants

**User Story:** As a developer, I want well-defined domain entities and status constants for OLT devices, so that business logic is consistent and type-safe across the codebase.

#### Acceptance Criteria

1. THE Network_Service SHALL define an OLT struct containing all fields from the database schema with appropriate Go types
2. THE Network_Service SHALL define OLTStatus constants: StatusOnline ("online"), StatusOffline ("offline"), StatusMaintenance ("maintenance")
3. THE Network_Service SHALL define valid OLT status transitions: offline→online, offline→maintenance, online→offline, online→maintenance, maintenance→online, maintenance→offline
4. WHEN an invalid OLT status transition is attempted, THE Network_Service SHALL return a domain error indicating the current status and allowed transitions
5. THE Network_Service SHALL define OLTBrand constants: BrandZTE ("zte"), BrandHuawei ("huawei"), BrandFiberHome ("fiberhome"), BrandVSOL ("vsol"), BrandHSGQ ("hsgq")
6. THE Network_Service SHALL define SNMPVersion constants: SNMPv2c ("v2c"), SNMPv3 ("v3")
7. THE Network_Service SHALL define CLIProtocol constants: CLIProtocolSSH ("ssh"), CLIProtocolTelnet ("telnet")
8. THE Network_Service SHALL define SignalLevel constants with thresholds: Normal (-8 to -25 dBm), Warning (-25 to -27 dBm), Weak (-27 to -30 dBm), Critical (below -30 dBm)

### Requirement 3: OLT Adapter Interface (Multi-Brand)

**User Story:** As a developer, I want an adapter interface per OLT brand, so that I can support multiple OLT brands with different SNMP OIDs and CLI commands without changing business logic.

#### Acceptance Criteria

1. THE Network_Service SHALL define an OLTAdapter interface with methods: GetSystemInfo(ctx) (OLTSystemInfo, error), GetPONPortStatus(ctx, portIndex) (PONPortStatus, error), GetAllPONPorts(ctx) ([]PONPortStatus, error), GetONTList(ctx, portIndex) ([]ONTStatus, error), GetONTSignal(ctx, portIndex, ontIndex) (ONTSignalInfo, error), GetAlarms(ctx) ([]OLTAlarm, error), GetSFPInfo(ctx, portIndex) (SFPInfo, error), GetTrafficStats(ctx, portIndex) (PONTrafficStats, error), Ping(ctx) error
2. THE Network_Service SHALL implement ZTE_Adapter, Huawei_Adapter, FiberHome_Adapter, VSOL_Adapter, and HSGQ_Adapter each implementing the OLTAdapter interface with brand-specific SNMP OIDs and CLI commands
3. THE Network_Service SHALL define an OLTAdapterFactory that creates the correct adapter instance based on the OLT brand field
4. WHILE NETWORK_MODE is set to "mock", THE Network_Service SHALL use a Mock_Adapter that returns predefined simulated responses for all OLT brands without establishing network connections
5. WHILE NETWORK_MODE is set to "live", THE Network_Service SHALL use the brand-specific adapter that communicates with actual OLT devices via SNMP and CLI
6. WHEN the Mock_Adapter receives a GetSystemInfo call, THE Mock_Adapter SHALL return a valid OLTSystemInfo with realistic simulated values (brand "zte", model "C320", firmware "V2.1.0", pon_ports 8, total_ont 245)

### Requirement 4: SNMP Connection Management

**User Story:** As a platform engineer, I want SNMP connection support for both v2c and v3, so that the system can communicate with OLT devices for monitoring regardless of their SNMP configuration.

#### Acceptance Criteria

1. THE SNMP_Connector SHALL support SNMP v2c connections using a community string for authentication
2. THE SNMP_Connector SHALL support SNMP v3 connections using username, auth protocol (MD5/SHA), auth password, privacy protocol (DES/AES), and privacy password
3. THE SNMP_Connector SHALL use the `gosnmp` library for all SNMP operations (GET, WALK, GETBULK)
4. THE SNMP_Connector SHALL set a connection timeout of 5 seconds and a request timeout of 10 seconds for SNMP operations
5. THE SNMP_Connector SHALL support SNMP port 161 as default, configurable per OLT
6. WHEN an SNMP operation fails due to timeout or authentication error, THE SNMP_Connector SHALL return a descriptive error indicating the failure reason

### Requirement 5: CLI Connection Management (SSH/Telnet)

**User Story:** As a platform engineer, I want SSH and Telnet connection support with connect-on-demand strategy, so that provisioning commands can be sent to OLT devices without maintaining persistent connections.

#### Acceptance Criteria

1. THE CLI_Connector SHALL support SSH connections using the `golang.org/x/crypto/ssh` library
2. THE CLI_Connector SHALL support Telnet connections for OLT brands that do not support SSH
3. THE CLI_Connector SHALL use connect-on-demand strategy: open session, send command, receive response, close session
4. THE CLI_Connector SHALL NOT maintain a connection pool for CLI sessions (unlike MikroTik RouterOS API)
5. THE CLI_Connector SHALL set a connection timeout of 10 seconds and a command timeout of 30 seconds
6. WHEN a CLI connection fails, THE CLI_Connector SHALL return a descriptive error indicating whether the failure is authentication, timeout, or network unreachable
7. THE CLI_Connector SHALL support enable password for OLT brands that require privileged mode escalation

### Requirement 6: OLT Registration dan Auto-Detect

**User Story:** As an ISP admin, I want to register a new OLT with SNMP and CLI credentials and have the system auto-detect brand, model, and firmware, so that I don't need to manually enter device information.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/olt/devices with valid OLT data, THE OLT_Manager SHALL create a new OLT record, test the SNMP connection, auto-detect brand/model/firmware via SNMP sysDescr, and return the created OLT with HTTP 201
2. WHEN the SNMP test connection succeeds, THE OLT_Manager SHALL parse the sysDescr response to determine brand, model, firmware version, and store them in the OLT record
3. WHEN the SNMP test connection fails during registration, THE OLT_Manager SHALL still create the OLT record with status "offline" and return HTTP 201 with a warning in the response
4. WHEN a POST request is made to /api/v1/olt/devices/:id/test-snmp, THE OLT_Manager SHALL test the SNMP connection and return the auto-detected system information
5. WHEN a POST request is made to /api/v1/olt/devices/:id/test-cli, THE OLT_Manager SHALL test the CLI connection (SSH or Telnet) and return the connection result
6. THE OLT_Manager SHALL detect the OLT brand from sysDescr patterns: "ZTE" or "ZXA10" for ZTE, "Huawei" or "MA56" for Huawei, "FiberHome" or "AN5516" for FiberHome, "VSOL" or "V1600" for VSOL, "HSGQ" for HSGQ

### Requirement 7: OLT Health Check

**User Story:** As an ISP admin, I want automatic periodic health checks on my OLT devices via SNMP, so that I am immediately notified when an OLT goes offline.

#### Acceptance Criteria

1. THE Health_Checker SHALL check each OLT's connectivity via SNMP ping at the interval configured in health_check_interval_sec (default 300 seconds / 5 minutes)
2. WHEN a health check succeeds, THE Health_Checker SHALL update the OLT's last_checked_at timestamp and reset failure_count to 0
3. WHEN a health check fails, THE Health_Checker SHALL increment the OLT's failure_count
4. WHEN an OLT's failure_count reaches 3 consecutive failures, THE Health_Checker SHALL update the OLT status to "offline" and publish an "olt.device_offline" event via TaskEnvelope to the Redis queue
5. WHEN a previously offline OLT responds to a health check successfully, THE Health_Checker SHALL update the OLT status to "online", reset failure_count to 0, and publish an "olt.device_online" event via TaskEnvelope
6. WHILE an OLT's status is "maintenance", THE Health_Checker SHALL skip health checks for that OLT

### Requirement 8: ODP/Splitter Management

**User Story:** As an ISP admin, I want to manage ODP (splitter) entities with capacity tracking and GPS coordinates, so that I can plan fiber network expansion and track physical infrastructure.

#### Acceptance Criteria

1. THE Network_Service SHALL store ODP entities in an `odps` table with columns: id (UUID), tenant_id (UUID), olt_id (UUID FK to olts), pon_port_index (INTEGER), name (VARCHAR 100), splitter_type (VARCHAR 10), capacity (INTEGER), used_ports (INTEGER DEFAULT 0), address (TEXT), latitude (DECIMAL 10,7 nullable), longitude (DECIMAL 10,7 nullable), notes (TEXT), deleted_at (TIMESTAMPTZ), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `odps` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (tenant_id, name) WHERE deleted_at IS NULL to prevent duplicate ODP names within a tenant
4. THE Network_Service SHALL define splitter_type values: "1:4" (capacity 4), "1:8" (capacity 8), "1:16" (capacity 16), "1:32" (capacity 32)
5. THE Network_Service SHALL automatically set the capacity field based on splitter_type: 4 for "1:4", 8 for "1:8", 16 for "1:16", 32 for "1:32"
6. WHEN an ODP's used_ports reaches its capacity, THE Network_Service SHALL return a warning indicating the ODP is full when queried

### Requirement 9: PON Port Monitoring

**User Story:** As an ISP admin, I want to monitor the status of each PON port on my OLT including ONT count and traffic, so that I can identify capacity issues and troubleshoot problems.

#### Acceptance Criteria

1. THE OLT_Adapter SHALL retrieve PON port status including: port_index, admin_status (up/down), oper_status (up/down), ont_count (total registered ONT), ont_online_count, description
2. WHEN a GET request is made to /api/v1/olt/devices/:id/pon-ports, THE Network_Service SHALL return the list of all PON ports with their current status for the specified OLT
3. WHEN a GET request is made to /api/v1/olt/devices/:id/pon-ports/:port/onts, THE Network_Service SHALL return the list of ONTs registered on the specified PON port with their status and signal level
4. THE OLT_Adapter SHALL retrieve PON port traffic statistics via SNMP: rx_bytes, rx_packets, tx_bytes, tx_packets per port

### Requirement 10: ONT Status Monitoring

**User Story:** As an ISP admin, I want to see the online/offline status and signal level of each ONT, so that I can proactively identify connectivity issues before customers complain.

#### Acceptance Criteria

1. THE OLT_Adapter SHALL retrieve ONT status information including: ont_index, serial_number, name, status (online/offline), rx_signal_dbm, distance_meters, uptime_seconds
2. THE Network_Service SHALL classify ONT signal level based on rx_signal_dbm: Normal (-8 to -25 dBm), Warning (-25 to -27 dBm), Weak (-27 to -30 dBm), Critical (below -30 dBm or LOS)
3. THE Signal_Store SHALL store ONT signal readings in Redis time-series with 30-day retention for historical signal graphs
4. THE Network_Service SHALL poll ONT status and signal every 5 minutes (configurable, same interval as OLT health check)

### Requirement 11: Alarm Management

**User Story:** As an ISP admin, I want to receive and manage alarms from OLT devices (both push via SNMP trap and pull via polling), so that I can respond quickly to network issues.

#### Acceptance Criteria

1. THE Alarm_Manager SHALL run an SNMP trap receiver listening on port 162 to receive push alarms from OLT devices
2. THE Alarm_Manager SHALL poll OLT alarm status via SNMP at the health check interval as a fallback for OLTs that do not support traps
3. THE Network_Service SHALL store alarms in an `olt_alarms` table with columns: id (UUID), tenant_id (UUID), olt_id (UUID FK), pon_port_index (INTEGER nullable), ont_index (INTEGER nullable), alarm_type (VARCHAR 50), severity (VARCHAR 20), message (TEXT), source (VARCHAR 20), status (VARCHAR 20 DEFAULT 'active'), cleared_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ)
4. THE Network_Service SHALL define alarm_type values: "ont_los" (Loss of Signal), "ont_dying_gasp" (power failure), "pon_port_down", "power_failure", "high_temperature", "ont_signal_degraded"
5. THE Network_Service SHALL define severity values: "critical", "major", "minor", "warning", "clear"
6. WHEN an alarm is received, THE Alarm_Manager SHALL publish an "olt.alarm" event via TaskEnvelope containing: olt_id, olt_name, tenant_id, alarm_type, severity, pon_port_index, ont_index, message
7. THE Network_Service SHALL retain alarm history for 90 days and automatically purge older records

### Requirement 12: SFP Module Monitoring

**User Story:** As an ISP admin, I want to monitor SFP module health (TX/RX power, temperature) per PON port, so that I can detect degrading hardware before it causes outages.

#### Acceptance Criteria

1. THE OLT_Adapter SHALL retrieve SFP module information per PON port including: tx_power_dbm, rx_power_dbm, temperature_celsius, sfp_type (e.g., "GPON C+", "GPON B+"), status
2. THE Network_Service SHALL classify SFP status: Normal (temperature below 45°C and power in range), Warm (temperature 45-60°C), Degraded (TX power significantly below nominal), Failed (out of range or undetected), Empty (no SFP installed)
3. WHEN a GET request is made to /api/v1/olt/devices/:id/sfp, THE Network_Service SHALL return SFP information for all PON ports of the specified OLT

### Requirement 13: OLT Periodic Sync

**User Story:** As a platform engineer, I want periodic synchronization between OLT physical data and the database, so that the database always reflects the actual state of the network with OLT as source of truth for physical data.

#### Acceptance Criteria

1. THE Sync_Engine SHALL run a periodic sync every 30 minutes for each online OLT
2. THE Sync_Engine SHALL retrieve all ONT data from the OLT via SNMP and compare with the database
3. WHEN an ONT exists on the OLT but not in the database, THE Sync_Engine SHALL mark it as "unmanaged" for display in the unregistered ONT list
4. WHEN an ONT exists in the database but not on the OLT, THE Sync_Engine SHALL mark it as "missing" and log the discrepancy
5. WHEN an ONT exists in both but data differs (port, status), THE Sync_Engine SHALL update the database to match the OLT data (OLT = source of truth for physical data)
6. THE Sync_Engine SHALL update the OLT's total_ont_count and each PON port's ont_count after each sync cycle

### Requirement 14: Traffic Monitoring

**User Story:** As an ISP admin, I want to see real-time and historical traffic data per PON port, so that I can monitor bandwidth utilization and plan capacity.

#### Acceptance Criteria

1. THE OLT_Adapter SHALL collect PON port traffic statistics via SNMP polling: rx_bytes, rx_packets, tx_bytes, tx_packets per port
2. THE Traffic_Store SHALL store traffic data points in Redis time-series with 7-day retention
3. THE Network_Service SHALL poll traffic statistics every 5 minutes (same interval as health check)
4. WHEN a GET request is made to /api/v1/olt/devices/:id/pon-ports/:port/traffic with from and to timestamp parameters, THE Network_Service SHALL return traffic history as time-series data

### Requirement 15: Capacity Planning

**User Story:** As an ISP admin, I want capacity planning data showing ONT slots per port, ODP utilization, and growth rate estimation, so that I can plan network expansion proactively.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/olt/devices/:id/capacity, THE Network_Service SHALL return capacity information: total_pon_ports, active_pon_ports, total_ont_slots, used_ont_slots, available_ont_slots, utilization_percent, growth_rate_per_month, estimated_months_remaining
2. THE Network_Service SHALL calculate growth_rate_per_month based on the average number of new ONTs added per month over the last 3 months
3. THE Network_Service SHALL calculate estimated_months_remaining as available_ont_slots divided by growth_rate_per_month
4. THE Network_Service SHALL return per-port capacity breakdown: port_index, ont_count, max_ont_per_port (default 64 for GPON), utilization_percent
5. WHEN a PON port utilization exceeds 90%, THE Network_Service SHALL include a warning in the capacity response for that port

### Requirement 16: HTTP API Endpoints

**User Story:** As a frontend developer, I want well-defined REST API endpoints for all OLT management operations, so that the dashboard can interact with the OLT management layer.

#### Acceptance Criteria

1. THE Network_Service SHALL expose POST /api/v1/olt/devices accepting OLT creation payload and returning the created OLT with auto-detected information
2. THE Network_Service SHALL expose GET /api/v1/olt/devices returning a paginated list of OLTs for the authenticated tenant with filtering by status and brand
3. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id returning full OLT detail including PON port summary and alarm count
4. THE Network_Service SHALL expose PUT /api/v1/olt/devices/:id accepting OLT update payload and returning the updated OLT
5. THE Network_Service SHALL expose DELETE /api/v1/olt/devices/:id performing soft-delete of the OLT record
6. THE Network_Service SHALL expose POST /api/v1/olt/devices/:id/test-snmp testing SNMP connectivity and returning auto-detected system info
7. THE Network_Service SHALL expose POST /api/v1/olt/devices/:id/test-cli testing CLI connectivity and returning connection result
8. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/pon-ports returning PON port status list
9. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/pon-ports/:port/onts returning ONT list for a specific port
10. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/pon-ports/:port/traffic returning traffic history for a specific port
11. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/alarms returning alarm list with filtering by severity and status
12. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/sfp returning SFP module status for all ports
13. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/capacity returning capacity planning data
14. THE Network_Service SHALL expose GET /api/v1/olt/summary returning OLT status summary: total_olts, online_count, offline_count, maintenance_count, active_alarm_count
15. THE Network_Service SHALL expose POST /api/v1/olt/odp accepting ODP creation payload and returning the created ODP
16. THE Network_Service SHALL expose GET /api/v1/olt/odp returning a paginated list of ODPs with filtering by olt_id and pon_port
17. THE Network_Service SHALL expose GET /api/v1/olt/odp/:id returning ODP detail including used_ports and linked ONT list
18. THE Network_Service SHALL expose PUT /api/v1/olt/odp/:id accepting ODP update payload and returning the updated ODP
19. THE Network_Service SHALL expose DELETE /api/v1/olt/odp/:id performing soft-delete of the ODP record
20. WHEN any OLT API request references an olt_id or odp_id that does not belong to the authenticated tenant, THE Network_Service SHALL return HTTP 404

### Requirement 17: Event Publishing

**User Story:** As a platform engineer, I want OLT status change and alarm events published to the Redis queue, so that other services (notification) can react to OLT events.

#### Acceptance Criteria

1. WHEN an OLT transitions from online to offline, THE Network_Service SHALL publish a TaskEnvelope with event_type "olt.device_offline" containing: olt_id, olt_name, tenant_id, brand, last_online_at
2. WHEN an OLT transitions from offline to online, THE Network_Service SHALL publish a TaskEnvelope with event_type "olt.device_online" containing: olt_id, olt_name, tenant_id, brand, downtime_duration
3. WHEN an alarm is received (trap or polling), THE Network_Service SHALL publish a TaskEnvelope with event_type "olt.alarm" containing: olt_id, olt_name, tenant_id, alarm_type, severity, pon_port_index, ont_index, message
4. THE Network_Service SHALL include a correlation_id in every published OLT event for distributed tracing

### Requirement 18: Credential Security

**User Story:** As a platform engineer, I want all OLT credentials (SNMP community strings, SNMP v3 passwords, CLI passwords) encrypted at rest using AES-256-GCM, so that sensitive data is protected even if the database is compromised.

#### Acceptance Criteria

1. WHEN an OLT is created or updated with credentials, THE Credential_Encryptor SHALL encrypt all sensitive fields (snmp_community, snmp_auth_password, snmp_priv_password, cli_password, cli_enable_password) using AES-256-GCM before storing in the database
2. WHEN OLT credentials are needed for connection (health check, monitoring, CLI command), THE Credential_Encryptor SHALL decrypt the credentials from the database using the same master key
3. THE Network_Service SHALL reuse the existing Credential_Encryptor implementation from the MikroTik Router Foundation Layer (same ENCRYPTION_KEY environment variable)
4. THE Network_Service SHALL never expose decrypted credentials in API responses — credential fields SHALL be masked or omitted
5. FOR ALL valid credential strings, encrypting then decrypting SHALL produce the original credential (round-trip property)
6. THE Network_Service SHALL recommend SNMP v3 over v2c in documentation and API response warnings when v2c is used without VPN tunnel

### Requirement 19: OLT Status Summary

**User Story:** As an ISP admin, I want a summary endpoint showing the overall status of all my OLTs, so that I can display a dashboard widget with quick health overview.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/olt/summary, THE Network_Service SHALL return a JSON object containing: total_olts, online_count, offline_count, maintenance_count, active_alarm_count for the authenticated tenant
2. THE Network_Service SHALL compute the summary from the current OLT statuses and alarm counts in the database without making live connections to OLT devices

### Requirement 20: OLT CRUD Operations

**User Story:** As an ISP admin, I want REST API endpoints to manage my OLT devices, so that I can add, view, update, and remove OLTs from the ISPBoss dashboard.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/olt/devices with query parameters, THE Network_Service SHALL support filtering by status, brand, and search (name or host), with pagination (page, page_size)
2. WHEN a PUT request is made to /api/v1/olt/devices/:id with updated data, THE OLT_Manager SHALL update the OLT record and return the updated OLT with HTTP 200
3. THE OLT_Manager SHALL allow updating: name, host, SNMP credentials, CLI credentials, health_check_interval_sec, notes, and status (to maintenance)
4. WHEN a DELETE request is made to /api/v1/olt/devices/:id, THE OLT_Manager SHALL soft-delete the OLT record and stop health check monitoring for that OLT
5. WHEN a PUT request changes the OLT status to "maintenance", THE Health_Checker SHALL stop monitoring that OLT until status is changed back
