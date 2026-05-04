package domain

import (
	"context"
	"time"
)

// =============================================================================
// RouterRepository — operasi data untuk tabel routers
// =============================================================================

// RouterRepository mendefinisikan operasi data untuk tabel routers.
// Diimplementasikan oleh repository.RouterRepo menggunakan sqlc.
type RouterRepository interface {
	// Create membuat router baru dan mengembalikan router yang dibuat.
	Create(ctx context.Context, router *Router) (*Router, error)

	// GetByID mengambil router berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*Router, error)

	// Update memperbarui data router dan mengembalikan router yang diperbarui.
	Update(ctx context.Context, router *Router) (*Router, error)

	// SoftDelete melakukan soft-delete router (set deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar router dengan paginasi (tenant-scoped via RLS).
	List(ctx context.Context, params RouterListParams) (*RouterListResult, error)

	// CountByStatus menghitung jumlah router per status untuk tenant.
	CountByStatus(ctx context.Context) (map[RouterStatus]int64, error)

	// GetActiveRouters mengambil semua router yang tidak di-delete dan bukan maintenance.
	GetActiveRouters(ctx context.Context) ([]*Router, error)

	// NameExists mengecek apakah nama router sudah ada di tenant.
	NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)

	// UpdateHealthCheck memperbarui field health check (last_checked_at, failure_count, status, last_uptime_sec).
	UpdateHealthCheck(ctx context.Context, id string, params HealthCheckUpdate) error
}

// =============================================================================
// MetricsStore — penyimpanan metrik router di Redis sorted sets
// =============================================================================

// MetricsStore menyimpan dan mengambil metrik router dari Redis.
// Menggunakan sorted set dengan score=unix timestamp dan 7-day TTL.
type MetricsStore interface {
	// Store menyimpan satu data point metrik untuk router.
	Store(ctx context.Context, routerID string, metrics RouterMetrics) error

	// Query mengambil data point metrik dalam rentang waktu tertentu.
	Query(ctx context.Context, routerID string, from, to time.Time) ([]RouterMetricsPoint, error)

	// GetLatest mengambil data point metrik terbaru untuk router.
	GetLatest(ctx context.Context, routerID string) (*RouterMetricsPoint, error)
}

// =============================================================================
// EventPublisher — publikasi event router ke Redis queue
// =============================================================================

// EventPublisher mempublikasikan event router ke Redis queue via pkg/queue.
// Best-effort: log error jika publish gagal, jangan return error ke caller.
type EventPublisher interface {
	// PublishRouterOffline mempublikasikan event router offline.
	PublishRouterOffline(ctx context.Context, router *Router) error

	// PublishRouterOnline mempublikasikan event router online.
	PublishRouterOnline(ctx context.Context, router *Router, downtimeDuration time.Duration) error

	// PublishUnexpectedReboot mempublikasikan event reboot tak terduga.
	PublishUnexpectedReboot(ctx context.Context, router *Router, prevUptime, currUptime int64) error
}

// =============================================================================
// CredentialEncryptor — enkripsi/dekripsi credential router (AES-256-GCM)
// =============================================================================

// CredentialEncryptor mengenkripsi dan mendekripsi credential router.
// Menggunakan AES-256-GCM dengan random nonce per operasi.
type CredentialEncryptor interface {
	// Encrypt mengenkripsi plaintext password menggunakan AES-256-GCM.
	Encrypt(plaintext string) (string, error)

	// Decrypt mendekripsi ciphertext kembali ke plaintext password.
	Decrypt(ciphertext string) (string, error)
}

// =============================================================================
// RouterOSAdapter — interface komunikasi dengan RouterOS API
// =============================================================================

// RouterOSAdapter mendefinisikan interface untuk komunikasi dengan RouterOS API.
// Diimplementasikan oleh MockAdapter dan LiveAdapter.
type RouterOSAdapter interface {
	// Connect membuka koneksi ke router dengan konfigurasi yang diberikan.
	Connect(ctx context.Context, cfg ConnectionConfig) error

	// Close menutup koneksi ke router.
	Close() error

	// Execute menjalankan perintah RouterOS dan mengembalikan hasil.
	Execute(ctx context.Context, command string, params map[string]string) ([]map[string]string, error)

	// GetSystemResource mengambil informasi sistem router (CPU, RAM, uptime, dll).
	GetSystemResource(ctx context.Context) (*SystemResource, error)

	// Ping memeriksa apakah koneksi ke router masih aktif.
	Ping(ctx context.Context) error
}

// =============================================================================
// ConnPool & PoolManager — connection pool per router
// =============================================================================

// ConnPool mengelola pool koneksi TCP ke satu router MikroTik.
// Max 5 koneksi per pool, lazy connect, priority queue saat pool penuh.
type ConnPool interface {
	// Get mengambil koneksi idle atau membuat koneksi baru (lazy).
	// Memblokir jika pool penuh sampai koneksi tersedia.
	// Priority menentukan urutan dequeue saat pool penuh.
	Get(ctx context.Context, priority CommandPriority) (RouterOSAdapter, error)

	// Put mengembalikan koneksi ke pool setelah selesai digunakan.
	Put(conn RouterOSAdapter)

	// Close menutup semua koneksi di pool.
	Close() error

	// Stats mengembalikan statistik pool (active, idle, total).
	Stats() PoolStats

	// WarmUp membuka koneksi hingga max capacity secara paralel.
	// Dipanggil saat antrian perintah melebihi warm-up threshold.
	WarmUp(ctx context.Context) error
}

// PoolManager mengelola pool koneksi untuk semua router.
type PoolManager interface {
	// GetPool mengembalikan pool untuk router tertentu (buat baru jika belum ada).
	GetPool(routerID string, cfg ConnectionConfig) ConnPool

	// ClosePool menutup pool untuk router tertentu.
	ClosePool(routerID string)

	// CloseAll menutup semua pool.
	CloseAll()
}

