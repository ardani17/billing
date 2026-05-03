// vlan_handler_test.go — unit test untuk VLANHandler.
// Menggunakan mock VLANManager dan Fiber app.Test().
// Validasi: CRUD endpoints, validation, error mapping.
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
// Mock VLANManager
// =============================================================================

type mockVLANManager struct {
	createFn     func(ctx context.Context, tenantID string, req domain.CreateVLANRequest) (*domain.VLANResponse, error)
	getByIDFn    func(ctx context.Context, id string) (*domain.VLANResponse, error)
	updateFn     func(ctx context.Context, id string, req domain.UpdateVLANRequest) (*domain.VLANResponse, error)
	deleteFn     func(ctx context.Context, id string) error
	listFn       func(ctx context.Context, oltID string, params domain.VLANListParams) (*domain.VLANListResult, error)
	resolveVLANFn func(ctx context.Context, oltID string, strategy domain.VLANStrategy, rctx domain.VLANResolveContext) (*domain.VLAN, error)
}

func (m *mockVLANManager) Create(ctx context.Context, tenantID string, req domain.CreateVLANRequest) (*domain.VLANResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockVLANManager) GetByID(ctx context.Context, id string) (*domain.VLANResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockVLANManager) Update(ctx context.Context, id string, req domain.UpdateVLANRequest) (*domain.VLANResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}
func (m *mockVLANManager) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockVLANManager) List(ctx context.Context, oltID string, params domain.VLANListParams) (*domain.VLANListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, oltID, params)
	}
	return nil, nil
}
func (m *mockVLANManager) ResolveVLAN(ctx context.Context, oltID string, strategy domain.VLANStrategy, rctx domain.VLANResolveContext) (*domain.VLAN, error) {
	if m.resolveVLANFn != nil {
		return m.resolveVLANFn(ctx, oltID, strategy, rctx)
	}
	return nil, nil
}

// =============================================================================
// Helper — setup test Fiber app untuk VLAN
// =============================================================================

const vlanTestTenantID = "tenant-vlan-test-123"

func setupVLANTestApp(mgr domain.VLANManager) *fiber.App {
	app := fiber.New()
	h := NewVLANHandler(mgr)

	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), vlanTestTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	// Route VLAN per OLT
	app.Post("/api/v1/olt/devices/:id/vlans", h.CreateVLAN)
	app.Get("/api/v1/olt/devices/:id/vlans", h.ListVLANs)
	// Route VLAN standalone
	app.Put("/api/v1/olt/vlans/:id", h.UpdateVLAN)
	app.Delete("/api/v1/olt/vlans/:id", h.DeleteVLAN)

	return app
}

func sampleVLANResponse() *domain.VLANResponse {
	now := time.Now()
	return &domain.VLANResponse{
		ID: "vlan-1", OLTID: "olt-1", VLANID: 100, Name: "VLAN Data",
		VLANType: "data", ActiveONTs: 5, CreatedAt: now, UpdatedAt: now,
	}
}

func parseVLANAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// =============================================================================
// Test CreateVLAN — POST /api/v1/olt/devices/:id/vlans
// =============================================================================

