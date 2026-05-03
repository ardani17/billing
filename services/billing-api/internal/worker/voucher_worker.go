// Package worker berisi asynq worker untuk memproses task async.
// VoucherWorker menangani dua jenis task:
// 1. voucher.async_generate — generate voucher dalam jumlah besar (>500) secara async
// 2. voucher.expiry_cron — cron harian untuk memproses voucher expired
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

// Konstanta tipe task yang diproses oleh VoucherWorker.
const (
	// TaskAsyncGenerate adalah tipe task untuk generate voucher secara async.
	TaskAsyncGenerate = "voucher.async_generate"

	// TaskExpiryCron adalah tipe task untuk cron expiry voucher harian.
	TaskExpiryCron = "voucher.expiry_cron"
)

// VoucherWorker menangani task asynq terkait voucher.
// Mendaftarkan handler untuk async generate dan expiry cron.
type VoucherWorker struct {
	voucherUsecase *usecase.VoucherUsecase
	expiryUsecase  *usecase.VoucherExpiryUsecase
	logger         zerolog.Logger
}

// NewVoucherWorker membuat instance baru VoucherWorker.
func NewVoucherWorker(
	voucherUsecase *usecase.VoucherUsecase,
	expiryUsecase *usecase.VoucherExpiryUsecase,
	logger zerolog.Logger,
) *VoucherWorker {
	return &VoucherWorker{
		voucherUsecase: voucherUsecase,
		expiryUsecase:  expiryUsecase,
		logger:         logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *VoucherWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskAsyncGenerate, w.handleAsyncGenerate)
	mux.HandleFunc(TaskExpiryCron, w.handleExpiryCron)
}

// handleAsyncGenerate memproses task generate voucher secara async.
// Payload di-deserialize dari format TaskEnvelope, lalu memanggil
// VoucherUsecase.Generate dengan data dari envelope.
func (w *VoucherWorker) handleAsyncGenerate(ctx context.Context, task *asynq.Task) error {
	// Decode envelope dari payload task
	envelope, err := queue.DecodeEnvelope(task)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal decode envelope async generate")
		return fmt.Errorf("worker: gagal decode envelope: %w", err)
	}

	w.logger.Info().
		Str("tenant_id", envelope.TenantID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses task async generate voucher")

	// Deserialize payload menjadi parameter generate
	var payload asyncGeneratePayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload async generate")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	// Bangun request dan actor info dari payload
	req := domain.GenerateVoucherRequest{
		PackageID:  payload.PackageID,
		Quantity:   payload.Quantity,
		CodeFormat: payload.CodeFormat,
		CodeLength: payload.CodeLength,
		Prefix:     payload.Prefix,
	}

	actor := domain.ActorInfo{
		ActorID:   payload.ActorID,
		ActorName: payload.ActorName,
	}

	// Panggil usecase untuk generate voucher
	result, err := w.voucherUsecase.Generate(ctx, envelope.TenantID, req, actor)
	if err != nil {
		w.logger.Error().Err(err).
			Str("tenant_id", envelope.TenantID).
			Str("package_id", payload.PackageID).
			Int("quantity", payload.Quantity).
			Msg("gagal generate voucher async")
		return fmt.Errorf("worker: gagal generate voucher: %w", err)
	}

	w.logger.Info().
		Str("tenant_id", envelope.TenantID).
		Int("total_generated", result.TotalGenerated).
		Int("total_failed", result.TotalFailed).
		Msg("selesai generate voucher async")

	return nil
}

// handleExpiryCron memproses task cron expiry voucher harian.
// Memanggil VoucherExpiryUsecase.ProcessExpiredVouchers untuk memproses
// semua voucher terjual yang sudah melewati expires_at.
func (w *VoucherWorker) handleExpiryCron(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron expiry voucher")

	if err := w.expiryUsecase.ProcessExpiredVouchers(ctx); err != nil {
		w.logger.Error().Err(err).Msg("gagal memproses voucher expired")
		return fmt.Errorf("worker: gagal proses expiry: %w", err)
	}

	w.logger.Info().Msg("selesai cron expiry voucher")
	return nil
}

// asyncGeneratePayload adalah struktur payload untuk task async generate voucher.
// Sesuai dengan format yang dikirim oleh VoucherUsecase.enqueueAsyncGenerate.
type asyncGeneratePayload struct {
	PackageID  string `json:"package_id"`
	Quantity   int    `json:"quantity"`
	CodeFormat string `json:"code_format"`
	CodeLength int    `json:"code_length"`
	Prefix     string `json:"prefix"`
	ActorID    string `json:"actor_id"`
	ActorName  string `json:"actor_name"`
}
