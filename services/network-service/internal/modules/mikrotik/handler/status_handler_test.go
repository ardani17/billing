// status_handler_test.go - unit test untuk StatusHandler.
// Menggunakan mockRouterUsecase dari router_handler_test.go (package yang sama).
// Tenant context di-bypass dengan middleware test yang memanggil pkg/tenant.
package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// setupStatusTestApp membuat Fiber app dengan StatusHandler dan middleware
// yang menyuntikkan tenant_id ke context (bypass JWT auth).
func setupStatusTestApp(uc domain.RouterUsecase) *fiber.App {
	app := fiber.New()
	handler := NewStatusHandler(uc)

	// Middleware test: atur tenant_id di Go context
	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), testTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	// Daftarkan route status summary
	app.Get("/api/v1/mikrotik/status/summary", handler.GetSummary)

	return app
}

// =============================================================================
// =============================================================================

// TestGetSummary_Success memverifikasi summary mengembalikan jumlah yang benar.
func TestGetSummary_Success(t *testing.T) {
	uc := &mockRouterUsecase{
		getStatusSummaryFn: func(_ context.Context) (*domain.StatusSummary, error) {
			return &domain.StatusSummary{
				TotalRouters:     10,
				OnlineCount:      6,
				OfflineCount:     3,
				MaintenanceCount: 1,
			}, nil
		},
	}
	app := setupStatusTestApp(uc)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/status/summary", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}

	data, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map, got %T", apiResp.Data)
	}

	// Verifikasi setiap field count
	tests := []struct {
		field    string
		expected float64
	}{
		{"total_routers", 10},
		{"online_count", 6},
		{"offline_count", 3},
		{"maintenance_count", 1},
	}

	for _, tc := range tests {
		val, exists := data[tc.field]
		if !exists {
			t.Errorf("field %s tidak ditemukan di response", tc.field)
			continue
		}
		num, ok := val.(float64)
		if !ok {
			t.Errorf("field %s bukan number, got %T", tc.field, val)
			continue
		}
		if num != tc.expected {
			t.Errorf("expected %s=%v, got %v", tc.field, tc.expected, num)
		}
	}
}

func TestGetSummary_Error(t *testing.T) {
	uc := &mockRouterUsecase{
		getStatusSummaryFn: func(_ context.Context) (*domain.StatusSummary, error) {
			return nil, errors.New("database connection lost")
		},
	}
	app := setupStatusTestApp(uc)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/status/summary", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Success {
		t.Fatal("expected success=false")
	}
	if apiResp.Error == nil || apiResp.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %v", apiResp.Error)
	}
}
