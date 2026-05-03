# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **OLT Provisioning Layer** di `services/network-service/`. Layer ini dibangun di atas **OLT Management Layer** (spec `olt-management`) yang sudah diimplementasikan. Management layer menangani OLT device CRUD, adapter pattern, SNMP/CLI connectors, health check, alarm management, sync engine, dan monitoring.

OLT Provisioning Layer menangani **provisioning plane**: manajemen entitas ONT, deteksi ONT unregistered, provisioning ONT (single dan bulk), decommission ONT, reboot ONT, VLAN management, service profile management, deteksi port migration, command builder per brand, audit trail, event integration, dan HTTP API. Setiap operasi provisioning menggunakan CLI command yang berbeda per brand OLT — ditangani oleh ekstensi adapter pattern yang sudah ada.

Scope spec ini mencakup **provisioning plane** saja. Management plane (OLT CRUD, monitoring, health check, alarm, sync) sudah dicover di spec `olt-management`.

## Glossary

- **Network_Service**: Go microservice (`services/network-service/`) yang menangani semua integrasi perangkat jaringan (MikroTik, OLT)
- **OLT**: Optical Line Terminal — perangkat utama jaringan fiber yang mengelola koneksi ke semua ONT pelanggan
- **ONT**: Optical Network Terminal — perangkat di sisi pelanggan yang terhubung ke OLT via fiber optik
- **ONT_Entity**: Record ONT di database yang menyimpan serial number, status, VLAN, service profile, dan relasi ke OLT/pelanggan/ODP
- **ONT_Status**: Status lifecycle ONT: registered (terdaftar di DB, belum di OLT), provisioned (aktif di OLT), unregistered (terdeteksi di OLT tapi belum di DB), missing (ada di DB tapi tidak di OLT), decommissioned (dihapus dari OLT)
- **Provisioning_State**: State proses provisioning ONT: pending, in_progress, completed, failed
- **Provisioning_Manager**: Komponen usecase yang mengelola lifecycle provisioning ONT: add, remove, reboot, bulk, auto-provision
- **Command_Builder**: Komponen yang membangun CLI command per brand OLT untuk operasi provisioning (add ONT, delete ONT, add service-port, delete service-port, reboot ONT)
- **OLT_Adapter**: Interface adapter per brand OLT yang sudah ada di management layer — diperluas dengan method provisioning
- **CLI_Connector**: Komponen connect-on-demand SSH/Telnet yang sudah ada — digunakan untuk mengirim provisioning command
- **Sync_Engine**: Komponen periodic sync yang sudah ada — mendeteksi ONT unmanaged dan missing
- **VLAN_Entity**: Record VLAN per OLT yang menyimpan VLAN ID, nama, tipe, dan assignment strategy
- **VLAN_Strategy**: Strategi assignment VLAN saat provisioning: single (semua pelanggan 1 VLAN), per_paket (per paket internet), per_odp (per ODP/splitter), per_pelanggan (VLAN unik per pelanggan)
- **Service_Profile**: Mapping antara paket ISPBoss dan OLT line/service profile untuk bandwidth/QoS
- **Audit_Log**: Record append-only yang mencatat semua provisioning command yang dikirim ke OLT
- **Bulk_Provisioning**: Proses provisioning banyak ONT sekaligus via upload CSV
- **Auto_Provisioning**: Fitur opsional yang otomatis mem-provision ONT baru jika serial number sudah terdaftar di record pelanggan
- **Port_Migration**: Kondisi saat ONT terdeteksi pindah dari satu PON port ke port lain
- **ODP**: Optical Distribution Point — splitter yang sudah dikelola di management layer
- **Decommission**: Proses menghapus ONT dari OLT dan memutus relasi ke pelanggan
- **TaskEnvelope**: Format standar event antar service via Redis queue (`pkg/queue`)
- **Tenant**: Organisasi ISP yang menggunakan platform ISPBoss (multi-tenant SaaS)
- **RLS**: Row-Level Security di PostgreSQL untuk isolasi data antar tenant

## Requirements

### Requirement 1: Database Schema untuk ONT Entity

**User Story:** As a platform engineer, I want a database schema for storing ONT records linked to OLT, PON port, customer, and ODP, so that the system can track the full lifecycle of each ONT device per tenant.

#### Acceptance Criteria

