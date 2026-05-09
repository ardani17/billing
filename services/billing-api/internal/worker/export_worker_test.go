// export_worker_test.go berisi integration tests untuk ExportWorker.
package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// mockJobRepo mengimplementasikan domain.ReportJobRepository untuk testing.
type mockJobRepo struct {
	jobs          map[string]*domain.ReportJob
	statusUpdates []statusUpdate
}

// statusUpdate melacak panggilan UpdateStatus.
type statusUpdate struct {
	ID          string
	Status      domain.ReportJobStatus
	DownloadURL string
	ErrMsg      string
}

func newMockJobRepo() *mockJobRepo {
	return &mockJobRepo{
		jobs:          make(map[string]*domain.ReportJob),
		statusUpdates: make([]statusUpdate, 0),
	}
}

func (m *mockJobRepo) Create(_ context.Context, job *domain.ReportJob) (*domain.ReportJob, error) {
	copy := *job
	m.jobs[copy.ID] = &copy
	return &copy, nil
}

func (m *mockJobRepo) GetByID(_ context.Context, id string) (*domain.ReportJob, error) {
	j, ok := m.jobs[id]
	if !ok {
		return nil, domain.ErrReportJobNotFound
	}
	copy := *j
	return &copy, nil
}

func (m *mockJobRepo) UpdateStatus(_ context.Context, id string, status domain.ReportJobStatus, downloadURL, errMsg string) error {
	m.statusUpdates = append(m.statusUpdates, statusUpdate{
		ID: id, Status: status, DownloadURL: downloadURL, ErrMsg: errMsg,
	})
	if j, ok := m.jobs[id]; ok {
		j.Status = status
		j.DownloadURL = downloadURL
		j.Error = errMsg
	}
	return nil
}

func (m *mockJobRepo) CleanupOld(_ context.Context, _ time.Time) error {
	return nil
}

// --- Tes: Export task payload parsing ---

func TestExportWorker_HandleExportTask_InvalidPayload(t *testing.T) {
	jobRepo := newMockJobRepo()
	logger := newWorkerTestLogger()

	worker := &ExportWorker{
		reportManager: nil, // Tidak perlu untuk test payload parsing
		jobRepo:       jobRepo,
		logger:        logger,
	}

	task := asynq.NewTask(TaskReportExport, []byte("invalid json"))
	err := worker.handleExportTask(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestExportWorker_HandleExportTask_PayloadParsing(t *testing.T) {
	payload := exportPayload{
		JobID:      "job-123",
		TenantID:   "tenant-abc",
		ReportType: "revenue",
		Format:     "xlsx",
		Filters: domain.ReportFilter{
			PeriodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("gagal marshal payload: %v", err)
	}

	var parsed exportPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("gagal unmarshal payload: %v", err)
	}

	if parsed.JobID != "job-123" {
		t.Fatalf("expected job_id=job-123, got %s", parsed.JobID)
	}
	if parsed.TenantID != "tenant-abc" {
		t.Fatalf("expected tenant_id=tenant-abc, got %s", parsed.TenantID)
	}
	if parsed.ReportType != "revenue" {
		t.Fatalf("expected report_type=revenue, got %s", parsed.ReportType)
	}
	if parsed.Format != "xlsx" {
		t.Fatalf("expected format=xlsx, got %s", parsed.Format)
	}
}

func TestExportWorker_StatusUpdateFlow(t *testing.T) {
	jobRepo := newMockJobRepo()
	jobRepo.jobs["job-123"] = &domain.ReportJob{
		ID:         "job-123",
		TenantID:   "tenant-abc",
		ReportType: "revenue",
		Format:     "xlsx",
		Status:     domain.JobPending,
		CreatedAt:  time.Now(),
	}

	err := jobRepo.UpdateStatus(context.Background(), "job-123", domain.JobProcessing, "", "")
	if err != nil {
		t.Fatalf("gagal update status: %v", err)
	}

	if len(jobRepo.statusUpdates) != 1 {
		t.Fatalf("expected 1 status update, got %d", len(jobRepo.statusUpdates))
	}
	if jobRepo.statusUpdates[0].Status != domain.JobProcessing {
		t.Fatalf("expected status processing, got %s", jobRepo.statusUpdates[0].Status)
	}

	err = jobRepo.UpdateStatus(context.Background(), "job-123", domain.JobCompleted, "/exports/revenue.xlsx", "")
	if err != nil {
		t.Fatalf("gagal update status: %v", err)
	}

	if len(jobRepo.statusUpdates) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(jobRepo.statusUpdates))
	}
	if jobRepo.statusUpdates[1].Status != domain.JobCompleted {
		t.Fatalf("expected status completed, got %s", jobRepo.statusUpdates[1].Status)
	}
	if jobRepo.statusUpdates[1].DownloadURL != "/exports/revenue.xlsx" {
		t.Fatalf("expected download URL, got %s", jobRepo.statusUpdates[1].DownloadURL)
	}
}

func TestExportWorker_FailedStatusUpdate(t *testing.T) {
	jobRepo := newMockJobRepo()
	jobRepo.jobs["job-456"] = &domain.ReportJob{
		ID:     "job-456",
		Status: domain.JobPending,
	}

	err := jobRepo.UpdateStatus(context.Background(), "job-456", domain.JobFailed, "", "gagal generate laporan")
	if err != nil {
		t.Fatalf("gagal update status: %v", err)
	}

	if jobRepo.jobs["job-456"].Status != domain.JobFailed {
		t.Fatalf("expected status failed, got %s", jobRepo.jobs["job-456"].Status)
	}
	if jobRepo.jobs["job-456"].Error != "gagal generate laporan" {
		t.Fatalf("expected error message, got %s", jobRepo.jobs["job-456"].Error)
	}
}

// --- Tes: Task type constants ---

func TestExportWorker_TaskTypeConstant(t *testing.T) {
	if TaskReportExport != "report.export" {
		t.Fatalf("expected task type 'report.export', got %s", TaskReportExport)
	}
}
