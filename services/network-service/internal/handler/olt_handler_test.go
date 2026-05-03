// olt_handler_test.go — unit test untuk OLTHandler dan ODPHandler.
// Menggunakan mock OLTManager/ODPManager/AlarmManager dan Fiber app.Test().
// Tenant context di-bypass dengan middleware test yang memanggil pkg/tenant.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock OLTManager
// =============================================================================

type mockOLTManager struct {
	createFn           func(ctx context.Context, tenantID string, req domain.CreateOLTRequest) (*domain.OLTResponse, error)
	getByIDFn          func(ctx context.Context, id string) (*domain.OLTDetailResponse, error)
	updateFn           func(ctx context.Context, id string, req domain.UpdateOLTRequest) (*domain.OLTResponse, error)
	deleteFn           func(ctx context.Context, id string) error
	listFn             func(ctx context.Context, params domain.OLTListParams) (*domain.OLTListResult, error)
	testSNMPFn         func(ctx context.Context, id string) (*domain.OLTSystemInfo, error)
	testCLIFn          func(ctx context.Context, id string) (*domain.CLITestResult, error)
	getStatusSummaryFn func(ctx context.Context) (*domain.OLTStatusSummary, error)
	getPONPortsFn      func(ctx context.Context, oltID string) ([]domain.PONPortStatus, error)
	getONTListFn       func(ctx context.Context, oltID string, portIndex int) ([]domain.ONTPortStatus, error)
	getSFPStatusFn     func(ctx context.Context, oltID string) ([]domain.SFPInfo, error)
	getCapacityFn      func(ctx context.Context, oltID string) (*domain.OLTCapacity, error)
}

func (m *mockOLTManager) Create(ctx context.Context, tenantID string, req domain.CreateOLTRequest) (*domain.OLTResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockOLTManager) GetByID(ctx context.Context, id string) (*domain.OLTDetailResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockOLTManager) Update(ctx context.Context, id string, req domain.UpdateOLTRequest) (*domain.OLTResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}
func (m *mockOLTManager) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockOLTManager) List(ctx context.Context, params domain.OLTListParams) (*domain.OLTListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, nil
}
func (m *mockOLTManager) TestSNMP(ctx context.Context, id string) (*domain.OLTSystemInfo, error) {
	if m.testSNMPFn != nil {
		return m.testSNMPFn(ctx, id)
	}
	return nil, nil
}
func (m *mockOLTManager) TestCLI(ctx context.Context, id string) (*domain.CLITestResult, error) {
	if m.testCLIFn != nil {
		return m.testCLIFn(ctx, id)
	}
	return nil, nil
}
func (m *mockOLTManager) GetStatusSummary(ctx context.Context) (*domain.OLTStatusSummary, error) {
	if m.getStatusSummaryFn != nil {
		return m.getStatusSummaryFn(ctx)
	}
	return nil, nil
}
func (m *mockOLTManager) GetPONPorts(ctx context.Context, oltID string) ([]domain.PONPortStatus, error) {
	if m.getPONPortsFn != nil {
		return m.getPONPortsFn(ctx, oltID)
	}
	return nil, nil
}
func (m *mockOLTManager) GetONTList(ctx context.Context, oltID string, portIndex int) ([]domain.ONTPortStatus, error) {
	if m.getONTListFn != nil {
		return m.getONTListFn(ctx, oltID, portIndex)
	}
	return nil, nil
}
func (m *mockOLTManager) GetSFPStatus(ctx context.Context, oltID string) ([]domain.SFPInfo, error) {
	if m.getSFPStatusFn != nil {
		return m.getSFPStatusFn(ctx, oltID)
	}
	return nil, nil
}
func (m *mockOLTManager) GetCapacity(ctx context.Context, oltID string) (*domain.OLTCapacity, error) {
	if m.getCapacityFn != nil {
		return m.getCapacityFn(ctx, oltID)
	}
	return nil, nil
}

// =============================================================================
// Mock AlarmManager
// =============================================================================

type mockAlarmManager struct {
	getAlarmsFn func(ctx context.Context, oltID string, params domain.AlarmListParams) (*domain.AlarmListResult, error)
}

