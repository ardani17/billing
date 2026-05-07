// router_handler_test.go - unit test untuk RouterHandler.
// Tenant context di-bypass dengan middleware test yang memanggil pkg/tenant.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

type mockRouterUsecase struct {
	createFn           func(ctx context.Context, tenantID string, req domain.CreateRouterRequest) (*domain.RouterResponse, error)
	getByIDFn          func(ctx context.Context, id string) (*domain.RouterDetailResponse, error)
	updateFn           func(ctx context.Context, id string, req domain.UpdateRouterRequest) (*domain.RouterResponse, error)
	deleteFn           func(ctx context.Context, id string) error
	listFn             func(ctx context.Context, params domain.RouterListParams) (*domain.RouterListResult, error)
	testConnectionFn   func(ctx context.Context, id string) (*domain.SystemResource, error)
	rebootFn           func(ctx context.Context, id string, confirmName string) error
	getStatusSummaryFn func(ctx context.Context) (*domain.StatusSummary, error)
}

func (m *mockRouterUsecase) Create(ctx context.Context, tenantID string, req domain.CreateRouterRequest) (*domain.RouterResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, req)
	}
	return nil, nil
}

func (m *mockRouterUsecase) GetByID(ctx context.Context, id string) (*domain.RouterDetailResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRouterUsecase) Update(ctx context.Context, id string, req domain.UpdateRouterRequest) (*domain.RouterResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}

func (m *mockRouterUsecase) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockRouterUsecase) List(ctx context.Context, params domain.RouterListParams) (*domain.RouterListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, nil
}

func (m *mockRouterUsecase) TestConnection(ctx context.Context, id string) (*domain.SystemResource, error) {
	if m.testConnectionFn != nil {
		return m.testConnectionFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRouterUsecase) Reboot(ctx context.Context, id string, confirmName string) error {
	if m.rebootFn != nil {
		return m.rebootFn(ctx, id, confirmName)
	}
	return nil
}

func (m *mockRouterUsecase) GetStatusSummary(ctx context.Context) (*domain.StatusSummary, error) {
	if m.getStatusSummaryFn != nil {
		return m.getStatusSummaryFn(ctx)
	}
	return nil, nil
}

// =============================================================================
// =============================================================================

const testTenantID = "tenant-test-123"

// setupTestApp membuat Fiber app dengan RouterHandler dan middleware
// yang menyuntikkan tenant_id ke context (bypass JWT auth).
func setupTestApp(uc domain.RouterUsecase) *fiber.App {
	app := fiber.New()
	handler := NewRouterHandler(uc)

	// Middleware test: atur tenant_id di Go context seperti yang dilakukan
	// pkg/tenant.Middleware, tanpa perlu JWT token.
	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), testTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	// Daftarkan route sesuai router.go
	routers := app.Group("/api/v1/mikrotik/routers")
	routers.Post("/", handler.Create)
	routers.Get("/", handler.List)
	routers.Get("/:id", handler.GetByID)
	routers.Put("/:id", handler.Update)
	routers.Delete("/:id", handler.Delete)
	routers.Post("/:id/test", handler.TestConnection)
	routers.Post("/:id/reboot", handler.Reboot)

	return app
}

