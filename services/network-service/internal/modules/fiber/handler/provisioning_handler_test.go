// provisioning_handler_test.go - unit test untuk ProvisioningHandler.
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
// =============================================================================

type mockProvisioningManager struct {
	provisionONTFn        func(ctx context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ONTResponse, error)
	previewProvisionFn    func(ctx context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ProvisioningDryRun, error)
	decommissionONTFn     func(ctx context.Context, ontID string, performedBy string) error
	rebootONTFn           func(ctx context.Context, ontID string, performedBy string) (*domain.ProvisioningResult, error)
	validateBulkFn        func(ctx context.Context, tenantID string, oltID string, csvData []byte) (*domain.BulkPreview, error)
	executeBulkFn         func(ctx context.Context, bulkID string, performedBy string) (*domain.BulkResult, error)
	getBulkTemplateFn     func() []byte
	handleUnregisteredFn  func(ctx context.Context, oltID string, ont domain.UnregisteredONT) error
	handlePortMigrationFn func(ctx context.Context, ontID string, oldPort, newPort, oldIdx, newIdx int) error
	confirmMigrationFn    func(ctx context.Context, ontID string) error
	handleCustTermFn      func(ctx context.Context, customerID, tenantID string) error
	getONTByIDFn          func(ctx context.Context, id string) (*domain.ONTDetailResponse, error)
	listONTsFn            func(ctx context.Context, params domain.ONTListParams) (*domain.ONTListResult, error)
	getUnregisteredFn     func(ctx context.Context, oltID string) ([]*domain.ONTResponse, error)
	getAuditLogsFn        func(ctx context.Context, params domain.AuditLogListParams) (*domain.AuditLogListResult, error)
	getSettingsFn         func(ctx context.Context, tenantID string) (*domain.ProvisioningSettings, error)
	updateSettingsFn      func(ctx context.Context, tenantID string, req domain.UpdateSettingsRequest) (*domain.ProvisioningSettings, error)
}

