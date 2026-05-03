# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **VPN Tunnel Management Layer** di `services/network-service/`. Layer ini dibangun di atas **MikroTik Router Foundation Layer** (spec `mikrotik-router`) dan **PPPoE Management Layer** (spec `mikrotik-pppoe`) yang sudah diimplementasikan.

ISPBoss adalah SaaS yang di-host di cloud, sementara router MikroTik dan OLT berada di lokasi tenant (on-premise). VPN Tunnel Management menyelesaikan masalah konektivitas: tenant tanpa IP publik, IP publik dinamis, keamanan koneksi RouterOS API tanpa enkripsi, dan kebutuhan manajemen terpusat untuk ISP multi-site.

Layer ini mencakup **management plane** saja: CRUD tunnel, konfigurasi generation, monitoring status, bandwidth tracking, dan API endpoints. Infrastruktur VPN server (WireGuard/L2TP server process) dikelola terpisah dan berada di luar scope spec ini.

## Glossary

- **Network_Service**: Go microservice (`services/network-service/`) yang menangani semua integrasi perangkat jaringan
- **VPN_Manager**: Komponen usecase yang mengelola lifecycle VPN tunnel: create, configure, monitor, delete
- **VPN_Tunnel**: Entitas koneksi VPN antara perangkat on-premise tenant dan VPN server ISPBoss
- **VPN_Server**: Server VPN ISPBoss yang menerima koneksi dari perangkat tenant (vpn.ispboss.id)
- **WireGuard**: Protokol VPN modern, cepat, dan ringan — direkomendasikan untuk RouterOS v7+
- **L2TP_IPSec**: Protokol VPN stabil yang didukung semua versi RouterOS (v6 dan v7)
- **PPTP**: Protokol VPN legacy, kurang aman, digunakan jika tidak ada opsi lain
- **SSTP**: Protokol VPN yang dapat melewati firewall/NAT ketat
- **OpenVPN**: Protokol VPN open source, fleksibel — alternatif jika WireGuard tidak tersedia
- **Tunnel_Status**: Status koneksi VPN tunnel: connected, disconnected, pending, error
- **VPN_IP**: Alamat IP yang diberikan ke perangkat dalam jaringan VPN (subnet 10.99.X.0/24)
- **Key_Pair**: Pasangan public key dan private key yang digunakan untuk autentikasi VPN (WireGuard)
- **Pre_Shared_Key**: Kunci tambahan opsional (PSK) untuk lapisan keamanan ekstra pada WireGuard
- **Setup_Wizard**: Proses 3 langkah untuk membuat VPN tunnel baru (pilih protokol → generate config → verifikasi)
- **Auto_Configure**: Fitur setup VPN otomatis via RouterOS API untuk router yang sudah terdaftar dan online
- **RSC_Script**: File script RouterOS (.rsc) yang berisi perintah konfigurasi VPN untuk dijalankan manual di router
- **Health_Monitor**: Komponen yang memonitor status koneksi VPN setiap 30 detik (ping, handshake, latency)
- **Bandwidth_Cap**: Batas bandwidth per tunnel berdasarkan tier tenant (Starter/Growth/Pro/Enterprise)
- **Failover_Endpoint**: VPN endpoint sekunder untuk geo-redundancy (vpn1.ispboss.id, vpn2.ispboss.id)
- **Tenant_Sequence**: Nomor urut tenant yang digunakan untuk alokasi subnet VPN (10.99.{tenant_seq}.0/24)
- **Persistent_Keepalive**: Mekanisme di sisi MikroTik untuk menjaga koneksi VPN tetap aktif
- **Rate_Limiting**: Pembatasan jumlah koneksi/request per tunnel untuk mencegah abuse
- **Maintenance_Notification**: Notifikasi ke semua tenant saat VPN server dijadwalkan maintenance

## Requirements

### Requirement 1: VPN Tunnel Entity dan Database Schema

**User Story:** As a platform engineer, I want a database schema for storing VPN tunnel configurations per tenant, so that the system can track which VPN tunnels exist, their status, and associated keys.