// =============================================================================
// HealthChecker — health check periodik untuk semua router
// =============================================================================

// HealthChecker menjalankan health check periodik untuk semua router.
// Satu goroutine ticker per router, skip router dengan status maintenance.
type HealthChecker interface {
	// Start memulai health check goroutine untuk semua router aktif.
	Start(ctx context.Context) error

	// Stop menghentikan semua health check goroutine.
	Stop()

	// AddRouter menambahkan router baru ke health check schedule.
	AddRouter(router *Router)

	// RemoveRouter menghapus router dari health check schedule.
	RemoveRouter(routerID string)

	// UpdateInterval mengubah interval health check untuk router tertentu.
	UpdateInterval(routerID string, intervalSec int)
}

// =============================================================================
// RouterUsecase — business logic untuk manajemen router
// =============================================================================

// RouterUsecase mendefinisikan business logic untuk manajemen router.
type RouterUsecase interface {
	// Create membuat router baru, test koneksi, dan auto-detect info.
	Create(ctx context.Context, tenantID string, req CreateRouterRequest) (*RouterResponse, error)

	// GetByID mengambil detail router termasuk live metrics jika online.
	GetByID(ctx context.Context, id string) (*RouterDetailResponse, error)

	// Update memperbarui data router.
	Update(ctx context.Context, id string, req UpdateRouterRequest) (*RouterResponse, error)

	// Delete soft-delete router dan tutup pool koneksi.
	Delete(ctx context.Context, id string) error

	// List mengambil daftar router dengan paginasi.
	List(ctx context.Context, params RouterListParams) (*RouterListResult, error)

	// TestConnection menguji koneksi ke router dan mengembalikan system info.
	TestConnection(ctx context.Context, id string) (*SystemResource, error)

	// Reboot mengirim perintah reboot ke router (dengan konfirmasi nama).
	Reboot(ctx context.Context, id string, confirmName string) error

	// GetStatusSummary mengembalikan ringkasan status semua router tenant.
	GetStatusSummary(ctx context.Context) (*StatusSummary, error)
}

// =============================================================================
// PPPoEEventPublisher — publikasi event hasil operasi PPPoE ke Redis queue
// =============================================================================

// PPPoEEventPublisher mempublikasikan event hasil operasi PPPoE ke Redis queue.
type PPPoEEventPublisher interface {
	// PublishCommandResult mempublikasikan hasil eksekusi perintah ke router.
	PublishCommandResult(ctx context.Context, result CommandResultPayload) error

	// PublishSyncFailed mempublikasikan event sinkronisasi gagal untuk notifikasi.
	PublishSyncFailed(ctx context.Context, payload SyncFailedPayload) error
}

// =============================================================================
// PPPoEUserRepository — operasi data untuk tabel pppoe_users
// =============================================================================

// PPPoEUserRepository mendefinisikan operasi data untuk tabel pppoe_users.
// Diimplementasikan oleh repository.PPPoEUserRepo menggunakan sqlc.
type PPPoEUserRepository interface {
	// Create membuat record PPPoE user baru.
	Create(ctx context.Context, user *PPPoEUser) (*PPPoEUser, error)

	// GetByID mengambil PPPoE user berdasarkan ID.
	GetByID(ctx context.Context, id string) (*PPPoEUser, error)

	// GetByUsername mengambil PPPoE user berdasarkan router_id dan username.
	GetByUsername(ctx context.Context, routerID, username string) (*PPPoEUser, error)

	// GetByCustomerID mengambil PPPoE user berdasarkan customer_id.
	GetByCustomerID(ctx context.Context, customerID string) (*PPPoEUser, error)

	// Update memperbarui record PPPoE user.
	Update(ctx context.Context, user *PPPoEUser) (*PPPoEUser, error)

	// SoftDelete melakukan soft-delete PPPoE user.
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar PPPoE user dengan paginasi per router.
	List(ctx context.Context, params PPPoEUserListParams) (*PPPoEUserListResult, error)

	// GetByRouterID mengambil semua PPPoE user aktif untuk satu router.
	GetByRouterID(ctx context.Context, routerID string) ([]*PPPoEUser, error)

	// GetSyncStatusSummary mengambil ringkasan sync status per router.
	GetSyncStatusSummary(ctx context.Context, routerID string) (*SyncStatusSummary, error)

	// UpdateSyncStatus memperbarui sync_status dan last_sync_at.
	UpdateSyncStatus(ctx context.Context, id string, status SyncStatus, syncAt *time.Time) error

	// BulkUpdateSyncStatus memperbarui sync_status untuk banyak user sekaligus.
	BulkUpdateSyncStatus(ctx context.Context, ids []string, status SyncStatus, syncAt *time.Time) error
}

// =============================================================================
// DHCPBindingRepository — operasi data untuk tabel dhcp_bindings
// =============================================================================

type DHCPBindingRepository interface {
	Create(ctx context.Context, binding *DHCPBinding) (*DHCPBinding, error)
	GetByID(ctx context.Context, id string) (*DHCPBinding, error)
	GetByRouterAndMAC(ctx context.Context, routerID, mac string) (*DHCPBinding, error)
	GetByRouterAndIP(ctx context.Context, routerID, ip string) (*DHCPBinding, error)
	Update(ctx context.Context, binding *DHCPBinding) (*DHCPBinding, error)
	SoftDelete(ctx context.Context, id string) error
	List(ctx context.Context, params DHCPBindingListParams) (*DHCPBindingListResult, error)
	UpdateSyncState(ctx context.Context, id, routerLeaseID, syncStatus string, syncAt *time.Time) error
}

type MikroTikCommandAuditRepository interface {
	Create(ctx context.Context, log MikroTikCommandAuditLog) error
}

