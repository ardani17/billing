# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **PPPoE Management Layer** di `services/network-service/`. Layer ini dibangun di atas **MikroTik Router Foundation Layer** yang sudah diimplementasikan (spec `mikrotik-router`), dan menangani seluruh lifecycle PPPoE user: pembuatan user saat pelanggan diaktivasi, isolir/buka isolir, suspend/terminate, upgrade/downgrade paket, sinkronisasi database↔router, dan manajemen active sessions.

Semua perintah ke router dijalankan secara **async** melalui Redis queue (asynq worker), bukan langsung dari HTTP request. Network Service menerima event dari Billing API dan mengeksekusi perintah RouterOS yang sesuai, lalu mempublikasikan event hasil (`mikrotik.command_result`).

## Glossary

- **Network_Service**: Go microservice (`services/network-service/`) yang menangani semua integrasi perangkat jaringan
- **PPPoE_Manager**: Komponen usecase yang mengelola lifecycle PPPoE user di router MikroTik
- **PPPoE_Secret**: Entitas user PPPoE di RouterOS (path: `/ppp/secret`)
- **PPPoE_Profile**: Profil bandwidth di RouterOS yang mendefinisikan rate-limit, burst, dan address pool (path: `/ppp/profile`)
- **PPPoE_Session**: Koneksi PPPoE aktif dari pelanggan ke router (path: `/ppp/active`)
- **Simple_Queue**: Antrian bandwidth per user di RouterOS untuk monitoring traffic individual (path: `/queue/simple`)
- **Walled_Garden**: Halaman redirect untuk pelanggan yang diisolir, menampilkan info tagihan
- **Isolir**: Proses penonaktifan layanan pelanggan karena tunggakan pembayaran (disable PPPoE + redirect ke walled garden)
- **Buka_Isolir**: Proses pengaktifan kembali layanan pelanggan setelah pembayaran diterima
- **Event_Worker**: Asynq worker yang memproses event dari Billing API dan mengeksekusi perintah ke router
- **Sync_Job**: Background job periodik yang membandingkan data PPPoE di router dengan database
- **Router_Foundation**: Layer dasar yang sudah diimplementasikan: Router CRUD, RouterOS adapter, connection pool, health checker, credential encryption
- **Command_Result**: Event hasil eksekusi perintah ke router (sukses/gagal) yang dipublikasikan kembali ke queue
- **Orphan_User**: PPPoE user yang ada di router tetapi tidak tercatat di database ISPBoss
- **DNS_Redirect**: Metode isolir yang mengarahkan DNS pelanggan ke server ISPBoss untuk resolve semua domain ke IP walled garden
- **Firewall_NAT_Redirect**: Metode isolir yang menggunakan firewall NAT rule untuk redirect HTTP traffic ke walled garden
- **PPPoE_User_Repository**: Komponen repository yang menyimpan data PPPoE user di database PostgreSQL

## Requirements

### Requirement 1: PPPoE User Entity dan Database Schema

**User Story:** As a platform engineer, I want a database schema for storing PPPoE user information per tenant, so that the system can track which PPPoE users exist on which routers and maintain sync state.

#### Acceptance Criteria

1. THE Network_Service SHALL store PPPoE user entities in a `pppoe_users` table with columns: id (UUID), tenant_id (UUID), customer_id (UUID), router_id (UUID FK to routers), username (VARCHAR 100), password_encrypted (TEXT), profile_name (VARCHAR 100), service (VARCHAR 20 DEFAULT 'pppoe'), remote_address (VARCHAR 45), comment (TEXT), disabled (BOOLEAN DEFAULT false), use_simple_queue (BOOLEAN DEFAULT false), status (VARCHAR 20), last_sync_at (TIMESTAMPTZ), sync_status (VARCHAR 20), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ), deleted_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `pppoe_users` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (router_id, username) WHERE deleted_at IS NULL to prevent duplicate PPPoE usernames per router
4. THE Network_Service SHALL store the comment field in format "ISPBoss:{customer_id}:{tenant_id}" for tracking ownership on the router
5. THE Network_Service SHALL define sync_status values: "synced", "pending_create", "pending_update", "pending_delete", "out_of_sync", "error"

### Requirement 2: PPPoE Profile Entity dan Database Schema

**User Story:** As a platform engineer, I want a database schema for storing PPPoE profile mappings, so that each ISPBoss package maps to a PPPoE profile on the router with correct bandwidth settings.