#### Acceptance Criteria

1. THE Network_Service SHALL store VPN tunnel entities in a `vpn_tunnels` table with columns: id (UUID), tenant_id (UUID), router_id (UUID nullable FK to routers), tunnel_name (VARCHAR 100), protocol (VARCHAR 20), vpn_ip (VARCHAR 45), server_endpoint (VARCHAR 255), server_public_key (TEXT), client_public_key (TEXT), client_private_key_encrypted (TEXT), pre_shared_key_encrypted (TEXT nullable), status (VARCHAR 20 DEFAULT 'pending'), listen_port (INTEGER), allowed_addresses (TEXT), persistent_keepalive (INTEGER DEFAULT 25), last_handshake_at (TIMESTAMPTZ), latency_ms (INTEGER), bandwidth_cap_mbps (INTEGER), notes (TEXT), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ), deleted_at (TIMESTAMPTZ)
2. THE Network_Service SHALL enforce Row-Level Security on the `vpn_tunnels` table so that queries only return rows matching the current tenant context
3. THE Network_Service SHALL enforce a unique constraint on (tenant_id, tunnel_name) WHERE deleted_at IS NULL to prevent duplicate tunnel names within a tenant
4. THE Network_Service SHALL enforce a unique constraint on (tenant_id, vpn_ip) WHERE deleted_at IS NULL to prevent duplicate VPN IP assignments within a tenant
5. THE Network_Service SHALL define status values: "connected", "disconnected", "pending", "error"
6. THE Network_Service SHALL define protocol values: "wireguard", "l2tp_ipsec", "pptp", "sstp", "openvpn"

### Requirement 2: VPN IP Address Allocation

**User Story:** As a platform engineer, I want automatic VPN IP address allocation per tenant using a dedicated subnet, so that each tenant has isolated addressing and can connect up to 253 devices.

#### Acceptance Criteria

1. THE VPN_Manager SHALL allocate VPN IP addresses from subnet 10.99.{tenant_seq}.0/24 where tenant_seq is a unique sequence number per tenant
2. THE Network_Service SHALL store tenant VPN subnet allocation in a `vpn_subnets` table with columns: id (UUID), tenant_id (UUID UNIQUE), subnet_prefix (VARCHAR 18), tenant_seq (INTEGER UNIQUE), server_ip (VARCHAR 45), next_client_ip_seq (INTEGER DEFAULT 2), created_at (TIMESTAMPTZ)
3. THE VPN_Manager SHALL reserve 10.99.{tenant_seq}.1 as the VPN server IP for each tenant subnet
4. THE VPN_Manager SHALL assign client IPs sequentially starting from 10.99.{tenant_seq}.2 up to 10.99.{tenant_seq}.254
5. IF a tenant attempts to create more than 253 VPN tunnels, THEN THE VPN_Manager SHALL return an error indicating the subnet is exhausted
6. WHEN a VPN tunnel is deleted, THE VPN_Manager SHALL NOT recycle the IP address immediately to prevent routing conflicts

### Requirement 3: VPN Setup Wizard — Step 1 (Protocol Selection)

**User Story:** As an ISP admin, I want to choose a VPN protocol and select a router for the tunnel, so that I can set up connectivity appropriate for my router version and network conditions.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/mikrotik/vpn/tunnels with protocol and optional router_id, THE VPN_Manager SHALL validate the protocol is one of: "wireguard", "l2tp_ipsec", "pptp", "sstp", "openvpn"
2. WHEN protocol is "wireguard" and the associated router has RouterOS version v6, THE VPN_Manager SHALL return a warning indicating WireGuard requires RouterOS v7 or later
3. THE VPN_Manager SHALL allow tunnel creation without a router_id for standalone tunnels (used for OLT access or routers not yet registered)
4. WHEN a router_id is provided, THE VPN_Manager SHALL validate that the router exists and belongs to the authenticated tenant

