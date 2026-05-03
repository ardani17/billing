// vpn_handler_test.go — unit test untuk VPNHandler.
// Menggunakan mock VPNManager dan Fiber app.Test() untuk HTTP testing.
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
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock VPNManager — mengembalikan respons yang sudah ditentukan
// =============================================================================

type mockVPNManager struct {
	createTunnelFn    func(ctx context.Context, tenantID string, req domain.CreateVPNTunnelRequest) (*domain.VPNTunnelResponse, error)
	getTunnelFn       func(ctx context.Context, id string) (*domain.VPNTunnelDetailResponse, error)
	updateTunnelFn    func(ctx context.Context, id string, req domain.UpdateVPNTunnelRequest) (*domain.VPNTunnelResponse, error)
	deleteTunnelFn    func(ctx context.Context, id string) error
	listTunnelsFn     func(ctx context.Context, params domain.VPNTunnelListParams) (*domain.VPNTunnelListResult, error)
	getSummaryFn      func(ctx context.Context) (*domain.VPNSummary, error)
	testConnectionFn  func(ctx context.Context, id string) (*domain.VPNTestResult, error)
	autoConfigureFn   func(ctx context.Context, id string) error
	generateScriptFn  func(ctx context.Context, id string) (string, error)
	getBandwidthFn    func(ctx context.Context, id string, from, to time.Time) (*domain.VPNBandwidthResult, error)
	updateRouterHostFn func(ctx context.Context, tunnelID string) error
}

func (m *mockVPNManager) CreateTunnel(ctx context.Context, tenantID string, req domain.CreateVPNTunnelRequest) (*domain.VPNTunnelResponse, error) {
	if m.createTunnelFn != nil {
		return m.createTunnelFn(ctx, tenantID, req)
	}
	return nil, nil
}

func (m *mockVPNManager) GetTunnel(ctx context.Context, id string) (*domain.VPNTunnelDetailResponse, error) {
	if m.getTunnelFn != nil {
		return m.getTunnelFn(ctx, id)
	}
	return nil, nil
}

func (m *mockVPNManager) UpdateTunnel(ctx context.Context, id string, req domain.UpdateVPNTunnelRequest) (*domain.VPNTunnelResponse, error) {
	if m.updateTunnelFn != nil {
		return m.updateTunnelFn(ctx, id, req)
	}
	return nil, nil
}

func (m *mockVPNManager) DeleteTunnel(ctx context.Context, id string) error {
	if m.deleteTunnelFn != nil {
		return m.deleteTunnelFn(ctx, id)
	}
	return nil
}

func (m *mockVPNManager) ListTunnels(ctx context.Context, params domain.VPNTunnelListParams) (*domain.VPNTunnelListResult, error) {
	if m.listTunnelsFn != nil {
		return m.listTunnelsFn(ctx, params)
	}
	return nil, nil
}

func (m *mockVPNManager) GetSummary(ctx context.Context) (*domain.VPNSummary, error) {
	if m.getSummaryFn != nil {
		return m.getSummaryFn(ctx)
	}
	return nil, nil
}

func (m *mockVPNManager) TestConnection(ctx context.Context, id string) (*domain.VPNTestResult, error) {
	if m.testConnectionFn != nil {
		return m.testConnectionFn(ctx, id)
	}
	return nil, nil
}

func (m *mockVPNManager) AutoConfigure(ctx context.Context, id string) error {
	if m.autoConfigureFn != nil {
		return m.autoConfigureFn(ctx, id)
	}
	return nil
}

func (m *mockVPNManager) GenerateScript(ctx context.Context, id string) (string, error) {
	if m.generateScriptFn != nil {
		return m.generateScriptFn(ctx, id)
	}
	return "", nil
}

func (m *mockVPNManager) GetBandwidth(ctx context.Context, id string, from, to time.Time) (*domain.VPNBandwidthResult, error) {
	if m.getBandwidthFn != nil {
		return m.getBandwidthFn(ctx, id, from, to)
	}
	return nil, nil
}

func (m *mockVPNManager) UpdateRouterHost(ctx context.Context, tunnelID string) error {
	if m.updateRouterHostFn != nil {
		return m.updateRouterHostFn(ctx, tunnelID)
	}
	return nil
}

