# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **MikroTik Router Foundation Layer** — layer dasar integrasi MikroTik di ISPBoss Network Service. Scope mencakup device registry (database schema), domain entities, RouterOS API adapter (mock + live), Router CRUD API, health check background job, status summary API, dan credential encryption. Fitur lanjutan (PPPoE management, isolir/buka isolir, hotspot, VPN tunnel, sync) akan dicover di spec terpisah.

## Glossary

- **Network_Service**: Go microservice (`services/network-service/`) yang menangani semua integrasi perangkat jaringan (MikroTik, OLT)
- **Router**: Entitas perangkat MikroTik yang terdaftar di sistem ISPBoss per tenant
- **RouterOS_API**: Protokol komunikasi bawaan MikroTik untuk manajemen router secara programatik (port 8728/8729)
- **Connection_Pool**: Kumpulan koneksi TCP ke RouterOS API per router, dikelola dengan strategi lazy connect
- **Health_Checker**: Background job yang memonitor status konektivitas dan metrik setiap router secara periodik
- **Credential_Encryptor**: Komponen yang mengenkripsi/mendekripsi credential router menggunakan AES-256-GCM
- **Router_Status**: Status konektivitas router: online, offline, atau maintenance
- **Mock_Adapter**: Implementasi RouterOS API adapter yang mengembalikan response simulasi tanpa koneksi ke router fisik
- **Live_Adapter**: Implementasi RouterOS API adapter yang berkomunikasi dengan router MikroTik sesungguhnya via library go-routeros
- **Tenant**: Organisasi ISP yang menggunakan platform ISPBoss (multi-tenant SaaS)
- **RLS**: Row-Level Security di PostgreSQL untuk isolasi data antar tenant
- **Metrics_Store**: Penyimpanan metrik router di Redis dengan format time-series dan retensi 7 hari
- **TaskEnvelope**: Format standar event antar service via Redis queue (pkg/queue)

## Requirements

### Requirement 1: Database Schema untuk Router Registry

**User Story:** As a platform engineer, I want a database schema for storing router device information per tenant, so that each ISP tenant can manage their own set of MikroTik routers in isolation.

#### Acceptance Criteria

1. THE Network_Service SHALL store Router entities in a `routers` table with columns: id (UUID), tenant_id (UUID), name (VARCHAR 100), host (VARCHAR 255), port (INTEGER), username (VARCHAR 100), password_encrypted (TEXT), use_ssl (BOOLEAN), service_types (JSONB NOT NULL DEFAULT '["pppoe"]'), router_os_version (VARCHAR 20), board_name (VARCHAR 100), cpu_count (INTEGER), total_ram_mb (INTEGER), identity (VARCHAR 255), status (VARCHAR 20), health_check_interval_sec (INTEGER), last_online_at (TIMESTAMPTZ), last_checked_at (TIMESTAMPTZ), last_uptime_sec (BIGINT), failure_count (INTEGER), notes (TEXT), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `routers` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (tenant_id, name) to prevent duplicate router names within a tenant
4. THE Network_Service SHALL set default values: port=8728, use_ssl=false, status='offline', health_check_interval_sec=60, failure_count=0

### Requirement 2: Domain Entities dan Status Constants

**User Story:** As a developer, I want well-defined domain entities and status constants for routers, so that business logic is consistent and type-safe across the codebase.

#### Acceptance Criteria

1. THE Network_Service SHALL define a Router struct containing all fields from the database schema with appropriate Go types
2. THE Network_Service SHALL define RouterStatus constants: StatusOnline ("online"), StatusOffline ("offline"), StatusMaintenance ("maintenance")
3. THE Network_Service SHALL define valid status transitions: offline→online, offline→maintenance, online→offline, online→maintenance, maintenance→online, maintenance→offline
4. WHEN an invalid status transition is attempted, THE Network_Service SHALL return a domain error indicating the current status and allowed transitions
5. THE Network_Service SHALL define a ConnectionConfig struct containing: Host, Port, Username, Password, UseSSL, ConnectTimeout, CommandTimeout
6. THE Network_Service SHALL define ServiceType constants: ServicePPPoE ("pppoe"), ServiceHotspot ("hotspot"), ServiceDHCP ("dhcp_binding"), ServiceStatic ("static") representing the types of network services a router can provide

### Requirement 3: RouterOS API Adapter Interface

**User Story:** As a developer, I want an adapter interface for RouterOS API communication, so that I can swap between mock and live implementations without changing business logic.

#### Acceptance Criteria

