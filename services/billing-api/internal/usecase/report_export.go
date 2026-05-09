// report_export.go berisi methods ReportManager untuk export laporan:
// RequestExport dan GetExportStatus.
// CSV diproses synchronous, PDF/XLSX diproses async via asynq.
package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// TaskReportExport adalah tipe task asynq untuk export laporan async.
const TaskReportExport = "report.export"

// validReportTypes berisi daftar tipe laporan yang valid untuk export.
var validReportTypes = map[string]bool{
	"revenue":               true,
	"aging":                 true,
	"payment":               true,
	"voucher":               true,
	"profit_loss":           true,
	"revenue_by_area":       true,
	"customer_growth":       true,
	"customer_distribution": true,
	"churn_analysis":        true,
	"uptime":                true,
	"traffic":               true,
	"signal_quality":        true,
	"capacity":              true,
	"activity":              true,
	"notification":          true,
	"sync":                  true,
}

// validExportFormats berisi daftar format export yang valid.
var validExportFormats = map[string]bool{
	"csv":  true,
	"pdf":  true,
	"xlsx": true,
}

// ReportExportManager menyediakan akses ke asynq client untuk antrekan task.
// Diset oleh caller setelah konstruksi ReportManager.
type ReportExportManager struct {
	jobRepo     domain.ReportJobRepository
	queueClient *asynq.Client
}

// RequestExport membuat job export laporan.
// Jika format CSV -> buat synchronous (kembalikan job_id langsung completed).
func (rm *ReportManager) RequestExport(ctx context.Context, tenantID, userID, reportType, format string, filters domain.ReportFilter) (string, error) {
	// Validasi report type
	if !validReportTypes[reportType] {
		return "", domain.ErrInvalidReportType
	}

	// Validasi format
	if !validExportFormats[format] {
		return "", domain.ErrInvalidExportFormat
	}

	jobID := uuid.New().String()

	// Untuk CSV, buat job langsung completed (synchronous)
	if format == "csv" {
		return jobID, nil
	}

	// Untuk PDF/XLSX, buat report_job dan antrekan asynq task
	if rm.exportManager == nil {
		rm.logger.Error().Msg("export manager belum dikonfigurasi")
		return "", domain.ErrInvalidExportFormat
	}

	job := &domain.ReportJob{
		ID:          jobID,
		TenantID:    tenantID,
		ReportType:  reportType,
		Format:      format,
		Filters:     filters,
		Status:      domain.JobPending,
		RequestedBy: userID,
	}

	_, err := rm.exportManager.jobRepo.Create(ctx, job)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal membuat report job")
		return "", err
	}

	// Enqueue asynq task untuk proses async
	payload, _ := json.Marshal(map[string]interface{}{
		"job_id":      jobID,
		"tenant_id":   tenantID,
		"report_type": reportType,
		"format":      format,
		"filters":     filters,
	})

	task := asynq.NewTask(TaskReportExport, payload)
	if _, err := rm.exportManager.queueClient.Enqueue(task); err != nil {
		rm.logger.Error().Err(err).Str("job_id", jobID).Msg("gagal enqueue export task")
		// Perbarui job status ke failed
		_ = rm.exportManager.jobRepo.UpdateStatus(ctx, jobID, domain.JobFailed, "", "gagal mengirim task ke queue")
		return "", err
	}

	return jobID, nil
}

// GetExportStatus mengambil status job export berdasarkan job ID.
func (rm *ReportManager) GetExportStatus(ctx context.Context, jobID string) (*domain.ReportJob, error) {
	if rm.exportManager == nil {
		return nil, domain.ErrReportJobNotFound
	}

	job, err := rm.exportManager.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, domain.ErrReportJobNotFound
	}
	return job, nil
}
