// export_worker.go berisi asynq worker untuk export laporan async.
// ExportWorker menangani task report.export — dequeue task, generate
// laporan dalam format PDF/XLSX, simpan file, dan update status job.
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
	"github.com/xuri/excelize/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// Konstanta tipe task yang diproses oleh ExportWorker.
const (
	// TaskReportExport adalah tipe task untuk export laporan async (PDF/XLSX).
	TaskReportExport = "report.export"
)

// exportPayload adalah struktur payload untuk task export laporan.
type exportPayload struct {
	JobID      string             `json:"job_id"`
	TenantID   string             `json:"tenant_id"`
	ReportType string             `json:"report_type"`
	Format     string             `json:"format"`
	Filters    domain.ReportFilter `json:"filters"`
}

// ExportWorker menangani task asynq untuk export laporan async.
// Mendaftarkan handler untuk task report.export.
type ExportWorker struct {
	reportManager *usecase.ReportManager
	jobRepo       domain.ReportJobRepository
	logger        zerolog.Logger
}

// NewExportWorker membuat instance baru ExportWorker.
func NewExportWorker(
	reportManager *usecase.ReportManager,
	jobRepo domain.ReportJobRepository,
	logger zerolog.Logger,
) *ExportWorker {
	return &ExportWorker{
		reportManager: reportManager,
		jobRepo:       jobRepo,
		logger:        logger,
	}
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *ExportWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskReportExport, w.handleExportTask)
}

// handleExportTask memproses task export laporan async.
// Alur: decode payload → update status processing → generate report →
// simpan file → update status completed + download_url.
// Jika gagal: update status failed + pesan error.
func (w *ExportWorker) handleExportTask(ctx context.Context, task *asynq.Task) error {
	var payload exportPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("gagal unmarshal payload export task")
		return fmt.Errorf("worker: gagal unmarshal payload: %w", err)
	}

	w.logger.Info().
		Str("job_id", payload.JobID).
		Str("tenant_id", payload.TenantID).
		Str("report_type", payload.ReportType).
		Str("format", payload.Format).
		Msg("memproses task export laporan")

	// Update status ke processing
	if err := w.jobRepo.UpdateStatus(ctx, payload.JobID, domain.JobProcessing, "", ""); err != nil {
		w.logger.Error().Err(err).Str("job_id", payload.JobID).Msg("gagal update status processing")
	}

	// Generate report data dan simpan file
	downloadURL, err := w.generateAndSave(ctx, payload)
	if err != nil {
		w.logger.Error().Err(err).
			Str("job_id", payload.JobID).
			Msg("gagal generate export laporan")
		// Update status ke failed
		_ = w.jobRepo.UpdateStatus(ctx, payload.JobID, domain.JobFailed, "", err.Error())
		return fmt.Errorf("worker: gagal export laporan: %w", err)
	}

	// Update status ke completed dengan download URL
	if err := w.jobRepo.UpdateStatus(ctx, payload.JobID, domain.JobCompleted, downloadURL, ""); err != nil {
		w.logger.Error().Err(err).Str("job_id", payload.JobID).Msg("gagal update status completed")
		return fmt.Errorf("worker: gagal update status completed: %w", err)
	}

	w.logger.Info().
		Str("job_id", payload.JobID).
		Str("download_url", downloadURL).
		Msg("selesai export laporan")
	return nil
}

// generateAndSave menghasilkan file laporan dan menyimpannya ke disk.
// Mengembalikan path file sebagai download URL.
func (w *ExportWorker) generateAndSave(ctx context.Context, p exportPayload) (string, error) {
	// Pastikan direktori export ada
	exportDir := filepath.Join("exports", p.TenantID)
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori export: %w", err)
	}

	filename := fmt.Sprintf("%s_%s_%s.%s",
		p.ReportType, p.TenantID[:8], time.Now().Format("20060102_150405"), p.Format)
	filePath := filepath.Join(exportDir, filename)

	switch p.Format {
	case "xlsx":
		return filePath, w.generateXLSX(ctx, p, filePath)
	case "pdf":
		return filePath, w.generatePDFPlaceholder(ctx, p, filePath)
	default:
		return "", fmt.Errorf("format tidak didukung: %s", p.Format)
	}
}

// generateXLSX menghasilkan file Excel menggunakan excelize.
func (w *ExportWorker) generateXLSX(ctx context.Context, p exportPayload, filePath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Laporan"
	f.SetSheetName("Sheet1", sheet)

	// Header laporan
	f.SetCellValue(sheet, "A1", "Laporan: "+p.ReportType)
	f.SetCellValue(sheet, "A2", "Periode: "+p.Filters.PeriodStart.Format("02/01/2006")+
		" - "+p.Filters.PeriodEnd.Format("02/01/2006"))
	f.SetCellValue(sheet, "A3", "Dibuat: "+time.Now().Format("02/01/2006 15:04"))

	// Ambil data laporan berdasarkan tipe dan tulis ke sheet
	if err := w.writeReportData(ctx, f, sheet, p); err != nil {
		return fmt.Errorf("gagal menulis data laporan: %w", err)
	}

	// Footer ISPBoss attribution
	f.SetCellValue(sheet, "A100", "Dihasilkan oleh ISPBoss")

	return f.SaveAs(filePath)
}

// writeReportData menulis data laporan ke sheet Excel berdasarkan tipe laporan.
func (w *ExportWorker) writeReportData(ctx context.Context, f *excelize.File, sheet string, p exportPayload) error {
	switch p.ReportType {
	case "revenue":
		report, err := w.reportManager.GetRevenueReport(ctx, p.TenantID, p.Filters)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, "A5", "Sumber Pendapatan")
		f.SetCellValue(sheet, "B5", "Jumlah")
		f.SetCellValue(sheet, "A6", "Langganan Bulanan")
		f.SetCellValue(sheet, "B6", report.Current.MonthlySubscription)
		f.SetCellValue(sheet, "A7", "Penjualan Voucher")
		f.SetCellValue(sheet, "B7", report.Current.VoucherSales)
		f.SetCellValue(sheet, "A8", "Biaya Instalasi")
		f.SetCellValue(sheet, "B8", report.Current.InstallationFees)
		f.SetCellValue(sheet, "A9", "Denda Keterlambatan")
		f.SetCellValue(sheet, "B9", report.Current.LateFees)
		f.SetCellValue(sheet, "A10", "Lainnya")
		f.SetCellValue(sheet, "B10", report.Current.Other)
		f.SetCellValue(sheet, "A11", "Total")
		f.SetCellValue(sheet, "B11", report.Current.Total)
	default:
		// Untuk tipe lain, tulis placeholder
		f.SetCellValue(sheet, "A5", "Data laporan "+p.ReportType)
		f.SetCellValue(sheet, "A6", "Export detail akan ditambahkan sesuai kebutuhan")
	}
	return nil
}

// generatePDFPlaceholder menghasilkan file PDF placeholder.
// TODO: Implementasi lengkap menggunakan gofpdf dengan tenant branding.
func (w *ExportWorker) generatePDFPlaceholder(_ context.Context, p exportPayload, filePath string) error {
	content := fmt.Sprintf("Laporan: %s\nPeriode: %s - %s\nDibuat: %s\n\nDihasilkan oleh ISPBoss\n",
		p.ReportType,
		p.Filters.PeriodStart.Format("02/01/2006"),
		p.Filters.PeriodEnd.Format("02/01/2006"),
		time.Now().Format("02/01/2006 15:04"),
	)
	return os.WriteFile(filePath, []byte(content), 0o644)
}