1. THE Network_Service SHALL store ONT entities in an `onts` table with columns: id (UUID), tenant_id (UUID), olt_id (UUID FK to olts), pon_port_index (INTEGER), ont_index (INTEGER), serial_number (VARCHAR 50), customer_id (UUID nullable), odp_id (UUID FK to odps nullable), vlan_id (UUID FK to vlans nullable), service_profile_id (UUID FK to service_profiles nullable), status (VARCHAR 30 DEFAULT 'registered'), provisioning_state (VARCHAR 20 DEFAULT 'pending'), description (TEXT nullable), last_provisioned_at (TIMESTAMPTZ nullable), last_decommissioned_at (TIMESTAMPTZ nullable), deleted_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `onts` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (tenant_id, serial_number) WHERE deleted_at IS NULL to prevent duplicate ONT serial numbers within a tenant
4. THE Network_Service SHALL enforce a unique constraint on (olt_id, pon_port_index, ont_index) WHERE deleted_at IS NULL to prevent duplicate ONT positions on the same OLT port
5. THE Network_Service SHALL define ONT status values: "registered" (terdaftar di DB, belum di OLT), "provisioned" (aktif di OLT), "unregistered" (terdeteksi di OLT tapi belum di DB), "missing" (ada di DB tapi tidak di OLT), "decommissioned" (dihapus dari OLT)
6. THE Network_Service SHALL define provisioning_state values: "pending" (menunggu provisioning), "in_progress" (sedang diproses), "completed" (berhasil), "failed" (gagal)

### Requirement 2: Unregistered ONT Detection

**User Story:** As an ISP admin, I want to see a list of ONTs detected on the OLT that are not yet registered in the database, so that I can provision them for customers.

#### Acceptance Criteria

1. WHEN the Sync_Engine detects an ONT on the OLT that does not exist in the `onts` table (by serial number), THE Provisioning_Manager SHALL create a temporary unregistered ONT record with status "unregistered" containing: serial_number, olt_id, pon_port_index, ont_index
2. WHEN a GET request is made to /api/v1/olt/devices/:id/unregistered-onts, THE Network_Service SHALL return a list of ONTs with status "unregistered" for the specified OLT, including serial_number, pon_port_index, ont_index, and detected_at timestamp
3. WHEN an unregistered ONT is provisioned, THE Provisioning_Manager SHALL update the ONT status from "unregistered" to "provisioned"
4. WHEN an unregistered ONT disappears from the OLT during the next sync cycle, THE Provisioning_Manager SHALL remove the unregistered ONT record from the database

### Requirement 3: Single ONT Provisioning

**User Story:** As an ISP admin, I want to provision a single ONT by selecting an unregistered ONT or entering a serial number manually, linking it to a customer, service profile, VLAN, and ODP, so that the customer gets internet service via fiber.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/olt/provisioning/ont with valid provisioning data (serial_number, olt_id, pon_port_index, customer_id, service_profile_id, vlan_id, odp_id), THE Provisioning_Manager SHALL execute the provisioning sequence: build CLI commands via Command_Builder, send commands to OLT via CLI_Connector, update ONT record in database
2. WHEN provisioning is initiated, THE Provisioning_Manager SHALL set the ONT provisioning_state to "in_progress" before sending CLI commands
3. WHEN all CLI commands execute successfully, THE Provisioning_Manager SHALL update the ONT status to "provisioned", set provisioning_state to "completed", record last_provisioned_at timestamp, and publish an "ont.provisioned" event
4. IF any CLI command fails during provisioning, THEN THE Provisioning_Manager SHALL set provisioning_state to "failed", log the error with the failed command and OLT response, and return a descriptive error to the caller
5. WHEN provisioning an ONT, THE Command_Builder SHALL generate brand-specific CLI commands: add ONT to PON port with line profile and service profile, then add service-port with VLAN assignment
6. IF the specified customer_id already has an active ONT (status "provisioned"), THEN THE Provisioning_Manager SHALL return an error indicating the customer already has an active ONT
7. IF the specified serial_number is already provisioned on the OLT, THEN THE Provisioning_Manager SHALL return an error indicating the ONT is already provisioned

### Requirement 4: Provisioning Command Builder (Per-Brand)

**User Story:** As a developer, I want a per-brand CLI command builder for provisioning operations, so that the system generates correct CLI commands for each OLT brand without changing business logic.

#### Acceptance Criteria