func (m *mockAlarmManager) StartTrapReceiver(_ context.Context) error { return nil }
func (m *mockAlarmManager) StopTrapReceiver()                        {}
func (m *mockAlarmManager) PollAlarms(_ context.Context, _ string) ([]domain.OLTAlarm, error) {
	return nil, nil
}
func (m *mockAlarmManager) GetAlarms(ctx context.Context, oltID string, params domain.AlarmListParams) (*domain.AlarmListResult, error) {
	if m.getAlarmsFn != nil {
		return m.getAlarmsFn(ctx, oltID, params)
	}
	return nil, nil
}
func (m *mockAlarmManager) PurgeOldAlarms(_ context.Context) (int64, error) { return 0, nil }

// =============================================================================
// Mock ODPManager
// =============================================================================

type mockODPManager struct {
	createFn  func(ctx context.Context, tenantID string, req domain.CreateODPRequest) (*domain.ODPResponse, error)
	getByIDFn func(ctx context.Context, id string) (*domain.ODPDetailResponse, error)
	updateFn  func(ctx context.Context, id string, req domain.UpdateODPRequest) (*domain.ODPResponse, error)
	deleteFn  func(ctx context.Context, id string) error
	listFn    func(ctx context.Context, params domain.ODPListParams) (*domain.ODPListResult, error)
}

func (m *mockODPManager) Create(ctx context.Context, tenantID string, req domain.CreateODPRequest) (*domain.ODPResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockODPManager) GetByID(ctx context.Context, id string) (*domain.ODPDetailResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockODPManager) Update(ctx context.Context, id string, req domain.UpdateODPRequest) (*domain.ODPResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}
func (m *mockODPManager) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockODPManager) List(ctx context.Context, params domain.ODPListParams) (*domain.ODPListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, nil
}

// =============================================================================
// Helper — setup test Fiber app untuk OLT dan ODP
// =============================================================================

const oltTestTenantID = "tenant-olt-test-123"

// setupOLTTestApp membuat Fiber app dengan OLTHandler dan ODPHandler.
func setupOLTTestApp(oltMgr domain.OLTManager, alarmMgr domain.AlarmManager, odpMgr domain.ODPManager) *fiber.App {
	app := fiber.New()
	oltHandler := NewOLTHandler(oltMgr, alarmMgr)
	odpHandler := NewODPHandler(odpMgr)

	// Middleware test: set tenant_id di Go context
	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), oltTestTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	// Route OLT
	devices := app.Group("/api/v1/olt/devices")
	devices.Post("/", oltHandler.CreateOLT)
	devices.Get("/", oltHandler.ListOLTs)
	devices.Get("/:id", oltHandler.GetOLT)
	devices.Put("/:id", oltHandler.UpdateOLT)
	devices.Delete("/:id", oltHandler.DeleteOLT)
	devices.Post("/:id/test-snmp", oltHandler.TestSNMP)
	devices.Post("/:id/test-cli", oltHandler.TestCLI)
	devices.Get("/:id/pon-ports", oltHandler.GetPONPorts)
	devices.Get("/:id/pon-ports/:port/onts", oltHandler.GetONTList)
	devices.Get("/:id/pon-ports/:port/traffic", oltHandler.GetTraffic)
	devices.Get("/:id/alarms", oltHandler.GetAlarms)
	devices.Get("/:id/sfp", oltHandler.GetSFP)
	devices.Get("/:id/capacity", oltHandler.GetCapacity)

	// Route Summary
	app.Get("/api/v1/olt/summary", oltHandler.GetSummary)

	// Route ODP
	odp := app.Group("/api/v1/olt/odp")
	odp.Post("/", odpHandler.CreateODP)
	odp.Get("/", odpHandler.ListODPs)
	odp.Get("/:id", odpHandler.GetODP)
	odp.Put("/:id", odpHandler.UpdateODP)
	odp.Delete("/:id", odpHandler.DeleteODP)

	return app
}

