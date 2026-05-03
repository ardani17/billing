package domain

import "time"

// =============================================================================
// Request DTOs — payload untuk VPN tunnel operations
// =============================================================================

// CreateVPNTunnelRequest adalah payload untuk POST /api/v1/mikrotik/vpn/tunnels.
// Digunakan untuk membuat VPN tunnel baru via setup wizard.
type CreateVPNTunnelRequest struct {
	TunnelName string `json:"tunnel_name" validate:"required,min=1,max=100"`
	Protocol   string `json:"protocol" validate:"required,oneof=wireguard l2tp_ipsec pptp sstp openvpn"`
	RouterID   string `json:"router_id,omitempty" validate:"omitempty,uuid"`
	Notes      string `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// UpdateVPNTunnelRequest adalah payload untuk PUT /api/v1/mikrotik/vpn/tunnels/:id.
// Field bersifat opsional — hanya field yang dikirim yang akan diupdate.
type UpdateVPNTunnelRequest struct {
	TunnelName          string `json:"tunnel_name,omitempty" validate:"omitempty,min=1,max=100"`
	Notes               string `json:"notes,omitempty" validate:"omitempty,max=500"`
	RouterID            string `json:"router_id,omitempty" validate:"omitempty,uuid"`
	PersistentKeepalive *int   `json:"persistent_keepalive,omitempty" validate:"omitempty,min=0,max=300"`
	AllowedAddresses    string `json:"allowed_addresses,omitempty" validate:"omitempty,max=500"`
}

// VPNTunnelListParams berisi parameter untuk list VPN tunnel dengan paginasi.
type VPNTunnelListParams struct {
	TenantID string
	Page     int
	PageSize int
	Status   string // filter berdasarkan status (opsional)
	Protocol string // filter berdasarkan protokol (opsional)
	Search   string // pencarian berdasarkan tunnel_name
}

// =============================================================================
// Response DTOs — format respons untuk VPN tunnel operations
// =============================================================================

// VPNTunnelResponse adalah respons untuk operasi create/update/list tunnel.
type VPNTunnelResponse struct {
	ID                  string       `json:"id"`
	TunnelName          string       `json:"tunnel_name"`
	RouterID            *string      `json:"router_id,omitempty"`
	RouterName          string       `json:"router_name,omitempty"`
	Protocol            VPNProtocol  `json:"protocol"`
	VPNIP               string       `json:"vpn_ip"`
	ServerEndpoint      string       `json:"server_endpoint"`
	ServerPublicKey     string       `json:"server_public_key,omitempty"`
	ClientPublicKey     string       `json:"client_public_key,omitempty"`
	Status              TunnelStatus `json:"status"`
	ListenPort          int          `json:"listen_port"`
	AllowedAddresses    string       `json:"allowed_addresses"`
	PersistentKeepalive int          `json:"persistent_keepalive"`
	LatencyMs           *int         `json:"latency_ms,omitempty"`
	BandwidthCapMbps    *int         `json:"bandwidth_cap_mbps,omitempty"`
	LastHandshakeAt     *time.Time   `json:"last_handshake_at,omitempty"`
	Notes               string       `json:"notes,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// VPNTunnelDetailResponse adalah respons untuk GET tunnel detail.
// Private key dan PSK di-mask, tidak pernah di-expose ke client.
type VPNTunnelDetailResponse struct {
	VPNTunnelResponse
	ClientPrivateKeyMasked string `json:"client_private_key"`        // selalu "********"
	PreSharedKeyMasked     string `json:"pre_shared_key"`            // selalu "********" atau kosong
	L2TPUsername           string `json:"l2tp_username,omitempty"`
	L2TPPasswordMasked     string `json:"l2tp_password"`             // selalu "********" atau kosong
	ActiveEndpoint         string `json:"active_endpoint,omitempty"`
}

// VPNTunnelListResult berisi hasil list VPN tunnel dengan metadata paginasi.
type VPNTunnelListResult struct {
	Data       []*VPNTunnelResponse `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

// VPNSummary berisi ringkasan status tunnel untuk dashboard.
type VPNSummary struct {
	TotalTunnels      int64 `json:"total_tunnels"`
	ConnectedCount    int64 `json:"connected_count"`
	DisconnectedCount int64 `json:"disconnected_count"`
	PendingCount      int64 `json:"pending_count"`
	ErrorCount        int64 `json:"error_count"`
}

// VPNTestResult berisi hasil test koneksi VPN.
type VPNTestResult struct {
	Status          TunnelStatus `json:"status"`
	LatencyMs       int          `json:"latency_ms"`
	LastHandshakeAt *time.Time   `json:"last_handshake_at,omitempty"`
	ErrorMessage    string       `json:"error_message,omitempty"`
	Diagnostic      string       `json:"diagnostic,omitempty"` // unreachable, handshake_timeout, auth_failure
}

// VPNBandwidthResult berisi statistik bandwidth untuk satu tunnel.
type VPNBandwidthResult struct {
	Current *VPNBandwidthPoint  `json:"current,omitempty"`
	History []VPNBandwidthPoint `json:"history"`
}

// =============================================================================
// VPN Event Payloads — dipublikasikan ke Redis queue via asynq
// =============================================================================

// VPNTunnelDownPayload adalah payload event mikrotik.vpn_tunnel_down.
// Dipublikasikan saat tunnel berubah dari connected ke disconnected.
type VPNTunnelDownPayload struct {
	CorrelationID   string     `json:"correlation_id"`
	TunnelID        string     `json:"tunnel_id"`
	TunnelName      string     `json:"tunnel_name"`
	TenantID        string     `json:"tenant_id"`
	RouterID        *string    `json:"router_id,omitempty"`
	Protocol        string     `json:"protocol"`
	VPNIP           string     `json:"vpn_ip"`
	LastHandshakeAt *time.Time `json:"last_handshake_at,omitempty"`
	DisconnectedAt  time.Time  `json:"disconnected_at"`
}

// VPNTunnelUpPayload adalah payload event mikrotik.vpn_tunnel_up.
// Dipublikasikan saat tunnel berubah dari disconnected ke connected.
type VPNTunnelUpPayload struct {
	CorrelationID string    `json:"correlation_id"`
	TunnelID      string    `json:"tunnel_id"`
	TunnelName    string    `json:"tunnel_name"`
	TenantID      string    `json:"tenant_id"`
	RouterID      *string   `json:"router_id,omitempty"`
	Protocol      string    `json:"protocol"`
	VPNIP         string    `json:"vpn_ip"`
	LatencyMs     int       `json:"latency_ms"`
	ConnectedAt   time.Time `json:"connected_at"`
}

// VPNTunnelCreatedPayload adalah payload event mikrotik.vpn_tunnel_created.
// Dipublikasikan saat tunnel baru dibuat (sukses atau gagal).
type VPNTunnelCreatedPayload struct {
	CorrelationID string `json:"correlation_id"`
	TunnelID      string `json:"tunnel_id"`
	TunnelName    string `json:"tunnel_name"`
	TenantID      string `json:"tenant_id"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// VPNServerBandwidthHighPayload adalah payload event mikrotik.vpn_server_bandwidth_high.
// Dipublikasikan saat total bandwidth VPN server melebihi 80% kapasitas.
type VPNServerBandwidthHighPayload struct {
	ServerEndpoint     string    `json:"server_endpoint"`
	CurrentUsageMbps   int64     `json:"current_usage_mbps"`
	CapacityMbps       int64     `json:"capacity_mbps"`
	UtilizationPercent int       `json:"utilization_percent"`
	Timestamp          time.Time `json:"timestamp"`
}

// VPNServerBandwidthNormalPayload adalah payload event mikrotik.vpn_server_bandwidth_normal.
// Dipublikasikan saat total bandwidth VPN server kembali di bawah 70%.
type VPNServerBandwidthNormalPayload struct {
	ServerEndpoint     string    `json:"server_endpoint"`
	CurrentUsageMbps   int64     `json:"current_usage_mbps"`
	CapacityMbps       int64     `json:"capacity_mbps"`
	UtilizationPercent int       `json:"utilization_percent"`
	Timestamp          time.Time `json:"timestamp"`
}

// VPNMaintenanceScheduledPayload adalah payload event mikrotik.vpn_maintenance_scheduled.
// Dipublikasikan ke setiap tenant yang memiliki tunnel aktif di server yang akan maintenance.
type VPNMaintenanceScheduledPayload struct {
	TenantID            string    `json:"tenant_id"`
	ServerEndpoint      string    `json:"server_endpoint"`
	ScheduledStart      time.Time `json:"scheduled_start"`
	ScheduledEnd        time.Time `json:"scheduled_end"`
	Description         string    `json:"description"`
	AffectedTunnelCount int       `json:"affected_tunnel_count"`
}