1. THE Command_Builder SHALL generate add-ONT commands per brand: ZTE uses `onu add sn {SN} ont-lineprofile-id {profile} ont-srvprofile-id {srv}`, Huawei uses `ont add {port} sn-auth {SN} omci ont-lineprofile-id {profile}`, and each brand follows its own CLI syntax
2. THE Command_Builder SHALL generate add-service-port commands per brand: ZTE uses `service-port add vlan {vlan} gpon {port} ont {id} gemport {gem}`, Huawei uses its own service-port syntax, and each brand follows its own CLI syntax
3. THE Command_Builder SHALL generate delete-ONT commands per brand for decommission operations
4. THE Command_Builder SHALL generate delete-service-port commands per brand for decommission operations
5. THE Command_Builder SHALL generate reboot-ONT commands per brand
6. THE Command_Builder SHALL extend the existing OLT_Adapter interface with provisioning methods: AddONT, RemoveONT, AddServicePort, RemoveServicePort, RebootONT
7. FOR ALL provisioning commands, THE Command_Builder SHALL produce non-empty command strings that contain the provided serial number and VLAN ID parameters
8. THE Command_Builder SHALL implement provisioning methods for ZTE_Adapter, Huawei_Adapter, FiberHome_Adapter, VSOL_Adapter, and HSGQ_Adapter
9. WHILE NETWORK_MODE is set to "mock", THE Mock_Adapter SHALL return simulated success responses for all provisioning commands without establishing network connections

### Requirement 5: Bulk Provisioning via CSV Upload

**User Story:** As an ISP admin, I want to upload a CSV file to provision multiple ONTs at once, so that I can efficiently handle mass installations or migrations.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/olt/provisioning/bulk with a CSV file and olt_id, THE Provisioning_Manager SHALL parse the CSV with columns: sn_ont, pelanggan_id, pon_port, vlan, odp, deskripsi
2. THE Provisioning_Manager SHALL validate all CSV rows before executing any provisioning: check serial number format, verify pelanggan_id exists, verify pon_port is valid for the OLT, verify VLAN exists, verify ODP exists
3. WHEN validation is complete, THE Provisioning_Manager SHALL return a preview response containing: total rows, valid count, error count, and per-row status (valid or error with reason)
4. WHEN a POST request is made to /api/v1/olt/provisioning/bulk/execute with the bulk_id, THE Provisioning_Manager SHALL execute provisioning for all valid rows sequentially, skipping rows with validation errors
5. THE Provisioning_Manager SHALL track bulk provisioning progress: total, completed, failed, and current row being processed
6. WHEN bulk provisioning completes, THE Provisioning_Manager SHALL return a summary: total, success_count, failure_count, and per-row results with error details for failures
7. IF a single row fails during bulk execution, THEN THE Provisioning_Manager SHALL continue processing remaining rows and include the failure in the summary
8. THE Provisioning_Manager SHALL provide a CSV template download endpoint at GET /api/v1/olt/provisioning/bulk/template

### Requirement 6: Auto-Provisioning (Optional Feature)

**User Story:** As an ISP admin, I want the system to automatically provision an ONT when it is detected as unregistered and its serial number is already linked to a customer record, so that field technicians don't need to manually provision from the dashboard.

#### Acceptance Criteria

1. WHERE auto-provisioning is enabled in tenant settings, WHEN the Sync_Engine detects an unregistered ONT whose serial number matches a customer record's pre-registered SN, THE Provisioning_Manager SHALL automatically execute the provisioning sequence using the customer's assigned service profile and VLAN
2. THE Network_Service SHALL store the auto-provisioning setting as a boolean flag per tenant, default OFF
3. WHEN auto-provisioning succeeds, THE Provisioning_Manager SHALL publish an "ont.auto_provisioned" event containing: ont_serial_number, customer_id, olt_id, pon_port_index
4. IF auto-provisioning fails, THEN THE Provisioning_Manager SHALL log the error, leave the ONT as "unregistered", and publish an "ont.auto_provision_failed" event for notification
5. WHERE auto-provisioning is disabled (default), THE Provisioning_Manager SHALL only display unregistered ONTs in the unregistered list without taking automatic action

### Requirement 7: ONT Decommission