// sampleOLTResponse mengembalikan OLTResponse contoh untuk testing.
func sampleOLTResponse() *domain.OLTResponse {
	now := time.Now()
	return &domain.OLTResponse{
		ID:                     "olt-1",
		Name:                   "OLT Utama",
		Host:                   "10.0.0.1",
		Brand:                  domain.BrandZTE,
		Model:                  "C320",
		Status:                 domain.OLTStatusOnline,
		PONPortCount:           8,
		TotalONTCount:          245,
		HealthCheckIntervalSec: 300,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

// parseOLTAPIResponse mem-parse body respons ke domain.APIResponse.
func parseOLTAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// =============================================================================
// Test OLT CreateOLT — POST /api/v1/olt/devices
// =============================================================================

func TestOLTCreate_Success(t *testing.T) {
	oltMgr := &mockOLTManager{
		createFn: func(_ context.Context, tenantID string, _ domain.CreateOLTRequest) (*domain.OLTResponse, error) {
			if tenantID != oltTestTenantID {
				t.Errorf("expected tenant %s, got %s", oltTestTenantID, tenantID)
			}
			return sampleOLTResponse(), nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	body, _ := json.Marshal(domain.CreateOLTRequest{
		Name:        "OLT Utama",
		Host:        "10.0.0.1",
		SNMPVersion: "v2c",
		CLIProtocol: "ssh",
		CLIPort:     22,
		CLIUsername:  "admin",
		CLIPassword:  "secret",
	})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseOLTAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestOLTCreate_InvalidBody(t *testing.T) {
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, &mockODPManager{})

	body, _ := json.Marshal(map[string]string{"notes": "test"})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestOLTCreate_MalformedJSON(t *testing.T) {
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestOLTCreate_DuplicateName(t *testing.T) {
	oltMgr := &mockOLTManager{
		createFn: func(_ context.Context, _ string, _ domain.CreateOLTRequest) (*domain.OLTResponse, error) {
			return nil, domain.ErrOLTNameExists
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	body, _ := json.Marshal(domain.CreateOLTRequest{
		Name: "OLT Duplikat", Host: "10.0.0.2", SNMPVersion: "v2c",
		CLIProtocol: "ssh", CLIPort: 22, CLIUsername: "admin", CLIPassword: "secret",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "OLT_NAME_EXISTS" {
		t.Fatalf("expected OLT_NAME_EXISTS, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test OLT GetByID — GET /api/v1/olt/devices/:id
// =============================================================================

func TestOLTGetByID_Success(t *testing.T) {
	oltMgr := &mockOLTManager{
		getByIDFn: func(_ context.Context, id string) (*domain.OLTDetailResponse, error) {
			if id != "olt-1" {
				t.Errorf("expected id olt-1, got %s", id)
			}
			return &domain.OLTDetailResponse{
				OLTResponse: *sampleOLTResponse(),
				SNMPVersion: domain.SNMPv2c,
				CLIProtocol: domain.CLIProtocolSSH,
				CLIPort:     22,
			}, nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestOLTGetByID_NotFound(t *testing.T) {
	oltMgr := &mockOLTManager{
		getByIDFn: func(_ context.Context, _ string) (*domain.OLTDetailResponse, error) {
			return nil, domain.ErrOLTNotFound
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "OLT_NOT_FOUND" {
		t.Fatalf("expected OLT_NOT_FOUND, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test OLT Update — PUT /api/v1/olt/devices/:id
// =============================================================================

func TestOLTUpdate_Success(t *testing.T) {
	resp := sampleOLTResponse()
	resp.Name = "OLT Updated"
	oltMgr := &mockOLTManager{
		updateFn: func(_ context.Context, id string, _ domain.UpdateOLTRequest) (*domain.OLTResponse, error) {
			if id != "olt-1" {
				t.Errorf("expected id olt-1, got %s", id)
			}
			return resp, nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	body, _ := json.Marshal(domain.UpdateOLTRequest{Name: "OLT Updated"})
	req := httptest.NewRequest("PUT", "/api/v1/olt/devices/olt-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if httpResp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		t.Fatalf("expected 200, got %d: %s", httpResp.StatusCode, string(respBody))
	}
}

func TestOLTUpdate_InvalidTransition(t *testing.T) {
	oltMgr := &mockOLTManager{
		updateFn: func(_ context.Context, _ string, _ domain.UpdateOLTRequest) (*domain.OLTResponse, error) {
			return nil, domain.ErrOLTInvalidStatusTransition
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	body, _ := json.Marshal(domain.UpdateOLTRequest{Status: "maintenance"})
	req := httptest.NewRequest("PUT", "/api/v1/olt/devices/olt-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test OLT Delete — DELETE /api/v1/olt/devices/:id
// =============================================================================

func TestOLTDelete_Success(t *testing.T) {
	oltMgr := &mockOLTManager{
		deleteFn: func(_ context.Context, id string) error {
			if id != "olt-1" {
				t.Errorf("expected id olt-1, got %s", id)
			}
			return nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("DELETE", "/api/v1/olt/devices/olt-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestOLTDelete_NotFound(t *testing.T) {
	oltMgr := &mockOLTManager{
		deleteFn: func(_ context.Context, _ string) error {
			return domain.ErrOLTNotFound
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("DELETE", "/api/v1/olt/devices/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test OLT List — GET /api/v1/olt/devices
// =============================================================================

func TestOLTList_Pagination(t *testing.T) {
	oltMgr := &mockOLTManager{
		listFn: func(_ context.Context, params domain.OLTListParams) (*domain.OLTListResult, error) {
			if params.Page != 2 {
				t.Errorf("expected page 2, got %d", params.Page)
			}
			if params.PageSize != 5 {
				t.Errorf("expected page_size 5, got %d", params.PageSize)
			}
			if params.TenantID != oltTestTenantID {
				t.Errorf("expected tenant %s, got %s", oltTestTenantID, params.TenantID)
			}
			return &domain.OLTListResult{
				Data: []*domain.OLTResponse{sampleOLTResponse()},
				Total: 10, Page: 2, PageSize: 5, TotalPages: 2,
			}, nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices?page=2&page_size=5", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestOLTList_FilterByStatus(t *testing.T) {
	oltMgr := &mockOLTManager{
		listFn: func(_ context.Context, params domain.OLTListParams) (*domain.OLTListResult, error) {
			if params.Status != "online" {
				t.Errorf("expected status 'online', got '%s'", params.Status)
			}
			if params.Brand != "zte" {
				t.Errorf("expected brand 'zte', got '%s'", params.Brand)
			}
			return &domain.OLTListResult{
				Data: []*domain.OLTResponse{sampleOLTResponse()},
				Total: 1, Page: 1, PageSize: 20, TotalPages: 1,
			}, nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices?status=online&brand=zte", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test Error Mapping — domain errors → HTTP status codes
// =============================================================================

func TestOLTErrorMapping_SNMPConnectionFailed(t *testing.T) {
	oltMgr := &mockOLTManager{
		testSNMPFn: func(_ context.Context, _ string) (*domain.OLTSystemInfo, error) {
			return nil, domain.ErrSNMPConnectionFailed
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/test-snmp", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "SNMP_CONNECTION_FAILED" {
		t.Fatalf("expected SNMP_CONNECTION_FAILED, got %v", apiResp.Error)
	}
}

func TestOLTErrorMapping_SNMPTimeout(t *testing.T) {
	oltMgr := &mockOLTManager{
		testSNMPFn: func(_ context.Context, _ string) (*domain.OLTSystemInfo, error) {
			return nil, domain.ErrSNMPTimeout
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/test-snmp", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
}

func TestOLTErrorMapping_CLIConnectionFailed(t *testing.T) {
	oltMgr := &mockOLTManager{
		testCLIFn: func(_ context.Context, _ string) (*domain.CLITestResult, error) {
			return nil, domain.ErrCLIConnectionFailed
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/test-cli", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "CLI_CONNECTION_FAILED" {
		t.Fatalf("expected CLI_CONNECTION_FAILED, got %v", apiResp.Error)
	}
}

func TestOLTErrorMapping_UnsupportedBrand(t *testing.T) {
	oltMgr := &mockOLTManager{
		testSNMPFn: func(_ context.Context, _ string) (*domain.OLTSystemInfo, error) {
			return nil, domain.ErrUnsupportedBrand
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/devices/olt-1/test-snmp", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestOLTErrorMapping_InternalError(t *testing.T) {
	oltMgr := &mockOLTManager{
		getByIDFn: func(_ context.Context, _ string) (*domain.OLTDetailResponse, error) {
			return nil, errors.New("unexpected error")
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test Monitoring Endpoints
// =============================================================================

func TestOLTGetPONPorts_Success(t *testing.T) {
	oltMgr := &mockOLTManager{
		getPONPortsFn: func(_ context.Context, oltID string) ([]domain.PONPortStatus, error) {
			if oltID != "olt-1" {
				t.Errorf("expected olt-1, got %s", oltID)
			}
			return []domain.PONPortStatus{
				{PortIndex: 0, AdminStatus: "up", OperStatus: "up", ONTCount: 32},
			}, nil
		},
	}
	app := setupOLTTestApp(oltMgr, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1/pon-ports", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestOLTGetONTList_InvalidPort(t *testing.T) {
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1/pon-ports/abc/onts", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestOLTGetTraffic_InvalidFromParam(t *testing.T) {
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1/pon-ports/0/traffic?from=invalid", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestOLTGetAlarms_Success(t *testing.T) {
	alarmMgr := &mockAlarmManager{
		getAlarmsFn: func(_ context.Context, oltID string, params domain.AlarmListParams) (*domain.AlarmListResult, error) {
			if oltID != "olt-1" {
				t.Errorf("expected olt-1, got %s", oltID)
			}
			return &domain.AlarmListResult{
				Data: []*domain.OLTAlarmRecord{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0,
			}, nil
		},
	}
	app := setupOLTTestApp(&mockOLTManager{}, alarmMgr, &mockODPManager{})

	req := httptest.NewRequest("GET", "/api/v1/olt/devices/olt-1/alarms", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Test ODP CRUD
// =============================================================================

func TestODPCreate_Success(t *testing.T) {
	now := time.Now()
	odpMgr := &mockODPManager{
		createFn: func(_ context.Context, tenantID string, _ domain.CreateODPRequest) (*domain.ODPResponse, error) {
			if tenantID != oltTestTenantID {
				t.Errorf("expected tenant %s, got %s", oltTestTenantID, tenantID)
			}
			return &domain.ODPResponse{
				ID: "odp-1", OLTID: "olt-1", Name: "ODP-01",
				SplitterType: "1:8", Capacity: 8, CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, odpMgr)

	body, _ := json.Marshal(domain.CreateODPRequest{
		OLTID: "550e8400-e29b-41d4-a716-446655440000", PONPortIndex: 1,
		Name: "ODP-01", SplitterType: "1:8",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/odp", bytes.NewReader(body))
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

func TestODPCreate_InvalidBody(t *testing.T) {
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, &mockODPManager{})

	body, _ := json.Marshal(map[string]string{"notes": "test"})
	req := httptest.NewRequest("POST", "/api/v1/olt/odp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestODPGetByID_NotFound(t *testing.T) {
	odpMgr := &mockODPManager{
		getByIDFn: func(_ context.Context, _ string) (*domain.ODPDetailResponse, error) {
			return nil, domain.ErrODPNotFound
		},
	}
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, odpMgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/odp/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ODP_NOT_FOUND" {
		t.Fatalf("expected ODP_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestODPDelete_Success(t *testing.T) {
	odpMgr := &mockODPManager{
		deleteFn: func(_ context.Context, id string) error {
			if id != "odp-1" {
				t.Errorf("expected id odp-1, got %s", id)
			}
			return nil
		},
	}
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, odpMgr)

	req := httptest.NewRequest("DELETE", "/api/v1/olt/odp/odp-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestODPErrorMapping_ODPFull(t *testing.T) {
	odpMgr := &mockODPManager{
		createFn: func(_ context.Context, _ string, _ domain.CreateODPRequest) (*domain.ODPResponse, error) {
			return nil, domain.ErrODPFull
		},
	}
	app := setupOLTTestApp(&mockOLTManager{}, &mockAlarmManager{}, odpMgr)

	body, _ := json.Marshal(domain.CreateODPRequest{
		OLTID: "550e8400-e29b-41d4-a716-446655440000", PONPortIndex: 1,
		Name: "ODP-02", SplitterType: "1:8",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/odp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	apiResp := parseOLTAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ODP_FULL" {
		t.Fatalf("expected ODP_FULL, got %v", apiResp.Error)
	}
}