#### Acceptance Criteria

1. THE Network_Service SHALL store PPPoE profile entities in a `pppoe_profiles` table with columns: id (UUID), tenant_id (UUID), package_id (UUID), profile_name (VARCHAR 100), download_limit (VARCHAR 20), upload_limit (VARCHAR 20), burst_download (VARCHAR 20), burst_upload (VARCHAR 20), burst_threshold_download (VARCHAR 20), burst_threshold_upload (VARCHAR 20), burst_time (VARCHAR 20), address_pool (VARCHAR 100), local_address (VARCHAR 45 DEFAULT 'gateway'), only_one (BOOLEAN DEFAULT true), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce a unique constraint on (tenant_id, profile_name) to prevent duplicate profile names within a tenant
3. THE Network_Service SHALL enforce Row-Level Security on the `pppoe_profiles` table
4. WHEN a profile is created, THE Network_Service SHALL generate the profile_name from the package name by replacing spaces with hyphens and removing special characters

### Requirement 3: PPPoE User Creation (customer.activated Event)

**User Story:** As an ISP admin, I want PPPoE users automatically created on the router when a customer is activated, so that customers can immediately connect to the internet.

#### Acceptance Criteria

1. WHEN the Event_Worker receives a "customer.activated" event with connection_method "pppoe", THE PPPoE_Manager SHALL create a PPPoE secret on the designated router using the RouterOS API command `/ppp/secret/add`
2. THE PPPoE_Manager SHALL set the PPPoE secret parameters: name={pppoe_username}, password={pppoe_password}, service=pppoe, profile={profile_name_from_package}, comment="ISPBoss:{customer_id}:{tenant_id}"
3. WHEN the router's RouterOS version is v6, THE PPPoE_Manager SHALL use API path `/ppp/secret/add` with v6-compatible parameters
4. WHEN the router's RouterOS version is v7, THE PPPoE_Manager SHALL use API path `/ppp/secret/add` with v7-compatible parameters
5. WHEN the PPPoE user is successfully created on the router, THE PPPoE_Manager SHALL save the PPPoE user record in the database with sync_status "synced" and publish a "mikrotik.command_result" event with status "success"
6. IF the PPPoE user creation fails on the router, THEN THE PPPoE_Manager SHALL save the record with sync_status "pending_create", schedule a retry with exponential backoff (max 5 retries), and publish a "mikrotik.command_result" event with status "failed" and error details
7. WHEN use_simple_queue is enabled in tenant settings, THE PPPoE_Manager SHALL also create a simple queue entry on the router with name={username}, target={remote_address}, max-limit={download_limit}/{upload_limit}, and burst settings from the profile

### Requirement 4: PPPoE User Disable/Enable (Isolir/Buka Isolir)

**User Story:** As an ISP admin, I want PPPoE users automatically disabled when a customer is isolated and re-enabled when unblocked, so that billing enforcement is automated.

#### Acceptance Criteria

1. WHEN the Event_Worker receives a "customer.isolir" event, THE PPPoE_Manager SHALL execute the isolir sequence: disable PPPoE user, disconnect active session, and add firewall redirect rules
2. THE PPPoE_Manager SHALL disable the PPPoE user by executing `/ppp/secret/set` with parameter disabled=yes, identified by the username
3. THE PPPoE_Manager SHALL disconnect any active PPPoE session for the user by executing `/ppp/active/remove` for the matching session
4. THE PPPoE_Manager SHALL add a firewall NAT redirect rule with comment "ISPBoss:isolir:{customer_id}" to redirect HTTP traffic to the walled garden IP
5. WHERE the tenant has DNS redirect enabled as isolir method, THE PPPoE_Manager SHALL add a DNS redirect NAT rule with comment "ISPBoss:dns-redirect:{customer_id}" to redirect DNS queries to the ISPBoss DNS server
6. WHEN the Event_Worker receives a "customer.un_isolir" event, THE PPPoE_Manager SHALL execute the buka isolir sequence: enable PPPoE user, remove all firewall redirect rules for that customer, and reset simple queue counters if use_simple_queue is enabled
7. THE PPPoE_Manager SHALL enable the PPPoE user by executing `/ppp/secret/set` with parameter disabled=no
8. THE PPPoE_Manager SHALL remove firewall NAT rules by finding rules with comment matching "ISPBoss:isolir:{customer_id}" and "ISPBoss:dns-redirect:{customer_id}"
9. WHEN isolir or buka isolir completes (success or failure), THE PPPoE_Manager SHALL publish a "mikrotik.command_result" event with the operation result
10. THE PPPoE_Manager SHALL execute isolir/buka isolir commands with CommandPriority High to ensure billing-related operations are processed before other commands

