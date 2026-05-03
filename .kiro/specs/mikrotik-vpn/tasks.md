# Tasks — VPN Tunnel Management Layer

## Task 1: Domain Entities, Constants, dan Errors

- [x] 1.1 Buat file `internal/domain/vpn.go` — VPN entities (VPNTunnel, VPNSubnet, VPNBandwidthMetrics, VPNBandwidthPoint, TunnelHealthUpdate), VPNProtocol constants (wireguard, l2tp_ipsec, pptp, sstp, openvpn), TunnelStatus constants (connected, disconnected, pending, error), ValidTunnelTransitions map, helper functions (IsValidVPNProtocol, CanTransitionTunnel, BuildClientIP, BuildServerIP, BuildSubnetPrefix, IsValidClientSeq), MaxClientsPerSubnet constant
  - Maksimal 200 baris per file
  - _Requirements: 1.1, 1.5, 1.6, 2.1, 2.3, 2.4, 2.5_

- [x] 1.2 Buat file `internal/domain/vpn_dto.go` — Request DTOs (CreateVPNTunnelRequest, UpdateVPNTunnelRequest, VPNTunnelListParams), Response DTOs (VPNTunnelResponse, VPNTunnelDetailResponse, VPNTunnelListResult, VPNSummary, VPNTestResult, VPNBandwidthResult), Event payloads (VPNTunnelDownPayload, VPNTunnelUpPayload, VPNTunnelCreatedPayload, VPNServerBandwidthHighPayload, VPNServerBandwidthNormalPayload, VPNMaintenanceScheduledPayload)
  - Maksimal 200 baris per file
  - _Requirements: 9.1, 9.3, 9.4, 16.1, 16.2, 16.3, 16.4, 17.1, 17.2, 17.3, 17.4, 19.2, 19.3, 20.2_

- [x] 1.3 Buat file `internal/domain/vpn_command.go` — RouterOS VPN command parameter structs (WireGuardInterfaceParams, WireGuardPeerParams, L2TPClientParams, IPSecProfileParams, IPSecProposalParams, PPTPClientParams, SSTPClientParams, OpenVPNClientParams, IPAddressParams, IPRouteParams, FirewallFilterParams)
  - Maksimal 200 baris per file
  - _Requirements: 4.4, 4.8, 6.2, 6.3, 7.2, 7.3, 7.4, 7.5, 7.6_

- [x] 1.4 Update file `internal/domain/errors.go` — Tambahkan VPN-specific domain errors (ErrVPNTunnelNotFound, ErrVPNTunnelNameExists, ErrVPNIPExists, ErrVPNSubnetExhausted, ErrInvalidVPNProtocol, ErrWireGuardRequiresV7, ErrInvalidTunnelTransition, ErrTunnelImmutableField, ErrVPNConnectionFailed, ErrVPNHandshakeTimeout, ErrVPNAuthFailure, ErrRouterNotOnline, ErrAutoConfigFailed, ErrKeyGenerationFailed, ErrVPNIPUpdateFailed, ErrTunnelDeleteWarning)
  - _Requirements: 2.5, 3.2, 5.3, 6.4, 6.5, 10.3, 10.6, 11.1_

- [x] 1.5 Buat file `internal/domain/vpn_test.go` — Property tests untuk IP allocation uniqueness dan subnet range (Property 1), tunnel status transition validity (Property 5), protocol-version compatibility (Property 6)
  - **Property 1: VPN IP allocation uniqueness and subnet range**
  - **Property 5: Tunnel status transition validity**
  - **Property 6: Protocol-version compatibility**
  - **Validates: Requirements 1.4, 1.5, 2.1, 2.3, 2.4, 3.1, 3.2, 5.5, 8.4, 8.5**

## Task 2: VPN Repository Interfaces dan Database Schema

- [x] 2.1 Update file `internal/domain/repository.go` — Tambahkan VPNTunnelRepository dan VPNSubnetRepository interfaces
  - _Requirements: 1.1, 1.2, 2.2_

- [x] 2.2 Buat SQL migration file `migrations/000005_create_vpn_subnets.up.sql` — CREATE TABLE vpn_subnets dengan semua kolom, UNIQUE constraints (tenant_id, tenant_seq), RLS policy
  - _Requirements: 2.2_

