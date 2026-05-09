// map_node_handler_test.go - unit test untuk MapNodeHandler.
// Tenant context di-bypass dengan middleware test yang memanggil pkg/tenant.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

type mockMapNodeManager struct {
	createNodeFn          func(ctx context.Context, tenantID string, req domain.CreateMapNodeRequest) (*domain.MapNodeResponse, error)
	getNodeFn             func(ctx context.Context, id string) (*domain.MapNodeDetailResponse, error)
	updateNodeFn          func(ctx context.Context, id string, req domain.UpdateMapNodeRequest) (*domain.MapNodeResponse, error)
	deleteNodeFn          func(ctx context.Context, id string, performedBy string) error
	listNodesFn           func(ctx context.Context, params domain.MapNodeListParams) ([]*domain.MapNodeWithRefResponse, error)
	searchFn              func(ctx context.Context, tenantID, query string) ([]*domain.MapSearchResult, error)
	uploadPhotoFn         func(ctx context.Context, nodeID string, file multipart.File, header *multipart.FileHeader, caption, uploadedBy string) (*domain.NodePhotoResponse, error)
	listPhotosFn          func(ctx context.Context, nodeID string) ([]*domain.NodePhotoResponse, error)
	deletePhotoFn         func(ctx context.Context, nodeID, photoID, performedBy string) error
	getHistoryFn          func(ctx context.Context, nodeID string, limit, offset int) ([]*domain.MapChangeHistoryResponse, error)
	listTrashedFn         func(ctx context.Context, tenantID string) ([]*domain.MapNodeResponse, error)
	restoreNodeFn         func(ctx context.Context, id, performedBy string) error
	getLabelSettingsFn    func(ctx context.Context, tenantID string) (*domain.MapLabelSettingsResponse, error)
	updateLabelSettingsFn func(ctx context.Context, tenantID string, req domain.UpdateLabelSettingsRequest) (*domain.MapLabelSettingsResponse, error)
}

