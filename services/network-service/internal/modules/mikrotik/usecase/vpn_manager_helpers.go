// File ini berisi helper methods untuk vpnManager.
// Dipisahkan agar setiap file tetap di bawah 200 baris.
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// buildDetailResponse membangun VPNTunnelDetailResponse dari entity.
// Private key, PSK, dan password selalu di-mask dengan "********".
func (m *vpnManager) buildDetailResponse(t *domain.VPNTunnel) *domain.VPNTunnelDetailResponse {
	base := m.buildTunnelResponse(t)

	detail := &domain.VPNTunnelDetailResponse{
		VPNTunnelResponse: *base,
		ActiveEndpoint:    t.ActiveEndpoint,
	}

	// Mask private key jika ada
	if t.ClientPrivateKeyEncrypted != "" {
		detail.ClientPrivateKeyMasked = "********"
	}
	// Mask pre-shared key jika ada
	if t.PreSharedKeyEncrypted != "" {
		detail.PreSharedKeyMasked = "********"
	}
	// Mask L2TP credential jika ada
	if t.L2TPUsername != "" {
		detail.L2TPUsername = t.L2TPUsername
	}
	if t.L2TPPasswordEncrypted != "" {
		detail.L2TPPasswordMasked = "********"
	}

	return detail
}

// buildTunnelResponse membangun VPNTunnelResponse dari entity.
func (m *vpnManager) buildTunnelResponse(t *domain.VPNTunnel) *domain.VPNTunnelResponse {
	return &domain.VPNTunnelResponse{
		ID:                  t.ID,
		TunnelName:          t.TunnelName,
		RouterID:            t.RouterID,
		Protocol:            t.Protocol,
		VPNIP:               t.VPNIP,
		ServerEndpoint:      t.ServerEndpoint,
		ServerPublicKey:     t.ServerPublicKey,
		ClientPublicKey:     t.ClientPublicKey,
		Status:              t.Status,
		ListenPort:          t.ListenPort,
		AllowedAddresses:    t.AllowedAddresses,
		PersistentKeepalive: t.PersistentKeepalive,
		LatencyMs:           t.LatencyMs,
		BandwidthCapMbps:    t.BandwidthCapMbps,
		LastHandshakeAt:     t.LastHandshakeAt,
		Notes:               t.Notes,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
}

// generateWireGuardKeys menghasilkan key pair WireGuard dan PSK, lalu encrypt private key.
func (m *vpnManager) generateWireGuardKeys(tunnel *domain.VPNTunnel) error {
	pubKey, privKey, err := m.keyGen.GenerateWireGuardKeyPair()
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrKeyGenerationFailed, err)
	}

	psk, err := m.keyGen.GeneratePreSharedKey()
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrKeyGenerationFailed, err)
	}

	// Encrypt private key dan PSK sebelum disimpan
	encPrivKey, err := m.crypto.Encrypt(privKey)
	if err != nil {
		return fmt.Errorf("%w: gagal encrypt private key", domain.ErrEncryptionFailed)
	}

	encPSK, err := m.crypto.Encrypt(psk)
	if err != nil {
		return fmt.Errorf("%w: gagal encrypt psk", domain.ErrEncryptionFailed)
	}

	tunnel.ServerPublicKey = m.serverCfg.ServerPublicKey
	tunnel.ClientPublicKey = pubKey
	tunnel.ClientPrivateKeyEncrypted = encPrivKey
	tunnel.PreSharedKeyEncrypted = encPSK

	return nil
}

// generateL2TPCredentials menghasilkan username, password, dan IPSec PSK untuk L2TP.
func (m *vpnManager) generateL2TPCredentials(tunnel *domain.VPNTunnel) error {
	username, password, err := m.keyGen.GenerateCredentials(tunnel.TunnelName)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrKeyGenerationFailed, err)
	}

	ipsecPSK, err := m.keyGen.GenerateIPSecPSK()
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrKeyGenerationFailed, err)
	}

	// Encrypt password dan IPSec PSK sebelum disimpan
	encPassword, err := m.crypto.Encrypt(password)
	if err != nil {
		return fmt.Errorf("%w: gagal encrypt password", domain.ErrEncryptionFailed)
	}

	encPSK, err := m.crypto.Encrypt(ipsecPSK)
	if err != nil {
		return fmt.Errorf("%w: gagal encrypt ipsec psk", domain.ErrEncryptionFailed)
	}

	tunnel.L2TPUsername = username
	tunnel.L2TPPasswordEncrypted = encPassword
	tunnel.PreSharedKeyEncrypted = encPSK

	return nil
}

// generateBasicCredentials menghasilkan username dan password untuk PPTP/SSTP/OpenVPN.
func (m *vpnManager) generateBasicCredentials(tunnel *domain.VPNTunnel) error {
	username, password, err := m.keyGen.GenerateCredentials(tunnel.TunnelName)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrKeyGenerationFailed, err)
	}

	// Encrypt password sebelum disimpan
	encPassword, err := m.crypto.Encrypt(password)
	if err != nil {
		return fmt.Errorf("%w: gagal encrypt password", domain.ErrEncryptionFailed)
	}

	tunnel.L2TPUsername = username
	tunnel.L2TPPasswordEncrypted = encPassword

	return nil
}

// buildServerEndpoint membangun string endpoint server dari konfigurasi.
// Format: "host:port"
func (m *vpnManager) buildServerEndpoint() string {
	return fmt.Sprintf("%s:%d", m.serverCfg.PrimaryEndpoint, m.serverCfg.ListenPort)
}

// publishTunnelCreated mempublikasikan event vpn_tunnel_created (best-effort).
func (m *vpnManager) publishTunnelCreated(ctx context.Context, tunnel *domain.VPNTunnel, tenantID string) {
	payload := domain.VPNTunnelCreatedPayload{
		CorrelationID: uuid.New().String(),
		TunnelID:      tunnel.ID,
		TunnelName:    tunnel.TunnelName,
		TenantID:      tenantID,
		Protocol:      string(tunnel.Protocol),
		Status:        string(tunnel.Status),
	}

	if err := m.eventPub.PublishTunnelCreated(ctx, payload); err != nil {
		m.logger.Error().Err(err).Str("tunnel_id", tunnel.ID).
			Msg("gagal publish event vpn_tunnel_created")
	}
}