type StaticIPAssignmentRepository interface {
	Create(ctx context.Context, assignment *StaticIPAssignment) (*StaticIPAssignment, error)
	GetByID(ctx context.Context, id string) (*StaticIPAssignment, error)
	GetByRouterAndIP(ctx context.Context, routerID, ip string) (*StaticIPAssignment, error)
	Update(ctx context.Context, assignment *StaticIPAssignment) (*StaticIPAssignment, error)
	SoftDelete(ctx context.Context, id string) error
	List(ctx context.Context, params StaticIPAssignmentListParams) (*StaticIPAssignmentListResult, error)
	UpdateSyncState(ctx context.Context, id, syncStatus string, syncAt *time.Time) error
}

// =============================================================================
// PPPoEProfileRepository — operasi data untuk tabel pppoe_profiles
// =============================================================================

// PPPoEProfileRepository mendefinisikan operasi data untuk tabel pppoe_profiles.
// Diimplementasikan oleh repository.PPPoEProfileRepo menggunakan sqlc.
type PPPoEProfileRepository interface {
	// Create membuat record PPPoE profile baru.
	Create(ctx context.Context, profile *PPPoEProfile) (*PPPoEProfile, error)

	// GetByID mengambil PPPoE profile berdasarkan ID.
	GetByID(ctx context.Context, id string) (*PPPoEProfile, error)

	// GetByPackageID mengambil PPPoE profile berdasarkan package_id.
	GetByPackageID(ctx context.Context, packageID string) (*PPPoEProfile, error)

	// GetByProfileName mengambil PPPoE profile berdasarkan tenant_id dan profile_name.
	GetByProfileName(ctx context.Context, tenantID, profileName string) (*PPPoEProfile, error)

	// Update memperbarui record PPPoE profile.
	Update(ctx context.Context, profile *PPPoEProfile) (*PPPoEProfile, error)

	// ListByTenant mengambil semua profile untuk satu tenant.
	ListByTenant(ctx context.Context, tenantID string) ([]*PPPoEProfile, error)
}

// =============================================================================
// VPNTunnelRepository — operasi data untuk tabel vpn_tunnels
// =============================================================================

// VPNTunnelRepository mendefinisikan operasi data untuk tabel vpn_tunnels.
// Diimplementasikan oleh repository.VPNTunnelRepo menggunakan sqlc.
type VPNTunnelRepository interface {
	// Create membuat record VPN tunnel baru.
	Create(ctx context.Context, tunnel *VPNTunnel) (*VPNTunnel, error)

	// GetByID mengambil VPN tunnel berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*VPNTunnel, error)

	// Update memperbarui record VPN tunnel.
	Update(ctx context.Context, tunnel *VPNTunnel) (*VPNTunnel, error)

	// SoftDelete melakukan soft-delete VPN tunnel (set deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar VPN tunnel dengan paginasi dan filter.
	List(ctx context.Context, params VPNTunnelListParams) (*VPNTunnelListResult, error)

	// GetByStatus mengambil semua tunnel dengan status tertentu.
	GetByStatus(ctx context.Context, status TunnelStatus) ([]*VPNTunnel, error)

	// CountByStatus menghitung jumlah tunnel per status untuk tenant.
	CountByStatus(ctx context.Context) (map[TunnelStatus]int64, error)

	// TunnelNameExists mengecek apakah tunnel_name sudah ada di tenant.
	TunnelNameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)

	// VPNIPExists mengecek apakah vpn_ip sudah digunakan di tenant.
	VPNIPExists(ctx context.Context, tenantID, vpnIP string) (bool, error)

	// UpdateStatus memperbarui status tunnel dan field terkait health check.
	UpdateStatus(ctx context.Context, id string, params TunnelHealthUpdate) error

	// GetConnectedTunnels mengambil semua tunnel dengan status "connected" (cross-tenant untuk health monitor).
	GetConnectedTunnels(ctx context.Context) ([]*VPNTunnel, error)

	// GetDisconnectedTunnels mengambil semua tunnel dengan status "disconnected" (cross-tenant untuk recovery check).
	GetDisconnectedTunnels(ctx context.Context) ([]*VPNTunnel, error)
}

// =============================================================================
// VPNSubnetRepository — operasi data untuk tabel vpn_subnets
// =============================================================================

// VPNSubnetRepository mendefinisikan operasi data untuk tabel vpn_subnets.
// Diimplementasikan oleh repository.VPNSubnetRepo menggunakan sqlc.
type VPNSubnetRepository interface {
	// GetByTenantID mengambil subnet allocation untuk tenant.
	GetByTenantID(ctx context.Context, tenantID string) (*VPNSubnet, error)

	// Create membuat subnet allocation baru untuk tenant.
	Create(ctx context.Context, subnet *VPNSubnet) (*VPNSubnet, error)

	// GetNextTenantSeq mengambil tenant_seq berikutnya yang tersedia.
	GetNextTenantSeq(ctx context.Context) (int, error)

	// IncrementNextClientIPSeq menaikkan next_client_ip_seq dan mengembalikan nilai sebelumnya.
	IncrementNextClientIPSeq(ctx context.Context, tenantID string) (int, error)
}

// =============================================================================
// VPNBandwidthStore — penyimpanan bandwidth metrics per tunnel di Redis
// =============================================================================

// VPNBandwidthStore menyimpan dan mengambil bandwidth metrics per tunnel dari Redis.
// Menggunakan sorted set dengan score=unix timestamp dan 24-hour TTL.
type VPNBandwidthStore interface {
	// Store menyimpan satu data point bandwidth untuk tunnel.
	Store(ctx context.Context, tunnelID string, metrics VPNBandwidthMetrics) error

	// Query mengambil data point bandwidth dalam rentang waktu tertentu.
	Query(ctx context.Context, tunnelID string, from, to time.Time) ([]VPNBandwidthPoint, error)

	// GetLatest mengambil data point bandwidth terbaru untuk tunnel.
	GetLatest(ctx context.Context, tunnelID string) (*VPNBandwidthPoint, error)
}