func TestVLANCreate_Success(t *testing.T) {
	mgr := &mockVLANManager{
		createFn: func(_ context.Context, tenantID string, req domain.CreateVLANRequest) (*domain.VLANResponse, error) {
			if tenantID != vlanTestTenantID {
				t.Errorf("expected tenant %s, got %s", vlanTestTenantID, tenantID)
			}
			if req.OLTID != "olt-1" {
				t.Errorf("expected olt_id olt-1, got %s", req.OLTID)
			}
			return sampleVLANResponse(), nil
		},
	}
	app := setupVLANTestApp(mgr)

	body, _ := json.Marshal(domain.CreateVLANRequest{
		VLANID: 100, Name: "VLAN Data", VLANType: "data",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/vlans", bytes.NewReader(body))
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

func TestVLANCreate_ValidationError(t *testing.T) {
	app := setupVLANTestApp(&mockVLANManager{})

	// Body tanpa required fields
	body, _ := json.Marshal(map[string]string{"description": "test"})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/vlans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	apiResp := parseVLANAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestVLANCreate_MalformedJSON(t *testing.T) {
	app := setupVLANTestApp(&mockVLANManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/vlans", bytes.NewReader([]byte("bukan json")))
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
// Test ListVLANs — GET /api/v1/olt/devices/:id/vlans
// =============================================================================

func TestVLANList_Pagination(t *testing.T) {
	mgr := &mockVLANManager{
		listFn: func(_ context.Context, oltID string, params domain.VLANListParams) (*domain.VLANListResult, error) {
			if oltID != "olt-1" {
				t.Errorf("expected olt_id olt-1, got %s", oltID)
			}
			if params.Page != 2 {
				t.Errorf("expected page 2, got %d", params.Page)
			}
			return &domain.VLANListResult{
				Data: []*domain.VLANResponse{sampleVLANResponse()},
				Total: 10, Page: 2, PageSize: 5, TotalPages: 2,
			}, nil
		},
	}
	app := setupVLANTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1/vlans?page=2&page_size=5", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test UpdateVLAN — PUT /api/v1/olt/vlans/:id
// =============================================================================

func TestVLANUpdate_Success(t *testing.T) {
	mgr := &mockVLANManager{
		updateFn: func(_ context.Context, id string, _ domain.UpdateVLANRequest) (*domain.VLANResponse, error) {
			if id != "vlan-1" {
				t.Errorf("expected id vlan-1, got %s", id)
			}
			resp := sampleVLANResponse()
			resp.Name = "VLAN Updated"
			return resp, nil
		},
	}
	app := setupVLANTestApp(mgr)

	body, _ := json.Marshal(domain.UpdateVLANRequest{Name: "VLAN Updated"})
	req := httptest.NewRequest("PUT", "/api/v1/olt/vlans/vlan-1", bytes.NewReader(body))
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
// Test DeleteVLAN — DELETE /api/v1/olt/vlans/:id
// =============================================================================

func TestVLANDelete_Success(t *testing.T) {
	mgr := &mockVLANManager{
		deleteFn: func(_ context.Context, id string) error {
			if id != "vlan-1" {
				t.Errorf("expected id vlan-1, got %s", id)
			}
			return nil
		},
	}
	app := setupVLANTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/vlans/vlan-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestVLANDelete_InUse_409(t *testing.T) {
	mgr := &mockVLANManager{
		deleteFn: func(_ context.Context, _ string) error {
			return domain.ErrVLANInUse
		},
	}
	app := setupVLANTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/vlans/vlan-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	apiResp := parseVLANAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VLAN_IN_USE" {
		t.Fatalf("expected VLAN_IN_USE, got %v", apiResp.Error)
	}
}

func TestVLANDelete_NotFound_404(t *testing.T) {
	mgr := &mockVLANManager{
		deleteFn: func(_ context.Context, _ string) error {
			return domain.ErrVLANNotFound
		},
	}
	app := setupVLANTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/vlans/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test Error Mapping — VLAN ID exists → 409
// =============================================================================

func TestVLANCreate_IDExists_409(t *testing.T) {
	mgr := &mockVLANManager{
		createFn: func(_ context.Context, _ string, _ domain.CreateVLANRequest) (*domain.VLANResponse, error) {
			return nil, domain.ErrVLANIDExists
		},
	}
	app := setupVLANTestApp(mgr)

	body, _ := json.Marshal(domain.CreateVLANRequest{
		VLANID: 100, Name: "VLAN Data", VLANType: "data",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/vlans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	apiResp := parseVLANAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VLAN_ID_EXISTS" {
		t.Fatalf("expected VLAN_ID_EXISTS, got %v", apiResp.Error)
	}
}
