// Package worker berisi komponen asynq worker untuk memproses event dari Redis queue.
package worker

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/notification/internal/usecase"
	"github.com/rs/zerolog"
)

// Daftar event type yang di-handle oleh EventConsumer.
// Event ini dipublikasikan oleh Billing API ke Redis queue.
const (
	EventInvoiceCreated          = "invoice.created"
	EventInvoiceReminder         = "invoice.reminder"
	EventInvoicePenaltyAdded     = "invoice.penalty_added"
	EventPaymentOnlineReceived   = "payment.online.received"
	EventPaymentRecorded         = "payment.recorded"
	EventCustomerIsolir          = "customer.isolir"
	EventCustomerUnIsolir        = "customer.un_isolir"
	EventCustomerSuspend         = "customer.suspend"
	EventNotificationIsolir      = "notification.isolir"
	EventNotificationUnIsolir    = "notification.un_isolir"
	EventNotificationSuspend     = "notification.suspend"
	EventNotificationReactivated = "notification.reactivated"
	EventPendingSyncFailed       = "notification.pending_sync_failed"
)

// allEventTypes berisi semua event type yang didaftarkan ke asynq ServeMux.
var allEventTypes = []string{
	EventInvoiceCreated,
	EventInvoiceReminder,
	EventInvoicePenaltyAdded,
	EventPaymentOnlineReceived,
	EventPaymentRecorded,
	EventCustomerIsolir,
	EventCustomerUnIsolir,
	EventCustomerSuspend,
	EventNotificationIsolir,
	EventNotificationUnIsolir,
	EventNotificationSuspend,
	EventNotificationReactivated,
	EventPendingSyncFailed,
}

// EventConsumer menangani event dari Redis queue via asynq.
// Setiap event di-decode dari TaskEnvelope lalu diteruskan ke DeliveryPipeline.
type EventConsumer struct {
	pipeline *usecase.DeliveryPipeline
	logger   zerolog.Logger
}

// NewEventConsumer membuat instance baru EventConsumer.
func NewEventConsumer(pipeline *usecase.DeliveryPipeline, logger zerolog.Logger) *EventConsumer {
	return &EventConsumer{
		pipeline: pipeline,
		logger:   logger.With().Str("component", "event_consumer").Logger(),
	}
}

// RegisterHandlers mendaftarkan semua handler event ke asynq ServeMux.
// Setiap event type di-route ke method handleEvent yang sama.
func (ec *EventConsumer) RegisterHandlers(mux *asynq.ServeMux) {
	for _, eventType := range allEventTypes {
		mux.HandleFunc(eventType, ec.handleEvent)
	}
	ec.logger.Info().Int("total_events", len(allEventTypes)).Msg("semua handler event berhasil didaftarkan")
}

// handleEvent memproses satu task dari asynq queue.
// Task di-decode menjadi TaskEnvelope lalu diteruskan ke DeliveryPipeline.
// Jika decode gagal, error di-log dan task di-skip (tidak di-retry).
func (ec *EventConsumer) handleEvent(ctx context.Context, task *asynq.Task) error {
	// Decode payload task menjadi TaskEnvelope
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		ec.logger.Warn().
			Err(err).
			Str("task_type", task.Type()).
			Msg("gagal decode TaskEnvelope, task di-skip")
		// Kembalikan nil agar asynq tidak me-retry task yang payload-nya invalid
		return nil
	}

	ec.logger.Info().
		Str("event_type", envelope.EventType).
		Str("tenant_id", envelope.TenantID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses event dari queue")

	// Teruskan envelope ke delivery pipeline untuk diproses
	if err := ec.pipeline.ProcessEvent(ctx, envelope); err != nil {
		ec.logger.Error().
			Err(err).
			Str("event_type", envelope.EventType).
			Str("tenant_id", envelope.TenantID).
			Str("correlation_id", envelope.CorrelationID).
			Msg("gagal memproses event")
		return err
	}

	ec.logger.Info().
		Str("event_type", envelope.EventType).
		Str("tenant_id", envelope.TenantID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("event berhasil diproses")

	return nil
}