// =============================================================================
// VPNEventPublisher — publikasi event VPN ke Redis queue via asynq
// =============================================================================

// VPNEventPublisher mempublikasikan event VPN ke Redis queue via asynq.
// Best-effort: log error jika publish gagal, jangan return error ke caller.
type VPNEventPublisher interface {
	// PublishTunnelDown mempublikasikan event tunnel disconnected.
	PublishTunnelDown(ctx context.Context, payload VPNTunnelDownPayload) error

	// PublishTunnelUp mempublikasikan event tunnel connected.
	PublishTunnelUp(ctx context.Context, payload VPNTunnelUpPayload) error

	// PublishTunnelCreated mempublikasikan event tunnel created.
	PublishTunnelCreated(ctx context.Context, payload VPNTunnelCreatedPayload) error

	// PublishServerBandwidthHigh mempublikasikan event bandwidth server melebihi 80% kapasitas.
	PublishServerBandwidthHigh(ctx context.Context, payload VPNServerBandwidthHighPayload) error

	// PublishServerBandwidthNormal mempublikasikan event bandwidth server kembali normal (< 70%).
	PublishServerBandwidthNormal(ctx context.Context, payload VPNServerBandwidthNormalPayload) error

	// PublishMaintenanceScheduled mempublikasikan event jadwal maintenance ke tenant terdampak.
	PublishMaintenanceScheduled(ctx context.Context, payload VPNMaintenanceScheduledPayload) error
}

// =============================================================================
// VPNKeyGenerator — generate key pair dan credential untuk VPN tunnel
// =============================================================================

// VPNKeyGenerator menghasilkan key pair dan credential untuk VPN tunnel.
// WireGuard: public/private key pair via curve25519.
// L2TP/PPTP/SSTP: random username/password/PSK.
type VPNKeyGenerator interface {
	// GenerateWireGuardKeyPair menghasilkan pasangan public key dan private key WireGuard.
	GenerateWireGuardKeyPair() (publicKey, privateKey string, err error)

	// GeneratePreSharedKey menghasilkan pre-shared key 256-bit untuk WireGuard.
	GeneratePreSharedKey() (string, error)

	// GenerateCredentials menghasilkan username dan password random untuk L2TP/PPTP/SSTP.
	GenerateCredentials(tunnelName string) (username, password string, err error)

	// GenerateIPSecPSK menghasilkan IPSec pre-shared key untuk L2TP/IPSec.
	GenerateIPSecPSK() (string, error)
}

// =============================================================================
// VPNCommandBuilder — membangun perintah RouterOS untuk konfigurasi VPN
// =============================================================================

// VPNCommandBuilder membangun perintah RouterOS untuk konfigurasi VPN.
// Extends CommandBuilder yang sudah ada dengan method VPN-specific.
type VPNCommandBuilder interface {
	// --- WireGuard Commands (RouterOS v7+ only) ---

	// CreateWireGuardInterface membangun perintah /interface/wireguard/add.
	CreateWireGuardInterface(params WireGuardInterfaceParams) (command string, args map[string]string)

	// AddWireGuardPeer membangun perintah /interface/wireguard/peers/add.
	AddWireGuardPeer(params WireGuardPeerParams) (command string, args map[string]string)

	// RemoveWireGuardInterface membangun perintah /interface/wireguard/remove.
	RemoveWireGuardInterface(name string) (command string, args map[string]string)

	// RemoveWireGuardPeer membangun perintah /interface/wireguard/peers/remove.
	RemoveWireGuardPeer(interfaceName string) (command string, args map[string]string)

	// --- L2TP Commands ---

	// CreateL2TPClient membangun perintah /interface/l2tp-client/add.
	CreateL2TPClient(params L2TPClientParams) (command string, args map[string]string)

	// RemoveL2TPClient membangun perintah /interface/l2tp-client/remove.
	RemoveL2TPClient(name string) (command string, args map[string]string)

	// CreateIPSecProfile membangun perintah /ip/ipsec/profile/add.
	CreateIPSecProfile(params IPSecProfileParams) (command string, args map[string]string)

	// CreateIPSecProposal membangun perintah /ip/ipsec/proposal/add.
	CreateIPSecProposal(params IPSecProposalParams) (command string, args map[string]string)

	// --- PPTP Commands ---

	// CreatePPTPClient membangun perintah /interface/pptp-client/add.
	CreatePPTPClient(params PPTPClientParams) (command string, args map[string]string)

	// RemovePPTPClient membangun perintah /interface/pptp-client/remove.
	RemovePPTPClient(name string) (command string, args map[string]string)

	// --- SSTP Commands ---

	// CreateSSTPClient membangun perintah /interface/sstp-client/add.
	CreateSSTPClient(params SSTPClientParams) (command string, args map[string]string)

	// RemoveSSTPClient membangun perintah /interface/sstp-client/remove.
	RemoveSSTPClient(name string) (command string, args map[string]string)

	// --- OpenVPN Commands ---

	// CreateOpenVPNClient membangun perintah /interface/ovpn-client/add.
	CreateOpenVPNClient(params OpenVPNClientParams) (command string, args map[string]string)

	// RemoveOpenVPNClient membangun perintah /interface/ovpn-client/remove.
	RemoveOpenVPNClient(name string) (command string, args map[string]string)

	// --- Common Commands ---

	// AddIPAddress membangun perintah /ip/address/add.
	AddIPAddress(params IPAddressParams) (command string, args map[string]string)

	// RemoveIPAddressByInterface membangun perintah /ip/address/remove by interface.
	RemoveIPAddressByInterface(interfaceName string) (command string, args map[string]string)

	// AddIPRoute membangun perintah /ip/route/add.
	AddIPRoute(params IPRouteParams) (command string, args map[string]string)

	// AddFirewallFilter membangun perintah /ip/firewall/filter/add.
	AddFirewallFilter(params FirewallFilterParams) (command string, args map[string]string)
}