**User Story:** As an ISP admin, I want to decommission an ONT when a customer terminates service, removing it from the OLT and updating the database, so that the ONT slot is freed for reuse.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/olt/provisioning/ont/:id/decommission, THE Provisioning_Manager SHALL execute the decommission sequence: remove service-port from OLT via CLI, remove ONT from OLT via CLI, update ONT record status to "decommissioned", clear customer_id link, record last_decommissioned_at timestamp
2. WHEN the Provisioning_Manager receives a "customer.terminated" event from the billing service, THE Provisioning_Manager SHALL look up the ONT linked to the terminated customer and execute the decommission sequence automatically
3. WHEN decommission completes successfully, THE Provisioning_Manager SHALL publish an "ont.decommissioned" event containing: ont_id, serial_number, customer_id, olt_id, pon_port_index
4. IF decommission CLI commands fail, THEN THE Provisioning_Manager SHALL set provisioning_state to "failed", log the error, and return a descriptive error to the caller
5. WHEN decommission is triggered by "customer.terminated" event and CLI commands fail, THE Provisioning_Manager SHALL retry with exponential backoff: delays of 30s, 1m, 2m, 5m, 10m (max 5 retries), consistent with the existing PPPoE event worker retry pattern
6. THE Provisioning_Manager SHALL allow manual decommission regardless of ONT status, enabling admin to force-remove ONTs from the OLT

### Requirement 8: ONT Reboot

**User Story:** As an ISP admin, I want to remotely reboot a specific ONT via the OLT CLI, so that I can troubleshoot customer connectivity issues without a field visit.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/olt/provisioning/ont/:id/reboot, THE Provisioning_Manager SHALL send a reboot command to the ONT via the OLT CLI using the brand-specific Command_Builder
2. WHEN the reboot command executes successfully, THE Provisioning_Manager SHALL return a success response and log the reboot action in the audit trail
3. IF the reboot command fails, THEN THE Provisioning_Manager SHALL return a descriptive error indicating the failure reason (timeout, OLT unreachable, ONT not found)
4. THE Provisioning_Manager SHALL only allow reboot for ONTs with status "provisioned" — reboot requests for ONTs in other statuses SHALL return an error

### Requirement 9: VLAN Management

**User Story:** As an ISP admin, I want to manage VLANs per OLT with configurable assignment strategies, so that I can organize network traffic according to my ISP's architecture.

#### Acceptance Criteria

1. THE Network_Service SHALL store VLAN entities in a `vlans` table with columns: id (UUID), tenant_id (UUID), olt_id (UUID FK to olts), vlan_id (INTEGER), name (VARCHAR 100), vlan_type (VARCHAR 30), description (TEXT nullable), deleted_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `vlans` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (olt_id, vlan_id) WHERE deleted_at IS NULL to prevent duplicate VLAN IDs on the same OLT
4. THE Network_Service SHALL define VLAN assignment strategy values stored in tenant settings: "single" (semua pelanggan 1 VLAN, default), "per_paket" (VLAN berbeda per paket internet), "per_odp" (VLAN berbeda per ODP), "per_pelanggan" (VLAN unik per pelanggan)
5. WHEN provisioning an ONT, THE Provisioning_Manager SHALL resolve the VLAN to assign based on the active VLAN strategy: for "single" use the default VLAN, for "per_paket" use the VLAN mapped to the customer's package, for "per_odp" use the VLAN mapped to the ODP, for "per_pelanggan" allocate a unique VLAN from the available pool
6. THE Network_Service SHALL expose CRUD endpoints for VLAN management: POST /api/v1/olt/devices/:id/vlans, GET /api/v1/olt/devices/:id/vlans, PUT /api/v1/olt/vlans/:id, DELETE /api/v1/olt/vlans/:id
7. WHEN a VLAN is deleted, THE Network_Service SHALL verify no active ONTs are using the VLAN before allowing deletion — IF active ONTs reference the VLAN, THEN THE Network_Service SHALL return an error indicating the VLAN is in use

### Requirement 10: Service Profile Management

**User Story:** As an ISP admin, I want to map ISPBoss packages to OLT service profiles, so that provisioning automatically applies the correct bandwidth and QoS settings for each customer's plan.

#### Acceptance Criteria

