// schedule_worker.go berisi asynq worker untuk jadwal laporan otomatis.
// ScheduleWorker menangani tiga jenis task:
// 1. report.scheduled — generate dan kirim laporan terjadwal
// 2. report.cleanup_jobs — bersihkan report jobs lama
// 3. report.cleanup_files — bersihkan file export lama
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// Konstanta tipe task yang diproses oleh ScheduleWorker.
const (
	// TaskScheduledReport adalah tipe task untuk generate laporan terjadwal.
	TaskScheduledReport = "report.scheduled"

	// TaskCleanupReportJobs adalah tipe task cron untuk bersihkan report jobs lama.
	TaskCleanupReportJobs = "report.cleanup_jobs"

	// TaskCleanupScheduledFiles adalah tipe task cron untuk bersihkan file export lama.
	TaskCleanupScheduledFiles = "report.cleanup_files"
)

// scheduledPayload adalah struktur payload untuk task laporan terjadwal.
type scheduledPayload struct {
	ScheduleType string `json:"schedule_type"`
}

// ScheduleWorker menangani task asynq terkait jadwal laporan otomatis.
// Mendaftarkan handler untuk scheduled report, cleanup jobs, dan cleanup files.
type ScheduleWorker struct {
	scheduleRepo  domain.ReportScheduleRepository
	reportManager *usecase.ReportManager
	jobRepo       domain.ReportJobRepository
	queueClient   *asynq.Client
	logger        zerolog.Logger
}

// NewScheduleWorker membuat instance baru ScheduleWorker.
func NewScheduleWorker(
	scheduleRepo domain.ReportScheduleRepository,
	reportManager *usecase.ReportManager,
	jobRepo domain.ReportJobRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *ScheduleWorker {
	return &ScheduleWorker{
		scheduleRepo:  scheduleRepo,
		reportManager: reportManager,
		jobRepo:       jobRepo,
		queueClient:   queueClient,
		logger:        logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *ScheduleWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskScheduledReport, w.handleScheduledReport)
	mux.HandleFunc(TaskCleanupReportJobs, w.handleCleanupReportJobs)
	mux.HandleFunc(TaskCleanupScheduledFiles, w.handleCleanupScheduledFiles)
}

// handleScheduledReport memproses task laporan terjadwal.
// Alur: decode schedule_type → query jadwal yang due → untuk setiap jadwal:
// enqueue task export untuk generate laporan.
func (w *ScheduleWorker) handleScheduledReport(ctx context.Context, task *asynq.Task) error {
	var payload scheduledPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload scheduled report")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	scheduleType := domain.ScheduleType(payload.ScheduleType)
	w.logger.Info().Str("schedule_type", payload.ScheduleType).Msg("memproses jadwal laporan")

	// Query jadwal yang perlu dijalankan
	schedules, err := w.scheduleRepo.ListDue(ctx, scheduleType)
	if err != nil {
		w.logger.Error().Err(err).Msg("gagal mengambil jadwal due")
		return fmt.Errorf("worker: gagal ambil jadwal due: %w", err)
	}

	if len(schedules) == 0 {
		w.logger.Info().Str("schedule_type", payload.ScheduleType).Msg("tidak ada jadwal yang perlu dijalankan")
		return nil
	}

	var successCount, failCount int
	for _, schedule := range schedules {
		if err := w.processSchedule(ctx, schedule); err != nil {
			w.logger.Error().Err(err).
				Str("schedule_id", schedule.ID).
				Str("tenant_id", schedule.TenantID).
				Msg("gagal memproses jadwal")
			failCount++
			continue
		}
		successCount++
	}

	w.logger.Info().
		Int("success", successCount).
		Int("failed", failCount).
		Int("total", len(schedules)).
		Msg("selesai memproses jadwal laporan")
	return nil
}

// processSchedule memproses satu jadwal laporan — enqueue task export.
func (w *ScheduleWorker) processSchedule(ctx context.Context, schedule *domain.ReportSchedule) error {
	// Bangun filter periode berdasarkan schedule type
	now := time.Now()
	filter := schedule.Filters
	filter.PeriodEnd = now

	switch schedule.ScheduleType {
	case domain.ScheduleDaily:
		filter.PeriodStart = now.AddDate(0, 0, -1)
	case domain.ScheduleWeekly:
		filter.PeriodStart = now.AddDate(0, 0, -7)
	case domain.ScheduleMonthly:
		filter.PeriodStart = now.AddDate(0, -1, 0)
	}

	// Enqueue task export untuk generate laporan
	exportPayload, _ := json.Marshal(map[string]interface{}{
		"job_id":      schedule.ID,
		"tenant_id":   schedule.TenantID,
		"report_type": schedule.ReportType,
		"format":      schedule.Format,
		"filters":     filter,
	})

	exportTask := asynq.NewTask(TaskReportExport, exportPayload)
	if _, err := w.queueClient.EnqueueContext(ctx, exportTask); err != nil {
		return fmt.Errorf("gagal enqueue export task: %w", err)
	}

	w.logger.Info().
		Str("schedule_id", schedule.ID).
		Str("report_type", schedule.ReportType).
		Msg("berhasil enqueue export untuk jadwal")
	return nil
}

// handleCleanupReportJobs memproses task cron pembersihan report jobs lama.
// Menghapus jobs yang lebih tua dari 30 hari.
func (w *ScheduleWorker) handleCleanupReportJobs(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron cleanup report jobs")

	cutoff := time.Now().AddDate(0, 0, -30)
	if err := w.jobRepo.CleanupOld(ctx, cutoff); err != nil {
		w.logger.Error().Err(err).Msg("gagal menghapus report jobs lama")
		return fmt.Errorf("worker: gagal cleanup report jobs: %w", err)
	}

	w.logger.Info().Msg("selesai cron cleanup report jobs")
	return nil
}

// handleCleanupScheduledFiles memproses task cron pembersihan file export lama.
// Menghapus file di direktori exports yang lebih tua dari 7 hari.
func (w *ScheduleWorker) handleCleanupScheduledFiles(ctx context.Context, task *asynq.Task) error {
	w.logger.Info().Msg("memulai cron cleanup file export")

	exportDir := "exports"
	cutoff := time.Now().AddDate(0, 0, -7)
	var deletedCount int

	err := filepath.Walk(exportDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if removeErr := os.Remove(path); removeErr != nil {
				w.logger.Warn().Err(removeErr).Str("path", path).Msg("gagal hapus file export lama")
				return nil
			}
			deletedCount++
		}
		return nil
	})
	if err != nil {
		w.logger.Warn().Err(err).Msg("gagal walk direktori exports")
	}

	w.logger.Info().Int("deleted_count", deletedCount).Msg("selesai cron cleanup file export")
	return nil
}