// =============================================================================
// VPNScriptGenerator — generate RouterOS script (.rsc) per protokol VPN
// =============================================================================

// VPNScriptGenerator menghasilkan RouterOS script (.rsc) per protokol VPN.
// Script berisi perintah lengkap untuk setup VPN di router MikroTik.
type VPNScriptGenerator interface {
	// Generate menghasilkan script .rsc berdasarkan tunnel configuration.
	// Script TIDAK boleh mengandung server private key.
	Generate(tunnel *VPNTunnel, subnet *VPNSubnet) (string, error)
}

// =============================================================================
// VPNHealthMonitor — health check periodik untuk semua VPN tunnel
// =============================================================================

// VPNHealthMonitor menjalankan health check periodik untuk semua VPN tunnel.
// Satu goroutine dengan ticker 30 detik, memeriksa semua tunnel connected.
type VPNHealthMonitor interface {
	// Start memulai health monitor goroutine.
	Start(ctx context.Context) error

	// Stop menghentikan health monitor goroutine.
	Stop()
}

// =============================================================================
// VPNManager — business logic untuk manajemen VPN tunnel
// =============================================================================

// VPNManager mendefinisikan business logic untuk manajemen VPN tunnel.
// Menangani lifecycle lengkap: create, configure, test, monitor, delete.
type VPNManager interface {
	// CreateTunnel membuat VPN tunnel baru dengan auto-generate key/credential dan IP allocation.
	CreateTunnel(ctx context.Context, tenantID string, req CreateVPNTunnelRequest) (*VPNTunnelResponse, error)

	// GetTunnel mengambil detail tunnel termasuk semua field (private key di-mask).
	GetTunnel(ctx context.Context, id string) (*VPNTunnelDetailResponse, error)

	// UpdateTunnel memperbarui field yang diizinkan (tunnel_name, notes, router_id, persistent_keepalive, allowed_addresses).
	UpdateTunnel(ctx context.Context, id string, req UpdateVPNTunnelRequest) (*VPNTunnelResponse, error)

	// DeleteTunnel soft-delete tunnel, remove peer dari VPN server, dan opsional remove interface dari router.
	DeleteTunnel(ctx context.Context, id string) error

	// ListTunnels mengambil daftar tunnel dengan paginasi dan filter.
	ListTunnels(ctx context.Context, params VPNTunnelListParams) (*VPNTunnelListResult, error)

	// GetSummary mengambil ringkasan status tunnel untuk dashboard.
	GetSummary(ctx context.Context) (*VPNSummary, error)

	// TestConnection menguji koneksi VPN dengan ping ke client VPN IP.
	TestConnection(ctx context.Context, id string) (*VPNTestResult, error)

	// AutoConfigure mengkonfigurasi VPN di router yang sudah online via RouterOS API.
	AutoConfigure(ctx context.Context, id string) error

	// GenerateScript menghasilkan RouterOS script (.rsc) untuk setup manual.
	GenerateScript(ctx context.Context, id string) (string, error)

	// GetBandwidth mengambil statistik bandwidth untuk satu tunnel.
	GetBandwidth(ctx context.Context, id string, from, to time.Time) (*VPNBandwidthResult, error)

	// UpdateRouterHost mengupdate host router ke VPN IP setelah tunnel terverifikasi.
	UpdateRouterHost(ctx context.Context, tunnelID string) error
}

// =============================================================================
// OLTRepository — operasi data untuk tabel olts
// =============================================================================

// OLTRepository mendefinisikan operasi data untuk tabel olts.
// Diimplementasikan oleh repository.OLTRepo menggunakan sqlc.
type OLTRepository interface {
	// Create membuat OLT baru dan mengembalikan OLT yang dibuat.
	Create(ctx context.Context, olt *OLT) (*OLT, error)

	// GetByID mengambil OLT berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*OLT, error)

	// Update memperbarui data OLT dan mengembalikan OLT yang diperbarui.
	Update(ctx context.Context, olt *OLT) (*OLT, error)

	// SoftDelete melakukan soft-delete OLT (set deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar OLT dengan paginasi dan filter (tenant-scoped via RLS).
	List(ctx context.Context, params OLTListParams) (*OLTListResult, error)

	// CountByStatus menghitung jumlah OLT per status untuk tenant.
	CountByStatus(ctx context.Context) (map[OLTStatus]int64, error)

	// GetActiveOLTs mengambil semua OLT yang tidak di-delete dan bukan maintenance.
	GetActiveOLTs(ctx context.Context) ([]*OLT, error)

	// GetOnlineOLTs mengambil semua OLT dengan status online.
	GetOnlineOLTs(ctx context.Context) ([]*OLT, error)

	// NameExists mengecek apakah nama OLT sudah ada di tenant.
	NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)

	// UpdateHealthCheck memperbarui field health check (last_checked_at, failure_count, status).
	UpdateHealthCheck(ctx context.Context, id string, params OLTHealthCheckUpdate) error

	// UpdateONTCounts memperbarui total_ont_count setelah sync.
	UpdateONTCounts(ctx context.Context, id string, totalONT int) error
}

// =============================================================================
// ODPRepository — operasi data untuk tabel odps
// =============================================================================

// ODPRepository mendefinisikan operasi data untuk tabel odps.
// Diimplementasikan oleh repository.ODPRepo menggunakan sqlc.
type ODPRepository interface {
	// Create membuat ODP baru.
	Create(ctx context.Context, odp *ODP) (*ODP, error)

	// GetByID mengambil ODP berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*ODP, error)

	// Update memperbarui data ODP.
	Update(ctx context.Context, odp *ODP) (*ODP, error)

	// SoftDelete melakukan soft-delete ODP.
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar ODP dengan paginasi dan filter.
	List(ctx context.Context, params ODPListParams) (*ODPListResult, error)

	// NameExists mengecek apakah nama ODP sudah ada di tenant.
	NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)

	// GetByOLTAndPort mengambil semua ODP untuk satu OLT dan PON port.
	GetByOLTAndPort(ctx context.Context, oltID string, ponPort int) ([]*ODP, error)
}