1. THE Network_Service SHALL store service profile entities in a `service_profiles` table with columns: id (UUID), tenant_id (UUID), olt_id (UUID FK to olts), name (VARCHAR 100), line_profile_id (INTEGER), service_profile_id (INTEGER), package_id (UUID nullable — FK to billing packages), description (TEXT nullable), deleted_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `service_profiles` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (olt_id, line_profile_id, service_profile_id) WHERE deleted_at IS NULL to prevent duplicate profile combinations on the same OLT
4. WHEN provisioning an ONT, THE Provisioning_Manager SHALL resolve the service profile from the customer's package mapping — IF no mapping exists, THEN THE Provisioning_Manager SHALL return an error indicating the package has no OLT service profile configured
5. THE Network_Service SHALL expose CRUD endpoints for service profile management: POST /api/v1/olt/devices/:id/service-profiles, GET /api/v1/olt/devices/:id/service-profiles, PUT /api/v1/olt/service-profiles/:id, DELETE /api/v1/olt/service-profiles/:id
6. WHEN a service profile is deleted, THE Network_Service SHALL verify no active ONTs are using the profile before allowing deletion — IF active ONTs reference the profile, THEN THE Network_Service SHALL return an error indicating the profile is in use

### Requirement 11: ONT Port Migration Detection

**User Story:** As an ISP admin, I want the system to detect when an ONT has been physically moved to a different PON port, so that I can verify whether the move was authorized and update records accordingly.

#### Acceptance Criteria

1. WHEN the Sync_Engine detects that an ONT's current PON port on the OLT differs from the port recorded in the database, THE Provisioning_Manager SHALL flag the ONT as "port_migrated" and record the old and new port information
2. WHEN a port migration is detected, THE Provisioning_Manager SHALL publish an "ont.port_migrated" event containing: ont_id, serial_number, olt_id, old_port_index, new_port_index, old_ont_index, new_ont_index
3. WHERE auto-port-migration is enabled in tenant settings, THE Provisioning_Manager SHALL automatically update the database to reflect the new port without admin intervention
4. WHERE auto-port-migration is disabled (default), THE Provisioning_Manager SHALL display the migration in a notification list and wait for admin confirmation before updating the database
5. WHEN an admin confirms a port migration via POST /api/v1/olt/provisioning/ont/:id/confirm-migration, THE Provisioning_Manager SHALL update the ONT's pon_port_index and ont_index in the database and optionally update the ODP assignment if the new port maps to a different ODP

### Requirement 12: Provisioning Audit Trail

**User Story:** As an ISP admin, I want an append-only audit log of all provisioning commands sent to OLT devices, so that I can trace every change made to the network for troubleshooting and compliance.

#### Acceptance Criteria

1. THE Network_Service SHALL store provisioning audit records in a `provisioning_audit_logs` table with columns: id (UUID), tenant_id (UUID), olt_id (UUID FK to olts), ont_id (UUID FK to onts nullable), action (VARCHAR 50), commands_sent (JSONB), command_responses (JSONB), status (VARCHAR 20), error_message (TEXT nullable), performed_by (VARCHAR 100), correlation_id (UUID), created_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `provisioning_audit_logs` table so that queries only return rows matching the current tenant context
3. WHEN any provisioning command is sent to an OLT (add ONT, remove ONT, add service-port, remove service-port, reboot ONT), THE Provisioning_Manager SHALL create an audit log entry with the exact commands sent and responses received
4. THE Network_Service SHALL define audit action values: "ont_provision", "ont_decommission", "ont_reboot", "service_port_add", "service_port_remove", "bulk_provision", "auto_provision"
5. THE Network_Service SHALL record the performer identity: username for manual actions, "system" for auto-provisioning, "event:customer.terminated" for event-triggered decommission
6. THE Network_Service SHALL expose GET /api/v1/olt/provisioning/audit-logs with filtering by olt_id, ont_id, action, date range, and pagination
7. THE provisioning_audit_logs table SHALL be append-only — THE Network_Service SHALL NOT provide update or delete endpoints for audit records

### Requirement 13: Event Integration

**User Story:** As a platform engineer, I want the provisioning layer to listen for customer lifecycle events and publish provisioning events, so that the system reacts automatically to customer changes and other services can respond to provisioning actions.

#### Acceptance Criteria