func (m *mockProvisioningManager) ProvisionONT(ctx context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ONTResponse, error) {
	if m.provisionONTFn != nil {
		return m.provisionONTFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockProvisioningManager) PreviewProvisionONT(ctx context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ProvisioningDryRun, error) {
	if m.previewProvisionFn != nil {
		return m.previewProvisionFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockProvisioningManager) DecommissionONT(ctx context.Context, ontID string, performedBy string) error {
	if m.decommissionONTFn != nil {
		return m.decommissionONTFn(ctx, ontID, performedBy)
	}
	return nil
}
func (m *mockProvisioningManager) RebootONT(ctx context.Context, ontID string, performedBy string) (*domain.ProvisioningResult, error) {
	if m.rebootONTFn != nil {
		return m.rebootONTFn(ctx, ontID, performedBy)
	}
	return nil, nil
}
func (m *mockProvisioningManager) ValidateBulk(ctx context.Context, tenantID string, oltID string, csvData []byte) (*domain.BulkPreview, error) {
	if m.validateBulkFn != nil {
		return m.validateBulkFn(ctx, tenantID, oltID, csvData)
	}
	return nil, nil
}
func (m *mockProvisioningManager) ExecuteBulk(ctx context.Context, bulkID string, performedBy string) (*domain.BulkResult, error) {
	if m.executeBulkFn != nil {
		return m.executeBulkFn(ctx, bulkID, performedBy)
	}
	return nil, nil
}
func (m *mockProvisioningManager) GetBulkTemplate() []byte {
	if m.getBulkTemplateFn != nil {
		return m.getBulkTemplateFn()
	}
	return []byte("sn_ont,pelanggan_id,pon_port,vlan,odp,deskripsi\n")
}
func (m *mockProvisioningManager) HandleUnregisteredONT(ctx context.Context, oltID string, ont domain.UnregisteredONT) error {
	if m.handleUnregisteredFn != nil {
		return m.handleUnregisteredFn(ctx, oltID, ont)
	}
	return nil
}
func (m *mockProvisioningManager) HandlePortMigration(ctx context.Context, ontID string, oldPort, newPort, oldIdx, newIdx int) error {
	if m.handlePortMigrationFn != nil {
		return m.handlePortMigrationFn(ctx, ontID, oldPort, newPort, oldIdx, newIdx)
	}
	return nil
}
func (m *mockProvisioningManager) ConfirmMigration(ctx context.Context, ontID string) error {
	if m.confirmMigrationFn != nil {
		return m.confirmMigrationFn(ctx, ontID)
	}
	return nil
}
func (m *mockProvisioningManager) HandleCustomerTerminated(ctx context.Context, customerID, tenantID string) error {
	if m.handleCustTermFn != nil {
		return m.handleCustTermFn(ctx, customerID, tenantID)
	}
	return nil
}
func (m *mockProvisioningManager) GetONTByID(ctx context.Context, id string) (*domain.ONTDetailResponse, error) {
	if m.getONTByIDFn != nil {
		return m.getONTByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockProvisioningManager) ListONTs(ctx context.Context, params domain.ONTListParams) (*domain.ONTListResult, error) {
	if m.listONTsFn != nil {
		return m.listONTsFn(ctx, params)
	}
	return nil, nil
}
func (m *mockProvisioningManager) GetUnregisteredONTs(ctx context.Context, oltID string) ([]*domain.ONTResponse, error) {
	if m.getUnregisteredFn != nil {
		return m.getUnregisteredFn(ctx, oltID)
	}
	return nil, nil
}
func (m *mockProvisioningManager) GetAuditLogs(ctx context.Context, params domain.AuditLogListParams) (*domain.AuditLogListResult, error) {
	if m.getAuditLogsFn != nil {
		return m.getAuditLogsFn(ctx, params)
	}
	return nil, nil
}
func (m *mockProvisioningManager) GetSettings(ctx context.Context, tenantID string) (*domain.ProvisioningSettings, error) {
	if m.getSettingsFn != nil {
		return m.getSettingsFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockProvisioningManager) UpdateSettings(ctx context.Context, tenantID string, req domain.UpdateSettingsRequest) (*domain.ProvisioningSettings, error) {
	if m.updateSettingsFn != nil {
		return m.updateSettingsFn(ctx, tenantID, req)
	}
	return nil, nil
}

// =============================================================================
// =============================================================================

const provTestTenantID = "tenant-prov-test-123"

func setupProvTestApp(mgr domain.ProvisioningManager) *fiber.App {
	app := fiber.New()
	h := NewProvisioningHandler(mgr)

	// Middleware test: atur tenant_id dan username di context
	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), provTestTenantID)
		c.SetUserContext(ctx)
		c.Locals("username", "admin-test")
		return c.Next()
	})

	// Route provisioning
	prov := app.Group("/api/v1/olt/provisioning")
	prov.Post("/ont", h.ProvisionONT)
	prov.Post("/ont/preview", h.PreviewProvisionONT)
	prov.Get("/onts", h.ListONTs)
	prov.Get("/onts/:id", h.GetONT)
	prov.Post("/ont/:id/decommission", h.DecommissionONT)
	prov.Post("/ont/:id/reboot", h.RebootONT)
	prov.Post("/ont/:id/confirm-migration", h.ConfirmMigration)
	prov.Post("/bulk", h.BulkUpload)
	prov.Post("/bulk/execute", h.BulkExecute)
	prov.Get("/bulk/template", h.BulkTemplate)
	prov.Get("/audit-logs", h.GetAuditLogs)
	prov.Get("/settings", h.GetSettings)
	prov.Put("/settings", h.UpdateSettings)

	// Route unregistered ONTs (di bawah devices)
	app.Get("/api/v1/olt/devices/:id/unregistered-onts", h.GetUnregisteredONTs)

	return app
}

func sampleONTResponse() *domain.ONTResponse {
	now := time.Now()
	return &domain.ONTResponse{
		ID: "ont-1", OLTID: "olt-1", PONPortIndex: 0, ONTIndex: 1,
		SerialNumber: "ZTEG12345678", Status: domain.ONTStatusProvisioned,
		ProvisioningState: domain.ProvisioningStateCompleted,
		CreatedAt:         now, UpdatedAt: now,
	}
}

func parseProvAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// =============================================================================
// Tes ProvisionONT - POST /api/v1/olt/provisioning/ont
// =============================================================================

