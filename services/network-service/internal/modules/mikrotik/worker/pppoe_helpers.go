// pppoe_helpers.go berisi helper functions untuk PPPoEEventWorker.
// Termasuk decode payload dan retry/permanent failure handling.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// decodePayload mendekode TaskEnvelope dari asynq.Task dan unmarshal payload.
// Mengembalikan envelope untuk akses correlation_id dan metadata lainnya.
func (w *PPPoEEventWorker) decodePayload(task *asynq.Task, dest interface{}) (*queue.TaskEnvelope, error) {
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		w.logger.Error().Err(err).
			Str("task_type", task.Type()).
			Msg("gagal decode envelope")
		return nil, fmt.Errorf("worker: gagal decode envelope: %w", err)
	}

	if err := json.Unmarshal(envelope.Payload, dest); err != nil {
		w.logger.Error().Err(err).
			Str("task_type", task.Type()).
			Msg("gagal unmarshal payload")
		return nil, fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	return envelope, nil
}

func (w *PPPoEEventWorker) canProcessMikroTik(ctx context.Context, tenantID, eventType string) (bool, error) {
	if w.moduleChecker == nil {
		return true, nil
	}

	enabled, err := w.moduleChecker.IsEnabled(ctx, tenantID, domain.ModuleMikroTik)
	if err != nil {
		w.logger.Error().Err(err).Str("tenant_id", tenantID).Str("event_type", eventType).Msg("gagal memeriksa modul mikrotik")
		return false, err
	}
	if !enabled {
		w.logger.Info().Str("tenant_id", tenantID).Str("event_type", eventType).Msg("skip event jaringan: modul mikrotik nonaktif")
		return false, nil
	}
	return true, nil
}

// handleRetryOrFail memeriksa apakah ini retry terakhir.
// Jika sudah mencapai maxRetries, terbitkan mikrotik.sync_failed event.
// Jika belum, kembalikan error agar asynq melakukan retry.
func (w *PPPoEEventWorker) handleRetryOrFail(
	ctx context.Context,
	envelope *queue.TaskEnvelope,
	operation, customerID, routerID, tenantID string,
	execErr error,
) error {
	retried, _ := asynq.GetRetryCount(ctx)
	w.logger.Error().Err(execErr).
		Str("operation", operation).
		Str("customer_id", customerID).
		Str("router_id", routerID).
		Int("retried", retried).
		Int("max_retries", maxRetries).
		Msg("gagal eksekusi operasi PPPoE")

	// Jika sudah mencapai retry terakhir, terbitkan sync_failed event
	if retried >= maxRetries-1 {
		w.logger.Error().
			Str("operation", operation).
			Str("customer_id", customerID).
			Str("router_id", routerID).
			Msg("semua retry gagal, tandai sebagai failed_permanent")

		syncFailed := domain.SyncFailedPayload{
			RouterID:     routerID,
			TenantID:     tenantID,
			Operation:    operation,
			ErrorMessage: execErr.Error(),
			FailedAt:     time.Now(),
		}

		if pubErr := w.eventPub.PublishSyncFailed(ctx, syncFailed); pubErr != nil {
			w.logger.Error().Err(pubErr).
				Msg("gagal publish mikrotik.sync_failed event")
		}
	}

	return fmt.Errorf("worker: gagal %s untuk customer %s: %w", operation, customerID, execErr)
}