### Requirement 4: VPN Setup Wizard — Step 2 (Configuration Generation)

**User Story:** As an ISP admin, I want VPN configuration automatically generated including key pairs and IP assignments, so that I can quickly set up the tunnel without manual cryptographic operations.

#### Acceptance Criteria

1. WHEN a VPN tunnel is created with protocol "wireguard", THE VPN_Manager SHALL generate a WireGuard key pair (public key + private key) for the client side
2. THE VPN_Manager SHALL store the client private key encrypted using the existing Credential_Encryptor (AES-256-GCM) and the client public key in plaintext
3. THE VPN_Manager SHALL assign the next available VPN IP from the tenant's subnet to the new tunnel
4. THE VPN_Manager SHALL generate a RouterOS configuration script (.rsc) containing: WireGuard interface creation, peer configuration with ISPBoss server public key and endpoint, and IP address assignment
5. WHEN a VPN tunnel is created with protocol "l2tp_ipsec", THE VPN_Manager SHALL generate a random IPSec pre-shared key and L2TP credentials
6. WHEN a VPN tunnel is created with protocol "pptp" or "sstp", THE VPN_Manager SHALL generate appropriate credentials for the selected protocol
7. THE VPN_Manager SHALL include persistent-keepalive=25 in WireGuard configurations to maintain connectivity through NAT
8. WHEN a VPN tunnel is created with protocol "openvpn", THE VPN_Manager SHALL generate OpenVPN credentials (username, password, certificate reference) and client configuration

### Requirement 5: VPN Setup Wizard — Step 3 (Connection Verification)

**User Story:** As an ISP admin, I want to verify that the VPN tunnel is working after setup, so that I can confirm connectivity before relying on it for router management.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/mikrotik/vpn/tunnels/:id/test, THE VPN_Manager SHALL test the VPN connection by pinging the client VPN IP from the server side
2. WHEN the VPN connection test succeeds, THE VPN_Manager SHALL return the connection status, measured latency in milliseconds, and last handshake timestamp
3. WHEN the VPN connection test fails, THE VPN_Manager SHALL return an error with diagnostic information: unreachable, handshake timeout, or authentication failure
4. WHEN the VPN connection test succeeds and the tunnel has an associated router_id, THE VPN_Manager SHALL offer to update the router's host field to the VPN IP address
5. THE VPN_Manager SHALL update the tunnel status to "connected" upon successful verification and record the last_handshake_at timestamp

### Requirement 6: Auto-Configure via RouterOS API

**User Story:** As an ISP admin, I want to automatically configure VPN on a router that is already registered and online, so that I don't need to manually run scripts on the router.

#### Acceptance Criteria

1. WHEN a POST request is made to /api/v1/mikrotik/vpn/tunnels/:id/auto-configure, THE VPN_Manager SHALL verify the associated router is online and accessible
2. WHEN auto-configuring WireGuard, THE VPN_Manager SHALL execute RouterOS commands to: create a WireGuard interface, add the ISPBoss VPN server as a peer with the correct public key and endpoint, and assign the VPN IP address to the interface
3. WHEN auto-configuring L2TP/IPSec, THE VPN_Manager SHALL execute RouterOS commands to create an L2TP client interface with the correct server address, credentials, and IPSec profile
4. IF the router is offline or unreachable, THEN THE VPN_Manager SHALL return an error indicating the router must be online for auto-configuration
5. IF auto-configuration fails due to a RouterOS command error, THEN THE VPN_Manager SHALL return the error details and suggest using the manual script method instead
6. WHEN auto-configuration succeeds, THE VPN_Manager SHALL update the tunnel status to "pending" and initiate a connection test

### Requirement 7: RSC Script Generation

