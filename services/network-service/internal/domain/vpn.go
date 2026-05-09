package domain

import (
	"fmt"
	"time"
)

// --- VPN Protocol ---

// VPNProtocol mendefinisikan protokol VPN yang didukung.
type VPNProtocol string

const (
	// ProtocolWireGuard untuk protokol WireGuard (RouterOS v7+ only).
	ProtocolWireGuard VPNProtocol = "wireguard"

	// ProtocolL2TPIPSec untuk protokol L2TP/IPSec (semua versi RouterOS).
	ProtocolL2TPIPSec VPNProtocol = "l2tp_ipsec"

	// ProtocolPPTP untuk protokol PPTP (legacy, kurang aman).
	ProtocolPPTP VPNProtocol = "pptp"

	// ProtocolSSTP untuk protokol SSTP (melewati firewall/NAT ketat).
	ProtocolSSTP VPNProtocol = "sstp"

	// ProtocolOpenVPN untuk protokol OpenVPN (alternatif jika WireGuard tidak tersedia).
	ProtocolOpenVPN VPNProtocol = "openvpn"
)

// ValidVPNProtocols berisi daftar protokol VPN yang valid.
var ValidVPNProtocols = []VPNProtocol{
	ProtocolWireGuard, ProtocolL2TPIPSec, ProtocolPPTP, ProtocolSSTP, ProtocolOpenVPN,
}

// IsValidVPNProtocol memeriksa apakah string adalah protokol VPN yang valid.
func IsValidVPNProtocol(s string) bool {
	for _, p := range ValidVPNProtocols {
		if string(p) == s {
			return true
		}
	}
	return false
}

// --- Tunnel Status ---

// TunnelStatus mendefinisikan status koneksi VPN tunnel.
type TunnelStatus string

const (
	// TunnelStatusConnected menandakan tunnel aktif dan terkoneksi.
	TunnelStatusConnected TunnelStatus = "connected"

	// TunnelStatusDisconnected menandakan tunnel terputus.
	TunnelStatusDisconnected TunnelStatus = "disconnected"

	// TunnelStatusPending menandakan tunnel menunggu verifikasi koneksi.
	TunnelStatusPending TunnelStatus = "pending"

	// TunnelStatusError menandakan tunnel mengalami error.
	TunnelStatusError TunnelStatus = "error"
)

// ValidTunnelStatuses berisi daftar status tunnel yang valid.
var ValidTunnelStatuses = []TunnelStatus{
	TunnelStatusConnected, TunnelStatusDisconnected,
	TunnelStatusPending, TunnelStatusError,
}

// ValidTunnelTransitions mendefinisikan transisi status tunnel yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidTunnelTransitions = map[TunnelStatus][]TunnelStatus{
	TunnelStatusPending:      {TunnelStatusConnected, TunnelStatusDisconnected, TunnelStatusError},
	TunnelStatusConnected:    {TunnelStatusDisconnected, TunnelStatusError},
	TunnelStatusDisconnected: {TunnelStatusConnected, TunnelStatusError},
	TunnelStatusError:        {TunnelStatusPending, TunnelStatusConnected},
}

