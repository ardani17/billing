// File ini berisi implementasi method VPN Manager untuk operasi setup & configure.
// Mencakup: TestConnection, GenerateScript, UpdateRouterHost, GetBandwidth.
package usecase

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// TestConnection menguji koneksi VPN dengan ping ke client VPN IP.
// Mengukur latency via TCP dial ke port 8728 (RouterOS API).
// Jika berhasil: perbarui status ke "connected", catat last_handshake_at dan latency_ms.
// Jika gagal: kembalikan VPNTestResult dengan diagnostic error.
func (m *vpnManager) TestConnection(ctx context.Context, id string) (*domain.VPNTestResult, error) {
	tunnel, err := m.tunnelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}

	// Ukur latency dengan TCP dial ke VPN IP port 8728
	start := time.Now()
	conn, dialErr := net.DialTimeout("tcp", tunnel.VPNIP+":8728", 5*time.Second)
	latency := int(time.Since(start).Milliseconds())

	if dialErr != nil {
		// Tentukan diagnostic berdasarkan jenis error
		diagnostic := "unreachable"
		if netErr, ok := dialErr.(net.Error); ok && netErr.Timeout() {
			diagnostic = "handshake_timeout"
		}

		m.logger.Warn().Str("tunnel_id", id).Str("vpn_ip", tunnel.VPNIP).
			Str("diagnostic", diagnostic).Err(dialErr).Msg("test koneksi vpn gagal")

		return &domain.VPNTestResult{
			Status:       tunnel.Status,
			LatencyMs:    latency,
			ErrorMessage: dialErr.Error(),
			Diagnostic:   diagnostic,
		}, nil
	}
	conn.Close()

	// Koneksi berhasil - perbarui status ke connected
	now := time.Now()
	connectedStatus := domain.TunnelStatusConnected
	healthUpdate := domain.TunnelHealthUpdate{
		Status:          &connectedStatus,
		LastHandshakeAt: &now,
		LatencyMs:       &latency,
	}
	if err := m.tunnelRepo.UpdateStatus(ctx, id, healthUpdate); err != nil {
		m.logger.Error().Err(err).Str("tunnel_id", id).
			Msg("gagal update status setelah test berhasil")
	}

	m.logger.Info().Str("tunnel_id", id).Int("latency_ms", latency).
		Msg("test koneksi vpn berhasil")

	return &domain.VPNTestResult{
		Status:          domain.TunnelStatusConnected,
		LatencyMs:       latency,
		LastHandshakeAt: &now,
	}, nil
}

// GenerateScript menghasilkan RouterOS script (.rsc) untuk setup manual.
// Mendekripsi key/credential terlebih dahulu, lalu delegasi ke VPNScriptGenerator.
func (m *vpnManager) GenerateScript(ctx context.Context, id string) (string, error) {
	tunnel, err := m.tunnelRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}

	// Ambil subnet untuk tenant
	subnet, err := m.subnetRepo.GetByTenantID(ctx, tunnel.TenantID)
	if err != nil {
		return "", fmt.Errorf("gagal ambil subnet tenant %s: %w", tunnel.TenantID, err)
	}

	// Buat copy tunnel dengan key/credential yang sudah di-dekripsi
	decrypted := *tunnel
	if tunnel.ClientPrivateKeyEncrypted != "" {
		plain, err := m.crypto.Decrypt(tunnel.ClientPrivateKeyEncrypted)
		if err != nil {
			return "", fmt.Errorf("gagal decrypt client private key: %w", err)
		}
		decrypted.ClientPrivateKeyEncrypted = plain
	}
	if tunnel.PreSharedKeyEncrypted != "" {
		plain, err := m.crypto.Decrypt(tunnel.PreSharedKeyEncrypted)
		if err != nil {
			return "", fmt.Errorf("gagal decrypt pre-shared key: %w", err)
		}
		decrypted.PreSharedKeyEncrypted = plain
	}
	if tunnel.L2TPPasswordEncrypted != "" {
		plain, err := m.crypto.Decrypt(tunnel.L2TPPasswordEncrypted)
		if err != nil {
			return "", fmt.Errorf("gagal decrypt l2tp password: %w", err)
		}
		decrypted.L2TPPasswordEncrypted = plain
	}

	// Delegasi ke script generator
	script, err := m.scriptGen.Generate(&decrypted, subnet)
	if err != nil {
		return "", fmt.Errorf("gagal generate script: %w", err)
	}

	m.logger.Info().Str("tunnel_id", id).Str("protocol", string(tunnel.Protocol)).
		Msg("script .rsc berhasil di-generate")

	return script, nil
}

