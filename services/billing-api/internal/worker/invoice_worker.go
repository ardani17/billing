// invoice_worker.go berisi asynq worker untuk job cron invoice.
// InvoiceWorker menangani dua jenis task:
// 1. invoice.generate_cron - cron harian untuk auto-buat invoice pelanggan
// 2. invoice.overdue_cron - cron harian untuk perbarui status invoice terlambat
package worker

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// Konstanta tipe task yang diproses oleh InvoiceWorker.
const (
	// TaskInvoiceGenerateCron adalah tipe task untuk cron auto-buat invoice harian.
	TaskInvoiceGenerateCron = "invoice.generate_cron"

	// TaskInvoiceOverdueCron adalah tipe task untuk cron perbarui status terlambat harian.
	TaskInvoiceOverdueCron = "invoice.overdue_cron"
)

// InvoiceWorker menangani task asynq terkait invoice cron jobs.
// Mendaftarkan handler untuk auto-buat dan terlambat perbarui.
type InvoiceWorker struct {
	cronUsecase *usecase.InvoiceCronUsecase
	logger      zerolog.Logger
}

// NewInvoiceWorker membuat instance baru InvoiceWorker.
func NewInvoiceWorker(
	cronUsecase *usecase.InvoiceCronUsecase,
	logger zerolog.Logger,
) *InvoiceWorker {
	return &InvoiceWorker{
		cronUsecase: cronUsecase,
		logger:      logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *InvoiceWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskInvoiceGenerateCron, w.handleGenerateCron)
	mux.HandleFunc(TaskInvoiceOverdueCron, w.handleOverdueCron)
}

// handleGenerateCron memproses task cron auto-buat invoice harian.
// Memanggil InvoiceCronUsecase.ProcessAutoGenerate untuk memproses
// semua tenant dan pelanggan yang eligible untuk invoice baru.
func (w *InvoiceWorker) handleGenerateCron(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron auto-generate invoice")

	if err := w.cronUsecase.ProcessAutoGenerate(ctx); err != nil {
		w.logger.Error().Err(err).Msg("gagal memproses auto-generate invoice")
		return fmt.Errorf("worker: gagal proses auto-generate invoice: %w", err)
	}

	w.logger.Info().Msg("selesai cron auto-generate invoice")
	return nil
}

// handleOverdueCron memproses task cron perbarui status terlambat harian.
// Memanggil InvoiceCronUsecase.ProcessOverdueUpdate untuk memperbarui
// status invoice yang sudah melewati tanggal jatuh tempo.
func (w *InvoiceWorker) handleOverdueCron(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron update overdue invoice")

	if err := w.cronUsecase.ProcessOverdueUpdate(ctx); err != nil {
		w.logger.Error().Err(err).Msg("gagal memproses overdue invoice")
		return fmt.Errorf("worker: gagal proses overdue invoice: %w", err)
	}

	w.logger.Info().Msg("selesai cron update overdue invoice")
	return nil
}