func (m *mockMapNodeManager) CreateNode(ctx context.Context, tenantID string, req domain.CreateMapNodeRequest) (*domain.MapNodeResponse, error) {
	if m.createNodeFn != nil {
		return m.createNodeFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockMapNodeManager) GetNode(ctx context.Context, id string) (*domain.MapNodeDetailResponse, error) {
	if m.getNodeFn != nil {
		return m.getNodeFn(ctx, id)
	}
	return nil, nil
}
func (m *mockMapNodeManager) UpdateNode(ctx context.Context, id string, req domain.UpdateMapNodeRequest) (*domain.MapNodeResponse, error) {
	if m.updateNodeFn != nil {
		return m.updateNodeFn(ctx, id, req)
	}
	return nil, nil
}
func (m *mockMapNodeManager) DeleteNode(ctx context.Context, id string, performedBy string) error {
	if m.deleteNodeFn != nil {
		return m.deleteNodeFn(ctx, id, performedBy)
	}
	return nil
}
func (m *mockMapNodeManager) ListNodes(ctx context.Context, params domain.MapNodeListParams) ([]*domain.MapNodeWithRefResponse, error) {
	if m.listNodesFn != nil {
		return m.listNodesFn(ctx, params)
	}
	return nil, nil
}
func (m *mockMapNodeManager) Search(ctx context.Context, tenantID, query string) ([]*domain.MapSearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, tenantID, query)
	}
	return nil, nil
}
func (m *mockMapNodeManager) UploadPhoto(ctx context.Context, nodeID string, file multipart.File, header *multipart.FileHeader, caption, uploadedBy string) (*domain.NodePhotoResponse, error) {
	if m.uploadPhotoFn != nil {
		return m.uploadPhotoFn(ctx, nodeID, file, header, caption, uploadedBy)
	}
	return nil, nil
}
func (m *mockMapNodeManager) ListPhotos(ctx context.Context, nodeID string) ([]*domain.NodePhotoResponse, error) {
	if m.listPhotosFn != nil {
		return m.listPhotosFn(ctx, nodeID)
	}
	return nil, nil
}
func (m *mockMapNodeManager) DeletePhoto(ctx context.Context, nodeID, photoID, performedBy string) error {
	if m.deletePhotoFn != nil {
		return m.deletePhotoFn(ctx, nodeID, photoID, performedBy)
	}
	return nil
}
func (m *mockMapNodeManager) GetHistory(ctx context.Context, nodeID string, limit, offset int) ([]*domain.MapChangeHistoryResponse, error) {
	if m.getHistoryFn != nil {
		return m.getHistoryFn(ctx, nodeID, limit, offset)
	}
	return nil, nil
}
func (m *mockMapNodeManager) ListTrashed(ctx context.Context, tenantID string) ([]*domain.MapNodeResponse, error) {
	if m.listTrashedFn != nil {
		return m.listTrashedFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockMapNodeManager) RestoreNode(ctx context.Context, id, performedBy string) error {
	if m.restoreNodeFn != nil {
		return m.restoreNodeFn(ctx, id, performedBy)
	}
	return nil
}
func (m *mockMapNodeManager) GetLabelSettings(ctx context.Context, tenantID string) (*domain.MapLabelSettingsResponse, error) {
	if m.getLabelSettingsFn != nil {
		return m.getLabelSettingsFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockMapNodeManager) UpdateLabelSettings(ctx context.Context, tenantID string, req domain.UpdateLabelSettingsRequest) (*domain.MapLabelSettingsResponse, error) {
	if m.updateLabelSettingsFn != nil {
		return m.updateLabelSettingsFn(ctx, tenantID, req)
	}
	return nil, nil
}

// =============================================================================
// =============================================================================

const mapTestTenantID = "tenant-map-test-123"

// setupMapNodeTestApp membuat Fiber app dengan MapNodeHandler.
func setupMapNodeTestApp(mgr domain.MapNodeManager) *fiber.App {
	app := fiber.New()
	handler := NewMapNodeHandler(mgr)

	// Middleware test: atur tenant_id di Go context
	app.Use(func(c *fiber.Ctx) error {
		ctx := tenant.SetForTest(c.UserContext(), mapTestTenantID)
		c.SetUserContext(ctx)
		return c.Next()
	})

	nodes := app.Group("/api/v1/network-map/nodes")
	nodes.Get("/", handler.ListNodes)
	nodes.Post("/", handler.CreateNode)
	nodes.Get("/:id", handler.GetNode)
	nodes.Put("/:id", handler.UpdateNode)
	nodes.Delete("/:id", handler.DeleteNode)
	nodes.Get("/:id/photos", handler.ListPhotos)
	nodes.Delete("/:id/photos/:photo_id", handler.DeletePhoto)
	nodes.Get("/:id/history", handler.GetHistory)

	return app
}

func parseMapAPIResponse(t *testing.T, body io.Reader) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("gagal parse response body: %v", err)
	}
	return resp
}

