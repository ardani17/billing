// share_handler_test.go — unit test untuk ShareHandler.
// Menggunakan mock ShareManager dan Fiber app.Test().
// Test: akses publik (tanpa auth), link kedaluwarsa (410), password salah (401).
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock ShareManager
// =============================================================================

type mockShareManager struct {
	createShareLinkFn func(ctx context.Context, tenantID, createdBy string, req domain.CreateShareLinkRequest) (*domain.ShareLinkResponse, error)
	getSharedMapFn    func(ctx context.Context, token, password string) (*domain.SharedMapData, error)
	deleteShareLinkFn func(ctx context.Context, token string) error
	listShareLinksFn  func(ctx context.Context, tenantID string) ([]*domain.ShareLinkResponse, error)
}

func (m *mockShareManager) CreateShareLink(ctx context.Context, tenantID, createdBy string, req domain.CreateShareLinkRequest) (*domain.ShareLinkResponse, error) {
	if m.createShareLinkFn != nil {
		return m.createShareLinkFn(ctx, tenantID, createdBy, req)
	}
	return nil, nil
}
func (m *mockShareManager) GetSharedMap(ctx context.Context, token, password string) (*domain.SharedMapData, error) {
	if m.getSharedMapFn != nil {
		return m.getSharedMapFn(ctx, token, password)
	}
	return nil, nil
}
func (m *mockShareManager) DeleteShareLink(ctx context.Context, token string) error {
	if m.deleteShareLinkFn != nil {
		return m.deleteShareLinkFn(ctx, token)
	}
	return nil
}
func (m *mockShareManager) ListShareLinks(ctx context.Context, tenantID string) ([]*domain.ShareLinkResponse, error) {
	if m.listShareLinksFn != nil {
		return m.listShareLinksFn(ctx, tenantID)
	}
	return nil, nil
}

// =============================================================================
// Helper — setup test Fiber app untuk ShareHandler
// =============================================================================

const shareTestTenantID = "tenant-share-test-123"

// setupShareTestApp membuat Fiber app dengan ShareHandler.
// Route GET /share/:token bersifat publik (tanpa auth middleware).
func setupShareTestApp(mgr domain.ShareManager) *fiber.App {
	app := fiber.New()
	handler := NewShareHandler(mgr)

	// Route publik — akses shared map tanpa auth
	app.Get("/api/v1/network-map/share/:token", handler.GetSharedMap)

	// Route yang memerlukan auth
	auth := app.Group("/api/v1/network-map")
	auth.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), shareTestTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})
	auth.Post("/share", handler.CreateShareLink)
	auth.Get("/share", handler.ListShareLinks)
	auth.Delete("/share/:token", handler.DeleteShareLink)

	return app
}

// parseShareAPIResponse mem-parse body respons ke domain.APIResponse.
func parseShareAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// =============================================================================
// Test GetSharedMap — akses publik tanpa auth
// =============================================================================

func TestShareGetSharedMap_PublicAccess(t *testing.T) {
	mgr := &mockShareManager{
		getSharedMapFn: func(_ context.Context, token, password string) (*domain.SharedMapData, error) {
			if token != "abc123token" {
				t.Errorf("expected token abc123token, got %s", token)
			}
			return &domain.SharedMapData{
				Nodes:         []domain.MapNodeWithRefResponse{},
				Cables:        []domain.CableRouteResponse{},
				VisibleLayers: json.RawMessage(`["olt","odp"]`),
			}, nil
		},
	}
	app := setupShareTestApp(mgr)

	// Request tanpa header Authorization — harus tetap berhasil
	req := httptest.NewRequest("GET", "/api/v1/network-map/share/abc123token", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseShareAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true untuk akses publik")
	}
}

// =============================================================================
// Test GetSharedMap — link kedaluwarsa (410 Gone)
// =============================================================================

func TestShareGetSharedMap_ExpiredLink(t *testing.T) {
	mgr := &mockShareManager{
		getSharedMapFn: func(_ context.Context, _ string, _ string) (*domain.SharedMapData, error) {
			return nil, domain.ErrShareLinkExpired
		},
	}
	app := setupShareTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/share/expired-token", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusGone {
		t.Fatalf("expected 410, got %d", resp.StatusCode)
	}

	apiResp := parseShareAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "SHARE_LINK_EXPIRED" {
		t.Fatalf("expected SHARE_LINK_EXPIRED, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test GetSharedMap — password salah (401 Unauthorized)
// =============================================================================

func TestShareGetSharedMap_WrongPassword(t *testing.T) {
	mgr := &mockShareManager{
		getSharedMapFn: func(_ context.Context, _ string, _ string) (*domain.SharedMapData, error) {
			return nil, domain.ErrShareLinkPassword
		},
	}
	app := setupShareTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/share/protected-token?password=wrong", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	apiResp := parseShareAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "SHARE_LINK_PASSWORD" {
		t.Fatalf("expected SHARE_LINK_PASSWORD, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test GetSharedMap — link tidak ditemukan (404)
// =============================================================================

func TestShareGetSharedMap_NotFound(t *testing.T) {
	mgr := &mockShareManager{
		getSharedMapFn: func(_ context.Context, _ string, _ string) (*domain.SharedMapData, error) {
			return nil, domain.ErrShareLinkNotFound
		},
	}
	app := setupShareTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/share/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test CreateShareLink — dengan auth
// =============================================================================

func TestShareCreate_Success(t *testing.T) {
	mgr := &mockShareManager{
		createShareLinkFn: func(_ context.Context, tenantID, _ string, _ domain.CreateShareLinkRequest) (*domain.ShareLinkResponse, error) {
			if tenantID != shareTestTenantID {
				t.Errorf("expected tenant %s, got %s", shareTestTenantID, tenantID)
			}
			return &domain.ShareLinkResponse{
				ID:    "link-1",
				Token: "generated-token",
				URL:   "https://app.ispboss.com/share/generated-token",
			}, nil
		},
	}
	app := setupShareTestApp(mgr)

	body, _ := json.Marshal(domain.CreateShareLinkRequest{
		VisibleLayers: json.RawMessage(`["olt","odp","ont"]`),
	})
	req := httptest.NewRequest("POST", "/api/v1/network-map/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseShareAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

// =============================================================================
// Test GetSharedMap — password via header X-Share-Password
// =============================================================================

func TestShareGetSharedMap_PasswordViaHeader(t *testing.T) {
	mgr := &mockShareManager{
		getSharedMapFn: func(_ context.Context, token, password string) (*domain.SharedMapData, error) {
			if password != "correct-password" {
				return nil, domain.ErrShareLinkPassword
			}
			return &domain.SharedMapData{
				Nodes:         []domain.MapNodeWithRefResponse{},
				Cables:        []domain.CableRouteResponse{},
				VisibleLayers: json.RawMessage(`["olt"]`),
			}, nil
		},
	}
	app := setupShareTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/share/protected-token", nil)
	req.Header.Set("X-Share-Password", "correct-password")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}
