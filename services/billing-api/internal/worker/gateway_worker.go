// gateway_worker.go berisi asynq worker untuk task async payment gateway.
// GatewayWorker menangani lima jenis task:
// 1. gateway.generate_payment_link — generate payment link via gateway
// 2. gateway.process_webhook — proses webhook dari gateway secara async
// 3. gateway.expire_payment_links — cron expire payment links yang sudah lewat waktu
// 4. gateway.cleanup_webhook_logs — cron pembersihan webhook logs lama
// 5. gateway.sync_payment_link_amount — sinkronisasi jumlah payment link setelah invoice berubah
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// Konstanta tipe task yang diproses oleh GatewayWorker.
const (
	// TaskGeneratePaymentLink adalah tipe task untuk generate payment link via gateway.
	TaskGeneratePaymentLink = "gateway.generate_payment_link"

	// TaskProcessWebhook adalah tipe task untuk memproses webhook secara async.
	TaskProcessWebhook = "gateway.process_webhook"

	// TaskExpirePaymentLinks adalah tipe task cron untuk expire payment links.
	TaskExpirePaymentLinks = "gateway.expire_payment_links"

	// TaskCleanupWebhookLogs adalah tipe task cron untuk pembersihan webhook logs lama.
	TaskCleanupWebhookLogs = "gateway.cleanup_webhook_logs"

	// TaskSyncPaymentLinkAmount adalah tipe task untuk sinkronisasi jumlah payment link.
	TaskSyncPaymentLinkAmount = "gateway.sync_payment_link_amount"
)

// GatewayWorker menangani task asynq terkait payment gateway.
// Mendaftarkan handler untuk generate link, proses webhook, expire, cleanup, dan sync.
type GatewayWorker struct {
	gatewayUsecase *usecase.GatewayUsecase
	webhookUsecase *usecase.WebhookUsecase
	linkRepo       domain.PaymentLinkRepository
	webhookRepo    domain.WebhookLogRepository
	retentionDays  int // Jumlah hari retensi webhook logs (default 90)
	logger         zerolog.Logger
}

// NewGatewayWorker membuat instance baru GatewayWorker.
func NewGatewayWorker(
	gatewayUsecase *usecase.GatewayUsecase,
	webhookUsecase *usecase.WebhookUsecase,
	linkRepo domain.PaymentLinkRepository,
	webhookRepo domain.WebhookLogRepository,
	retentionDays int,
	logger zerolog.Logger,
) *GatewayWorker {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	return &GatewayWorker{
		gatewayUsecase: gatewayUsecase,
		webhookUsecase: webhookUsecase,
		linkRepo:       linkRepo,
		webhookRepo:    webhookRepo,
		retentionDays:  retentionDays,
		logger:         logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *GatewayWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskGeneratePaymentLink, w.handleGeneratePaymentLink)
	mux.HandleFunc(TaskProcessWebhook, w.handleProcessWebhook)
	mux.HandleFunc(TaskExpirePaymentLinks, w.handleExpirePaymentLinks)
	mux.HandleFunc(TaskCleanupWebhookLogs, w.handleCleanupWebhookLogs)
	mux.HandleFunc(TaskSyncPaymentLinkAmount, w.handleSyncPaymentLinkAmount)
}

// handleGeneratePaymentLink memproses task generate payment link.
// Deserialize GeneratePaymentLinkRequest dari payload, panggil usecase.
func (w *GatewayWorker) handleGeneratePaymentLink(ctx context.Context, task *asynq.Task) error {
	var req domain.GeneratePaymentLinkRequest
	if err := json.Unmarshal(task.Payload(), &req); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload generate payment link")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().
		Str("tenant_id", req.TenantID).
		Str("customer_id", req.CustomerID).
		Msg("memproses task generate payment link")

	if _, err := w.gatewayUsecase.GeneratePaymentLink(ctx, req); err != nil {
		w.logger.Error().Err(err).
			Str("customer_id", req.CustomerID).
			Msg("gagal generate payment link")
		return fmt.Errorf("worker: gagal generate payment link: %w", err)
	}

	w.logger.Info().Str("customer_id", req.CustomerID).Msg("selesai generate payment link")
	return nil
}

// handleProcessWebhook memproses task webhook secara async.
// Deserialize webhookLogID dari payload, panggil webhookUsecase.ProcessWebhook.
func (w *GatewayWorker) handleProcessWebhook(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		WebhookLogID string `json:"webhook_log_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload process webhook")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().Str("webhook_log_id", payload.WebhookLogID).Msg("memproses task webhook")

	if err := w.webhookUsecase.ProcessWebhook(ctx, payload.WebhookLogID); err != nil {
		w.logger.Error().Err(err).
			Str("webhook_log_id", payload.WebhookLogID).
			Msg("gagal memproses webhook")
		return fmt.Errorf("worker: gagal proses webhook: %w", err)
	}

	w.logger.Info().Str("webhook_log_id", payload.WebhookLogID).Msg("selesai memproses webhook")
	return nil
}

// handleExpirePaymentLinks memproses task cron expire payment links.
// Ambil batch payment links yang sudah expired, expire satu per satu.
func (w *GatewayWorker) handleExpirePaymentLinks(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron expire payment links")

	const batchSize = 100
	expired, err := w.linkRepo.FindExpired(ctx, batchSize)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal mengambil payment links expired")
		return fmt.Errorf("worker: gagal ambil expired links: %w", err)
	}

	var expiredCount int
	for _, link := range expired {
		if err := w.linkRepo.ExpireByID(ctx, link.ID); err != nil {
			w.logger.Warn().Err(err).Str("link_id", link.ID).Msg("gagal expire payment link")
			continue
		}
		expiredCount++
	}

	w.logger.Info().
		Int("total_found", len(expired)).
		Int("total_expired", expiredCount).
		Msg("selesai cron expire payment links")
	return nil
}

// handleCleanupWebhookLogs memproses task cron pembersihan webhook logs lama.
// Hapus logs yang lebih tua dari retentionDays (default 90 hari).
func (w *GatewayWorker) handleCleanupWebhookLogs(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Int("retention_days", w.retentionDays).Msg("memulai cron cleanup webhook logs")

	cutoff := time.Now().AddDate(0, 0, -w.retentionDays)
	deleted, err := w.webhookRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal menghapus webhook logs lama")
		return fmt.Errorf("worker: gagal cleanup webhook logs: %w", err)
	}

	w.logger.Info().Int64("deleted_count", deleted).Msg("selesai cron cleanup webhook logs")
	return nil
}

// handleSyncPaymentLinkAmount memproses task sinkronisasi jumlah payment link.
// Deserialize invoiceID dari payload, panggil gatewayUsecase.SyncPaymentLinkAmount.
func (w *GatewayWorker) handleSyncPaymentLinkAmount(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		InvoiceID string `json:"invoice_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload sync payment link amount")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().Str("invoice_id", payload.InvoiceID).Msg("memproses task sync payment link amount")

	if err := w.gatewayUsecase.SyncPaymentLinkAmount(ctx, payload.InvoiceID); err != nil {
		w.logger.Error().Err(err).
			Str("invoice_id", payload.InvoiceID).
			Msg("gagal sync payment link amount")
		return fmt.Errorf("worker: gagal sync payment link amount: %w", err)
	}

	w.logger.Info().Str("invoice_id", payload.InvoiceID).Msg("selesai sync payment link amount")
	return nil
}
