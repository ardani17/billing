// File ini mengimplementasikan VPNHealthMonitor untuk monitoring periodik VPN tunnel.
package usecase

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const (
	vpnCheckInterval     = 30 * time.Second
	vpnPingTimeout       = 3 * time.Second
	vpnPingPort          = 8728
	vpnHandshakeMaxSec   = 150
	vpnBandwidthCheckMod = 2  // cek bandwidth setiap N tick
	vpnBwHighThreshold   = 80 // persen kapasitas server
	vpnBwNormThreshold   = 70
)

// vpnHealthMonitor mengimplementasikan domain.VPNHealthMonitor.
type vpnHealthMonitor struct {
	tunnelRepo     domain.VPNTunnelRepository
	events         domain.VPNEventPublisher
	bwStore        domain.VPNBandwidthStore
	logger         zerolog.Logger
	serverCapacity int64
	serverEndpoint string
	mu             sync.Mutex
	cancel         context.CancelFunc
	stopped, bandwidthHigh bool
	tickCount      int
}

func NewVPNHealthMonitor(
	tunnelRepo domain.VPNTunnelRepository, events domain.VPNEventPublisher,
	bwStore domain.VPNBandwidthStore, logger zerolog.Logger,
	serverCapacityMbps int64, serverEndpoint string,
) domain.VPNHealthMonitor {
	return &vpnHealthMonitor{
		tunnelRepo: tunnelRepo, events: events, bwStore: bwStore,
		logger: logger, serverCapacity: serverCapacityMbps, serverEndpoint: serverEndpoint,
	}
}
// Start memulai health monitor goroutine.
func (m *vpnHealthMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return fmt.Errorf("health monitor sudah dihentikan")
	}
	monCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go m.run(monCtx)
	m.logger.Info().Msg("vpn health monitor dimulai")
	return nil
}
// Stop menghentikan health monitor goroutine.
func (m *vpnHealthMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	if m.cancel != nil {
		m.cancel()
	}
	m.logger.Info().Msg("vpn health monitor dihentikan")
}

func (m *vpnHealthMonitor) run(ctx context.Context) {
	ticker := time.NewTicker(vpnCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			m.tickCount++
			tick := m.tickCount
			m.mu.Unlock()
			m.checkConnected(ctx)
			m.checkDisconnected(ctx)
			if tick%vpnBandwidthCheckMod == 0 {
				m.checkBandwidth(ctx)
			}
		}
	}
}
// checkConnected memeriksa tunnel connected, update latency atau transisi ke disconnected.
func (m *vpnHealthMonitor) checkConnected(ctx context.Context) {
	tunnels, err := m.tunnelRepo.GetConnectedTunnels(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("gagal ambil connected tunnels")
		return
	}
	for _, t := range tunnels {
		latency, ok := pingVPNClient(t.VPNIP)
		hsExpired := t.LastHandshakeAt != nil && time.Since(*t.LastHandshakeAt).Seconds() > vpnHandshakeMaxSec
		if !ok || hsExpired {
			disc := domain.TunnelStatusDisconnected
			_ = m.tunnelRepo.UpdateStatus(ctx, t.ID, domain.TunnelHealthUpdate{Status: &disc})
			_ = m.events.PublishTunnelDown(ctx, domain.VPNTunnelDownPayload{
				TunnelID: t.ID, TunnelName: t.TunnelName, TenantID: t.TenantID,
				RouterID: t.RouterID, Protocol: string(t.Protocol), VPNIP: t.VPNIP,
				LastHandshakeAt: t.LastHandshakeAt, DisconnectedAt: time.Now(),
			})
			continue
		}
		now := time.Now()
		ms := int(latency.Milliseconds())
		_ = m.tunnelRepo.UpdateStatus(ctx, t.ID, domain.TunnelHealthUpdate{
			LatencyMs: &ms, LastHandshakeAt: &now,
		})
	}
}
// checkDisconnected memeriksa tunnel disconnected untuk recovery detection.
func (m *vpnHealthMonitor) checkDisconnected(ctx context.Context) {
	tunnels, err := m.tunnelRepo.GetDisconnectedTunnels(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("gagal ambil disconnected tunnels")
		return
	}
	for _, t := range tunnels {
		latency, ok := pingVPNClient(t.VPNIP)
		if !ok {
			continue
		}
		conn := domain.TunnelStatusConnected
		now := time.Now()
		ms := int(latency.Milliseconds())
		_ = m.tunnelRepo.UpdateStatus(ctx, t.ID, domain.TunnelHealthUpdate{
			Status: &conn, LatencyMs: &ms, LastHandshakeAt: &now,
		})
		_ = m.events.PublishTunnelUp(ctx, domain.VPNTunnelUpPayload{
			TunnelID: t.ID, TunnelName: t.TunnelName, TenantID: t.TenantID,
			RouterID: t.RouterID, Protocol: string(t.Protocol), VPNIP: t.VPNIP,
			LatencyMs: ms, ConnectedAt: now,
		})
	}
}

// checkBandwidth memeriksa aggregate bandwidth dan publish event jika melebihi threshold.
func (m *vpnHealthMonitor) checkBandwidth(ctx context.Context) {
	tunnels, err := m.tunnelRepo.GetConnectedTunnels(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("gagal ambil tunnels untuk bandwidth check")
		return
	}
	var totalBps int64
	for _, t := range tunnels {
		pt, err := m.bwStore.GetLatest(ctx, t.ID)
		if err != nil || pt == nil {
			continue
		}
		totalBps += pt.Metrics.TXRateBps + pt.Metrics.RXRateBps
	}
	if m.serverCapacity <= 0 {
		return
	}
	totalMbps := totalBps / 1_000_000
	util := int(totalMbps * 100 / m.serverCapacity)
	now := time.Now()
	m.mu.Lock()
	wasHigh := m.bandwidthHigh
	m.mu.Unlock()

	if util > vpnBwHighThreshold && !wasHigh {
		m.mu.Lock()
		m.bandwidthHigh = true
		m.mu.Unlock()
		_ = m.events.PublishServerBandwidthHigh(ctx, domain.VPNServerBandwidthHighPayload{
			ServerEndpoint: m.serverEndpoint, CurrentUsageMbps: totalMbps,
			CapacityMbps: m.serverCapacity, UtilizationPercent: util, Timestamp: now,
		})
	} else if util < vpnBwNormThreshold && wasHigh {
		m.mu.Lock()
		m.bandwidthHigh = false
		m.mu.Unlock()
		_ = m.events.PublishServerBandwidthNormal(ctx, domain.VPNServerBandwidthNormalPayload{
			ServerEndpoint: m.serverEndpoint, CurrentUsageMbps: totalMbps,
			CapacityMbps: m.serverCapacity, UtilizationPercent: util, Timestamp: now,
		})
	}
}

// pingVPNClient melakukan TCP dial ke client VPN IP pada port RouterOS API.
func pingVPNClient(vpnIP string) (time.Duration, bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", vpnIP, vpnPingPort), vpnPingTimeout)
	if err != nil {
		return 0, false
	}
	_ = conn.Close()
	return time.Since(start), true
}
