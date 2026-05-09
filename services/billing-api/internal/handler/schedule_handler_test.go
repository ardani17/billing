// schedule_handler_test.go berisi integration tests untuk jadwal endpoints.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// mockScheduleUsecase mengimplementasikan domain.ScheduleUsecase untuk testing.
type mockScheduleUsecase struct {
	schedules map[string]*domain.ReportSchedule
	seqID     int
}

func newMockScheduleUsecase() *mockScheduleUsecase {
	return &mockScheduleUsecase{
		schedules: make(map[string]*domain.ReportSchedule),
	}
}

func (m *mockScheduleUsecase) Create(_ context.Context, tenantID string, req domain.CreateScheduleRequest, actor domain.ActorInfo) (*domain.ReportSchedule, error) {
	m.seqID++
	id := fmt.Sprintf("sched-%d", m.seqID)

	recipients := make([]domain.Recipient, len(req.Recipients))
	for i, r := range req.Recipients {
		recipients[i] = domain.Recipient{Type: r.Type, Address: r.Address}
	}

	schedule := &domain.ReportSchedule{
		ID:           id,
		TenantID:     tenantID,
		ReportType:   req.ReportType,
		ScheduleType: domain.ScheduleType(req.ScheduleType),
		Format:       req.Format,
		Recipients:   recipients,
		IsActive:     true,
		CreatedByID:  actor.ActorID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.schedules[id] = schedule
	return schedule, nil
}

func (m *mockScheduleUsecase) Update(_ context.Context, id string, req domain.UpdateScheduleRequest) (*domain.ReportSchedule, error) {
	s, ok := m.schedules[id]
	if !ok {
		return nil, domain.ErrReportScheduleNotFound
	}
	if req.ReportType != "" {
		s.ReportType = req.ReportType
	}
	if req.Format != "" {
		s.Format = req.Format
	}
	s.UpdatedAt = time.Now()
	copy := *s
	return &copy, nil
}

func (m *mockScheduleUsecase) Delete(_ context.Context, id string) error {
	if _, ok := m.schedules[id]; !ok {
		return domain.ErrReportScheduleNotFound
	}
	m.schedules[id].IsActive = false
	return nil
}

func (m *mockScheduleUsecase) List(_ context.Context, tenantID string) ([]*domain.ReportSchedule, error) {
	var result []*domain.ReportSchedule
	for _, s := range m.schedules {
		if s.TenantID == tenantID && s.IsActive {
			copy := *s
			result = append(result, &copy)
		}
	}
	return result, nil
}

func setupScheduleTestApp(mock *mockScheduleUsecase) *fiber.App {
	logger := zerolog.New(io.Discard)
	handler := NewScheduleHandler(mock, logger)

	app := fiber.New()

	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	schedules := app.Group("/api/v1/reports/schedules", setLocals)
	schedules.Get("/", handler.List)
	schedules.Post("/", handler.Create)
	schedules.Put("/:id", handler.Update)
	schedules.Delete("/:id", handler.Delete)

	return app
}

// --- Tes: Schedule CRUD ---

func TestScheduleHandler_Create_Success(t *testing.T) {
	mock := newMockScheduleUsecase()
	app := setupScheduleTestApp(mock)

	body, _ := json.Marshal(domain.CreateScheduleRequest{
		ReportType:   "revenue",
		ScheduleType: "daily",
		Format:       "pdf",
		Recipients: []domain.RecipientRequest{
			{Type: "email", Address: "admin@isp.com"},
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/reports/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestScheduleHandler_Create_InvalidBody(t *testing.T) {
	mock := newMockScheduleUsecase()
	app := setupScheduleTestApp(mock)

	req := httptest.NewRequest("POST", "/api/v1/reports/schedules", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestScheduleHandler_Create_ValidationError(t *testing.T) {
	mock := newMockScheduleUsecase()
	app := setupScheduleTestApp(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"report_type":   "revenue",
		"schedule_type": "invalid_type",
		"format":        "pdf",
		"recipients":    []map[string]string{{"type": "email", "address": "a@b.com"}},
	})

	req := httptest.NewRequest("POST", "/api/v1/reports/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestScheduleHandler_List_Success(t *testing.T) {
	mock := newMockScheduleUsecase()
	mock.schedules["sched-1"] = &domain.ReportSchedule{
		ID: "sched-1", TenantID: "test-tenant-id", ReportType: "revenue",
		ScheduleType: domain.ScheduleDaily, Format: "pdf", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupScheduleTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/reports/schedules", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestScheduleHandler_Update_Success(t *testing.T) {
	mock := newMockScheduleUsecase()
	mock.schedules["sched-1"] = &domain.ReportSchedule{
		ID: "sched-1", TenantID: "test-tenant-id", ReportType: "revenue",
		ScheduleType: domain.ScheduleDaily, Format: "pdf", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupScheduleTestApp(mock)

	body, _ := json.Marshal(domain.UpdateScheduleRequest{
		Format: "xlsx",
	})

	req := httptest.NewRequest("PUT", "/api/v1/reports/schedules/sched-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestScheduleHandler_Update_NotFound(t *testing.T) {
	mock := newMockScheduleUsecase()
	app := setupScheduleTestApp(mock)

	body, _ := json.Marshal(domain.UpdateScheduleRequest{Format: "xlsx"})
	req := httptest.NewRequest("PUT", "/api/v1/reports/schedules/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestScheduleHandler_Delete_Success(t *testing.T) {
	mock := newMockScheduleUsecase()
	mock.schedules["sched-1"] = &domain.ReportSchedule{
		ID: "sched-1", TenantID: "test-tenant-id", ReportType: "revenue",
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupScheduleTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/reports/schedules/sched-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestScheduleHandler_Delete_NotFound(t *testing.T) {
	mock := newMockScheduleUsecase()
	app := setupScheduleTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/reports/schedules/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestScheduleHandler_Unauthorized(t *testing.T) {
	mock := newMockScheduleUsecase()
	logger := zerolog.New(io.Discard)
	handler := NewScheduleHandler(mock, logger)

	app := fiber.New()
	// Tanpa middleware atur tenant_id
	app.Get("/api/v1/reports/schedules", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/reports/schedules", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