### Requirement 5: PPPoE User Removal (Suspend/Terminate)

**User Story:** As an ISP admin, I want PPPoE users removed from the router when a customer is suspended or terminated, so that deactivated customers cannot access the network.

#### Acceptance Criteria

1. WHEN the Event_Worker receives a "customer.suspend" or "customer.terminated" event, THE PPPoE_Manager SHALL execute the removal sequence: disconnect active session, remove PPPoE secret, remove simple queue (if exists), and remove any firewall rules for that customer
2. THE PPPoE_Manager SHALL disconnect the active session by executing `/ppp/active/remove` for the matching session
3. THE PPPoE_Manager SHALL remove the PPPoE secret by executing `/ppp/secret/remove` identified by the username
4. WHEN a simple queue exists for the user, THE PPPoE_Manager SHALL remove it by executing `/queue/simple/remove` with find by name={username}
5. THE PPPoE_Manager SHALL remove any firewall NAT rules with comment containing "ISPBoss:isolir:{customer_id}" or "ISPBoss:dns-redirect:{customer_id}"
6. WHEN the PPPoE user is successfully removed from the router, THE PPPoE_Manager SHALL soft-delete the PPPoE user record in the database and publish a "mikrotik.command_result" event with status "success"
7. IF the removal fails on the router, THEN THE PPPoE_Manager SHALL mark the record with sync_status "pending_delete" and schedule a retry
8. THE PPPoE_Manager SHALL treat "customer.terminated" events identically to "customer.suspend" events — both execute the same removal sequence

### Requirement 6: PPPoE Profile Sync (Package Create/Update)

**User Story:** As an ISP admin, I want PPPoE profiles automatically created and updated on all routers when I create or edit a package, so that bandwidth settings are always consistent.

#### Acceptance Criteria

1. WHEN a package is created in the billing system with connection_method "pppoe", THE PPPoE_Manager SHALL create a corresponding PPPoE profile on all routers that have service_type "pppoe"
2. THE PPPoE_Manager SHALL create the profile using `/ppp/profile/add` with parameters: name={profile_name}, local-address={local_address}, remote-address={address_pool}, rate-limit={download_limit}/{upload_limit}, only-one={only_one}
3. WHERE burst settings are configured for the profile, THE PPPoE_Manager SHALL include burst parameters: burst-limit={burst_download}/{burst_upload}, burst-threshold={burst_threshold_download}/{burst_threshold_upload}, burst-time={burst_time}
4. WHEN a package is updated, THE PPPoE_Manager SHALL update the corresponding profile on all routers using `/ppp/profile/set` with the changed parameters
5. WHEN a profile already exists on the router (detected by name), THE PPPoE_Manager SHALL update it instead of creating a duplicate
6. THE PPPoE_Manager SHALL sync profiles to all routers with service_type "pppoe" in parallel using goroutines, with errors logged per router without blocking other routers
7. IF a profile sync fails for a specific router, THEN THE PPPoE_Manager SHALL log the error and mark the profile as "pending_sync" for that router

### Requirement 7: Package Change (Upgrade/Downgrade)

**User Story:** As an ISP admin, I want PPPoE profile assignments updated when a customer changes their package, so that bandwidth changes take effect immediately.

#### Acceptance Criteria

1. WHEN the Event_Worker receives a "package.changed" event, THE PPPoE_Manager SHALL update the PPPoE user's profile assignment on the router
2. THE PPPoE_Manager SHALL update the profile by executing `/ppp/secret/set` with parameter profile={new_profile_name} for the user
3. WHEN use_simple_queue is enabled, THE PPPoE_Manager SHALL update the simple queue bandwidth limits by executing `/queue/simple/set` with new max-limit and burst settings
4. THE PPPoE_Manager SHALL disconnect the active PPPoE session after updating the profile to force reconnection with the new profile settings
5. WHEN the package change completes successfully, THE PPPoE_Manager SHALL update the PPPoE user record in the database with the new profile_name and publish a "mikrotik.command_result" event
6. IF the new profile does not exist on the router, THEN THE PPPoE_Manager SHALL create the profile first before assigning it to the user