// CanTransitionTunnel memeriksa apakah transisi status tunnel valid.
func CanTransitionTunnel(current, target TunnelStatus) bool {
	targets, ok := ValidTunnelTransitions[current]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// --- VPN Tunnel Entitas ---

// VPNTunnel merepresentasikan koneksi VPN antara perangkat tenant dan VPN server ISPBoss.
type VPNTunnel struct {
	ID                        string       `json:"id"`
	TenantID                  string       `json:"tenant_id"`
	RouterID                  *string      `json:"router_id,omitempty"`
	TunnelName                string       `json:"tunnel_name"`
	Protocol                  VPNProtocol  `json:"protocol"`
	VPNIP                     string       `json:"vpn_ip"`
	ServerEndpoint            string       `json:"server_endpoint"`
	ServerPublicKey           string       `json:"server_public_key,omitempty"`
	ClientPublicKey           string       `json:"client_public_key,omitempty"`
	ClientPrivateKeyEncrypted string       `json:"-"`
	PreSharedKeyEncrypted     string       `json:"-"`
	L2TPUsername              string       `json:"l2tp_username,omitempty"`
	L2TPPasswordEncrypted     string       `json:"-"`
	Status                    TunnelStatus `json:"status"`
	ListenPort                int          `json:"listen_port"`
	AllowedAddresses          string       `json:"allowed_addresses"`
	PersistentKeepalive       int          `json:"persistent_keepalive"`
	LastHandshakeAt           *time.Time   `json:"last_handshake_at,omitempty"`
	LatencyMs                 *int         `json:"latency_ms,omitempty"`
	BandwidthCapMbps          *int         `json:"bandwidth_cap_mbps,omitempty"`
	RateLimitPps              int          `json:"rate_limit_pps"`
	ActiveEndpoint            string       `json:"active_endpoint,omitempty"`
	Notes                     string       `json:"notes,omitempty"`
	CreatedAt                 time.Time    `json:"created_at"`
	UpdatedAt                 time.Time    `json:"updated_at"`
	DeletedAt                 *time.Time   `json:"deleted_at,omitempty"`
}

// --- VPN Subnet Entitas ---

// VPNSubnet merepresentasikan alokasi subnet VPN per tenant.
// Setiap tenant mendapat 1 subnet /24: 10.99.{tenant_seq}.0/24.
type VPNSubnet struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	SubnetPrefix    string    `json:"subnet_prefix"`
	TenantSeq       int       `json:"tenant_seq"`
	ServerIP        string    `json:"server_ip"`
	NextClientIPSeq int       `json:"next_client_ip_seq"`
	CreatedAt       time.Time `json:"created_at"`
}

// --- IP Address Fungsi bantus ---

// BuildClientIP menghasilkan IP address client dari tenant sequence dan client sequence.
// Format: 10.99.{tenant_seq}.{client_seq}
func BuildClientIP(tenantSeq, clientSeq int) string {
	return fmt.Sprintf("10.99.%d.%d", tenantSeq, clientSeq)
}

// BuildServerIP menghasilkan IP address server dari tenant sequence.
// Format: 10.99.{tenant_seq}.1
func BuildServerIP(tenantSeq int) string {
	return fmt.Sprintf("10.99.%d.1", tenantSeq)
}

// BuildSubnetPrefix menghasilkan subnet prefix dari tenant sequence.
// Format: 10.99.{tenant_seq}.0/24
func BuildSubnetPrefix(tenantSeq int) string {
	return fmt.Sprintf("10.99.%d.0/24", tenantSeq)
}

// IsValidClientSeq memeriksa apakah client sequence number valid (2-254).
func IsValidClientSeq(seq int) bool {
	return seq >= 2 && seq <= 254
}

// MaxClientsPerSubnet adalah jumlah maksimum client per subnet /24.
const MaxClientsPerSubnet = 253 // 2-254

// --- VPN Bandwidth Metrics ---

// VPNBandwidthMetrics berisi metrik bandwidth per tunnel.
type VPNBandwidthMetrics struct {
	TXBytes   int64 `json:"tx_bytes"`
	RXBytes   int64 `json:"rx_bytes"`
	TXRateBps int64 `json:"tx_rate_bps"`
	RXRateBps int64 `json:"rx_rate_bps"`
}

// VPNBandwidthPoint berisi metrik bandwidth dengan timestamp.
type VPNBandwidthPoint struct {
	Timestamp time.Time           `json:"timestamp"`
	Metrics   VPNBandwidthMetrics `json:"metrics"`
}

// --- Tunnel Health Perbarui ---

// TunnelHealthUpdate berisi field yang diupdate saat health cek.
type TunnelHealthUpdate struct {
	Status          *TunnelStatus
	LastHandshakeAt *time.Time
	LatencyMs       *int
	ActiveEndpoint  string
}