// UpdateRouterHost mengupdate host router ke VPN IP setelah tunnel terverifikasi.
// Jika koneksi via VPN IP gagal, revert ke host asli.
func (m *vpnManager) UpdateRouterHost(ctx context.Context, tunnelID string) error {
	tunnel, err := m.tunnelRepo.GetByID(ctx, tunnelID)
	if err != nil {
		return fmt.Errorf("gagal ambil tunnel %s: %w", tunnelID, err)
	}

	if tunnel.RouterID == nil {
		return fmt.Errorf("tunnel %s tidak memiliki router_id", tunnelID)
	}
	if tunnel.Status != domain.TunnelStatusConnected {
		return fmt.Errorf("tunnel harus berstatus connected, saat ini: %s", tunnel.Status)
	}

	router, err := m.routerRepo.GetByID(ctx, *tunnel.RouterID)
	if err != nil {
		return fmt.Errorf("gagal ambil router %s: %w", *tunnel.RouterID, err)
	}

	// Simpan host asli untuk revert jika gagal
	originalHost := router.Host

	// Perbarui host router ke VPN IP
	router.Host = tunnel.VPNIP
	router.UpdatedAt = time.Now()
	if _, err := m.routerRepo.Update(ctx, router); err != nil {
		return fmt.Errorf("gagal update host router: %w", err)
	}

	// Tes koneksi via VPN IP
	conn, dialErr := net.DialTimeout("tcp", tunnel.VPNIP+":8728", 5*time.Second)
	if dialErr != nil {
		m.logger.Warn().Str("tunnel_id", tunnelID).Str("vpn_ip", tunnel.VPNIP).
			Err(dialErr).Msg("koneksi via vpn ip gagal, revert ke host asli")

		router.Host = originalHost
		router.UpdatedAt = time.Now()
		if _, revertErr := m.routerRepo.Update(ctx, router); revertErr != nil {
			m.logger.Error().Err(revertErr).Msg("gagal revert host router")
		}
		return domain.ErrVPNIPUpdateFailed
	}
	conn.Close()

	m.logger.Info().Str("tunnel_id", tunnelID).Str("vpn_ip", tunnel.VPNIP).
		Str("original_host", originalHost).Msg("host router berhasil diupdate ke vpn ip")

	return nil
}

// GetBandwidth mengambil statistik bandwidth untuk satu tunnel.
// Mengembalikan data point terbaru dan history dalam rentang waktu.
func (m *vpnManager) GetBandwidth(ctx context.Context, id string, from, to time.Time) (*domain.VPNBandwidthResult, error) {
	if _, err := m.tunnelRepo.GetByID(ctx, id); err != nil {
		return nil, fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}

	// Ambil data point terbaru
	current, err := m.bwStore.GetLatest(ctx, id)
	if err != nil {
		m.logger.Warn().Err(err).Str("tunnel_id", id).
			Msg("gagal ambil bandwidth terbaru")
	}

	// Ambil history dalam rentang waktu
	history, err := m.bwStore.Query(ctx, id, from, to)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil history bandwidth: %w", err)
	}

	return &domain.VPNBandwidthResult{
		Current: current,
		History: history,
	}, nil
}
