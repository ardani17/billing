package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// --- Mock repositories for area handler tests ---

type mockHandlerAreaRepo struct {
	areas map[string]*domain.Area
}

func newMockHandlerAreaRepo() *mockHandlerAreaRepo {
	return &mockHandlerAreaRepo{areas: make(map[string]*domain.Area)}
}

func (m *mockHandlerAreaRepo) Create(_ context.Context, area *domain.Area) (*domain.Area, error) {
	if area.ID == "" {
		area.ID = fmt.Sprintf("area-%d", len(m.areas)+1)
	}
	copy := *area
	m.areas[copy.ID] = &copy
	return &copy, nil
}

func (m *mockHandlerAreaRepo) GetByID(_ context.Context, id string) (*domain.Area, error) {
	a, ok := m.areas[id]
	if !ok {
		return nil, domain.ErrAreaNotFound
	}
	copy := *a
	return &copy, nil
}

func (m *mockHandlerAreaRepo) Update(_ context.Context, area *domain.Area) (*domain.Area, error) {
	if _, ok := m.areas[area.ID]; !ok {
		return nil, domain.ErrAreaNotFound
	}
	copy := *area
	m.areas[copy.ID] = &copy
	return &copy, nil
}

func (m *mockHandlerAreaRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.areas[id]; !ok {
		return domain.ErrAreaNotFound
	}
	delete(m.areas, id)
	return nil
}

func (m *mockHandlerAreaRepo) List(_ context.Context, _ string) ([]*domain.Area, error) {
	var result []*domain.Area
	for _, a := range m.areas {
		copy := *a
		result = append(result, &copy)
	}
	return result, nil
}