**User Story:** As an ISP admin, I want to download a RouterOS script file for manual VPN setup, so that I can configure routers that are not yet accessible from ISPBoss.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/mikrotik/vpn/tunnels/:id/script, THE VPN_Manager SHALL return a complete RouterOS script (.rsc) for the tunnel's protocol
2. THE VPN_Manager SHALL generate WireGuard scripts containing: interface creation with private key, peer addition with server public key and endpoint, IP address assignment, and optional firewall rules
3. THE VPN_Manager SHALL generate L2TP/IPSec scripts containing: IPSec profile and proposal, L2TP client interface, IP route for VPN subnet, and connection settings
4. THE VPN_Manager SHALL generate PPTP scripts containing: PPTP client interface creation and IP route configuration
5. THE VPN_Manager SHALL generate SSTP scripts containing: SSTP client interface creation with certificate settings and IP route configuration
6. THE VPN_Manager SHALL generate OpenVPN scripts containing: OpenVPN client interface creation, certificate import commands, IP route configuration, and connection settings
7. THE VPN_Manager SHALL return the script with Content-Type "text/plain" and Content-Disposition header for file download with filename format "{tunnel_name}.rsc"
8. THE VPN_Manager SHALL NOT include the server private key in any generated script

### Requirement 8: VPN Health Monitoring

**User Story:** As an ISP admin, I want continuous monitoring of VPN tunnel health, so that I am immediately aware when a tunnel goes down and can see latency trends.

#### Acceptance Criteria

1. THE Health_Monitor SHALL check VPN tunnel connectivity every 30 seconds for all tunnels with status "connected"
2. THE Health_Monitor SHALL measure round-trip latency by pinging the client VPN IP and store the result in the tunnel's latency_ms field
3. THE Health_Monitor SHALL check the last WireGuard handshake timestamp and consider a tunnel disconnected if no handshake has occurred in the last 150 seconds (2.5 minutes)
4. WHEN a tunnel transitions from "connected" to "disconnected", THE Health_Monitor SHALL publish a "mikrotik.vpn_tunnel_down" event containing tunnel_id, tunnel_name, tenant_id, router_id, and last_handshake_at
5. WHEN a previously disconnected tunnel responds to a health check successfully, THE Health_Monitor SHALL update the status to "connected" and publish a "mikrotik.vpn_tunnel_up" event
6. THE Health_Monitor SHALL skip monitoring for tunnels with status "pending" or "error"
7. WHILE a tunnel has status "connected", THE Health_Monitor SHALL update last_handshake_at and latency_ms on each successful check

### Requirement 9: VPN Dashboard and Tunnel Listing

**User Story:** As an ISP admin, I want a dashboard showing all VPN tunnels with their status, protocol, IP, uptime, and latency, so that I can monitor connectivity at a glance.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/mikrotik/vpn/tunnels, THE Network_Service SHALL return a paginated list of VPN tunnels for the authenticated tenant including: id, tunnel_name, router_id, router_name, protocol, vpn_ip, status, latency_ms, last_handshake_at, created_at
2. WHEN a GET request is made to /api/v1/mikrotik/vpn/tunnels with query parameter status={status}, THE Network_Service SHALL filter tunnels by the specified status
3. WHEN a GET request is made to /api/v1/mikrotik/vpn/tunnels/:id, THE Network_Service SHALL return the full tunnel detail including all configuration fields (excluding decrypted private keys)
4. THE Network_Service SHALL provide a summary endpoint at GET /api/v1/mikrotik/vpn/summary returning: total_tunnels, connected_count, disconnected_count, pending_count, error_count
5. THE Network_Service SHALL never expose decrypted private keys or pre-shared keys in API responses

### Requirement 10: VPN Tunnel CRUD Operations

**User Story:** As an ISP admin, I want to edit, disconnect, reconnect, and delete VPN tunnels, so that I can manage the lifecycle of my VPN connections.

#### Acceptance Criteria