// =============================================================================
// Helper — setup test Fiber app untuk VPN handler
// =============================================================================

// setupVPNTestApp membuat Fiber app dengan VPNHandler dan middleware
// yang menyuntikkan tenant_id ke context (bypass JWT auth).
func setupVPNTestApp(mgr domain.VPNManager) *fiber.App {
	app := fiber.New()
	logger := zerolog.Nop()
	handler := NewVPNHandler(mgr, logger)

	// Middleware test: set tenant_id di Go context
	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), testTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	// Daftarkan route sesuai spesifikasi
	vpn := app.Group("/api/v1/mikrotik/vpn")
	vpn.Get("/tunnels", handler.ListTunnels)
	vpn.Post("/tunnels", handler.CreateTunnel)
	vpn.Get("/tunnels/:id", handler.GetTunnel)
	vpn.Put("/tunnels/:id", handler.UpdateTunnel)
	vpn.Delete("/tunnels/:id", handler.DeleteTunnel)
	vpn.Post("/tunnels/:id/test", handler.TestConnection)
	vpn.Post("/tunnels/:id/auto-configure", handler.AutoConfigure)
	vpn.Get("/tunnels/:id/script", handler.GenerateScript)
	vpn.Get("/tunnels/:id/bandwidth", handler.GetBandwidth)
	vpn.Get("/summary", handler.GetSummary)

	return app
}

// =============================================================================
// Test CreateTunnel — POST /api/v1/mikrotik/vpn/tunnels
// =============================================================================