func (m *mockHandlerAreaRepo) NameExists(_ context.Context, tenantID, name, excludeID string) (bool, error) {
	for _, a := range m.areas {
		if a.TenantID == tenantID && a.Name == name && a.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockHandlerAreaRepo) CustomerCount(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// mockHandlerAreaRepoWithCustomers returns a non-zero customer count for testing.
type mockHandlerAreaRepoWithCustomers struct {
	mockHandlerAreaRepo
	customerCounts map[string]int
}

func newMockHandlerAreaRepoWithCustomers() *mockHandlerAreaRepoWithCustomers {
	return &mockHandlerAreaRepoWithCustomers{
		mockHandlerAreaRepo: mockHandlerAreaRepo{areas: make(map[string]*domain.Area)},
		customerCounts:      make(map[string]int),
	}
}

func (m *mockHandlerAreaRepoWithCustomers) CustomerCount(_ context.Context, id string) (int, error) {
	return m.customerCounts[id], nil
}

type areaTestSetup struct {
	app      *fiber.App
	areaRepo *mockHandlerAreaRepo
}

func setupAreaTestApp() *areaTestSetup {
	areaRepo := newMockHandlerAreaRepo()
	auditLogRepo := newMockHandlerAuditLogRepo()
	logger := zerolog.New(io.Discard)

	uc := usecase.NewAreaUsecase(areaRepo, auditLogRepo, logger)
	handler := NewAreaHandler(uc, logger)

	app := fiber.New()

	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	areas := app.Group("/api/v1/areas", setLocals)
	areas.Get("/", handler.List)
	areas.Post("/", handler.Create)
	areas.Get("/:id", handler.Get)
	areas.Put("/:id", handler.Update)
	areas.Delete("/:id", handler.Delete)

	return &areaTestSetup{
		app:      app,
		areaRepo: areaRepo,
	}
}

// --- Area Create tests ---

func TestAreaHandler_Create_Success(t *testing.T) {
	setup := setupAreaTestApp()

	body, _ := json.Marshal(domain.CreateAreaRequest{
		Name:        "Area Sukamaju",
		Description: "Wilayah Sukamaju",
	})

	req := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestAreaHandler_Create_ValidationError(t *testing.T) {
	setup := setupAreaTestApp()

	// Name too short (min 2)
	body, _ := json.Marshal(domain.CreateAreaRequest{
		Name: "A",
	})

	req := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestAreaHandler_Create_DuplicateName(t *testing.T) {
	setup := setupAreaTestApp()

	body, _ := json.Marshal(domain.CreateAreaRequest{
		Name: "Area Sukamaju",
	})

	// Create first area
	req1 := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	setup.app.Test(req1, -1)

	// Create second area with same name
	body2, _ := json.Marshal(domain.CreateAreaRequest{
		Name: "Area Sukamaju",
	})
	req2 := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req2, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "AREA_NAME_DUPLICATE" {
		t.Fatalf("expected AREA_NAME_DUPLICATE, got %v", apiResp.Error)
	}
}

// --- Area Get tests ---

func TestAreaHandler_Get_NotFound(t *testing.T) {
	setup := setupAreaTestApp()

	req := httptest.NewRequest("GET", "/api/v1/areas/nonexistent-id", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "AREA_NOT_FOUND" {
		t.Fatalf("expected AREA_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestAreaHandler_Get_Success(t *testing.T) {
	setup := setupAreaTestApp()

	// Create an area first
	body, _ := json.Marshal(domain.CreateAreaRequest{Name: "Area Test"})
	createReq := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Get the area
	getReq := httptest.NewRequest("GET", "/api/v1/areas/"+createApiResp.Data.ID, nil)
	resp, err := setup.app.Test(getReq, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Area List tests ---

func TestAreaHandler_List_Success(t *testing.T) {
	setup := setupAreaTestApp()

	req := httptest.NewRequest("GET", "/api/v1/areas", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Area Update tests ---

func TestAreaHandler_Update_NotFound(t *testing.T) {
	setup := setupAreaTestApp()

	body, _ := json.Marshal(domain.UpdateAreaRequest{Name: "New Name"})
	req := httptest.NewRequest("PUT", "/api/v1/areas/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAreaHandler_Update_Success(t *testing.T) {
	setup := setupAreaTestApp()

	// Create an area first
	createBody, _ := json.Marshal(domain.CreateAreaRequest{Name: "Area Old"})
	createReq := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Update the area
	updateBody, _ := json.Marshal(domain.UpdateAreaRequest{Name: "Area New"})
	updateReq := httptest.NewRequest("PUT", "/api/v1/areas/"+createApiResp.Data.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(updateReq, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// --- Area Delete tests ---

func TestAreaHandler_Delete_NotFound(t *testing.T) {
	setup := setupAreaTestApp()

	req := httptest.NewRequest("DELETE", "/api/v1/areas/nonexistent-id", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAreaHandler_Delete_Success(t *testing.T) {
	setup := setupAreaTestApp()

	// Create an area first
	createBody, _ := json.Marshal(domain.CreateAreaRequest{Name: "Area To Delete"})
	createReq := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Delete the area
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/areas/"+createApiResp.Data.ID, nil)

	resp, err := setup.app.Test(deleteReq, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestAreaHandler_Delete_HasCustomers(t *testing.T) {
	// Use a special repo that returns non-zero customer count
	areaRepo := newMockHandlerAreaRepoWithCustomers()
	auditLogRepo := newMockHandlerAuditLogRepo()
	logger := zerolog.New(io.Discard)

	uc := usecase.NewAreaUsecase(areaRepo, auditLogRepo, logger)
	handler := NewAreaHandler(uc, logger)

	app := fiber.New()
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}
	areas := app.Group("/api/v1/areas", setLocals)
	areas.Post("/", handler.Create)
	areas.Delete("/:id", handler.Delete)

	// Create an area
	createBody, _ := json.Marshal(domain.CreateAreaRequest{Name: "Area With Customers"})
	createReq := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Set customer count for this area
	areaRepo.customerCounts[createApiResp.Data.ID] = 5

	// Try to delete the area
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/areas/"+createApiResp.Data.ID, nil)

	resp, err := app.Test(deleteReq, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "AREA_HAS_CUSTOMERS" {
		t.Fatalf("expected AREA_HAS_CUSTOMERS, got %v", apiResp.Error)
	}
}

func TestAreaHandler_Create_InvalidBody(t *testing.T) {
	setup := setupAreaTestApp()

	req := httptest.NewRequest("POST", "/api/v1/areas", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