1. THE Network_Service SHALL define a RouterOSAdapter interface with methods: Connect(ctx, ConnectionConfig) error, Close() error, Execute(ctx, command string, params map[string]string) ([]map[string]string, error), GetSystemResource(ctx) (SystemResource, error), Ping(ctx) error
2. THE Network_Service SHALL define a SystemResource struct containing: Version, BoardName, CPUCount, CPULoad, TotalRAM, FreeRAM, Uptime, Architecture, Identity
3. WHILE NETWORK_MODE is set to "mock", THE Network_Service SHALL use the Mock_Adapter that returns predefined simulated responses without establishing network connections
4. WHILE NETWORK_MODE is set to "live", THE Network_Service SHALL use the Live_Adapter that communicates with actual MikroTik routers via the go-routeros library
5. WHEN the Mock_Adapter receives a GetSystemResource call, THE Mock_Adapter SHALL return a valid SystemResource with realistic simulated values (version "6.49.10", board "RB750Gr3", CPU count 2, total RAM 256 MB, uptime 3888000 seconds)
6. WHEN the Live_Adapter receives an Execute call and the router is unreachable, THE Live_Adapter SHALL return an error within the configured ConnectTimeout duration
7. THE Live_Adapter SHALL detect the RouterOS version (v6 or v7) from the SystemResource.Version field and store it in the Router entity, so that future specs (PPPoE, hotspot) can use version-specific API paths

### Requirement 4: Connection Pool per Router

**User Story:** As a platform engineer, I want a connection pool per router with lazy connect strategy, so that connections are efficiently reused without wasting resources on idle routers.

#### Acceptance Criteria

1. THE Connection_Pool SHALL maintain a maximum of 5 concurrent connections per router
2. THE Connection_Pool SHALL create connections lazily (only when a command needs to be executed and no idle connection is available)
3. WHEN a connection has been idle for more than 5 minutes, THE Connection_Pool SHALL close that connection automatically
4. WHEN a connection has been alive for more than 1 hour, THE Connection_Pool SHALL close that connection and create a new one on next demand
5. THE Connection_Pool SHALL perform a health ping every 30 seconds on idle connections and remove connections that fail the ping
6. WHEN all connections in the pool are busy and pool is at maximum capacity, THE Connection_Pool SHALL queue the command and execute it when a connection becomes available
7. WHEN a connection attempt fails, THE Connection_Pool SHALL return an error within 5 seconds (connect timeout)
8. THE Connection_Pool SHALL enforce a rate limit of maximum 10 commands per second per router
9. WHEN the pending command queue for a router exceeds 10 commands, THE Connection_Pool SHALL warm-up the pool to maximum capacity by opening connections in parallel (event-driven warm-up)
10. THE Connection_Pool SHALL support three command priority levels: High (isolir, buka isolir, disconnect), Medium (CRUD user, update profile), Low (sync, monitoring, backup). Higher priority commands SHALL be dequeued before lower priority commands when the pool is busy

### Requirement 5: Router CRUD API

**User Story:** As an ISP admin, I want REST API endpoints to manage my MikroTik routers, so that I can add, view, update, and remove routers from the ISPBoss dashboard.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/mikrotik/routers with valid router data, THE Network_Service SHALL create a new router record, test the connection, auto-detect router info (version, board, CPU, RAM, uptime, identity), and return the created router with HTTP 201
2. WHEN a POST request is made to /api/v1/mikrotik/routers and the test connection fails, THE Network_Service SHALL still create the router record with status "offline" and return HTTP 201 with a warning in the response
3. WHEN a GET request is made to /api/v1/mikrotik/routers, THE Network_Service SHALL return a paginated list of routers belonging to the authenticated tenant
4. WHEN a GET request is made to /api/v1/mikrotik/routers/:id, THE Network_Service SHALL return the router detail including live status metrics (CPU, RAM, uptime) if the router is online
5. WHEN a PUT request is made to /api/v1/mikrotik/routers/:id with updated data, THE Network_Service SHALL update the router record and return the updated router with HTTP 200
6. WHEN a DELETE request is made to /api/v1/mikrotik/routers/:id, THE Network_Service SHALL soft-delete the router record and close all pool connections for that router
7. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/test, THE Network_Service SHALL test the connection to the router and return the connection result with auto-detected system info
8. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/reboot with a valid confirmation (router name matches), THE Network_Service SHALL send a reboot command to the router via RouterOS API
9. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/reboot without valid confirmation, THE Network_Service SHALL reject the request with HTTP 400 and a message indicating confirmation is required
10. WHEN any router API request references a router_id that does not belong to the authenticated tenant, THE Network_Service SHALL return HTTP 404

### Requirement 6: Health Check Background Job

**User Story:** As an ISP admin, I want automatic periodic health checks on my routers, so that I am immediately notified when a router goes offline or experiences an unexpected reboot.

