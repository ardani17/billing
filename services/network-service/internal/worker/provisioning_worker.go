// Package worker berisi asynq worker untuk memproses event dari service lain.
// ProvisioningEventWorker menangani event "customer.terminated" untuk
// auto-decommission ONT pelanggan yang diterminasi.
// Pattern sama dengan PPPoEEventWorker yang sudah ada.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// provisioningMaxRetries adalah jumlah maksimal retry sebelum menyerah.
const provisioningMaxRetries = 5

// ProvisioningRetryDelays adalah jadwal delay retry dengan exponential backoff.
// Konsisten dengan PPPoERetryDelays: 30s, 1m, 2m, 5m, 10m.
var ProvisioningRetryDelays = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	300 * time.Second,
	600 * time.Second,
}

// ProvisioningEventWorker memproses event provisioning dari service lain via asynq.
// Saat ini menangani "customer.terminated" untuk auto-decommission ONT.
type ProvisioningEventWorker struct {
	manager domain.ProvisioningManager
	logger  zerolog.Logger
}

// NewProvisioningEventWorker membuat instance baru ProvisioningEventWorker.
func NewProvisioningEventWorker(
	manager domain.ProvisioningManager,
	logger zerolog.Logger,
) *ProvisioningEventWorker {
	return &ProvisioningEventWorker{
		manager: manager,
		logger:  logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
// - "customer.terminated" → handleCustomerTerminated
func (w *ProvisioningEventWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(EventCustomerTerminated, w.handleCustomerTerminated)
}

// ProvisioningRetryDelay menghitung delay retry berdasarkan nomor percobaan.
// Digunakan sebagai asynq.RetryDelayFunc untuk provisioning worker.
func ProvisioningRetryDelay(n int, err error, task *asynq.Task) time.Duration {
	if n < len(ProvisioningRetryDelays) {
		return ProvisioningRetryDelays[n]
	}
	return ProvisioningRetryDelays[len(ProvisioningRetryDelays)-1]
}

// handleCustomerTerminated memproses event customer.terminated.
// Decode TaskEnvelope, ambil customer_id dan tenant_id dari payload,
// lalu delegate ke ProvisioningManager.HandleCustomerTerminated.
func (w *ProvisioningEventWorker) handleCustomerTerminated(
	ctx context.Context,
	task *asynq.Task,
) error {
	// Decode TaskEnvelope dari asynq task
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		w.logger.Error().Err(err).
			Str("task_type", task.Type()).
			Msg("gagal decode envelope")
		return fmt.Errorf("worker: gagal decode envelope: %w", err)
	}

	// Unmarshal payload untuk mendapatkan customer_id dan tenant_id
	var payload domain.CustomerTerminatedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		w.logger.Error().Err(err).
			Str("task_type", task.Type()).
			Msg("gagal unmarshal payload customer.terminated")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	// Validasi field wajib
	if payload.CustomerID == "" {
		w.logger.Error().Msg("payload customer.terminated: customer_id kosong")
		return fmt.Errorf("worker: payload customer.terminated: customer_id kosong")
	}

	// Gunakan tenant_id dari envelope jika payload tidak punya
	tenantID := payload.TenantID
	if tenantID == "" {
		tenantID = envelope.TenantID
	}

	if tenantID == "" {
		w.logger.Error().Msg("payload customer.terminated: tenant_id kosong")
		return fmt.Errorf("worker: payload customer.terminated: tenant_id kosong")
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("tenant_id", tenantID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses customer.terminated untuk decommission ONT")

	// Delegate ke ProvisioningManager
	if err := w.manager.HandleCustomerTerminated(ctx, payload.CustomerID, tenantID); err != nil {
		retried, _ := asynq.GetRetryCount(ctx)
		w.logger.Error().Err(err).
			Str("customer_id", payload.CustomerID).
			Str("tenant_id", tenantID).
			Int("retried", retried).
			Int("max_retries", provisioningMaxRetries).
			Msg("gagal decommission ONT untuk customer terminated")

		if retried >= provisioningMaxRetries-1 {
			w.logger.Error().
				Str("customer_id", payload.CustomerID).
				Str("tenant_id", tenantID).
				Msg("semua retry gagal untuk decommission ONT")
		}

		return fmt.Errorf(
			"worker: gagal decommission ONT untuk customer %s: %w",
			payload.CustomerID, err,
		)
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("tenant_id", tenantID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("berhasil decommission ONT untuk customer terminated")

	return nil
}