### Requirement 8: Database ↔ Router Periodic Sync

**User Story:** As an ISP admin, I want periodic synchronization between the database and router PPPoE users, so that discrepancies are detected and corrected automatically.

#### Acceptance Criteria

1. THE Sync_Job SHALL run periodically at a configurable interval (default 15 minutes) for each online router with service_type "pppoe"
2. THE Sync_Job SHALL retrieve all PPPoE secrets from the router using `/ppp/secret/print` and compare them with the database records for that router
3. WHEN a PPPoE user exists on the router but not in the database (orphan), THE Sync_Job SHALL mark it as "orphan" in the sync report without deleting it from the router
4. WHEN a PPPoE user exists in the database with status active but not on the router (missing), THE Sync_Job SHALL auto-create the user on the router and update sync_status to "synced"
5. WHEN a PPPoE user exists in both but with different profile or disabled state (out-of-sync), THE Sync_Job SHALL update the router to match the database (database is source of truth) and update sync_status to "synced"
6. THE Sync_Job SHALL update the last_sync_at timestamp and sync_status for each PPPoE user record after sync completes
7. THE Sync_Job SHALL store sync results per router: total_users, synced_count, orphan_count, missing_count, out_of_sync_count, error_count
8. IF the sync job fails to connect to a router, THEN THE Sync_Job SHALL skip that router and log the error without affecting other routers
9. THE Sync_Job SHALL identify ISPBoss-managed users on the router by checking the comment field for the "ISPBoss:" prefix

### Requirement 9: PPPoE Active Sessions

**User Story:** As an ISP admin, I want to view and manage active PPPoE sessions on my routers, so that I can monitor connected customers and disconnect sessions when needed.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/mikrotik/routers/:id/pppoe/sessions, THE Network_Service SHALL retrieve active PPPoE sessions from the router using `/ppp/active/print` and return them as a JSON array
2. THE Network_Service SHALL return session data including: username, caller_id (MAC), address (IP), uptime, bytes_in, bytes_out, service, encoding
3. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/pppoe/sessions/:session_id/disconnect, THE Network_Service SHALL disconnect the specified session using `/ppp/active/remove`
4. WHEN a GET request is made to /api/v1/mikrotik/routers/:id/pppoe/sessions/count, THE Network_Service SHALL return the total count of active PPPoE sessions on that router
5. IF the router is offline, THEN THE Network_Service SHALL return an error indicating the router is not reachable

### Requirement 10: Event Worker dan Retry Mechanism

**User Story:** As a platform engineer, I want a robust event worker that processes billing events and retries failed commands, so that no customer state change is lost.

#### Acceptance Criteria

1. THE Event_Worker SHALL listen for events: "customer.activated", "customer.isolir", "customer.un_isolir", "customer.suspend", "customer.terminated", "package.changed" from the Redis queue
2. WHEN an event is received, THE Event_Worker SHALL validate the payload, resolve the router and PPPoE user, and execute the appropriate command sequence
3. IF a command execution fails due to router connectivity issues, THEN THE Event_Worker SHALL retry with exponential backoff: delays of 30s, 1m, 2m, 5m, 10m (max 5 retries)
4. IF all retries are exhausted, THEN THE Event_Worker SHALL mark the operation as "failed", update the PPPoE user sync_status to "error", publish a "mikrotik.command_result" event with status "failed_permanent", and publish a "mikrotik.sync_failed" event for notification purposes
5. THE Event_Worker SHALL process events sequentially per router (to prevent race conditions) but in parallel across different routers
6. THE Event_Worker SHALL publish a "mikrotik.command_result" event after each operation containing: correlation_id, customer_id, router_id, tenant_id, operation (create/isolir/un_isolir/suspend/package_change), status (success/failed/failed_permanent), error_message (if failed), executed_at

### Requirement 11: PPPoE User CRUD API (Manual Management)