#### Acceptance Criteria

1. THE Health_Checker SHALL check each router's connectivity at the interval configured in health_check_interval_sec (default 60 seconds)
2. WHEN a health check succeeds, THE Health_Checker SHALL update the router's last_checked_at timestamp, reset failure_count to 0, and collect metrics (CPU load, RAM usage, uptime, active sessions)
3. WHEN a health check fails, THE Health_Checker SHALL increment the router's failure_count
4. WHEN a router's failure_count reaches 3 consecutive failures, THE Health_Checker SHALL update the router status to "offline" and publish a "mikrotik.router_offline" event via TaskEnvelope to the Redis queue
5. WHEN a previously offline router responds to a health check successfully, THE Health_Checker SHALL update the router status to "online", reset failure_count to 0, and publish a "mikrotik.router_online" event via TaskEnvelope to the Redis queue
6. WHEN the Health_Checker detects that a router's current uptime is significantly less than the previously recorded uptime (indicating a reboot), THE Health_Checker SHALL publish a "mikrotik.router_unexpected_reboot" event containing the previous uptime and current uptime
7. WHILE a router's status is "maintenance", THE Health_Checker SHALL skip health checks for that router
8. THE Health_Checker SHALL store collected metrics (CPU load, RAM usage, uptime, active sessions count) in the Metrics_Store with a key format "router:{router_id}:metrics" and 7-day retention

### Requirement 7: Router Status Summary API

**User Story:** As an ISP admin, I want a summary endpoint showing the overall status of all my routers, so that I can display a dashboard widget with quick health overview.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/mikrotik/status/summary, THE Network_Service SHALL return a JSON object containing: total_routers, online_count, offline_count, maintenance_count for the authenticated tenant
2. THE Network_Service SHALL compute the summary from the current router statuses in the database without making live connections to routers

### Requirement 8: Credential Encryption

**User Story:** As a platform engineer, I want router credentials encrypted at rest using AES-256-GCM, so that sensitive passwords are protected even if the database is compromised.

#### Acceptance Criteria

1. WHEN a router is created or updated with a password, THE Credential_Encryptor SHALL encrypt the password using AES-256-GCM before storing it in the database
2. WHEN a router's password is needed for connection (health check, test connection, command execution), THE Credential_Encryptor SHALL decrypt the password from the database using the same master key
3. THE Credential_Encryptor SHALL use a 32-byte master key loaded from the ENCRYPTION_KEY environment variable
4. IF the ENCRYPTION_KEY environment variable is not set or is not exactly 32 bytes, THEN THE Network_Service SHALL fail to start with a clear error message
5. THE Credential_Encryptor SHALL generate a unique random nonce for each encryption operation to ensure identical passwords produce different ciphertexts
6. FOR ALL valid passwords, encrypting then decrypting SHALL produce the original password (round-trip property)
7. WHEN decryption fails due to a wrong key or corrupted data, THE Credential_Encryptor SHALL return a descriptive error without exposing the master key or plaintext

### Requirement 9: Router Metrics Time-Series Storage

**User Story:** As an ISP admin, I want router metrics stored over time, so that I can view historical CPU, RAM, and session data for trend analysis.

#### Acceptance Criteria

1. THE Metrics_Store SHALL store each metric data point with a timestamp in Redis using sorted sets with score as Unix timestamp
2. THE Metrics_Store SHALL automatically expire metric entries older than 7 days
3. WHEN metrics are queried for a router, THE Metrics_Store SHALL return data points within the requested time range sorted by timestamp ascending
4. THE Metrics_Store SHALL store metrics with the following structure per router: cpu_load (percentage 0-100), ram_usage_percent (percentage 0-100), uptime_seconds (integer), active_sessions (integer)

### Requirement 10: Router Event Publishing

**User Story:** As a platform engineer, I want router status change events published to the Redis queue, so that other services (notification, billing) can react to router state changes.

#### Acceptance Criteria

1. WHEN a router transitions from online to offline, THE Network_Service SHALL publish a TaskEnvelope with event_type "mikrotik.router_offline" containing router_id, router_name, tenant_id, and last_online_at
2. WHEN a router transitions from offline to online, THE Network_Service SHALL publish a TaskEnvelope with event_type "mikrotik.router_online" containing router_id, router_name, tenant_id, and downtime_duration
3. WHEN an unexpected reboot is detected, THE Network_Service SHALL publish a TaskEnvelope with event_type "mikrotik.router_unexpected_reboot" containing router_id, router_name, tenant_id, previous_uptime_seconds, and current_uptime_seconds
4. THE Network_Service SHALL include a correlation_id in every published event for distributed tracing
