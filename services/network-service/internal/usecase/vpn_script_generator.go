package usecase

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// vpnScriptGenerator — implementasi VPNScriptGenerator menggunakan text/template
// =============================================================================

// ScriptTemplateData berisi semua data yang dibutuhkan template .rsc.
// Caller (VPN Manager) bertanggung jawab mengisi field dengan nilai yang sudah di-dekripsi.
type ScriptTemplateData struct {
	TunnelID                string
	TunnelName              string
	GeneratedAt             string
	ClientPrivateKey        string // sudah di-dekripsi oleh caller
	ServerPublicKey         string
	SecondaryServerPublicKey string
	PreSharedKey            string // sudah di-dekripsi, kosong jika tidak ada
	PrimaryEndpoint         string
	SecondaryEndpoint       string
	ListenPort              int
	VPNIP                   string
	AllowedAddresses        string
	PersistentKeepalive     int
	L2TPUsername            string
	L2TPPassword            string // sudah di-dekripsi oleh caller
	IPSecPSK                string // sudah di-dekripsi oleh caller
}

// VPNScriptConfig berisi konfigurasi server VPN untuk script generator.
type VPNScriptConfig struct {
	PrimaryEndpoint          string
	SecondaryEndpoint        string
	ServerPublicKey          string
	SecondaryServerPublicKey string
}

// vpnScriptGenerator menghasilkan RouterOS script (.rsc) per protokol VPN.
type vpnScriptGenerator struct {
	config    VPNScriptConfig
	templates map[domain.VPNProtocol]*template.Template
}

// NewVPNScriptGenerator membuat instance VPNScriptGenerator baru.
// Semua template di-parse saat inisialisasi untuk fail-fast jika ada error.
func NewVPNScriptGenerator(cfg VPNScriptConfig) domain.VPNScriptGenerator {
	g := &vpnScriptGenerator{
		config:    cfg,
		templates: make(map[domain.VPNProtocol]*template.Template),
	}
	g.templates[domain.ProtocolWireGuard] = template.Must(
		template.New("wireguard").Parse(wireguardTplStr),
	)
	g.templates[domain.ProtocolL2TPIPSec] = template.Must(
		template.New("l2tp").Parse(l2tpTplStr),
	)
	g.templates[domain.ProtocolPPTP] = template.Must(
		template.New("pptp").Parse(pptpTplStr),
	)
	g.templates[domain.ProtocolSSTP] = template.Must(
		template.New("sstp").Parse(sstpTplStr),
	)
	g.templates[domain.ProtocolOpenVPN] = template.Must(
		template.New("openvpn").Parse(openvpnTplStr),
	)
	return g
}

// Generate menghasilkan script .rsc berdasarkan tunnel configuration.
// Tunnel yang diterima harus sudah memiliki key/credential yang di-dekripsi
// di field encrypted-nya (caller bertanggung jawab mendekripsi sebelum memanggil).
// Script TIDAK boleh mengandung server private key.
func (g *vpnScriptGenerator) Generate(tunnel *domain.VPNTunnel, subnet *domain.VPNSubnet) (string, error) {
	if tunnel == nil {
		return "", fmt.Errorf("tunnel tidak boleh nil")
	}
	if subnet == nil {
		return "", fmt.Errorf("subnet tidak boleh nil")
	}

	tmpl, ok := g.templates[tunnel.Protocol]
	if !ok {
		return "", fmt.Errorf("%w: %s", domain.ErrInvalidVPNProtocol, tunnel.Protocol)
	}

	data := g.buildTemplateData(tunnel)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("gagal generate script %s: %w", tunnel.Protocol, err)
	}

	return buf.String(), nil
}

// buildTemplateData membangun ScriptTemplateData dari VPNTunnel dan config server.
// Field encrypted dibaca langsung — caller harus sudah mendekripsi sebelumnya.
func (g *vpnScriptGenerator) buildTemplateData(tunnel *domain.VPNTunnel) ScriptTemplateData {
	// Pisahkan host:port dari ServerEndpoint untuk PrimaryEndpoint
	primaryEndpoint := g.config.PrimaryEndpoint
	if primaryEndpoint == "" {
		primaryEndpoint = extractHost(tunnel.ServerEndpoint)
	}

	return ScriptTemplateData{
		TunnelID:                 tunnel.ID,
		TunnelName:               tunnel.TunnelName,
		GeneratedAt:              time.Now().Format(time.RFC3339),
		ClientPrivateKey:         tunnel.ClientPrivateKeyEncrypted,
		ServerPublicKey:          tunnel.ServerPublicKey,
		SecondaryServerPublicKey: g.config.SecondaryServerPublicKey,
		PreSharedKey:             tunnel.PreSharedKeyEncrypted,
		PrimaryEndpoint:          primaryEndpoint,
		SecondaryEndpoint:        g.config.SecondaryEndpoint,
		ListenPort:               tunnel.ListenPort,
		VPNIP:                    tunnel.VPNIP,
		AllowedAddresses:         tunnel.AllowedAddresses,
		PersistentKeepalive:      tunnel.PersistentKeepalive,
		L2TPUsername:             tunnel.L2TPUsername,
		L2TPPassword:             tunnel.L2TPPasswordEncrypted,
		IPSecPSK:                 tunnel.PreSharedKeyEncrypted,
	}
}

// extractHost mengekstrak hostname dari format "host:port".
// Jika tidak ada port, mengembalikan string asli.
func extractHost(endpoint string) string {
	if idx := strings.LastIndex(endpoint, ":"); idx > 0 {
		return endpoint[:idx]
	}
	return endpoint
}