1. WHEN a PUT request is made to /api/v1/mikrotik/vpn/tunnels/:id with updated data, THE VPN_Manager SHALL update the tunnel record and return the updated tunnel
2. THE VPN_Manager SHALL allow updating: tunnel_name, notes, router_id association, persistent_keepalive, and allowed_addresses
3. THE VPN_Manager SHALL NOT allow updating: vpn_ip, protocol, or key pairs after creation (require delete and recreate)
4. WHEN a DELETE request is made to /api/v1/mikrotik/vpn/tunnels/:id, THE VPN_Manager SHALL soft-delete the tunnel record and remove the VPN peer configuration from the server side
5. WHEN a tunnel is deleted and has an associated router that is online, THE VPN_Manager SHALL attempt to remove the VPN interface from the router via API (best-effort, log error if fails)
6. IF the router's host field was updated to the VPN IP, THEN THE VPN_Manager SHALL warn the admin that deleting the tunnel will make the router unreachable via VPN

### Requirement 11: VPN Security

**User Story:** As a platform engineer, I want VPN keys and credentials stored securely and access restricted through firewall rules, so that the VPN infrastructure is protected from unauthorized access.

#### Acceptance Criteria

1. THE VPN_Manager SHALL encrypt all private keys and pre-shared keys using the existing Credential_Encryptor (AES-256-GCM) before storing in the database
2. THE VPN_Manager SHALL generate key pairs locally on the server and never transmit private keys over the network unencrypted
3. THE VPN_Manager SHALL include firewall configuration in generated scripts that restricts VPN traffic to only RouterOS API ports (8728, 8729) and SNMP port (161)
4. THE Network_Service SHALL log all VPN tunnel creation, deletion, and configuration changes in the audit trail with tenant_id, user_id, operation, and timestamp
5. WHEN a GET request is made to /api/v1/mikrotik/vpn/tunnels/:id, THE Network_Service SHALL return client_public_key and server_public_key but SHALL mask client_private_key_encrypted and pre_shared_key_encrypted fields
6. FOR ALL valid private keys, encrypting then decrypting SHALL produce the original key (round-trip property)

### Requirement 12: Bandwidth Monitoring per Tunnel

**User Story:** As an ISP admin, I want to see real-time bandwidth usage per VPN tunnel and historical graphs, so that I can monitor traffic patterns and identify issues.

#### Acceptance Criteria

1. WHEN a GET request is made to /api/v1/mikrotik/vpn/tunnels/:id/bandwidth, THE Network_Service SHALL return current bandwidth statistics: tx_bytes, rx_bytes, tx_rate_bps, rx_rate_bps, and timestamp
2. THE Network_Service SHALL store bandwidth data points in Redis with 24-hour retention for per-tunnel traffic graphs
3. THE Network_Service SHALL collect bandwidth statistics from the VPN server interface counters every 30 seconds for each connected tunnel
4. THE Network_Service SHALL return bandwidth history as time-series data when queried with from and to timestamp parameters

### Requirement 13: Bandwidth Cap per Tenant Tier

**User Story:** As a platform engineer, I want bandwidth caps enforced per VPN tunnel based on the tenant's subscription tier, so that no single tenant monopolizes VPN server resources.

#### Acceptance Criteria

1. THE VPN_Manager SHALL assign bandwidth_cap_mbps based on tenant tier: Starter=10, Growth=50, Pro=200, Enterprise=custom value from tenant settings
2. WHEN a tunnel's traffic exceeds the bandwidth cap, THE VPN_Manager SHALL throttle (rate-limit) the traffic rather than dropping packets
3. THE VPN_Manager SHALL store the bandwidth cap in the vpn_tunnels record and apply it via traffic shaping on the VPN server interface
4. WHEN a tenant's tier changes, THE VPN_Manager SHALL update the bandwidth_cap_mbps for all active tunnels belonging to that tenant
5. THE Network_Service SHALL include bandwidth_cap_mbps and current utilization percentage in the tunnel detail API response

### Requirement 14: VPN High Availability

**User Story:** As a platform engineer, I want VPN endpoints geo-redundant with automatic failover, so that tenant connectivity is maintained even if one VPN server goes down.

#### Acceptance Criteria

