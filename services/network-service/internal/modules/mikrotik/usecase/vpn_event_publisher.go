// Package usecase berisi implementasi business logic untuk network-service.
package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Event type constants untuk VPN events.
const (
	EventVPNTunnelDown           = "mikrotik.vpn_tunnel_down"
	EventVPNTunnelUp             = "mikrotik.vpn_tunnel_up"
	EventVPNTunnelCreated        = "mikrotik.vpn_tunnel_created"
	EventVPNServerBandwidthHigh  = "mikrotik.vpn_server_bandwidth_high"
	EventVPNServerBandwidthNorm  = "mikrotik.vpn_server_bandwidth_normal"
	EventVPNMaintenanceScheduled = "mikrotik.vpn_maintenance_scheduled"
)

// vpnEventPublisher mengimplementasikan domain.VPNEventPublisher.
// Menggunakan pkg/queue.EnqueueTask untuk terbitkan event ke Redis queue.
// Best-effort: log error jika terbitkan gagal, jangan kembalikan error ke caller.
type vpnEventPublisher struct {
	client *asynq.Client
	logger zerolog.Logger
}

// NewVPNEventPublisher membuat instance baru vpnEventPublisher.
// Menerima asynq.Klien yang sudah terkoneksi ke Redis.
func NewVPNEventPublisher(client *asynq.Client, logger zerolog.Logger) domain.VPNEventPublisher {
	return &vpnEventPublisher{
		client: client,
		logger: logger,
	}
}

// PublishTunnelDown mempublikasikan event tunnel disconnected.
// Buat correlation_id jika belum diset pada payload.
func (ep *vpnEventPublisher) PublishTunnelDown(ctx context.Context, payload domain.VPNTunnelDownPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(EventVPNTunnelDown, payload.TenantID, payload)
	return nil
}

// PublishTunnelUp mempublikasikan event tunnel connected.
// Buat correlation_id jika belum diset pada payload.
func (ep *vpnEventPublisher) PublishTunnelUp(ctx context.Context, payload domain.VPNTunnelUpPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(EventVPNTunnelUp, payload.TenantID, payload)
	return nil
}

// PublishTunnelCreated mempublikasikan event tunnel created.
// Buat correlation_id jika belum diset pada payload.
func (ep *vpnEventPublisher) PublishTunnelCreated(ctx context.Context, payload domain.VPNTunnelCreatedPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(EventVPNTunnelCreated, payload.TenantID, payload)
	return nil
}

// PublishServerBandwidthHigh mempublikasikan event bandwidth server melebihi 80%.
// Event ini tidak memiliki tenant_id, gunakan server_endpoint sebagai identifier.
func (ep *vpnEventPublisher) PublishServerBandwidthHigh(ctx context.Context, payload domain.VPNServerBandwidthHighPayload) error {
	ep.publish(EventVPNServerBandwidthHigh, payload.ServerEndpoint, payload)
	return nil
}

// PublishServerBandwidthNormal mempublikasikan event bandwidth server kembali normal.
// Event ini tidak memiliki tenant_id, gunakan server_endpoint sebagai identifier.
func (ep *vpnEventPublisher) PublishServerBandwidthNormal(ctx context.Context, payload domain.VPNServerBandwidthNormalPayload) error {
	ep.publish(EventVPNServerBandwidthNorm, payload.ServerEndpoint, payload)
	return nil
}

// PublishMaintenanceScheduled mempublikasikan event jadwal maintenance ke tenant.
func (ep *vpnEventPublisher) PublishMaintenanceScheduled(ctx context.Context, payload domain.VPNMaintenanceScheduledPayload) error {
	ep.publish(EventVPNMaintenanceScheduled, payload.TenantID, payload)
	return nil
}

// terbitkan adalah helper internal untuk membuat TaskEnvelope dan mengirim ke queue.
// Best-effort: log error jika gagal, tidak kembalikan error ke caller.
func (ep *vpnEventPublisher) publish(eventType, tenantID string, payload interface{}) {
	// Serialisasi payload ke JSON
	raw, err := json.Marshal(payload)
	if err != nil {
		ep.logger.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("gagal serialisasi vpn event payload")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   raw,
	}

	if err := queue.EnqueueTask(ep.client, envelope); err != nil {
		ep.logger.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("gagal publish vpn event ke queue")
		return
	}

	ep.logger.Debug().
		Str("event_type", eventType).
		Str("tenant_id", tenantID).
		Msg("vpn event berhasil dipublish ke queue")
}