1. THE Network_Service SHALL listen for "customer.terminated" events from the Redis queue and trigger ONT decommission for the terminated customer's linked ONT
2. WHEN an ONT is successfully provisioned, THE Network_Service SHALL publish a TaskEnvelope with event_type "ont.provisioned" containing: ont_id, serial_number, customer_id, olt_id, olt_name, pon_port_index, vlan_id, tenant_id, correlation_id
3. WHEN an ONT is successfully decommissioned, THE Network_Service SHALL publish a TaskEnvelope with event_type "ont.decommissioned" containing: ont_id, serial_number, customer_id, olt_id, olt_name, pon_port_index, tenant_id, correlation_id
4. WHEN an ONT is auto-provisioned, THE Network_Service SHALL publish a TaskEnvelope with event_type "ont.auto_provisioned" containing: ont_id, serial_number, customer_id, olt_id, pon_port_index, tenant_id, correlation_id
5. WHEN auto-provisioning fails, THE Network_Service SHALL publish a TaskEnvelope with event_type "ont.auto_provision_failed" containing: serial_number, olt_id, pon_port_index, error_message, tenant_id, correlation_id
6. WHEN a port migration is detected, THE Network_Service SHALL publish a TaskEnvelope with event_type "ont.port_migrated" containing: ont_id, serial_number, olt_id, old_port_index, new_port_index, tenant_id, correlation_id
7. THE Network_Service SHALL include a correlation_id (UUID v4) in every published provisioning event for distributed tracing

### Requirement 14: HTTP API Endpoints untuk Provisioning

**User Story:** As a frontend developer, I want well-defined REST API endpoints for all provisioning operations, so that the dashboard can interact with the provisioning layer.

#### Acceptance Criteria

1. THE Network_Service SHALL expose POST /api/v1/olt/provisioning/ont accepting single ONT provisioning payload (serial_number, olt_id, pon_port_index, customer_id, service_profile_id, vlan_id, odp_id, description) and returning the provisioned ONT with HTTP 201
2. THE Network_Service SHALL expose GET /api/v1/olt/devices/:id/unregistered-onts returning a list of unregistered ONTs detected on the specified OLT
3. THE Network_Service SHALL expose POST /api/v1/olt/provisioning/bulk accepting a CSV file and olt_id, returning a validation preview with per-row status
4. THE Network_Service SHALL expose POST /api/v1/olt/provisioning/bulk/execute accepting a bulk_id and executing provisioning for all valid rows
5. THE Network_Service SHALL expose GET /api/v1/olt/provisioning/bulk/template returning a CSV template file for bulk provisioning
6. THE Network_Service SHALL expose POST /api/v1/olt/provisioning/ont/:id/decommission triggering ONT decommission and returning the updated ONT record
7. THE Network_Service SHALL expose POST /api/v1/olt/provisioning/ont/:id/reboot triggering ONT reboot and returning the command result
8. THE Network_Service SHALL expose POST /api/v1/olt/provisioning/ont/:id/confirm-migration confirming a detected port migration and updating the database
9. THE Network_Service SHALL expose GET /api/v1/olt/provisioning/onts returning a paginated list of ONTs with filtering by olt_id, status, provisioning_state, customer_id, and search (serial_number)
10. THE Network_Service SHALL expose GET /api/v1/olt/provisioning/onts/:id returning full ONT detail including linked customer, ODP, VLAN, service profile, and provisioning history
11. THE Network_Service SHALL expose GET /api/v1/olt/provisioning/audit-logs returning a paginated list of provisioning audit logs with filtering by olt_id, ont_id, action, and date range
12. WHEN any provisioning API request references an entity that does not belong to the authenticated tenant, THE Network_Service SHALL return HTTP 404

### Requirement 15: Provisioning Settings per Tenant

**User Story:** As an ISP admin, I want configurable provisioning settings per tenant, so that I can control auto-provisioning behavior, VLAN strategy, and port migration handling according to my ISP's operational preferences.

#### Acceptance Criteria

1. THE Network_Service SHALL store provisioning settings in a `provisioning_settings` table with columns: id (UUID), tenant_id (UUID UNIQUE), auto_provisioning_enabled (BOOLEAN DEFAULT false), auto_port_migration_enabled (BOOLEAN DEFAULT false), vlan_strategy (VARCHAR 30 DEFAULT 'single'), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `provisioning_settings` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL expose GET /api/v1/olt/provisioning/settings returning the current provisioning settings for the authenticated tenant
4. THE Network_Service SHALL expose PUT /api/v1/olt/provisioning/settings accepting updated settings and returning the saved settings
5. WHEN a tenant has no provisioning settings record, THE Network_Service SHALL use default values: auto_provisioning_enabled=false, auto_port_migration_enabled=false, vlan_strategy="single"