1. THE VPN_Manager SHALL support two VPN server endpoints: vpn1.ispboss.id (primary) and vpn2.ispboss.id (secondary)
2. THE VPN_Manager SHALL include both endpoints in generated WireGuard scripts as multi-peer configuration for client-side failover
3. THE VPN_Manager SHALL include failover script logic in L2TP/PPTP/SSTP configurations that switches to the secondary endpoint if the primary is unreachable for 30 seconds
4. THE VPN_Manager SHALL store the active endpoint per tunnel and update it when failover occurs
5. THE Network_Service SHALL target 99.9% uptime SLA for VPN connectivity measured across both endpoints

### Requirement 15: Router Integration after VPN Establishment

**User Story:** As an ISP admin, I want the option to update my router's connection IP to the VPN IP after the tunnel is established, so that all subsequent router management uses the secure VPN connection.

#### Acceptance Criteria

1. WHEN a VPN tunnel is verified as connected and has an associated router_id, THE VPN_Manager SHALL offer to update the router's host field to the VPN IP address
2. WHEN the admin confirms the IP update, THE VPN_Manager SHALL update the router record's host field to the tunnel's vpn_ip and test connectivity via the new IP
3. IF the connectivity test via VPN IP fails, THEN THE VPN_Manager SHALL revert the router's host field to the original IP and return an error
4. THE VPN_Manager SHALL allow creating tunnels without a router association (standalone) for accessing OLTs or other devices through the VPN
5. WHEN a router's host is updated to VPN IP, THE Network_Service SHALL use the VPN IP for all subsequent operations: health checks, PPPoE management, and command execution

### Requirement 16: VPN Event Publishing

**User Story:** As a platform engineer, I want VPN status change events published to the Redis queue, so that the notification service can alert admins about tunnel connectivity issues.

#### Acceptance Criteria

1. WHEN a VPN tunnel transitions from "connected" to "disconnected", THE Network_Service SHALL publish a TaskEnvelope with event_type "mikrotik.vpn_tunnel_down" containing: tunnel_id, tunnel_name, tenant_id, router_id, protocol, vpn_ip, last_handshake_at, disconnected_at
2. WHEN a VPN tunnel transitions from "disconnected" to "connected", THE Network_Service SHALL publish a TaskEnvelope with event_type "mikrotik.vpn_tunnel_up" containing: tunnel_id, tunnel_name, tenant_id, router_id, protocol, vpn_ip, latency_ms, connected_at
3. WHEN a VPN tunnel creation completes (success or failure), THE Network_Service SHALL publish a TaskEnvelope with event_type "mikrotik.vpn_tunnel_created" containing: tunnel_id, tunnel_name, tenant_id, protocol, status, error_message (if failed)
4. THE Network_Service SHALL include a correlation_id in every published VPN event for distributed tracing

### Requirement 17: HTTP API Endpoints

**User Story:** As a frontend developer, I want well-defined REST API endpoints for VPN tunnel management, so that the dashboard can interact with the VPN management layer.

#### Acceptance Criteria

1. THE Network_Service SHALL expose GET /api/v1/mikrotik/vpn/tunnels returning a paginated list of tunnels with filtering by status and protocol
2. THE Network_Service SHALL expose POST /api/v1/mikrotik/vpn/tunnels accepting tunnel creation payload (protocol, router_id, tunnel_name) and returning the created tunnel with generated configuration
3. THE Network_Service SHALL expose GET /api/v1/mikrotik/vpn/tunnels/:id returning full tunnel detail
4. THE Network_Service SHALL expose PUT /api/v1/mikrotik/vpn/tunnels/:id accepting tunnel update payload and returning the updated tunnel
5. THE Network_Service SHALL expose DELETE /api/v1/mikrotik/vpn/tunnels/:id performing soft-delete and server-side cleanup
6. THE Network_Service SHALL expose POST /api/v1/mikrotik/vpn/tunnels/:id/test performing connection verification and returning status with latency
7. THE Network_Service SHALL expose POST /api/v1/mikrotik/vpn/tunnels/:id/auto-configure performing automatic router configuration via RouterOS API
8. THE Network_Service SHALL expose GET /api/v1/mikrotik/vpn/tunnels/:id/script returning the RouterOS configuration script for the tunnel
9. THE Network_Service SHALL expose GET /api/v1/mikrotik/vpn/tunnels/:id/bandwidth returning bandwidth statistics and history
10. THE Network_Service SHALL expose GET /api/v1/mikrotik/vpn/summary returning tunnel status summary counts
11. WHEN any VPN API request references a tunnel_id that does not belong to the authenticated tenant, THE Network_Service SHALL return HTTP 404

