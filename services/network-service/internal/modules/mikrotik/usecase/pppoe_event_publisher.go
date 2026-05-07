// Package usecase berisi implementasi business logic untuk network-service.
package usecase

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Event type constants untuk PPPoE events.
const (
	EventCommandResult = "mikrotik.command_result"
	EventSyncFailed    = "mikrotik.sync_failed"
)

// pppoeEventPublisher mengimplementasikan domain.PPPoEEventPublisher.
// Menggunakan pkg/queue.EnqueueTask untuk terbitkan event ke Redis queue.
// Best-effort: log error jika terbitkan gagal, jangan kembalikan error ke caller.
type pppoeEventPublisher struct {
	client *asynq.Client
	logger zerolog.Logger
}

// NewPPPoEEventPublisher membuat instance baru pppoeEventPublisher.
// Menerima asynq.Klien yang sudah terkoneksi ke Redis.
func NewPPPoEEventPublisher(client *asynq.Client, logger zerolog.Logger) domain.PPPoEEventPublisher {
	return &pppoeEventPublisher{
		client: client,
		logger: logger,
	}
}

// PublishCommandResult mempublikasikan hasil eksekusi perintah ke router.
// Payload berisi correlation_id, customer_id, router_id, tenant_id,
// operation, status, error_message, executed_at, duration_ms.
func (ep *pppoeEventPublisher) PublishCommandResult(ctx context.Context, result domain.CommandResultPayload) error {
	ep.publish(EventCommandResult, result.TenantID, result)
	return nil
}

// PublishSyncFailed mempublikasikan event sinkronisasi gagal untuk notifikasi.
// Payload berisi router_id, router_name, tenant_id, operation, error_message, failed_at.
func (ep *pppoeEventPublisher) PublishSyncFailed(ctx context.Context, payload domain.SyncFailedPayload) error {
	ep.publish(EventSyncFailed, payload.TenantID, payload)
	return nil
}

// terbitkan adalah helper internal untuk membuat TaskEnvelope dan mengirim ke queue.
// Best-effort: log error jika gagal, tidak kembalikan error ke caller.
func (ep *pppoeEventPublisher) publish(eventType, tenantID string, payload interface{}) {
	// Serialisasi payload ke JSON
	raw, err := json.Marshal(payload)
	if err != nil {
		ep.logger.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("gagal serialisasi event payload")
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
			Msg("gagal publish event ke queue")
		return
	}

	ep.logger.Debug().
		Str("event_type", eventType).
		Str("tenant_id", tenantID).
		Msg("event berhasil dipublish ke queue")
}