// sampleMapNodeResponse mengembalikan MapNodeResponse contoh untuk testing.
func sampleMapNodeResponse() *domain.MapNodeResponse {
	now := time.Now()
	return &domain.MapNodeResponse{
		ID:          "node-1",
		NodeType:    "odp",
		ReferenceID: "ref-1",
		Latitude:    -6.2088,
		Longitude:   106.8456,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// =============================================================================
// Tes CreateNode - POST /api/v1/network-map/nodes
// =============================================================================

func TestMapNodeCreate_Success(t *testing.T) {
	mgr := &mockMapNodeManager{
		createNodeFn: func(_ context.Context, tenantID string, _ domain.CreateMapNodeRequest) (*domain.MapNodeResponse, error) {
			if tenantID != mapTestTenantID {
				t.Errorf("expected tenant %s, got %s", mapTestTenantID, tenantID)
			}
			return sampleMapNodeResponse(), nil
		},
	}
	app := setupMapNodeTestApp(mgr)

	body, _ := json.Marshal(domain.CreateMapNodeRequest{
		NodeType:    "odp",
		ReferenceID: "550e8400-e29b-41d4-a716-446655440000",
		Latitude:    -6.2088,
		Longitude:   106.8456,
	})

	req := httptest.NewRequest("POST", "/api/v1/network-map/nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseMapAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestMapNodeCreate_ValidationError(t *testing.T) {
	app := setupMapNodeTestApp(&mockMapNodeManager{})

	// Body tanpa field required
	body, _ := json.Marshal(map[string]string{"notes": "test"})
	req := httptest.NewRequest("POST", "/api/v1/network-map/nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}

	apiResp := parseMapAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestMapNodeCreate_MalformedJSON(t *testing.T) {
	app := setupMapNodeTestApp(&mockMapNodeManager{})

	req := httptest.NewRequest("POST", "/api/v1/network-map/nodes", bytes.NewReader([]byte("bukan json")))
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

func TestMapNodeCreate_DuplicateError(t *testing.T) {
	mgr := &mockMapNodeManager{
		createNodeFn: func(_ context.Context, _ string, _ domain.CreateMapNodeRequest) (*domain.MapNodeResponse, error) {
			return nil, domain.ErrMapNodeDuplicate
		},
	}
	app := setupMapNodeTestApp(mgr)

	body, _ := json.Marshal(domain.CreateMapNodeRequest{
		NodeType: "odp", ReferenceID: "550e8400-e29b-41d4-a716-446655440000",
		Latitude: -6.2088, Longitude: 106.8456,
	})
	req := httptest.NewRequest("POST", "/api/v1/network-map/nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	apiResp := parseMapAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "MAP_NODE_DUPLICATE" {
		t.Fatalf("expected MAP_NODE_DUPLICATE, got %v", apiResp.Error)
	}
}

func TestMapNodeGetByID_NotFound(t *testing.T) {
	mgr := &mockMapNodeManager{
		getNodeFn: func(_ context.Context, _ string) (*domain.MapNodeDetailResponse, error) {
			return nil, domain.ErrMapNodeNotFound
		},
	}
	app := setupMapNodeTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/nodes/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	apiResp := parseMapAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "MAP_NODE_NOT_FOUND" {
		t.Fatalf("expected MAP_NODE_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestMapNodeGetByID_Deleted(t *testing.T) {
	mgr := &mockMapNodeManager{
		getNodeFn: func(_ context.Context, _ string) (*domain.MapNodeDetailResponse, error) {
			return nil, domain.ErrMapNodeDeleted
		},
	}
	app := setupMapNodeTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/nodes/deleted-node", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusGone {
		t.Fatalf("expected 410, got %d", resp.StatusCode)
	}

	apiResp := parseMapAPIResponse(t, resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "MAP_NODE_DELETED" {
		t.Fatalf("expected MAP_NODE_DELETED, got %v", apiResp.Error)
	}
}

func TestMapNodeDelete_Success(t *testing.T) {
	mgr := &mockMapNodeManager{
		deleteNodeFn: func(_ context.Context, id string, _ string) error {
			if id != "node-1" {
				t.Errorf("expected id node-1, got %s", id)
			}
			return nil
		},
	}
	app := setupMapNodeTestApp(mgr)

	req := httptest.NewRequest("DELETE", "/api/v1/network-map/nodes/node-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestMapNodeErrorMapping_InternalError(t *testing.T) {
	mgr := &mockMapNodeManager{
		getNodeFn: func(_ context.Context, _ string) (*domain.MapNodeDetailResponse, error) {
			return nil, errors.New("unexpected error")
		},
	}
	app := setupMapNodeTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/nodes/node-1", nil)
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

func TestMapNodeGetByID_ResponseFormat(t *testing.T) {
	now := time.Now()
	mgr := &mockMapNodeManager{
		getNodeFn: func(_ context.Context, id string) (*domain.MapNodeDetailResponse, error) {
			return &domain.MapNodeDetailResponse{
				MapNodeResponse: domain.MapNodeResponse{
					ID: id, NodeType: "odp", ReferenceID: "ref-1",
					Latitude: -6.2088, Longitude: 106.8456,
					CreatedAt: now, UpdatedAt: now,
				},
				Photos:  []domain.NodePhotoResponse{},
				History: []domain.MapChangeHistoryResponse{},
			}, nil
		},
	}
	app := setupMapNodeTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/nodes/node-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	apiResp := parseMapAPIResponse(t, resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
	if apiResp.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
}

func TestMapNodeList_Success(t *testing.T) {
	mgr := &mockMapNodeManager{
		listNodesFn: func(_ context.Context, params domain.MapNodeListParams) ([]*domain.MapNodeWithRefResponse, error) {
			if params.TenantID != mapTestTenantID {
				t.Errorf("expected tenant %s, got %s", mapTestTenantID, params.TenantID)
			}
			return []*domain.MapNodeWithRefResponse{}, nil
		},
	}
	app := setupMapNodeTestApp(mgr)

	req := httptest.NewRequest("GET", "/api/v1/network-map/nodes?min_lat=-7&max_lat=-6&min_lng=106&max_lng=107", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
