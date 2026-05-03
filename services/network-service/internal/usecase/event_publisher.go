// Package usecase berisi implementasi business logic untuk network-service.
package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Event type constants untuk router events.
const (
	EventRouterOffline          = "mikrotik.router_offline"
	EventRouterOnline           = "mikrotik.router_online"
	EventRouterUnexpectedReboot = "mikrotik.router_unexpected_reboot"
)

// eventPublisher mengimplementasikan domain.EventPublisher.
// Menggunakan pkg/queue.EnqueueTask untuk publish event ke Redis queue.
// Best-effort: log error jika publish gagal, jangan return error ke caller.
type eventPublisher struct {
	client *asynq.Client
}

// NewEventPublisher membuat instance baru eventPublisher.
// Menerima asynq.Client yang sudah terkoneksi ke Redis.
func NewEventPublisher(client *asynq.Client) domain.EventPublisher {
	return &eventPublisher{client: client}
}

// PublishRouterOffline mempublikasikan event router offline ke queue.
// Payload berisi router_id, router_name, tenant_id, dan last_online_at.
func (ep *eventPublisher) PublishRouterOffline(ctx context.Context, router *domain.Router) error {
	// Tentukan last_online_at — gunakan zero time jika belum pernah online
	lastOnline := time.Time{}
	if router.LastOnlineAt != nil {
		lastOnline = *router.LastOnlineAt
	}

	payload := domain.RouterOfflinePayload{
		RouterID:     router.ID,
		RouterName:   router.Name,
		TenantID:     router.TenantID,
		LastOnlineAt: lastOnline,
	}

	ep.publish(EventRouterOffline, router.TenantID, payload)
	return nil
}

// PublishRouterOnline mempublikasikan event router online ke queue.
// Payload berisi router_id, router_name, tenant_id, dan downtime_duration.
func (ep *eventPublisher) PublishRouterOnline(ctx context.Context, router *domain.Router, downtimeDuration time.Duration) error {
	payload := domain.RouterOnlinePayload{
		RouterID:         router.ID,
		RouterName:       router.Name,
		TenantID:         router.TenantID,
		DowntimeDuration: downtimeDuration,
	}

	ep.publish(EventRouterOnline, router.TenantID, payload)
	return nil
}

// PublishUnexpectedReboot mempublikasikan event reboot tak terduga ke queue.
// Payload berisi router_id, router_name, tenant_id, previous dan current uptime.
func (ep *eventPublisher) PublishUnexpectedReboot(ctx context.Context, router *domain.Router, prevUptime, currUptime int64) error {
	payload := domain.RouterRebootPayload{
		RouterID:              router.ID,
		RouterName:            router.Name,
		TenantID:              router.TenantID,
		PreviousUptimeSeconds: prevUptime,
		CurrentUptimeSeconds:  currUptime,
	}

	ep.publish(EventRouterUnexpectedReboot, router.TenantID, payload)
	return nil
}

// publish adalah helper internal untuk membuat TaskEnvelope dan mengirim ke queue.
// Best-effort: log error jika gagal, tidak return error ke caller.
func (ep *eventPublisher) publish(eventType, tenantID string, payload interface{}) {
	// Serialisasi payload ke JSON
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("gagal serialisasi event payload")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   raw,
		// CorrelationID dan Timestamp akan di-generate otomatis oleh EnqueueTask
	}

	if err := queue.EnqueueTask(ep.client, envelope); err != nil {
		log.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("gagal publish event ke queue")
		return
	}

	log.Debug().
		Str("event_type", eventType).
		Str("tenant_id", tenantID).
		Msg("event berhasil dipublish ke queue")
}
