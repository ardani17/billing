// schedule_worker_test.go berisi integration tests untuk ScheduleWorker.
// Test: payload parsing, scheduled report processing, cleanup jobs.
package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock ReportScheduleRepository untuk schedule worker tests ---

// mockScheduleRepo mengimplementasikan domain.ReportScheduleRepository untuk testing.
type mockScheduleRepo struct {
	schedules map[string]*domain.ReportSchedule
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{
		schedules: make(map[string]*domain.ReportSchedule),
	}
}

func (m *mockScheduleRepo) Create(_ context.Context, s *domain.ReportSchedule) (*domain.ReportSchedule, error) {
	copy := *s
	m.schedules[copy.ID] = &copy
	return &copy, nil
}

func (m *mockScheduleRepo) GetByID(_ context.Context, id string) (*domain.ReportSchedule, error) {
	s, ok := m.schedules[id]
	if !ok {
		return nil, domain.ErrReportScheduleNotFound
	}
	copy := *s
	return &copy, nil
}

func (m *mockScheduleRepo) Update(_ context.Context, s *domain.ReportSchedule) (*domain.ReportSchedule, error) {
	if _, ok := m.schedules[s.ID]; !ok {
		return nil, domain.ErrReportScheduleNotFound
	}
	copy := *s
	m.schedules[copy.ID] = &copy
	return &copy, nil
}

func (m *mockScheduleRepo) Deactivate(_ context.Context, id string) error {
	s, ok := m.schedules[id]
	if !ok {
		return domain.ErrReportScheduleNotFound
	}
	s.IsActive = false
	return nil
}

func (m *mockScheduleRepo) ListByTenant(_ context.Context, tenantID string) ([]*domain.ReportSchedule, error) {
	var result []*domain.ReportSchedule
	for _, s := range m.schedules {
		if s.TenantID == tenantID && s.IsActive {
			copy := *s
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) ListDue(_ context.Context, scheduleType domain.ScheduleType) ([]*domain.ReportSchedule, error) {
	var result []*domain.ReportSchedule
	for _, s := range m.schedules {
		if s.ScheduleType == scheduleType && s.IsActive {
			copy := *s
			result = append(result, &copy)
		}
	}
	return result, nil
}

// --- Test: Scheduled report payload parsing ---

func TestScheduleWorker_HandleScheduledReport_InvalidPayload(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()
	jobRepo := newMockJobRepo()
	logger := newWorkerTestLogger()

	worker := &ScheduleWorker{
		scheduleRepo:  scheduleRepo,
		reportManager: nil,
		jobRepo:       jobRepo,
		queueClient:   nil,
		logger:        logger,
	}

	task := asynq.NewTask(TaskScheduledReport, []byte("invalid json"))
	err := worker.handleScheduledReport(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestScheduleWorker_HandleScheduledReport_NoSchedulesDue(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()
	jobRepo := newMockJobRepo()
	logger := newWorkerTestLogger()

	worker := &ScheduleWorker{
		scheduleRepo:  scheduleRepo,
		reportManager: nil,
		jobRepo:       jobRepo,
		queueClient:   nil,
		logger:        logger,
	}

	payload, _ := json.Marshal(scheduledPayload{ScheduleType: "daily"})
	task := asynq.NewTask(TaskScheduledReport, payload)

	err := worker.handleScheduledReport(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error when no schedules due, got: %v", err)
	}
}

func TestScheduleWorker_ScheduledPayloadParsing(t *testing.T) {
	payload := scheduledPayload{ScheduleType: "weekly"}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("gagal marshal payload: %v", err)
	}

	var parsed scheduledPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("gagal unmarshal payload: %v", err)
	}

	if parsed.ScheduleType != "weekly" {
		t.Fatalf("expected schedule_type=weekly, got %s", parsed.ScheduleType)
	}
}

// --- Test: Cleanup report jobs ---

func TestScheduleWorker_HandleCleanupReportJobs(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()
	jobRepo := newMockJobRepo()
	logger := newWorkerTestLogger()

	worker := &ScheduleWorker{
		scheduleRepo:  scheduleRepo,
		reportManager: nil,
		jobRepo:       jobRepo,
		queueClient:   nil,
		logger:        logger,
	}

	task := asynq.NewTask(TaskCleanupReportJobs, nil)
	err := worker.handleCleanupReportJobs(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error for cleanup, got: %v", err)
	}
}

// --- Test: Task type constants ---

func TestScheduleWorker_TaskTypeConstants(t *testing.T) {
	if TaskScheduledReport != "report.scheduled" {
		t.Fatalf("expected 'report.scheduled', got %s", TaskScheduledReport)
	}
	if TaskCleanupReportJobs != "report.cleanup_jobs" {
		t.Fatalf("expected 'report.cleanup_jobs', got %s", TaskCleanupReportJobs)
	}
	if TaskCleanupScheduledFiles != "report.cleanup_files" {
		t.Fatalf("expected 'report.cleanup_files', got %s", TaskCleanupScheduledFiles)
	}
}

// --- Test: ListDue returns correct schedules ---

func TestScheduleWorker_ListDue_FiltersByType(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()

	// Tambahkan jadwal dengan tipe berbeda
	scheduleRepo.schedules["s-1"] = &domain.ReportSchedule{
		ID: "s-1", TenantID: "t-1", ReportType: "revenue",
		ScheduleType: domain.ScheduleDaily, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	scheduleRepo.schedules["s-2"] = &domain.ReportSchedule{
		ID: "s-2", TenantID: "t-1", ReportType: "aging",
		ScheduleType: domain.ScheduleWeekly, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	scheduleRepo.schedules["s-3"] = &domain.ReportSchedule{
		ID: "s-3", TenantID: "t-1", ReportType: "profit-loss",
		ScheduleType: domain.ScheduleDaily, IsActive: false, // Inactive
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	// Query daily schedules
	daily, err := scheduleRepo.ListDue(context.Background(), domain.ScheduleDaily)
	if err != nil {
		t.Fatalf("gagal list due: %v", err)
	}
	if len(daily) != 1 {
		t.Fatalf("expected 1 daily schedule (active only), got %d", len(daily))
	}
	if daily[0].ID != "s-1" {
		t.Fatalf("expected schedule s-1, got %s", daily[0].ID)
	}

	// Query weekly schedules
	weekly, err := scheduleRepo.ListDue(context.Background(), domain.ScheduleWeekly)
	if err != nil {
		t.Fatalf("gagal list due: %v", err)
	}
	if len(weekly) != 1 {
		t.Fatalf("expected 1 weekly schedule, got %d", len(weekly))
	}
}
