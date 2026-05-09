package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// =============================================================================

type mockLogRepo struct {
	logs       map[string]*domain.NotificationLog
	listResult *domain.LogListResult
	listErr    error
	getErr     error
}

func newMockLogRepo() *mockLogRepo {
	return &mockLogRepo{
		logs: make(map[string]*domain.NotificationLog),
	}
}

func (m *mockLogRepo) Create(_ context.Context, log *domain.NotificationLog) (*domain.NotificationLog, error) {
	copy := *log
	m.logs[copy.ID] = &copy
	return &copy, nil
}

func (m *mockLogRepo) GetByID(_ context.Context, id string) (*domain.NotificationLog, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	l, ok := m.logs[id]
	if !ok {
		return nil, domain.ErrLogNotFound
	}
	copy := *l
	return &copy, nil
}

func (m *mockLogRepo) Update(_ context.Context, _ *domain.NotificationLog) error {
	return nil
}

func (m *mockLogRepo) List(_ context.Context, _ domain.LogListParams) (*domain.LogListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listResult != nil {
		return m.listResult, nil
	}
	return &domain.LogListResult{
		Data:       []*domain.NotificationLog{},
		Total:      0,
		Page:       1,
		PageSize:   25,
		TotalPages: 0,
	}, nil
}

func (m *mockLogRepo) FindByDedupKey(_ context.Context, _ string, _ int) (*domain.NotificationLog, error) {
	return nil, domain.ErrLogNotFound
}

func (m *mockLogRepo) CountTodayByCustomer(_ context.Context, _, _, _ string) (int, error) {
	return 0, nil
}

func (m *mockLogRepo) LastSentToCustomer(_ context.Context, _, _ string) (*time.Time, error) {
	return nil, nil
}

// =============================================================================
// =============================================================================

func testTenantMiddleware(tenantID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenantID)
		return c.Next()
	}
}

// =============================================================================
// =============================================================================

func TestLogHandler_List_Success(t *testing.T) {
	repo := newMockLogRepo()
	now := time.Now()

	repo.listResult = &domain.LogListResult{
		Data: []*domain.NotificationLog{
			{
				ID:           "log-1",
				TenantID:     "tenant-1",
				CustomerID:   "cust-1",
				Channel:      domain.ChannelWhatsApp,
				Provider:     "fonnte",
				Recipient:    "08123456789",
				Body:         "Halo pelanggan",
				Status:       domain.StatusSent,
				CustomerName: "Ahmad",
				TemplateName: "invoice_new",
				CreatedAt:    now,
			},
			{
				ID:           "log-2",
				TenantID:     "tenant-1",
				CustomerID:   "cust-2",
				Channel:      domain.ChannelSMS,
				Provider:     "zenziva",
				Recipient:    "08198765432",
				Body:         "Tagihan Anda",
				Status:       domain.StatusDelivered,
				CustomerName: "Budi",
				TemplateName: "reminder_h1",
				CreatedAt:    now,
			},
		},
		Total:      2,
		Page:       1,
		PageSize:   25,
		TotalPages: 1,
	}

	handler := NewLogHandler(repo)
	app := fiber.New()
	app.Get("/api/v1/notifications/logs", testTenantMiddleware("tenant-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/notifications/logs?page=1&page_size=25", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Fatalf("expected success=true, got false")
	}

	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", apiResp.Data)
	}

	if total, ok := dataMap["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total=2, got %v", dataMap["total"])
	}

	if page, ok := dataMap["page"].(float64); !ok || int(page) != 1 {
		t.Fatalf("expected page=1, got %v", dataMap["page"])
	}

	if pageSize, ok := dataMap["page_size"].(float64); !ok || int(pageSize) != 25 {
		t.Fatalf("expected page_size=25, got %v", dataMap["page_size"])
	}

	items, ok := dataMap["items"].([]interface{})
	if !ok {
		t.Fatalf("expected items to be an array, got %T", dataMap["items"])
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

// =============================================================================
// Tes: LogHandler.GetByID - 200 sukses
// =============================================================================

func TestLogHandler_GetByID_Success(t *testing.T) {
	repo := newMockLogRepo()
	now := time.Now()

	repo.logs["log-abc"] = &domain.NotificationLog{
		ID:           "log-abc",
		TenantID:     "tenant-1",
		CustomerID:   "cust-1",
		Channel:      domain.ChannelWhatsApp,
		Provider:     "fonnte",
		Recipient:    "08123456789",
		Body:         "Halo pelanggan",
		Status:       domain.StatusSent,
		CustomerName: "Ahmad",
		TemplateName: "invoice_new",
		CreatedAt:    now,
	}

	handler := NewLogHandler(repo)
	app := fiber.New()
	app.Get("/api/v1/notifications/logs/:id", testTenantMiddleware("tenant-1"), handler.GetByID)

	req := httptest.NewRequest("GET", "/api/v1/notifications/logs/log-abc", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Fatalf("expected success=true, got false")
	}

	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", apiResp.Data)
	}

	if id, ok := dataMap["id"].(string); !ok || id != "log-abc" {
		t.Fatalf("expected id=log-abc, got %v", dataMap["id"])
	}
}

// =============================================================================
// Tes: LogHandler.GetByID - 404 tidak ditemukan
// =============================================================================

func TestLogHandler_GetByID_NotFound(t *testing.T) {
	repo := newMockLogRepo()

	handler := NewLogHandler(repo)
	app := fiber.New()
	app.Get("/api/v1/notifications/logs/:id", testTenantMiddleware("tenant-1"), handler.GetByID)

	req := httptest.NewRequest("GET", "/api/v1/notifications/logs/nonexistent-id", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if apiResp.Success {
		t.Fatalf("expected success=false, got true")
	}

	if apiResp.Error == nil || apiResp.Error.Code != "LOG_NOT_FOUND" {
		t.Fatalf("expected error code LOG_NOT_FOUND, got %v", apiResp.Error)
	}
}
