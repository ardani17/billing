// recurring_expense_worker.go berisi asynq worker untuk pengeluaran berulang.
// RecurringExpenseWorker menangani task expense.recurring — query pengeluaran
// berulang yang jatuh pada hari ini dan membuat record pengeluaran baru.
package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Konstanta tipe task yang diproses oleh RecurringExpenseWorker.
const (
	// TaskRecurringExpense adalah tipe task cron untuk auto-create pengeluaran berulang.
	TaskRecurringExpense = "expense.recurring"
)

// RecurringExpenseWorker menangani task asynq untuk pengeluaran berulang.
// Mendaftarkan handler untuk task expense.recurring.
type RecurringExpenseWorker struct {
	expenseRepo domain.ExpenseRepository
	logger      zerolog.Logger
}

// NewRecurringExpenseWorker membuat instance baru RecurringExpenseWorker.
func NewRecurringExpenseWorker(
	expenseRepo domain.ExpenseRepository,
	logger zerolog.Logger,
) *RecurringExpenseWorker {
	return &RecurringExpenseWorker{
		expenseRepo: expenseRepo,
		logger:      logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *RecurringExpenseWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskRecurringExpense, w.handleRecurringExpense)
}

// handleRecurringExpense memproses task cron pengeluaran berulang.
// Alur: query semua expense dengan is_recurring=true → filter yang recurring_day
// sama dengan hari ini → buat record pengeluaran baru untuk masing-masing.
func (w *RecurringExpenseWorker) handleRecurringExpense(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron pengeluaran berulang")

	today := time.Now()
	currentDay := today.Day()

	// Ambil semua pengeluaran berulang yang aktif
	recurring, err := w.expenseRepo.ListRecurring(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal mengambil pengeluaran berulang")
		return fmt.Errorf("worker: gagal ambil pengeluaran berulang: %w", err)
	}

	var createdCount, skippedCount, failedCount int
	for _, expense := range recurring {
		// Lewati jika recurring_day tidak sesuai hari ini
		if expense.RecurringDay == nil || *expense.RecurringDay != currentDay {
			skippedCount++
			continue
		}

		// Buat record pengeluaran baru berdasarkan template recurring
		newExpense := &domain.Expense{
			ID:          uuid.New().String(),
			TenantID:    expense.TenantID,
			CategoryID:  expense.CategoryID,
			Amount:      expense.Amount,
			Description: expense.Description,
			ExpenseDate: today,
			IsRecurring: false, // Record baru bukan recurring
			CreatedByID: expense.CreatedByID,
		}

		if _, err := w.expenseRepo.Create(ctx, newExpense); err != nil {
			w.logger.Error().Err(err).
				Str("expense_id", expense.ID).
				Str("tenant_id", expense.TenantID).
				Msg("gagal membuat pengeluaran berulang")
			failedCount++
			continue
		}
		createdCount++
	}

	w.logger.Info().
		Int("total_recurring", len(recurring)).
		Int("created", createdCount).
		Int("skipped", skippedCount).
		Int("failed", failedCount).
		Msg("selesai cron pengeluaran berulang")
	return nil
}
