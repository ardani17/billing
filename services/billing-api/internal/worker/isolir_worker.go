// isolir_worker.go berisi asynq worker untuk task isolir, un-isolir, suspend, dan sync.
// IsolirWorker menangani enam jenis task:
// 1. isolir.auto_isolir_cron — cron harian auto-isolir pelanggan dengan invoice terlambat
// 2. isolir.suspend_cron — cron harian suspend pelanggan yang melewati batas toleransi
// 3. isolir.periodic_sync — periodic sync pending_syncs setiap 15 menit
// 4. payment.online.received — buka isolir setelah pembayaran online diterima
// 5. payment.recorded — buka isolir setelah pembayaran manual dicatat
// 6. payment.voided.re_isolir — re-isolir setelah pembayaran di-void
package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// IsolirWorker menangani task asynq terkait modul isolir.
// Mendaftarkan handler untuk cron isolir/suspend, periodic sync, dan event pembayaran.
type IsolirWorker struct {
	isolirUsecase *usecase.IsolirUsecase
	logger        zerolog.Logger
}

// NewIsolirWorker membuat instance baru IsolirWorker.
func NewIsolirWorker(isolirUsecase *usecase.IsolirUsecase, logger zerolog.Logger) *IsolirWorker {
	return &IsolirWorker{
		isolirUsecase: isolirUsecase,
		logger:        logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *IsolirWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(domain.TaskAutoIsolirCron, w.handleAutoIsolirCron)
	mux.HandleFunc(domain.TaskSuspendCron, w.handleSuspendCron)
	mux.HandleFunc(domain.TaskPeriodicSync, w.handlePeriodicSync)
	mux.HandleFunc(domain.TaskPaymentOnlineReceived, w.handlePaymentOnlineReceived)
	mux.HandleFunc(domain.TaskPaymentRecorded, w.handlePaymentRecorded)
	mux.HandleFunc(domain.TaskPaymentVoidedReIsolir, w.handlePaymentVoidedReIsolir)
}

// handleAutoIsolirCron memproses cron harian auto-isolir pelanggan.
// Memanggil IsolirUsecase.ProcessAutoIsolir untuk memproses semua tenant.
func (w *IsolirWorker) handleAutoIsolirCron(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron auto-isolir")

	if err := w.isolirUsecase.ProcessAutoIsolir(ctx); err != nil {
		w.logger.Error().Err(err).Msg("gagal memproses auto-isolir")
		return fmt.Errorf("worker: gagal proses auto-isolir: %w", err)
	}

	w.logger.Info().Msg("selesai cron auto-isolir")
	return nil
}

// handleSuspendCron memproses cron harian suspend pelanggan.
// Memanggil IsolirUsecase.ProcessSuspend untuk memproses semua tenant.
func (w *IsolirWorker) handleSuspendCron(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron suspend")

	if err := w.isolirUsecase.ProcessSuspend(ctx); err != nil {
		w.logger.Error().Err(err).Msg("gagal memproses suspend")
		return fmt.Errorf("worker: gagal proses suspend: %w", err)
	}

	w.logger.Info().Msg("selesai cron suspend")
	return nil
}

// handlePeriodicSync memproses periodic sync pending_syncs.
// Memanggil IsolirUsecase.ProcessPeriodicSync untuk retry sinkronisasi router.
func (w *IsolirWorker) handlePeriodicSync(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai periodic sync")

	if err := w.isolirUsecase.ProcessPeriodicSync(ctx); err != nil {
		w.logger.Error().Err(err).Msg("gagal memproses periodic sync")
		return fmt.Errorf("worker: gagal proses periodic sync: %w", err)
	}

	w.logger.Info().Msg("selesai periodic sync")
	return nil
}

// paymentEventPayload adalah struct internal untuk deserialize payload event pembayaran.
type paymentEventPayload struct {
	TenantID   string `json:"tenant_id"`
	CustomerID string `json:"customer_id"`
}

// handlePaymentOnlineReceived memproses event pembayaran online diterima.
// Deserialize payload dari TaskEnvelope, panggil ProcessUnIsolir.
// Catatan: event ini juga diproses oleh GatewayWorker untuk tujuan berbeda.
func (w *IsolirWorker) handlePaymentOnlineReceived(ctx context.Context, task *asynq.Task) error {
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal decode envelope payment.online.received")
		return fmt.Errorf("worker: gagal decode envelope: %w", err)
	}

	var payload paymentEventPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload payment.online.received")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().
		Str("tenant_id", payload.TenantID).
		Str("customer_id", payload.CustomerID).
		Msg("memproses payment.online.received untuk un-isolir")

	if err := w.isolirUsecase.ProcessUnIsolir(ctx, payload.TenantID, payload.CustomerID, "payment_received"); err != nil {
		w.logger.Error().Err(err).
			Str("customer_id", payload.CustomerID).
			Msg("gagal memproses un-isolir dari payment online")
		return fmt.Errorf("worker: gagal proses un-isolir: %w", err)
	}

	return nil
}

// handlePaymentRecorded memproses event pembayaran manual dicatat.
// Deserialize payload dari TaskEnvelope, panggil ProcessUnIsolir.
func (w *IsolirWorker) handlePaymentRecorded(ctx context.Context, task *asynq.Task) error {
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal decode envelope payment.recorded")
		return fmt.Errorf("worker: gagal decode envelope: %w", err)
	}

	var payload paymentEventPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload payment.recorded")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().
		Str("tenant_id", payload.TenantID).
		Str("customer_id", payload.CustomerID).
		Msg("memproses payment.recorded untuk un-isolir")

	if err := w.isolirUsecase.ProcessUnIsolir(ctx, payload.TenantID, payload.CustomerID, "payment_received"); err != nil {
		w.logger.Error().Err(err).
			Str("customer_id", payload.CustomerID).
			Msg("gagal memproses un-isolir dari payment recorded")
		return fmt.Errorf("worker: gagal proses un-isolir: %w", err)
	}

	return nil
}

// handlePaymentVoidedReIsolir memproses event void pembayaran untuk re-isolir.
// Deserialize payload dari TaskEnvelope, panggil ProcessReIsolir.
func (w *IsolirWorker) handlePaymentVoidedReIsolir(ctx context.Context, task *asynq.Task) error {
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal decode envelope payment.voided.re_isolir")
		return fmt.Errorf("worker: gagal decode envelope: %w", err)
	}

	var payload paymentEventPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload payment.voided.re_isolir")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().
		Str("tenant_id", payload.TenantID).
		Str("customer_id", payload.CustomerID).
		Msg("memproses payment.voided.re_isolir untuk re-isolir")

	if err := w.isolirUsecase.ProcessReIsolir(ctx, payload.TenantID, payload.CustomerID); err != nil {
		w.logger.Error().Err(err).
			Str("customer_id", payload.CustomerID).
			Msg("gagal memproses re-isolir dari payment voided")
		return fmt.Errorf("worker: gagal proses re-isolir: %w", err)
	}

	return nil
}