func TestProvisionONT_Success(t *testing.T) {
	mgr := &mockProvisioningManager{
		provisionONTFn: func(_ context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ONTResponse, error) {
			if tenantID != provTestTenantID {
				t.Errorf("expected tenant %s, got %s", provTestTenantID, tenantID)
			}
			if req.SerialNumber != "ZTEG12345678" {
				t.Errorf("expected SN ZTEG12345678, got %s", req.SerialNumber)
			}
			return sampleONTResponse(), nil
		},
	}
	app := setupProvTestApp(mgr)

	body, _ := json.Marshal(domain.ProvisionONTRequest{
		SerialNumber: "ZTEG12345678", OLTID: "550e8400-e29b-41d4-a716-446655440000",
		CustomerID: "550e8400-e29b-41d4-a716-446655440001", ServiceProfileID: "550e8400-e29b-41d4-a716-446655440002",
		VLANID: "550e8400-e29b-41d4-a716-446655440003",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}
	apiResp := parseProvAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestProvisionONT_ValidationError(t *testing.T) {
	app := setupProvTestApp(&mockProvisioningManager{})

	// Body tanpa required field
	body, _ := json.Marshal(map[string]string{"description": "test"})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}
	apiResp := parseProvAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestProvisionONT_MalformedJSON(t *testing.T) {
	app := setupProvTestApp(&mockProvisioningManager{})

	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPreviewProvisionONT_Success(t *testing.T) {
	mgr := &mockProvisioningManager{
		previewProvisionFn: func(_ context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ProvisioningDryRun, error) {
			if tenantID != provTestTenantID {
				t.Errorf("expected tenant %s, got %s", provTestTenantID, tenantID)
			}
			if req.SerialNumber != "ZTEG12345678" {
				t.Errorf("expected SN ZTEG12345678, got %s", req.SerialNumber)
			}
			return &domain.ProvisioningDryRun{
				OLTID:            req.OLTID,
				Brand:            domain.BrandZTE,
				Transport:        "cli",
				Operation:        "provision_ont_preview",
				PONPortIndex:     req.PONPortIndex,
				ONTIndex:         1,
				VLANID:           100,
				LineProfileID:    1,
				ServiceProfileID: 1,
				Commands:         []string{"interface gpon-olt_1/0", "onu 1 type auto sn ZTEG12345678"},
			}, nil
		},
	}
	app := setupProvTestApp(mgr)

	body, _ := json.Marshal(domain.ProvisionONTRequest{
		SerialNumber: "ZTEG12345678", OLTID: "550e8400-e29b-41d4-a716-446655440000",
		CustomerID: "550e8400-e29b-41d4-a716-446655440001", ServiceProfileID: "550e8400-e29b-41d4-a716-446655440002",
		VLANID: "550e8400-e29b-41d4-a716-446655440003",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
	apiResp := parseProvAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

// =============================================================================
// =============================================================================

func TestProvisionONT_SNExists_409(t *testing.T) {
	mgr := &mockProvisioningManager{
		provisionONTFn: func(_ context.Context, _ string, _ domain.ProvisionONTRequest) (*domain.ONTResponse, error) {
			return nil, domain.ErrONTSerialNumberExists
		},
	}
	app := setupProvTestApp(mgr)

	body, _ := json.Marshal(domain.ProvisionONTRequest{
		SerialNumber: "ZTEG12345678", OLTID: "550e8400-e29b-41d4-a716-446655440000",
		CustomerID: "550e8400-e29b-41d4-a716-446655440001", ServiceProfileID: "550e8400-e29b-41d4-a716-446655440002",
		VLANID: "550e8400-e29b-41d4-a716-446655440003",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	apiResp := parseProvAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ONT_SN_EXISTS" {
		t.Fatalf("expected ONT_SN_EXISTS, got %v", apiResp.Error)
	}
}

func TestGetONT_NotFound_404(t *testing.T) {
	mgr := &mockProvisioningManager{
		getONTByIDFn: func(_ context.Context, _ string) (*domain.ONTDetailResponse, error) {
			return nil, domain.ErrONTNotFound
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/provisioning/onts/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	apiResp := parseProvAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "ONT_NOT_FOUND" {
		t.Fatalf("expected ONT_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestDecommissionONT_ProvisioningFailed_502(t *testing.T) {
	mgr := &mockProvisioningManager{
		decommissionONTFn: func(_ context.Context, _ string, _ string) error {
			return domain.ErrDecommissionFailed
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont/ont-1/decommission", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
	apiResp := parseProvAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "DECOMMISSION_FAILED" {
		t.Fatalf("expected DECOMMISSION_FAILED, got %v", apiResp.Error)
	}
}

func TestRebootONT_NotProvisioned_422(t *testing.T) {
	mgr := &mockProvisioningManager{
		rebootONTFn: func(_ context.Context, _ string, _ string) (*domain.ProvisioningResult, error) {
			return nil, domain.ErrONTNotProvisioned
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont/ont-1/reboot", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestRebootONT_CLITimeout_504(t *testing.T) {
	mgr := &mockProvisioningManager{
		rebootONTFn: func(_ context.Context, _ string, _ string) (*domain.ProvisioningResult, error) {
			return nil, domain.ErrCLITimeout
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont/ont-1/reboot", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", resp.StatusCode)
	}
}

func TestProvisionONT_InternalError_500(t *testing.T) {
	mgr := &mockProvisioningManager{
		provisionONTFn: func(_ context.Context, _ string, _ domain.ProvisionONTRequest) (*domain.ONTResponse, error) {
			return nil, errors.New("unexpected error")
		},
	}
	app := setupProvTestApp(mgr)

	body, _ := json.Marshal(domain.ProvisionONTRequest{
		SerialNumber: "ZTEG12345678", OLTID: "550e8400-e29b-41d4-a716-446655440000",
		CustomerID: "550e8400-e29b-41d4-a716-446655440001", ServiceProfileID: "550e8400-e29b-41d4-a716-446655440002",
		VLANID: "550e8400-e29b-41d4-a716-446655440003",
	})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/ont", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

// =============================================================================
// =============================================================================

func TestListONTs_Pagination(t *testing.T) {
	mgr := &mockProvisioningManager{
		listONTsFn: func(_ context.Context, params domain.ONTListParams) (*domain.ONTListResult, error) {
			if params.Page != 2 {
				t.Errorf("expected page 2, got %d", params.Page)
			}
			if params.PageSize != 5 {
				t.Errorf("expected page_size 5, got %d", params.PageSize)
			}
			if params.TenantID != provTestTenantID {
				t.Errorf("expected tenant %s, got %s", provTestTenantID, params.TenantID)
			}
			return &domain.ONTListResult{
				Data:  []*domain.ONTResponse{sampleONTResponse()},
				Total: 10, Page: 2, PageSize: 5, TotalPages: 2,
			}, nil
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/provisioning/onts?page=2&page_size=5", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// =============================================================================
// Tes GetSettings / UpdateSettings
// =============================================================================

func TestGetSettings_Success(t *testing.T) {
	mgr := &mockProvisioningManager{
		getSettingsFn: func(_ context.Context, tenantID string) (*domain.ProvisioningSettings, error) {
			return domain.DefaultProvisioningSettings(tenantID), nil
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/provisioning/settings", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateSettings_InvalidStrategy_422(t *testing.T) {
	app := setupProvTestApp(&mockProvisioningManager{})

	body, _ := json.Marshal(map[string]string{"vlan_strategy": "invalid_strategy"})
	req := httptest.NewRequest("PUT", "/api/v1/olt/provisioning/settings", bytes.NewReader(body))
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

func TestBulkTemplate_Success(t *testing.T) {
	mgr := &mockProvisioningManager{
		getBulkTemplateFn: func() []byte {
			return []byte("sn_ont,pelanggan_id,pon_port,vlan,odp,deskripsi\n")
		},
	}
	app := setupProvTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/olt/provisioning/bulk/template", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("expected Content-Type text/csv, got %s", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); cd == "" {
		t.Fatal("expected Content-Disposition header")
	}
}

// =============================================================================
// Tes BulkExecute - POST /api/v1/olt/provisioning/bulk/execute
// =============================================================================

func TestBulkExecute_MissingBulkID_400(t *testing.T) {
	app := setupProvTestApp(&mockProvisioningManager{})

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/bulk/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBulkExecute_NotFound_404(t *testing.T) {
	mgr := &mockProvisioningManager{
		executeBulkFn: func(_ context.Context, _ string, _ string) (*domain.BulkResult, error) {
			return nil, domain.ErrBulkNotFound
		},
	}
	app := setupProvTestApp(mgr)

	body, _ := json.Marshal(map[string]string{"bulk_id": "nonexistent"})
	req := httptest.NewRequest("POST", "/api/v1/olt/provisioning/bulk/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