- [x] 2.3 Buat SQL migration file `migrations/000006_create_vpn_tunnels.up.sql` — CREATE TABLE vpn_tunnels dengan semua kolom (termasuk rate_limit_pps), unique indexes (tenant_id+tunnel_name, tenant_id+vpn_ip WHERE deleted_at IS NULL), indexes (tenant_id, status, router_id), RLS policy
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 18.2_

- [x] 2.3b Buat SQL migration file `migrations/000007_create_vpn_maintenance_windows.up.sql` — CREATE TABLE vpn_maintenance_windows dengan kolom: id, server_endpoint, scheduled_start, scheduled_end, description, created_by, created_at. Index pada scheduled_start
  - _Requirements: 20.3_

- [x] 2.4 Buat sqlc queries file `queries/vpn_subnets.sql` — CRUD queries (GetByTenantID, Create, GetNextTenantSeq, IncrementNextClientIPSeq)
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 2.5 Buat sqlc queries file `queries/vpn_tunnels.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, List, GetByStatus, CountByStatus, TunnelNameExists, VPNIPExists, UpdateStatus, GetConnectedTunnels, GetDisconnectedTunnels)
  - _Requirements: 1.1, 8.1, 9.1, 9.2, 10.1, 10.4_

- [x] 2.6 Jalankan `sqlc generate` dan buat repository wrapper `internal/repository/vpn_tunnel_repo.go`
  - _Requirements: 1.1, 1.2_

- [x] 2.7 Buat repository wrapper `internal/repository/vpn_subnet_repo.go`
  - _Requirements: 2.2_

## Task 3: VPN Key Generator

- [x] 3.1 Buat file `internal/usecase/vpn_key_generator.go` — VPNKeyGenerator implementation: GenerateWireGuardKeyPair (curve25519), GeneratePreSharedKey (256-bit random), GenerateCredentials (random username/password), GenerateIPSecPSK (random PSK)
  - Maksimal 200 baris per file
  - _Requirements: 4.1, 4.2, 4.5, 4.6, 11.2_

- [x] 3.2 Buat file `internal/usecase/vpn_key_generator_test.go` — Property test untuk key encryption round-trip (Property 2)
  - **Property 2: Key encryption round-trip**
  - **Validates: Requirements 4.2, 11.1, 11.6**

## Task 4: RouterOS VPN Command Builder

- [x] 4.1 Buat file `internal/adapter/vpn_command_builder.go` — VPNCommandBuilder implementation: WireGuard commands (CreateWireGuardInterface, AddWireGuardPeer, RemoveWireGuardInterface, RemoveWireGuardPeer), L2TP commands (CreateL2TPClient, RemoveL2TPClient, CreateIPSecProfile, CreateIPSecProposal), PPTP commands (CreatePPTPClient, RemovePPTPClient), SSTP commands (CreateSSTPClient, RemoveSSTPClient), OpenVPN commands (CreateOpenVPNClient, RemoveOpenVPNClient), Common commands (AddIPAddress, RemoveIPAddressByInterface, AddIPRoute, AddFirewallFilter)
  - Maksimal 200 baris per file, split jika perlu
  - _Requirements: 6.2, 6.3_

- [x] 4.2 Buat file `internal/adapter/vpn_command_builder_test.go` — Unit tests: command parameter completeness per protocol, correct RouterOS paths, argument mapping
  - _Requirements: 6.2, 6.3_

## Task 5: VPN Script Generator

- [x] 5.1 Buat file `internal/usecase/vpn_script_generator.go` — VPNScriptGenerator implementation: template-based .rsc generation per protocol (WireGuard, L2TP/IPSec, PPTP, SSTP, OpenVPN), include firewall rules, include failover endpoints, include comments
  - Maksimal 200 baris per file, split templates jika perlu
  - _Requirements: 4.4, 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 11.3, 14.2, 14.3_

- [x] 5.2 Buat file `internal/usecase/vpn_script_generator_test.go` — Property tests untuk script generation completeness per protocol (Property 3) dan script security no server private key exposure (Property 4)
  - **Property 3: Script generation completeness per protocol**
  - **Property 4: Script security — no server private key exposure**
  - **Validates: Requirements 4.4, 7.1, 7.2, 7.3, 7.4, 7.5, 7.7, 11.3**

## Task 6: VPN Manager Usecase — Core Operations

- [x] 6.1 Buat file `internal/usecase/vpn_manager.go` — Struct vpnManager dengan dependencies (VPNTunnelRepo, VPNSubnetRepo, RouterRepo, PoolManager, CredentialEncryptor, VPNKeyGenerator, VPNScriptGenerator, VPNEventPublisher, VPNCommandBuilder, VPNBandwidthStore), constructor NewVPNManager
  - _Requirements: 3.1, 4.1, 5.1_

- [x] 6.2 Implementasi `CreateTunnel` — Validate protocol, check router version (WireGuard v7+), GetOrCreateSubnet, AllocateNextIP, generate key/credential per protocol, encrypt keys, save to DB, publish vpn_tunnel_created event
  - _Requirements: 2.1, 2.4, 3.1, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3, 4.5, 4.6, 4.7, 13.1_

- [x] 6.3 Implementasi `GetTunnel` — Retrieve tunnel by ID, mask private keys, return VPNTunnelDetailResponse
  - _Requirements: 9.3, 11.5_

- [x] 6.4 Implementasi `UpdateTunnel` — Validate allowed fields (tunnel_name, notes, router_id, persistent_keepalive, allowed_addresses), reject immutable fields (vpn_ip, protocol, keys), update DB
  - _Requirements: 10.1, 10.2, 10.3_

- [x] 6.5 Implementasi `DeleteTunnel` — Soft-delete tunnel, remove VPN peer from server (best-effort), attempt remove interface from router if online, warn if router uses VPN IP as host
  - _Requirements: 10.4, 10.5, 10.6_

- [x] 6.6 Implementasi `ListTunnels` — Query DB with pagination, filter by status/protocol, return VPNTunnelListResult
  - _Requirements: 9.1, 9.2, 17.1_

- [x] 6.7 Implementasi `GetSummary` — CountByStatus, return VPNSummary (total, connected, disconnected, pending, error)
  - _Requirements: 9.4, 17.10_

## Task 7: VPN Manager Usecase — Setup & Configure

- [x] 7.1 Implementasi `TestConnection` — Ping client VPN IP, measure latency, check last handshake, return VPNTestResult with diagnostic info, update status to connected on success
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 17.6_

- [x] 7.2 Implementasi `AutoConfigure` — Verify router online, build VPN commands per protocol via VPNCommandBuilder, execute via pool, update status to pending, initiate test
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 17.7_

- [x] 7.3 Implementasi `GenerateScript` — Delegate to VPNScriptGenerator, return script content with Content-Type text/plain
  - _Requirements: 7.1, 7.6, 7.7, 17.8_

- [x] 7.4 Implementasi `UpdateRouterHost` — Update router host field to VPN IP, test connectivity via new IP, revert on failure
  - _Requirements: 15.1, 15.2, 15.3, 15.5_

- [x] 7.5 Implementasi `GetBandwidth` — Query VPNBandwidthStore for tunnel metrics, return VPNBandwidthResult with current and history
  - _Requirements: 12.1, 12.4, 17.9_

## Task 8: Checkpoint — Core Logic

- [x] 8. Checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Task 9: VPN Event Publisher

- [x] 9.1 Buat file `internal/usecase/vpn_event_publisher.go` — VPNEventPublisher implementation: PublishTunnelDown (event type "mikrotik.vpn_tunnel_down"), PublishTunnelUp (event type "mikrotik.vpn_tunnel_up"), PublishTunnelCreated (event type "mikrotik.vpn_tunnel_created"), PublishServerBandwidthHigh (event type "mikrotik.vpn_server_bandwidth_high"), PublishServerBandwidthNormal (event type "mikrotik.vpn_server_bandwidth_normal"), PublishMaintenanceScheduled (event type "mikrotik.vpn_maintenance_scheduled"), best-effort publish with error logging, correlation_id generation
  - _Requirements: 16.1, 16.2, 16.3, 16.4, 19.2, 19.3, 20.2_

- [x] 9.2 Buat file `internal/usecase/vpn_event_publisher_test.go` — Property test untuk VPN event payload completeness (Property 7)
  - **Property 7: VPN event payload completeness**
  - **Validates: Requirements 16.1, 16.2, 16.3, 16.4**

## Task 10: VPN Health Monitor

- [x] 10.1 Buat file `internal/usecase/vpn_health_monitor.go` — VPNHealthMonitor implementation: background goroutine, 30-second ticker, GetConnectedTunnels, ping client VPN IP per tunnel, update latency_ms dan last_handshake_at on success, transition to disconnected if ping fails or handshake > 150s, publish vpn_tunnel_down event, recovery detection for disconnected tunnels, publish vpn_tunnel_up event on recovery, aggregate bandwidth check setiap 60 detik (publish vpn_server_bandwidth_high jika > 80%, publish vpn_server_bandwidth_normal jika turun < 70%)
  - Maksimal 200 baris per file
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 19.1, 19.2, 19.3, 19.5_

## Task 11: VPN Bandwidth Store

- [x] 11.1 Buat file `internal/usecase/vpn_bandwidth_store.go` — VPNBandwidthStore implementation: Redis sorted sets (key: vpn:bw:{tunnel_id}), Store data point with unix timestamp as score, Query by time range, GetLatest, 24-hour TTL per key
  - _Requirements: 12.1, 12.2, 12.3, 12.4_

## Task 12: HTTP Handlers

- [x] 12.1 Buat file `internal/handler/vpn_handler.go` — VPNHandler struct dengan methods: ListTunnels (GET /tunnels), CreateTunnel (POST /tunnels), GetTunnel (GET /tunnels/:id), UpdateTunnel (PUT /tunnels/:id), DeleteTunnel (DELETE /tunnels/:id), TestConnection (POST /tunnels/:id/test), AutoConfigure (POST /tunnels/:id/auto-configure), GenerateScript (GET /tunnels/:id/script), GetBandwidth (GET /tunnels/:id/bandwidth), GetSummary (GET /summary), ScheduleMaintenance (POST /admin/vpn/maintenance), GetUpcomingMaintenance (GET /vpn/maintenance)
  - Maksimal 200 baris per file, split jika perlu
  - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8, 17.9, 17.10, 17.11, 20.1, 20.4, 20.5_

- [x] 12.2 Buat file `internal/handler/vpn_handler_test.go` — Unit tests: request validation, response format, error mapping (400/404/409/422/503), Content-Type/Content-Disposition for script download
  - _Requirements: 17.1, 17.2, 17.11_

- [x] 12.3 Update file `internal/handler/router.go` — Tambahkan VPNHandler ke RouterConfig struct, register VPN route group (/api/v1/mikrotik/vpn/tunnels, /api/v1/mikrotik/vpn/summary)
  - _Requirements: 17.1, 17.10_

## Task 13: Wiring dan Integration

- [x] 13.1 Update `internal/config/config.go` — Tambahkan VPN config fields: VPNServerEndpoint, VPNSecondaryEndpoint, VPNServerPublicKey, VPNSecondaryServerPublicKey, VPNListenPort, VPNHealthCheckInterval, VPNBandwidthCollectInterval, VPNServerCapacityMbps (default 1000)
  - _Requirements: 14.1, 14.2, 19.4_

- [x] 13.2 Update `cmd/main.go` — Wire VPN dependencies: VPNTunnelRepo, VPNSubnetRepo, VPNKeyGenerator, VPNCommandBuilder, VPNScriptGenerator, VPNBandwidthStore, VPNEventPublisher, VPNManager, VPNHealthMonitor, VPNHandler. Start VPNHealthMonitor goroutine, stop on shutdown
  - _Requirements: 8.1_

- [x] 13.3 Register VPN routes dan handler — Tambahkan VPNHandler ke RegisterRoutes, pastikan auth dan tenant middleware aktif
  - _Requirements: 17.11_

- [x] 13.4 Integration test end-to-end — Test full flow: create tunnel → generate script → test connection → connected. Test auto-configure flow. Test health monitor cycle (connected → disconnected → recovery). Test cross-tenant isolation (RLS)
  - _Requirements: 1.2, 5.1, 6.1, 8.4, 8.5_

## Task 14: Final Checkpoint

- [x] 14. Final checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- Maksimal 200 baris per file Go — split ke file terpisah jika melebihi
- Gunakan `pgregory.net/rapid` untuk property-based testing (sudah ada di go.mod)
- Semua komentar dalam bahasa Indonesia
