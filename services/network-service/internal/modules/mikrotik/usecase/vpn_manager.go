// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan struct vpnManager beserta constructor dan method
// GetTunnel, UpdateTunnel, DeleteTunnel, ListTunnels, GetSummary.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// VPNServerConfig berisi konfigurasi VPN server (endpoints, public keys, listen port).
type VPNServerConfig struct {
	PrimaryEndpoint          string
	SecondaryEndpoint        string
	ServerPublicKey          string
	SecondaryServerPublicKey string
	ListenPort               int
}

// vpnManager mengimplementasikan domain.VPNManager.
// Mengelola lifecycle VPN tunnel: buat, configure, test, monitor, hapus.
type vpnManager struct {
	tunnelRepo  domain.VPNTunnelRepository
	subnetRepo  domain.VPNSubnetRepository
	routerRepo  domain.RouterRepository
	poolManager domain.PoolManager
	crypto      domain.CredentialEncryptor
	keyGen      domain.VPNKeyGenerator
	scriptGen   domain.VPNScriptGenerator
	eventPub    domain.VPNEventPublisher
	cmdBuilder  func(routerOSVersion string) domain.VPNCommandBuilder
	bwStore     domain.VPNBandwidthStore
	serverCfg   VPNServerConfig
	logger      zerolog.Logger
}

// NewVPNManager membuat instance VPNManager baru dengan semua dependensi.
func NewVPNManager(
	tunnelRepo domain.VPNTunnelRepository,
	subnetRepo domain.VPNSubnetRepository,
	routerRepo domain.RouterRepository,
	poolManager domain.PoolManager,
	crypto domain.CredentialEncryptor,
	keyGen domain.VPNKeyGenerator,
	scriptGen domain.VPNScriptGenerator,
	eventPub domain.VPNEventPublisher,
	cmdBuilder func(routerOSVersion string) domain.VPNCommandBuilder,
	bwStore domain.VPNBandwidthStore,
	serverCfg VPNServerConfig,
	logger zerolog.Logger,
) domain.VPNManager {
	return &vpnManager{
		tunnelRepo:  tunnelRepo,
		subnetRepo:  subnetRepo,
		routerRepo:  routerRepo,
		poolManager: poolManager,
		crypto:      crypto,
		keyGen:      keyGen,
		scriptGen:   scriptGen,
		eventPub:    eventPub,
		cmdBuilder:  cmdBuilder,
		bwStore:     bwStore,
		serverCfg:   serverCfg,
		logger:      logger,
	}
}

// GetTunnel mengambil detail tunnel termasuk semua field (private key di-mask).
func (m *vpnManager) GetTunnel(ctx context.Context, id string) (*domain.VPNTunnelDetailResponse, error) {
	tunnel, err := m.tunnelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}
	return m.buildDetailResponse(tunnel), nil
}

// UpdateTunnel memperbarui field yang diizinkan pada tunnel.
// Field yang boleh diubah: tunnel_name, notes, router_id, persistent_keepalive, allowed_addresses.
func (m *vpnManager) UpdateTunnel(ctx context.Context, id string, req domain.UpdateVPNTunnelRequest) (*domain.VPNTunnelResponse, error) {
	tunnel, err := m.tunnelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}

	// Cek uniqueness tunnel_name jika berubah
	if req.TunnelName != "" && req.TunnelName != tunnel.TunnelName {
		exists, err := m.tunnelRepo.TunnelNameExists(ctx, tunnel.TenantID, req.TunnelName, tunnel.ID)
		if err != nil {
			return nil, fmt.Errorf("gagal cek nama tunnel: %w", err)
		}
		if exists {
			return nil, domain.ErrVPNTunnelNameExists
		}
		tunnel.TunnelName = req.TunnelName
	}

	// Perbarui field opsional
	if req.Notes != "" {
		tunnel.Notes = req.Notes
	}
	if req.RouterID != "" {
		tunnel.RouterID = &req.RouterID
	}
	if req.PersistentKeepalive != nil {
		tunnel.PersistentKeepalive = *req.PersistentKeepalive
	}
	if req.AllowedAddresses != "" {
		tunnel.AllowedAddresses = req.AllowedAddresses
	}

	tunnel.UpdatedAt = time.Now()

	updated, err := m.tunnelRepo.Update(ctx, tunnel)
	if err != nil {
		return nil, fmt.Errorf("gagal update tunnel %s: %w", id, err)
	}

	m.logger.Info().Str("tunnel_id", id).Msg("tunnel berhasil diupdate")
	return m.buildTunnelResponse(updated), nil
}

// DeleteTunnel melakukan hapus lunak tunnel.
// Jika router menggunakan VPN IP sebagai host, kembalikan warning error.
func (m *vpnManager) DeleteTunnel(ctx context.Context, id string) error {
	tunnel, err := m.tunnelRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}

	// Cek apakah router menggunakan VPN IP sebagai host
	if tunnel.RouterID != nil {
		router, err := m.routerRepo.GetByID(ctx, *tunnel.RouterID)
		if err == nil && router.Host == tunnel.VPNIP {
			return domain.ErrTunnelDeleteWarning
		}
	}

	// Soft-hapus tunnel di database
	if err := m.tunnelRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("gagal hapus tunnel %s: %w", id, err)
	}

	// Best-effort: log info setelah berhasil hapus
	m.logger.Info().Str("tunnel_id", id).Str("tunnel_name", tunnel.TunnelName).
		Msg("tunnel berhasil dihapus (soft-delete)")

	return nil
}

// ListTunnels mengambil daftar tunnel dengan paginasi dan filter.
func (m *vpnManager) ListTunnels(ctx context.Context, params domain.VPNTunnelListParams) (*domain.VPNTunnelListResult, error) {
	return m.tunnelRepo.List(ctx, params)
}

// GetSummary mengambil ringkasan status tunnel untuk dashboard.
func (m *vpnManager) GetSummary(ctx context.Context) (*domain.VPNSummary, error) {
	counts, err := m.tunnelRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal hitung status tunnel: %w", err)
	}

	summary := &domain.VPNSummary{
		ConnectedCount:    counts[domain.TunnelStatusConnected],
		DisconnectedCount: counts[domain.TunnelStatusDisconnected],
		PendingCount:      counts[domain.TunnelStatusPending],
		ErrorCount:        counts[domain.TunnelStatusError],
	}
	summary.TotalTunnels = summary.ConnectedCount + summary.DisconnectedCount +
		summary.PendingCount + summary.ErrorCount

	return summary, nil
}