// TestVPN_CreateTunnel_Success memverifikasi request valid menghasilkan 201.
func TestVPN_CreateTunnel_Success(t *testing.T) {
	now := time.Now()
	mgr := &mockVPNManager{
		createTunnelFn: func(_ context.Context, tenantID string, req domain.CreateVPNTunnelRequest) (*domain.VPNTunnelResponse, error) {
			if tenantID != testTenantID {
				t.Errorf("expected tenant %s, got %s", testTenantID, tenantID)
			}
			return &domain.VPNTunnelResponse{
				ID: "tunnel-1", TunnelName: req.TunnelName,
				Protocol: domain.VPNProtocol(req.Protocol), VPNIP: "10.99.1.2",
				Status: domain.TunnelStatusPending, CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	app := setupVPNTestApp(mgr)

	body, _ := json.Marshal(domain.CreateVPNTunnelRequest{
		TunnelName: "Tunnel Utama", Protocol: "wireguard",
	})
	req := httptest.NewRequest("POST", "/api/v1/mikrotik/vpn/tunnels", bytes.NewReader(body))
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

// TestVPN_CreateTunnel_InvalidProtocol memverifikasi protokol tidak valid menghasilkan 422.
func TestVPN_CreateTunnel_InvalidProtocol(t *testing.T) {
	mgr := &mockVPNManager{}
	app := setupVPNTestApp(mgr)

	body, _ := json.Marshal(map[string]string{
		"tunnel_name": "Test", "protocol": "invalid_proto",
	})
	req := httptest.NewRequest("POST", "/api/v1/mikrotik/vpn/tunnels", bytes.NewReader(body))
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

// TestVPN_CreateTunnel_MissingFields memverifikasi body tanpa field required menghasilkan 422.
func TestVPN_CreateTunnel_MissingFields(t *testing.T) {
	mgr := &mockVPNManager{}
	app := setupVPNTestApp(mgr)

	body, _ := json.Marshal(map[string]string{"notes": "test"})
	req := httptest.NewRequest("POST", "/api/v1/mikrotik/vpn/tunnels", bytes.NewReader(body))
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
// Test GetTunnel — GET /api/v1/mikrotik/vpn/tunnels/:id
// =============================================================================

// TestVPN_GetTunnel_NotFound memverifikasi tunnel tidak ditemukan menghasilkan 404.
func TestVPN_GetTunnel_NotFound(t *testing.T) {
	mgr := &mockVPNManager{
		getTunnelFn: func(_ context.Context, _ string) (*domain.VPNTunnelDetailResponse, error) {
			return nil, domain.ErrVPNTunnelNotFound
		},
	}
	app := setupVPNTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/vpn/tunnels/nonexistent", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	apiResp := parseAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VPN_TUNNEL_NOT_FOUND" {
		t.Fatalf("expected VPN_TUNNEL_NOT_FOUND, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test DeleteTunnel — DELETE /api/v1/mikrotik/vpn/tunnels/:id
// =============================================================================

// TestVPN_DeleteTunnel_Success memverifikasi delete berhasil menghasilkan 204.
func TestVPN_DeleteTunnel_Success(t *testing.T) {
	mgr := &mockVPNManager{
		deleteTunnelFn: func(_ context.Context, id string) error {
			if id != "tunnel-1" {
				t.Errorf("expected id tunnel-1, got %s", id)
			}
			return nil
		},
	}
	app := setupVPNTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/mikrotik/vpn/tunnels/tunnel-1", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// =============================================================================
// Test GenerateScript — GET /api/v1/mikrotik/vpn/tunnels/:id/script
// =============================================================================

// TestVPN_GenerateScript_ContentType memverifikasi script download mengembalikan text/plain.
func TestVPN_GenerateScript_ContentType(t *testing.T) {
	mgr := &mockVPNManager{
		generateScriptFn: func(_ context.Context, id string) (string, error) {
			return "# RouterOS VPN Script\n/interface wireguard add name=wg-ispboss", nil
		},
	}
	app := setupVPNTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/mikrotik/vpn/tunnels/tunnel-1/script", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verifikasi Content-Type text/plain
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("expected Content-Type 'text/plain; charset=utf-8', got '%s'", contentType)
	}

	// Verifikasi Content-Disposition attachment
	disposition := resp.Header.Get("Content-Disposition")
	expected := "attachment; filename=\"vpn-tunnel-tunnel-1.rsc\""
	if disposition != expected {
		t.Fatalf("expected Content-Disposition '%s', got '%s'", expected, disposition)
	}

	// Verifikasi body berisi script
	bodyBytes, _ := io.ReadAll(resp.Body)
	if len(bodyBytes) == 0 {
		t.Fatal("expected non-empty script body")
	}
}

// =============================================================================
// Test Error Mapping — verifikasi domain errors → HTTP status codes
// =============================================================================

// TestVPN_ErrorMapping memverifikasi berbagai domain error dipetakan ke status code yang benar.
func TestVPN_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"TunnelNotFound", domain.ErrVPNTunnelNotFound, 404, "VPN_TUNNEL_NOT_FOUND"},
		{"TunnelNameExists", domain.ErrVPNTunnelNameExists, 409, "VPN_TUNNEL_NAME_EXISTS"},
		{"VPNIPExists", domain.ErrVPNIPExists, 409, "VPN_IP_EXISTS"},
		{"InvalidProtocol", domain.ErrInvalidVPNProtocol, 422, "INVALID_VPN_PROTOCOL"},
		{"WireGuardRequiresV7", domain.ErrWireGuardRequiresV7, 422, "WIREGUARD_REQUIRES_V7"},
		{"ImmutableField", domain.ErrTunnelImmutableField, 422, "TUNNEL_IMMUTABLE_FIELD"},
		{"SubnetExhausted", domain.ErrVPNSubnetExhausted, 422, "VPN_SUBNET_EXHAUSTED"},
		{"RouterNotOnline", domain.ErrRouterNotOnline, 503, "ROUTER_NOT_ONLINE"},
		{"AutoConfigFailed", domain.ErrAutoConfigFailed, 503, "AUTO_CONFIG_FAILED"},
		{"DeleteWarning", domain.ErrTunnelDeleteWarning, 409, "TUNNEL_DELETE_WARNING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &mockVPNManager{
				getTunnelFn: func(_ context.Context, _ string) (*domain.VPNTunnelDetailResponse, error) {
					return nil, tt.err
				},
			}
			app := setupVPNTestApp(mgr)

			req := httptest.NewRequest("GET", "/api/v1/mikrotik/vpn/tunnels/test-id", nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("request gagal: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				respBody, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, resp.StatusCode, string(respBody))
			}

			apiResp := parseAPIResponse(t, resp.Body)
			if apiResp.Error == nil || apiResp.Error.Code != tt.wantCode {
				t.Fatalf("expected code %s, got %v", tt.wantCode, apiResp.Error)
			}
		})
	}
}