### Requirement 18: Rate Limiting per Tunnel

**User Story:** As a platform engineer, I want rate limiting per VPN tunnel to prevent abuse and protect the VPN server from excessive connection attempts or traffic spikes.

#### Acceptance Criteria

1. THE VPN_Manager SHALL enforce a maximum connection rate per tunnel to prevent abuse (default: 100 packets/second)
2. THE Network_Service SHALL store rate_limit_pps (packets per second) in the vpn_tunnels table as a configurable field per tunnel
3. WHEN a tunnel's traffic exceeds the rate limit, THE VPN_Manager SHALL drop excess packets and log a warning with tunnel_id and tenant_id
4. THE Network_Service SHALL include rate_limit_pps in the tunnel detail API response
5. THE VPN_Manager SHALL apply rate limiting via traffic shaping rules on the VPN server interface, separate from bandwidth cap (bandwidth cap = throughput limit, rate limit = packet rate limit)

### Requirement 19: VPN Server Aggregate Bandwidth Alert

**User Story:** As a platform admin (ISPBoss operator), I want alerts when total VPN server bandwidth exceeds 80%, so that I can proactively scale infrastructure before it impacts tenants.

#### Acceptance Criteria

1. THE Health_Monitor SHALL track aggregate bandwidth usage across all connected VPN tunnels on the VPN server
2. WHEN total VPN server bandwidth usage exceeds 80% of server capacity, THE Network_Service SHALL publish a "mikrotik.vpn_server_bandwidth_high" event containing: server_endpoint, current_usage_mbps, capacity_mbps, utilization_percent, timestamp
3. WHEN total VPN server bandwidth usage drops below 70% after being above 80%, THE Network_Service SHALL publish a "mikrotik.vpn_server_bandwidth_normal" event
4. THE Network_Service SHALL store VPN server capacity_mbps as a configuration value (default: 1000 Mbps)
5. THE Health_Monitor SHALL check aggregate bandwidth every 60 seconds

### Requirement 20: Scheduled Maintenance Notification

**User Story:** As a platform admin, I want to schedule VPN server maintenance windows and automatically notify all affected tenants, so that ISP admins can prepare for temporary connectivity disruptions.

#### Acceptance Criteria

1. THE Network_Service SHALL expose POST /api/v1/admin/vpn/maintenance accepting: server_endpoint, scheduled_start (TIMESTAMPTZ), scheduled_end (TIMESTAMPTZ), description (TEXT)
2. WHEN a maintenance window is scheduled, THE Network_Service SHALL publish a "mikrotik.vpn_maintenance_scheduled" event for each tenant that has active tunnels on the affected server endpoint, containing: tenant_id, server_endpoint, scheduled_start, scheduled_end, description, affected_tunnel_count
3. THE Network_Service SHALL store scheduled maintenance windows in a `vpn_maintenance_windows` table with columns: id (UUID), server_endpoint (VARCHAR 255), scheduled_start (TIMESTAMPTZ), scheduled_end (TIMESTAMPTZ), description (TEXT), created_by (UUID), created_at (TIMESTAMPTZ)
4. WHEN a GET request is made to /api/v1/mikrotik/vpn/maintenance, THE Network_Service SHALL return upcoming maintenance windows that affect the authenticated tenant's tunnels
5. THE Network_Service SHALL include a maintenance_warning field in the VPN summary response if there is an upcoming maintenance window within the next 24 hours
