// service_profile_handler_test.go - unit test untuk ServiceProfileHandler.
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

type mockServiceProfileManager struct {
	createFn         func(ctx context.Context, tenantID string, req domain.CreateServiceProfileRequest) (*domain.ServiceProfileResponse, error)
	getByIDFn        func(ctx context.Context, id string) (*domain.ServiceProfileResponse, error)
	updateFn         func(ctx context.Context, id string, req domain.UpdateServiceProfileRequest) (*domain.ServiceProfileResponse, error)
	deleteFn         func(ctx context.Context, id string) error
	listFn           func(ctx context.Context, oltID string, params domain.ServiceProfileListParams) (*domain.ServiceProfileListResult, error)
	resolveProfileFn func(ctx context.Context, oltID string, packageID string) (*domain.ServiceProfile, error)
}

func (m *mockServiceProfileManager) Create(ctx context.Context, tenantID string, req domain.CreateServiceProfileRequest) (*domain.ServiceProfileResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockServiceProfileManager) GetByID(ctx context.Context, id string) (*domain.ServiceProfileResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockServiceProfileManager) Update(ctx context.Context, id string, req domain.UpdateServiceProfileRequest) (*domain.ServiceProfileResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}
func (m *mockServiceProfileManager) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockServiceProfileManager) List(ctx context.Context, oltID string, params domain.ServiceProfileListParams) (*domain.ServiceProfileListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, oltID, params)
	}
	return nil, nil
}
func (m *mockServiceProfileManager) ResolveProfile(ctx context.Context, oltID string, packageID string) (*domain.ServiceProfile, error) {
	if m.resolveProfileFn != nil {
		return m.resolveProfileFn(ctx, oltID, packageID)
	}
	return nil, nil
}

// =============================================================================
// =============================================================================

const spTestTenantID = "tenant-sp-test-123"

func setupSPTestApp(mgr domain.ServiceProfileManager) *fiber.App {
	app := fiber.New()
	h := NewServiceProfileHandler(mgr)

	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), spTestTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	// Route service profile per OLT
	app.Post("/api/v1/olt/devices/:id/service-profiles", h.CreateServiceProfile)
	app.Get("/api/v1/olt/devices/:id/service-profiles", h.ListServiceProfiles)
	// Route service profile standalone
	app.Put("/api/v1/olt/service-profiles/:id", h.UpdateServiceProfile)
	app.Delete("/api/v1/olt/service-profiles/:id", h.DeleteServiceProfile)

	return app
}

func sampleSPResponse() *domain.ServiceProfileResponse {
	now := time.Now()
	return &domain.ServiceProfileResponse{
		ID: "sp-1", OLTID: "olt-1", Name: "Profile 10Mbps",
		LineProfileID: 1, ServiceProfileID: 1,
		ActiveONTs: 3, CreatedAt: now, UpdatedAt: now,
	}
}

func parseSPAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// =============================================================================
// Tes CreateServiceProfile - POST /api/v1/olt/devices/:id/service-profiles
// =============================================================================

func TestSPCreate_Success(t *testing.T) {
	mgr := &mockServiceProfileManager{
		createFn: func(_ context.Context, tenantID string, req domain.CreateServiceProfileRequest) (*domain.ServiceProfileResponse, error) {
			if tenantID != spTestTenantID {
				t.Errorf("expected tenant %s, got %s", spTestTenantID, tenantID)
			}
			if req.OLTID != "olt-1" {
				t.Errorf("expected olt_id olt-1, got %s", req.OLTID)
			}
			return sampleSPResponse(), nil
		},
	}
	app := setupSPTestApp(mgr)

	body, _ := json.Marshal(domain.CreateServiceProfileRequest{
		Name: "Profile 10Mbps", LineProfileID: 1, ServiceProfileID: 1,
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/service-profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestSPCreate_ValidationError(t *testing.T) {
	app := setupSPTestApp(&mockServiceProfileManager{})

	// Body tanpa required field
	body, _ := json.Marshal(map[string]string{"description": "test"})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/service-profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	apiResp := parseSPAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestSPCreate_MalformedJSON(t *testing.T) {
	app := setupSPTestApp(&mockServiceProfileManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/service-profiles", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// =============================================================================
// =============================================================================

func TestSPList_Pagination(t *testing.T) {
	mgr := &mockServiceProfileManager{
		listFn: func(_ context.Context, oltID string, params domain.ServiceProfileListParams) (*domain.ServiceProfileListResult, error) {
			if oltID != "olt-1" {
				t.Errorf("expected olt_id olt-1, got %s", oltID)
			}
			if params.Page != 2 {
				t.Errorf("expected page 2, got %d", params.Page)
			}
			return &domain.ServiceProfileListResult{
				Data:  []*domain.ServiceProfileResponse{sampleSPResponse()},
				Total: 10, Page: 2, PageSize: 5, TotalPages: 2,
			}, nil
		},
	}
	app := setupSPTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1/service-profiles?page=2&page_size=5", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Tes UpdateServiceProfile - PUT /api/v1/olt/service-profiles/:id
// =============================================================================

func TestSPUpdate_Success(t *testing.T) {
	mgr := &mockServiceProfileManager{
		updateFn: func(_ context.Context, id string, _ domain.UpdateServiceProfileRequest) (*domain.ServiceProfileResponse, error) {
			if id != "sp-1" {
				t.Errorf("expected id sp-1, got %s", id)
			}
			resp := sampleSPResponse()
			resp.Name = "Profile Updated"
			return resp, nil
		},
	}
	app := setupSPTestApp(mgr)

	body, _ := json.Marshal(domain.UpdateServiceProfileRequest{Name: "Profile Updated"})
	req := httptest.NewRequest("PUT", "/api/v1/olt/service-profiles/sp-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// =============================================================================

func TestSPDelete_Success(t *testing.T) {
	mgr := &mockServiceProfileManager{
		deleteFn: func(_ context.Context, id string) error {
			if id != "sp-1" {
				t.Errorf("expected id sp-1, got %s", id)
			}
			return nil
		},
	}
	app := setupSPTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/service-profiles/sp-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestSPDelete_InUse_409(t *testing.T) {
	mgr := &mockServiceProfileManager{
		deleteFn: func(_ context.Context, _ string) error {
			return domain.ErrServiceProfileInUse
		},
	}
	app := setupSPTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/service-profiles/sp-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	apiResp := parseSPAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "SERVICE_PROFILE_IN_USE" {
		t.Fatalf("expected SERVICE_PROFILE_IN_USE, got %v", apiResp.Error)
	}
}

func TestSPDelete_NotFound_404(t *testing.T) {
	mgr := &mockServiceProfileManager{
		deleteFn: func(_ context.Context, _ string) error {
			return domain.ErrServiceProfileNotFound
		},
	}
	app := setupSPTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/service-profiles/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// =============================================================================
// =============================================================================

func TestSPCreate_ProfileExists_409(t *testing.T) {
	mgr := &mockServiceProfileManager{
		createFn: func(_ context.Context, _ string, _ domain.CreateServiceProfileRequest) (*domain.ServiceProfileResponse, error) {
			return nil, domain.ErrServiceProfileExists
		},
	}
	app := setupSPTestApp(mgr)

	body, _ := json.Marshal(domain.CreateServiceProfileRequest{
		Name: "Profile 10Mbps", LineProfileID: 1, ServiceProfileID: 1,
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/service-profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	apiResp := parseSPAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "SERVICE_PROFILE_EXISTS" {
		t.Fatalf("expected SERVICE_PROFILE_EXISTS, got %v", apiResp.Error)
	}
}
