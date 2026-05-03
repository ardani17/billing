// File ini mengimplementasikan method CreateTunnel pada vpnManager.
// Dipisahkan dari vpn_manager.go agar setiap file tetap di bawah 200 baris.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// CreateTunnel membuat VPN tunnel baru dengan auto-generate key/credential dan IP allocation.
// Langkah: validasi → cek router → get/create subnet → alokasi IP → generate key → encrypt → simpan → publish event.
func (m *vpnManager) CreateTunnel(ctx context.Context, tenantID string, req domain.CreateVPNTunnelRequest) (*domain.VPNTunnelResponse, error) {
	// Validasi protokol VPN
	if !domain.IsValidVPNProtocol(req.Protocol) {
		return nil, domain.ErrInvalidVPNProtocol
	}

	// Jika router_id diberikan, cek versi router (WireGuard butuh v7+)
	if req.RouterID != "" {
		if err := m.validateRouterVersion(ctx, req.RouterID, req.Protocol); err != nil {
			return nil, err
		}
	}

	// GetOrCreateSubnet: cek apakah tenant sudah punya subnet, jika belum buat baru
	subnet, err := m.getOrCreateSubnet(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// AllocateNextIP: ambil IP berikutnya dari subnet
	clientIP, err := m.allocateNextIP(ctx, tenantID, subnet)
	if err != nil {
		return nil, err
	}

	// Generate key/credential sesuai protokol
	tunnel, err := m.generateTunnelCredentials(req, tenantID, subnet, clientIP)
	if err != nil {
		return nil, err
	}

	// Simpan tunnel ke database
	created, err := m.tunnelRepo.Create(ctx, tunnel)
	if err != nil {
		return nil, fmt.Errorf("gagal simpan tunnel: %w", err)
	}

	// Publish event vpn_tunnel_created (best-effort, log error jika gagal)
	m.publishTunnelCreated(ctx, created, tenantID)

	m.logger.Info().Str("tunnel_id", created.ID).Str("protocol", req.Protocol).
		Str("vpn_ip", clientIP).Msg("tunnel vpn berhasil dibuat")

	return m.buildTunnelResponse(created), nil
}

// validateRouterVersion memeriksa versi router untuk kompatibilitas protokol.
// WireGuard membutuhkan RouterOS v7 atau lebih baru.
func (m *vpnManager) validateRouterVersion(ctx context.Context, routerID, protocol string) error {
	router, err := m.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return fmt.Errorf("gagal ambil router %s: %w", routerID, err)
	}

	if protocol == string(domain.ProtocolWireGuard) && !domain.IsRouterOSv7(router.RouterOSVersion) {
		return domain.ErrWireGuardRequiresV7
	}

	return nil
}

// getOrCreateSubnet mengambil subnet tenant yang sudah ada, atau membuat baru jika belum ada.
func (m *vpnManager) getOrCreateSubnet(ctx context.Context, tenantID string) (*domain.VPNSubnet, error) {
	subnet, err := m.subnetRepo.GetByTenantID(ctx, tenantID)
	if err == nil && subnet != nil {
		return subnet, nil
	}

	// Subnet belum ada (nil, nil dari repo) atau error not found — buat baru
	nextSeq, err := m.subnetRepo.GetNextTenantSeq(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil tenant seq berikutnya: %w", err)
	}

	newSubnet := &domain.VPNSubnet{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		SubnetPrefix:    domain.BuildSubnetPrefix(nextSeq),
		TenantSeq:       nextSeq,
		ServerIP:        domain.BuildServerIP(nextSeq),
		NextClientIPSeq: 2,
		CreatedAt:       time.Now(),
	}

	created, err := m.subnetRepo.Create(ctx, newSubnet)
	if err != nil {
		return nil, fmt.Errorf("gagal buat subnet untuk tenant %s: %w", tenantID, err)
	}

	m.logger.Info().Str("tenant_id", tenantID).Str("subnet", created.SubnetPrefix).
		Msg("subnet vpn baru dialokasikan")

	return created, nil
}

// allocateNextIP mengambil IP client berikutnya dari subnet tenant.
// Mengembalikan error jika subnet sudah penuh (253 client).
func (m *vpnManager) allocateNextIP(ctx context.Context, tenantID string, subnet *domain.VPNSubnet) (string, error) {
	clientSeq, err := m.subnetRepo.IncrementNextClientIPSeq(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("gagal alokasi IP berikutnya: %w", err)
	}

	if !domain.IsValidClientSeq(clientSeq) {
		return "", domain.ErrVPNSubnetExhausted
	}

	return domain.BuildClientIP(subnet.TenantSeq, clientSeq), nil
}

// generateTunnelCredentials membuat entity VPNTunnel dengan key/credential sesuai protokol.
func (m *vpnManager) generateTunnelCredentials(
	req domain.CreateVPNTunnelRequest,
	tenantID string,
	subnet *domain.VPNSubnet,
	clientIP string,
) (*domain.VPNTunnel, error) {
	tunnel := &domain.VPNTunnel{
		TenantID:            tenantID,
		TunnelName:          req.TunnelName,
		Protocol:            domain.VPNProtocol(req.Protocol),
		VPNIP:               clientIP,
		ServerEndpoint:      m.buildServerEndpoint(),
		Status:              domain.TunnelStatusPending,
		ListenPort:          m.serverCfg.ListenPort,
		AllowedAddresses:    "10.99.0.0/16",
		PersistentKeepalive: 25,
		RateLimitPps:        100,
		Notes:               req.Notes,
	}

	// Bandwidth cap default — akan diupdate berdasarkan tier tenant oleh billing service
	defaultBwCap := 50 // default Growth tier (Mbps)
	tunnel.BandwidthCapMbps = &defaultBwCap

	if req.RouterID != "" {
		tunnel.RouterID = &req.RouterID
	}

	protocol := domain.VPNProtocol(req.Protocol)

	switch protocol {
	case domain.ProtocolWireGuard:
		if err := m.generateWireGuardKeys(tunnel); err != nil {
			return nil, err
		}
	case domain.ProtocolL2TPIPSec:
		if err := m.generateL2TPCredentials(tunnel); err != nil {
			return nil, err
		}
	case domain.ProtocolPPTP, domain.ProtocolSSTP, domain.ProtocolOpenVPN:
		if err := m.generateBasicCredentials(tunnel); err != nil {
			return nil, err
		}
	}

	return tunnel, nil
}