**User Story:** As an ISP admin, I want REST API endpoints to manually manage PPPoE users on routers, so that I can perform operations outside the automated event flow.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/mikrotik/routers/:id/pppoe/users, THE Network_Service SHALL return a paginated list of PPPoE users for that router from the database
2. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/pppoe/users with valid PPPoE user data, THE Network_Service SHALL create the PPPoE user on the router and in the database
3. WHEN a DELETE request is made to /api/v1/mikrotik/routers/:id/pppoe/users/:user_id, THE Network_Service SHALL remove the PPPoE user from the router and soft-delete the database record
4. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/pppoe/users/:user_id/disconnect, THE Network_Service SHALL disconnect the active session for that user
5. WHEN a GET request is made to /api/v1/mikrotik/routers/:id/pppoe/sync-status, THE Network_Service SHALL return the sync status summary for that router including: synced_count, orphan_count, missing_count, out_of_sync_count, last_sync_at
6. WHEN a POST request is made to /api/v1/mikrotik/routers/:id/pppoe/sync, THE Network_Service SHALL trigger an immediate sync job for that router

### Requirement 12: Command Result Event Publishing

**User Story:** As a platform engineer, I want command execution results published as events, so that the billing service and other consumers can track the outcome of router operations.

#### Acceptance Criteria

1. THE PPPoE_Manager SHALL publish a "mikrotik.command_result" TaskEnvelope after every router operation (create, isolir, un_isolir, suspend, package_change)
2. THE command_result payload SHALL contain: correlation_id, customer_id, router_id, tenant_id, operation, status, error_message, executed_at, duration_ms
3. WHEN the operation status is "success", THE payload SHALL include the router-assigned values (e.g., remote_address if assigned from pool)
4. WHEN the operation status is "failed" or "failed_permanent", THE payload SHALL include a descriptive error_message without exposing sensitive data (passwords, encryption keys)
5. THE Network_Service SHALL include the original event's correlation_id in the command_result for distributed tracing

### Requirement 13: RouterOS Version Compatibility

**User Story:** As a platform engineer, I want PPPoE commands to work correctly on both RouterOS v6 and v7, so that ISPs with different router versions are supported.

#### Acceptance Criteria

1. THE PPPoE_Manager SHALL determine the RouterOS version from the router entity's router_os_version field before executing commands
2. WHEN the router runs RouterOS v6, THE PPPoE_Manager SHALL use the v6 API paths: `/ppp/secret/add`, `/ppp/secret/set`, `/ppp/secret/remove`, `/ppp/secret/print`, `/ppp/active/print`, `/ppp/active/remove`, `/ppp/profile/add`, `/ppp/profile/set`, `/queue/simple/add`, `/queue/simple/set`, `/queue/simple/remove`
3. WHEN the router runs RouterOS v7, THE PPPoE_Manager SHALL use the v7 API paths (same base paths but with v7-specific parameter handling where applicable)
4. THE PPPoE_Manager SHALL handle differences in parameter naming between v6 and v7 (e.g., address vs remote-address variations) transparently

### Requirement 14: Walled Garden Firewall Management

**User Story:** As an ISP admin, I want firewall rules for walled garden redirect managed automatically, so that isolated customers see the billing page instead of accessing the internet.

#### Acceptance Criteria

1. THE PPPoE_Manager SHALL support two isolir methods configurable per tenant: "firewall_nat_redirect" (HTTP redirect via NAT rule) and "dns_redirect" (DNS hijack to ISPBoss DNS server)
2. WHEN isolir method is "firewall_nat_redirect", THE PPPoE_Manager SHALL create a NAT rule: chain=dstnat, src-address={user_remote_ip}, protocol=tcp, dst-port=80, action=dst-nat, to-addresses={walled_garden_ip}, comment="ISPBoss:isolir:{customer_id}"
3. WHEN isolir method is "dns_redirect", THE PPPoE_Manager SHALL create a NAT rule: chain=dstnat, src-address={user_remote_ip}, protocol=udp, dst-port=53, action=dst-nat, to-addresses={ispboss_dns_ip}, comment="ISPBoss:dns-redirect:{customer_id}"
4. WHEN buka isolir is executed, THE PPPoE_Manager SHALL remove all NAT rules matching comment "ISPBoss:isolir:{customer_id}" and "ISPBoss:dns-redirect:{customer_id}" using the find-by-comment pattern
5. THE PPPoE_Manager SHALL use the comment field as the primary identifier for managing firewall rules (never by rule number/position, which can change)
6. IF a firewall rule cannot be found during buka isolir (already removed manually), THEN THE PPPoE_Manager SHALL log a warning and continue without error

