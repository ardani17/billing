// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan OLTEventPublisher untuk terbitkan event OLT ke Redis queue.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: oltEventPublisher harus mengimplementasikan domain.OLTEventPublisher.
var _ domain.OLTEventPublisher = (*oltEventPublisher)(nil)

// oltEventPublisher mengimplementasikan domain.OLTEventPublisher.
// Menggunakan pkg/queue.EnqueueTask untuk terbitkan event ke Redis queue.
// Best-effort: log error jika terbitkan gagal, jangan kembalikan error ke caller.
type oltEventPublisher struct {
	client *asynq.Client
	logger zerolog.Logger
}

// NewOLTEventPublisher membuat instance baru oltEventPublisher.
// Menerima asynq.Klien yang sudah terkoneksi ke Redis.
func NewOLTEventPublisher(client *asynq.Client, logger zerolog.Logger) domain.OLTEventPublisher {
	return &oltEventPublisher{
		client: client,
		logger: logger,
	}
}

// PublishDeviceOffline mempublikasikan event OLT offline ke queue.
// Buat correlation_id jika belum diset pada payload.
func (ep *oltEventPublisher) PublishDeviceOffline(ctx context.Context, payload domain.OLTDeviceOfflinePayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventOLTDeviceOffline, payload.TenantID, payload)
	return nil
}

// PublishDeviceOnline mempublikasikan event OLT online ke queue.
// Buat correlation_id jika belum diset pada payload.
func (ep *oltEventPublisher) PublishDeviceOnline(ctx context.Context, payload domain.OLTDeviceOnlinePayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventOLTDeviceOnline, payload.TenantID, payload)
	return nil
}

// PublishAlarm mempublikasikan event alarm OLT ke queue.
// Buat correlation_id jika belum diset pada payload.
func (ep *oltEventPublisher) PublishAlarm(ctx context.Context, payload domain.OLTAlarmPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventOLTAlarm, payload.TenantID, payload)
	return nil
}

// PublishONTProvisioned mempublikasikan event ONT berhasil di-provision ke queue.
func (ep *oltEventPublisher) PublishONTProvisioned(ctx context.Context, payload domain.ONTProvisionedPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventONTProvisioned, payload.TenantID, payload)
	return nil
}

// PublishONTDecommissioned mempublikasikan event ONT berhasil di-decommission ke queue.
func (ep *oltEventPublisher) PublishONTDecommissioned(ctx context.Context, payload domain.ONTDecommissionedPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventONTDecommissioned, payload.TenantID, payload)
	return nil
}

// PublishONTAutoProvisioned mempublikasikan event ONT berhasil di-auto-provision ke queue.
func (ep *oltEventPublisher) PublishONTAutoProvisioned(ctx context.Context, payload domain.ONTAutoProvisionedPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventONTAutoProvisioned, payload.TenantID, payload)
	return nil
}

// PublishONTAutoProvisionFailed mempublikasikan event auto-provisioning gagal ke queue.
func (ep *oltEventPublisher) PublishONTAutoProvisionFailed(ctx context.Context, payload domain.ONTAutoProvisionFailedPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventONTAutoProvisionFail, payload.TenantID, payload)
	return nil
}

// PublishONTPortMigrated mempublikasikan event port migration terdeteksi ke queue.
func (ep *oltEventPublisher) PublishONTPortMigrated(ctx context.Context, payload domain.ONTPortMigratedPayload) error {
	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.New().String()
	}
	ep.publish(domain.EventONTPortMigrated, payload.TenantID, payload)
	return nil
}

// terbitkan adalah helper internal untuk membuat TaskEnvelope dan mengirim ke queue.
// Best-effort: log error jika gagal, tidak kembalikan error ke caller.
func (ep *oltEventPublisher) publish(eventType, tenantID string, payload interface{}) {
	// Serialisasi payload ke JSON
	raw, err := json.Marshal(payload)
	if err != nil {
		ep.logger.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("gagal serialisasi olt event payload")
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
			Msg(fmt.Sprintf("gagal publish olt event ke queue: %s", eventType))
		return
	}

	ep.logger.Debug().
		Str("event_type", eventType).
		Str("tenant_id", tenantID).
		Msg("olt event berhasil dipublish ke queue")
}