// sampleRouter mengembalikan Router contoh untuk testing.
func sampleRouter() *domain.Router {
	now := time.Now()
	return &domain.Router{
		ID:                     "router-1",
		TenantID:               testTenantID,
		Name:                   "Router Utama",
		Host:                   "192.168.1.1",
		Port:                   8728,
		Username:               "admin",
		UseSSL:                 false,
		ServiceTypes:           []string{"pppoe"},
		Status:                 domain.StatusOnline,
		HealthCheckIntervalSec: 60,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func parseAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// =============================================================================
// =============================================================================

func TestCreate_Success(t *testing.T) {
	router := sampleRouter()
	uc := &mockRouterUsecase{
		createFn: func(_ context.Context, tenantID string, req domain.CreateRouterRequest) (*domain.RouterResponse, error) {
			if tenantID != testTenantID {
				t.Errorf("expected tenant %s, got %s", testTenantID, tenantID)
			}
			return &domain.RouterResponse{Router: router}, nil
		},
	}
	app := setupTestApp(uc)

	body, _ := json.Marshal(domain.CreateRouterRequest{
		Name:     "Router Utama",
		Host:     "192.168.1.1",
		Username: "admin",
		Password: "secret",
	})

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	uc := &mockRouterUsecase{}
	app := setupTestApp(uc)

	// Body tanpa field required (name, host, username, password)
	body, _ := json.Marshal(map[string]string{"notes": "test"})

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

// TestCreate_DuplicateName memverifikasi nama duplikat menghasilkan 409.
func TestCreate_DuplicateName(t *testing.T) {
	uc := &mockRouterUsecase{
		createFn: func(_ context.Context, _ string, _ domain.CreateRouterRequest) (*domain.RouterResponse, error) {
			return nil, domain.ErrRouterNameExists
		},
	}
	app := setupTestApp(uc)

	body, _ := json.Marshal(domain.CreateRouterRequest{
		Name:     "Router Duplikat",
		Host:     "192.168.1.2",
		Username: "admin",
		Password: "secret",
	})

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ROUTER_NAME_EXISTS" {
		t.Fatalf("expected ROUTER_NAME_EXISTS, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

// TestGetByID_Success memverifikasi router ditemukan menghasilkan 200.
func TestGetByID_Success(t *testing.T) {
	router := sampleRouter()
	uc := &mockRouterUsecase{
		getByIDFn: func(_ context.Context, id string) (*domain.RouterDetailResponse, error) {
			if id != "router-1" {
				t.Errorf("expected id router-1, got %s", id)
			}
			return &domain.RouterDetailResponse{Router: router}, nil
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/routers/router-1", nil)

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
}

// TestGetByID_NotFound memverifikasi router tidak ditemukan menghasilkan 404.
func TestGetByID_NotFound(t *testing.T) {
	uc := &mockRouterUsecase{
		getByIDFn: func(_ context.Context, _ string) (*domain.RouterDetailResponse, error) {
			return nil, domain.ErrRouterNotFound
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/routers/nonexistent", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ROUTER_NOT_FOUND" {
		t.Fatalf("expected ROUTER_NOT_FOUND, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestUpdate_Success(t *testing.T) {
	router := sampleRouter()
	router.Name = "Router Updated"
	uc := &mockRouterUsecase{
		updateFn: func(_ context.Context, id string, _ domain.UpdateRouterRequest) (*domain.RouterResponse, error) {
			if id != "router-1" {
				t.Errorf("expected id router-1, got %s", id)
			}
			return &domain.RouterResponse{Router: router}, nil
		},
	}
	app := setupTestApp(uc)

	body, _ := json.Marshal(domain.UpdateRouterRequest{
		Name: "Router Updated",
	})

	req := httptest.NewRequest("PUT", "/api/v1/mikrotik/routers/router-1", bytes.NewReader(body))
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

func TestUpdate_InvalidTransition(t *testing.T) {
	uc := &mockRouterUsecase{
		updateFn: func(_ context.Context, _ string, _ domain.UpdateRouterRequest) (*domain.RouterResponse, error) {
			return nil, domain.ErrInvalidStatusTransition
		},
	}
	app := setupTestApp(uc)

	body, _ := json.Marshal(domain.UpdateRouterRequest{
		Status: "maintenance",
	})

	req := httptest.NewRequest("PUT", "/api/v1/mikrotik/routers/router-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestDelete_Success(t *testing.T) {
	uc := &mockRouterUsecase{
		deleteFn: func(_ context.Context, id string) error {
			if id != "router-1" {
				t.Errorf("expected id router-1, got %s", id)
			}
			return nil
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("DELETE", "/api/v1/mikrotik/routers/router-1", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestDelete_NotFound(t *testing.T) {
	uc := &mockRouterUsecase{
		deleteFn: func(_ context.Context, _ string) error {
			return domain.ErrRouterNotFound
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("DELETE", "/api/v1/mikrotik/routers/nonexistent", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ROUTER_NOT_FOUND" {
		t.Fatalf("expected ROUTER_NOT_FOUND, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestList_Pagination(t *testing.T) {
	uc := &mockRouterUsecase{
		listFn: func(_ context.Context, params domain.RouterListParams) (*domain.RouterListResult, error) {
			// Verifikasi parameter paginasi diteruskan dengan benar
			if params.Page != 2 {
				t.Errorf("expected page 2, got %d", params.Page)
			}
			if params.PageSize != 5 {
				t.Errorf("expected page_size 5, got %d", params.PageSize)
			}
			if params.TenantID != testTenantID {
				t.Errorf("expected tenant %s, got %s", testTenantID, params.TenantID)
			}
			return &domain.RouterListResult{
				Data:       []*domain.Router{sampleRouter()},
				Total:      10,
				Page:       2,
				PageSize:   5,
				TotalPages: 2,
			}, nil
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/routers?page=2&page_size=5", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

// TestList_FilterByStatus memverifikasi filter status diteruskan ke usecase.
func TestList_FilterByStatus(t *testing.T) {
	uc := &mockRouterUsecase{
		listFn: func(_ context.Context, params domain.RouterListParams) (*domain.RouterListResult, error) {
			if params.Status != "online" {
				t.Errorf("expected status filter 'online', got '%s'", params.Status)
			}
			return &domain.RouterListResult{
				Data:       []*domain.Router{sampleRouter()},
				Total:      1,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			}, nil
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/routers?status=online", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Tes TestConnection - POST /api/v1/mikrotik/routers/:id/test
// =============================================================================

// TestTestConnection_Online memverifikasi router online menghasilkan 200 dengan system info.
func TestTestConnection_Online(t *testing.T) {
	uc := &mockRouterUsecase{
		testConnectionFn: func(_ context.Context, id string) (*domain.SystemResource, error) {
			if id != "router-1" {
				t.Errorf("expected id router-1, got %s", id)
			}
			return &domain.SystemResource{
				Version:   "6.49.10",
				BoardName: "RB750Gr3",
				CPUCount:  2,
				TotalRAM:  268435456,
				Identity:  "MikroTik",
			}, nil
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/test", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

// TestTestConnection_Offline memverifikasi router offline menghasilkan 502.
func TestTestConnection_Offline(t *testing.T) {
	uc := &mockRouterUsecase{
		testConnectionFn: func(_ context.Context, _ string) (*domain.SystemResource, error) {
			return nil, domain.ErrConnectionFailed
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/test", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadGateway {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 502, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "CONNECTION_FAILED" {
		t.Fatalf("expected CONNECTION_FAILED, got %v", apiResp.Error)
	}
}

// =============================================================================
// Tes Reboot - POST /api/v1/mikrotik/routers/:id/reboot
// =============================================================================

func TestReboot_ValidConfirmation(t *testing.T) {
	uc := &mockRouterUsecase{
		rebootFn: func(_ context.Context, id string, confirmName string) error {
			if id != "router-1" {
				t.Errorf("expected id router-1, got %s", id)
			}
			if confirmName != "Router Utama" {
				t.Errorf("expected confirmName 'Router Utama', got '%s'", confirmName)
			}
			return nil
		},
	}
	app := setupTestApp(uc)

	body, _ := json.Marshal(domain.RebootRequest{
		ConfirmationName: "Router Utama",
	})

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/reboot", bytes.NewReader(body))
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

// TestReboot_Mismatch memverifikasi konfirmasi tidak cocok menghasilkan 400.
func TestReboot_Mismatch(t *testing.T) {
	uc := &mockRouterUsecase{
		rebootFn: func(_ context.Context, _ string, _ string) error {
			return domain.ErrConfirmationMismatch
		},
	}
	app := setupTestApp(uc)

	body, _ := json.Marshal(domain.RebootRequest{
		ConfirmationName: "Nama Salah",
	})

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/reboot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "CONFIRMATION_MISMATCH" {
		t.Fatalf("expected CONFIRMATION_MISMATCH, got %v", apiResp.Error)
	}
}

// TestReboot_MissingConfirmation memverifikasi body tanpa confirmation_name menghasilkan 422.
func TestReboot_MissingConfirmation(t *testing.T) {
	uc := &mockRouterUsecase{}
	app := setupTestApp(uc)

	body, _ := json.Marshal(map[string]string{})

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/reboot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// =============================================================================
// =============================================================================

// TestErrorMapping_ConnectionTimeout memverifikasi timeout menghasilkan 504.
func TestErrorMapping_ConnectionTimeout(t *testing.T) {
	uc := &mockRouterUsecase{
		testConnectionFn: func(_ context.Context, _ string) (*domain.SystemResource, error) {
			return nil, domain.ErrConnectionTimeout
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/test", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", resp.StatusCode)
	}
}

// TestErrorMapping_PoolExhausted memverifikasi pool penuh menghasilkan 503.
func TestErrorMapping_PoolExhausted(t *testing.T) {
	uc := &mockRouterUsecase{
		testConnectionFn: func(_ context.Context, _ string) (*domain.SystemResource, error) {
			return nil, domain.ErrPoolExhausted
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/test", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}

// TestErrorMapping_RateLimited memverifikasi rate limit menghasilkan 429.
func TestErrorMapping_RateLimited(t *testing.T) {
	uc := &mockRouterUsecase{
		testConnectionFn: func(_ context.Context, _ string) (*domain.SystemResource, error) {
			return nil, domain.ErrRateLimited
		},
	}
	app := setupTestApp(uc)

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/router-1/test", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
}

func TestCreate_MalformedJSON(t *testing.T) {
	uc := &mockRouterUsecase{}
	app := setupTestApp(uc)

	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