// =============================================================================
// AlarmRepository — operasi data untuk tabel olt_alarms
// =============================================================================

// AlarmRepository mendefinisikan operasi data untuk tabel olt_alarms.
// Diimplementasikan oleh repository.AlarmRepo menggunakan sqlc.
type AlarmRepository interface {
	// Create menyimpan alarm baru.
	Create(ctx context.Context, alarm *OLTAlarmRecord) (*OLTAlarmRecord, error)

	// List mengambil daftar alarm dengan paginasi dan filter.
	List(ctx context.Context, oltID string, params AlarmListParams) (*AlarmListResult, error)

	// CountActive menghitung jumlah alarm aktif per OLT.
	CountActive(ctx context.Context, oltID string) (int64, error)

	// CountActiveByTenant menghitung total alarm aktif untuk tenant.
	CountActiveByTenant(ctx context.Context) (int64, error)

	// ClearAlarm mengubah status alarm menjadi cleared.
	ClearAlarm(ctx context.Context, id string) error

	// PurgeOlderThan menghapus alarm lebih tua dari durasi tertentu.
	PurgeOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// =============================================================================
// SignalStore — penyimpanan signal data ONT di Redis time-series
// =============================================================================

// SignalStore menyimpan dan mengambil signal data ONT dari Redis time-series.
// Menggunakan sorted set dengan score=unix timestamp dan 30-day TTL.
type SignalStore interface {
	// Store menyimpan satu data point signal untuk ONT.
	Store(ctx context.Context, oltID string, portIndex int, ontIndex int, signal ONTSignalPoint) error

	// Query mengambil data point signal dalam rentang waktu tertentu.
	Query(ctx context.Context, oltID string, portIndex int, ontIndex int, from, to time.Time) ([]ONTSignalPoint, error)

	// GetLatest mengambil data point signal terbaru untuk ONT.
	GetLatest(ctx context.Context, oltID string, portIndex int, ontIndex int) (*ONTSignalPoint, error)
}

// =============================================================================
// TrafficStore — penyimpanan traffic data PON port di Redis time-series
// =============================================================================

// TrafficStore menyimpan dan mengambil traffic data PON port dari Redis time-series.
// Menggunakan sorted set dengan score=unix timestamp dan 7-day TTL.
type TrafficStore interface {
	// Store menyimpan satu data point traffic untuk PON port.
	Store(ctx context.Context, oltID string, portIndex int, traffic PONTrafficPoint) error

	// Query mengambil data point traffic dalam rentang waktu tertentu.
	Query(ctx context.Context, oltID string, portIndex int, from, to time.Time) ([]PONTrafficPoint, error)

	// GetLatest mengambil data point traffic terbaru untuk PON port.
	GetLatest(ctx context.Context, oltID string, portIndex int) (*PONTrafficPoint, error)
}

// =============================================================================
// OLTEventPublisher — publikasi event OLT ke Redis queue via asynq
// =============================================================================

// OLTEventPublisher mempublikasikan event OLT ke Redis queue via asynq.
// Best-effort: log error jika publish gagal, jangan return error ke caller.
// Pattern sama dengan EventPublisher dan VPNEventPublisher yang sudah ada.
type OLTEventPublisher interface {
	// PublishDeviceOffline mempublikasikan event OLT offline.
	PublishDeviceOffline(ctx context.Context, payload OLTDeviceOfflinePayload) error

	// PublishDeviceOnline mempublikasikan event OLT online.
	PublishDeviceOnline(ctx context.Context, payload OLTDeviceOnlinePayload) error

	// PublishAlarm mempublikasikan event alarm OLT.
	PublishAlarm(ctx context.Context, payload OLTAlarmPayload) error

	// --- Provisioning event methods ---

	// PublishONTProvisioned mempublikasikan event ONT berhasil di-provision.
	PublishONTProvisioned(ctx context.Context, payload ONTProvisionedPayload) error

	// PublishONTDecommissioned mempublikasikan event ONT berhasil di-decommission.
	PublishONTDecommissioned(ctx context.Context, payload ONTDecommissionedPayload) error

	// PublishONTAutoProvisioned mempublikasikan event ONT berhasil di-auto-provision.
	PublishONTAutoProvisioned(ctx context.Context, payload ONTAutoProvisionedPayload) error

	// PublishONTAutoProvisionFailed mempublikasikan event auto-provisioning gagal.
	PublishONTAutoProvisionFailed(ctx context.Context, payload ONTAutoProvisionFailedPayload) error

	// PublishONTPortMigrated mempublikasikan event port migration terdeteksi.
	PublishONTPortMigrated(ctx context.Context, payload ONTPortMigratedPayload) error
}

// =============================================================================
// OLTAdapter — interface komunikasi dengan OLT device (multi-brand)
// =============================================================================

// OLTAdapter mendefinisikan interface untuk komunikasi dengan OLT device.
// Diimplementasikan per brand: ZTEAdapter, HuaweiAdapter, FiberHomeAdapter, VSOLAdapter, HSGQAdapter, MockOLTAdapter.
// Setiap adapter mengabstraksi perbedaan SNMP OID dan CLI command antar brand.
type OLTAdapter interface {
	// GetSystemInfo mengambil informasi sistem OLT (brand, model, firmware, uptime, pon_ports, total_ont).
	GetSystemInfo(ctx context.Context) (*OLTSystemInfo, error)

	// GetPONPortStatus mengambil status satu PON port (admin/oper status, ONT count, description).
	GetPONPortStatus(ctx context.Context, portIndex int) (*PONPortStatus, error)

	// GetAllPONPorts mengambil status semua PON port pada OLT.
	GetAllPONPorts(ctx context.Context) ([]PONPortStatus, error)

	// GetONTList mengambil daftar ONT yang terdaftar pada satu PON port.
	GetONTList(ctx context.Context, portIndex int) ([]ONTPortStatus, error)

	// GetONTSignal mengambil informasi signal ONT (rx_power, distance, uptime).
	GetONTSignal(ctx context.Context, portIndex int, ontIndex int) (*ONTSignalInfo, error)

	// GetAlarms mengambil daftar alarm aktif dari OLT.
	GetAlarms(ctx context.Context) ([]OLTAlarm, error)

	// GetSFPInfo mengambil informasi SFP module pada satu PON port.
	GetSFPInfo(ctx context.Context, portIndex int) (*SFPInfo, error)

	// GetTrafficStats mengambil statistik traffic pada satu PON port.
	GetTrafficStats(ctx context.Context, portIndex int) (*PONTrafficStats, error)

	// Ping memeriksa konektivitas OLT via SNMP GET sysUpTime.
	Ping(ctx context.Context) error

	// --- Provisioning methods ---

	// AddONT menambahkan ONT ke PON port dengan line profile dan service profile.
	// Menghasilkan CLI command per brand: ZTE `onu add sn`, Huawei `ont add sn-auth`, dll.
	AddONT(ctx context.Context, params AddONTParams) (*ProvisioningResult, error)

	// RemoveONT menghapus ONT dari PON port.
	// Menghasilkan CLI command per brand: ZTE `onu delete`, Huawei `ont delete`, dll.
	RemoveONT(ctx context.Context, params RemoveONTParams) (*ProvisioningResult, error)

	// AddServicePort menambahkan service-port dengan VLAN assignment.
	// Menghasilkan CLI command per brand: ZTE `service-port add vlan`, dll.
	AddServicePort(ctx context.Context, params AddServicePortParams) (*ProvisioningResult, error)

	// RemoveServicePort menghapus service-port.
	// Menghasilkan CLI command per brand: ZTE `service-port delete`, dll.
	RemoveServicePort(ctx context.Context, params RemoveServicePortParams) (*ProvisioningResult, error)

	// RebootONT mengirim perintah reboot ke ONT tertentu.
	// Menghasilkan CLI command per brand: ZTE `onu reset`, Huawei `ont reset`, dll.
	RebootONT(ctx context.Context, params RebootONTParams) (*ProvisioningResult, error)

	// GetUnregisteredONTs mengambil daftar ONT yang terdeteksi tapi belum terdaftar.
	GetUnregisteredONTs(ctx context.Context) ([]UnregisteredONT, error)
}

// =============================================================================
// OLTAdapterFactory — factory untuk membuat OLTAdapter berdasarkan brand
// =============================================================================

// OLTAdapterFactory membuat instance OLTAdapter berdasarkan brand dan konfigurasi koneksi.
// Jika NETWORK_MODE=mock, selalu mengembalikan MockOLTAdapter.
type OLTAdapterFactory interface {
	// CreateAdapter membuat adapter sesuai brand OLT dengan konfigurasi SNMP dan CLI.
	CreateAdapter(brand OLTBrand, snmpCfg SNMPConfig, cliCfg CLIConfig) (OLTAdapter, error)
}

// =============================================================================
// SNMPConnector — koneksi SNMP ke OLT untuk monitoring
// =============================================================================

// SNMPConnector mengelola koneksi SNMP ke OLT untuk monitoring.
// Menggunakan library gosnmp untuk operasi GET, WALK, GETBULK.
type SNMPConnector interface {
	// Get melakukan SNMP GET untuk satu atau lebih OID.
	Get(ctx context.Context, cfg SNMPConfig, oids []string) ([]SNMPResult, error)

	// Walk melakukan SNMP WALK pada subtree OID.
	Walk(ctx context.Context, cfg SNMPConfig, rootOID string) ([]SNMPResult, error)

	// GetBulk melakukan SNMP GETBULK untuk efisiensi pada tabel besar.
	GetBulk(ctx context.Context, cfg SNMPConfig, oids []string, maxRepetitions int) ([]SNMPResult, error)

	// Ping melakukan SNMP GET sysUpTime untuk cek konektivitas.
	Ping(ctx context.Context, cfg SNMPConfig) error
}

// =============================================================================
// CLIConnector — koneksi CLI (SSH/Telnet) ke OLT untuk provisioning
// =============================================================================

// CLIConnector mengelola koneksi CLI ke OLT untuk provisioning command.
// Connect-on-demand: buka session → kirim command → terima response → tutup session.
// TIDAK menggunakan connection pool (berbeda dari MikroTik RouterOS API).
type CLIConnector interface {
	// Execute membuka session, mengirim command, dan mengembalikan output.
	// Session ditutup setelah command selesai.
	Execute(ctx context.Context, cfg CLIConfig, command string) (string, error)

	// ExecuteMultiple mengirim beberapa command dalam satu session.
	// Berguna untuk provisioning yang butuh beberapa langkah.
	ExecuteMultiple(ctx context.Context, cfg CLIConfig, commands []string) ([]string, error)

	// TestConnection menguji koneksi CLI dan mengembalikan banner/prompt.
	TestConnection(ctx context.Context, cfg CLIConfig) (string, error)
}

// =============================================================================
// OLTManager — business logic untuk manajemen OLT device
// =============================================================================

// OLTManager mendefinisikan business logic untuk manajemen OLT device.
// Menangani CRUD, registrasi dengan auto-detect, test connection, dan status summary.
type OLTManager interface {
	// Create membuat OLT baru, test SNMP, auto-detect brand/model/firmware.
	Create(ctx context.Context, tenantID string, req CreateOLTRequest) (*OLTResponse, error)

	// GetByID mengambil detail OLT termasuk PON port summary dan alarm count.
	GetByID(ctx context.Context, id string) (*OLTDetailResponse, error)

	// Update memperbarui data OLT (name, host, credentials, interval, notes, status).
	Update(ctx context.Context, id string, req UpdateOLTRequest) (*OLTResponse, error)

	// Delete soft-delete OLT dan stop health check monitoring.
	Delete(ctx context.Context, id string) error

	// List mengambil daftar OLT dengan paginasi dan filter (status, brand, search).
	List(ctx context.Context, params OLTListParams) (*OLTListResult, error)

	// TestSNMP menguji koneksi SNMP dan mengembalikan auto-detected system info.
	TestSNMP(ctx context.Context, id string) (*OLTSystemInfo, error)

	// TestCLI menguji koneksi CLI (SSH/Telnet) dan mengembalikan hasil.
	TestCLI(ctx context.Context, id string) (*CLITestResult, error)

	// GetStatusSummary mengembalikan ringkasan status semua OLT tenant.
	GetStatusSummary(ctx context.Context) (*OLTStatusSummary, error)

	// GetPONPorts mengambil status semua PON port untuk satu OLT.
	GetPONPorts(ctx context.Context, oltID string) ([]PONPortStatus, error)

	// GetONTList mengambil daftar ONT pada satu PON port.
	GetONTList(ctx context.Context, oltID string, portIndex int) ([]ONTPortStatus, error)

	// GetSFPStatus mengambil status SFP module semua PON port.
	GetSFPStatus(ctx context.Context, oltID string) ([]SFPInfo, error)

	// GetCapacity mengambil data capacity planning untuk satu OLT.
	GetCapacity(ctx context.Context, oltID string) (*OLTCapacity, error)
}

// =============================================================================
// ODPManager — business logic untuk manajemen ODP/splitter
// =============================================================================

// ODPManager mendefinisikan business logic untuk manajemen ODP/splitter.
type ODPManager interface {
	// Create membuat ODP baru dengan auto-set capacity berdasarkan splitter_type.
	Create(ctx context.Context, tenantID string, req CreateODPRequest) (*ODPResponse, error)

	// GetByID mengambil detail ODP termasuk used_ports dan linked ONT list.
	GetByID(ctx context.Context, id string) (*ODPDetailResponse, error)

	// Update memperbarui data ODP.
	Update(ctx context.Context, id string, req UpdateODPRequest) (*ODPResponse, error)

	// Delete soft-delete ODP.
	Delete(ctx context.Context, id string) error

	// List mengambil daftar ODP dengan paginasi dan filter (olt_id, pon_port).
	List(ctx context.Context, params ODPListParams) (*ODPListResult, error)
}

// =============================================================================
// OLTHealthChecker — health check periodik untuk semua OLT aktif
// =============================================================================

// OLTHealthChecker menjalankan health check periodik untuk semua OLT aktif.
// Satu goroutine ticker per OLT, skip OLT dengan status maintenance.
// Pattern sama dengan MikroTik HealthChecker yang sudah ada.
type OLTHealthChecker interface {
	// Start memulai health check goroutine untuk semua OLT aktif.
	Start(ctx context.Context) error

	// Stop menghentikan semua health check goroutine.
	Stop()

	// AddOLT menambahkan OLT baru ke health check schedule.
	AddOLT(olt *OLT)

	// RemoveOLT menghapus OLT dari health check schedule.
	RemoveOLT(oltID string)

	// UpdateInterval mengubah interval health check untuk OLT tertentu.
	UpdateInterval(oltID string, intervalSec int)
}

// =============================================================================
// AlarmManager — manajemen alarm OLT (trap receiver + polling)
// =============================================================================

// AlarmManager mengelola alarm dari OLT (trap receiver + polling).
// Menyimpan alarm ke database, publish event, dan manage alarm lifecycle.
type AlarmManager interface {
	// StartTrapReceiver memulai SNMP trap receiver pada port 162.
	StartTrapReceiver(ctx context.Context) error

	// StopTrapReceiver menghentikan SNMP trap receiver.
	StopTrapReceiver()

	// PollAlarms mengambil alarm dari OLT via SNMP polling (fallback).
	PollAlarms(ctx context.Context, oltID string) ([]OLTAlarm, error)

	// GetAlarms mengambil daftar alarm dengan filter (severity, status, olt_id).
	GetAlarms(ctx context.Context, oltID string, params AlarmListParams) (*AlarmListResult, error)

	// PurgeOldAlarms menghapus alarm lebih tua dari 90 hari.
	PurgeOldAlarms(ctx context.Context) (int64, error)
}

// =============================================================================
// SyncEngine — periodic sync antara OLT dan database
// =============================================================================

// SyncEngine menjalankan periodic sync antara OLT dan database.
// OLT = source of truth untuk data fisik (SN, port, signal, status).
type SyncEngine interface {
	// Start memulai periodic sync goroutine (interval 30 menit).
	Start(ctx context.Context) error

	// Stop menghentikan sync goroutine.
	Stop()

	// SyncOLT menjalankan sync untuk satu OLT secara manual.
	SyncOLT(ctx context.Context, oltID string) (*OLTSyncResult, error)
}

// OLTSyncResult berisi hasil sinkronisasi satu OLT.
type OLTSyncResult struct {
	OLTID          string    `json:"olt_id"`
	TotalONT       int       `json:"total_ont"`
	UnmanagedCount int       `json:"unmanaged_count"` // ada di OLT tapi tidak di DB
	MissingCount   int       `json:"missing_count"`   // ada di DB tapi tidak di OLT
	UpdatedCount   int       `json:"updated_count"`   // data berbeda, DB diupdate
	SyncedAt       time.Time `json:"synced_at"`
}
