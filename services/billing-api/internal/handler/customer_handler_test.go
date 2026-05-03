package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// Feature: customer-crud, Property 7: Validation Error Aggregation
// **Validates: Requirements 22.8**
//
// For any request body containing multiple invalid fields, the validation
// response SHALL return HTTP 400 with error code VALIDATION_ERROR and an
// array of field-level error details covering ALL invalid fields in a single
// response (not just the first error encountered).

// invalidFieldGenerator generates a CreateCustomerRequest with a random
// subset of fields set to invalid values. It returns the request and the
// set of field names that are expected to fail validation.
func generateInvalidRequest(t *rapid.T) (domain.CreateCustomerRequest, map[string]bool) {
	expectedErrors := make(map[string]bool)

	req := domain.CreateCustomerRequest{
		// Start with valid defaults
		Name:             "Valid Name",
		Phone:            "+6281234567890",
		Address:          "Jl. Valid Address No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "pppoe",
	}

	// Randomly invalidate fields (at least 2 must be invalid)
	invalidCount := 0

	// Name: make invalid (too short)
	if rapid.Bool().Draw(t, "invalidName") {
		req.Name = "AB" // less than 3 chars
		expectedErrors["name"] = true
		invalidCount++
	}

	// Phone: make invalid (wrong format)
	if rapid.Bool().Draw(t, "invalidPhone") {
		req.Phone = "08123" // doesn't start with +62
		expectedErrors["phone"] = true
		invalidCount++
	}

	// Address: make invalid (empty)
	if rapid.Bool().Draw(t, "invalidAddress") {
		req.Address = ""
		expectedErrors["address"] = true
		invalidCount++
	}

	// DueDate: make invalid (out of range)
	if rapid.Bool().Draw(t, "invalidDueDate") {
		req.DueDate = 30 // > 28
		expectedErrors["due_date"] = true
		invalidCount++
	}

	// ConnectionMethod: make invalid
	if rapid.Bool().Draw(t, "invalidConnectionMethod") {
		req.ConnectionMethod = "invalid_method"
		expectedErrors["connection_method"] = true
		invalidCount++
	}

	// PackageID: make invalid (not UUID)
	if rapid.Bool().Draw(t, "invalidPackageID") {
		req.PackageID = "not-a-uuid"
		expectedErrors["package_id"] = true
		invalidCount++
	}

	// ActivationDate: make invalid (wrong format)
	if rapid.Bool().Draw(t, "invalidActivationDate") {
		req.ActivationDate = "not-a-date"
		expectedErrors["activation_date"] = true
		invalidCount++
	}

	// Email: make invalid (bad format)
	if rapid.Bool().Draw(t, "invalidEmail") {
		req.Email = "not-an-email"
		expectedErrors["email"] = true
		invalidCount++
	}

	// Ensure at least 2 fields are invalid
	if invalidCount < 2 {
		req.Name = "AB"
		expectedErrors["name"] = true
		req.Phone = "08123"
		expectedErrors["phone"] = true
	}

	return req, expectedErrors
}

func TestProperty_ValidationErrorAggregation(t *testing.T) {
	v := validator.New()
	RegisterCustomValidators(v)

	// Create a minimal Fiber app for testing
	app := fiber.New()

	// Test endpoint that validates CreateCustomerRequest
	app.Post("/test-validate", func(c *fiber.Ctx) error {
		var req domain.CreateCustomerRequest
		if err := c.BodyParser(&req); err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
		}

		if err := v.Struct(req); err != nil {
			var ve validator.ValidationErrors
			if ok := err.(validator.ValidationErrors); ok != nil {
				ve = ok
				return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
			}
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		}

		return domain.SuccessResponse(c, fiber.StatusOK, nil)
	})

	rapid.Check(t, func(t *rapid.T) {
		req, expectedErrors := generateInvalidRequest(t)

		// Marshal request to JSON
		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		// Create HTTP request
		httpReq := httptest.NewRequest("POST", "/test-validate", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, err := app.Test(httpReq, -1)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		// Property 1: Response status must be 400
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected status 400, got %d. Body: %s", resp.StatusCode, string(respBody))
		}

		// Parse response
		var apiResp domain.APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Property 2: Error code must be VALIDATION_ERROR
		if apiResp.Error == nil {
			t.Fatal("expected error in response, got nil")
		}
		if apiResp.Error.Code != "VALIDATION_ERROR" {
			t.Fatalf("expected error code VALIDATION_ERROR, got %s", apiResp.Error.Code)
		}

		// Property 3: Details must contain errors for ALL invalid fields
		returnedFields := make(map[string]bool)
		for _, detail := range apiResp.Error.Details {
			returnedFields[detail.Field] = true
		}

		for field := range expectedErrors {
			if !returnedFields[field] {
				t.Fatalf("expected validation error for field %q but it was not returned. Expected: %v, Got: %v",
					field, expectedErrors, returnedFields)
			}
		}

		// Property 4: Number of returned errors must be >= number of expected errors
		// (there may be additional errors from dependent validations)
		if len(apiResp.Error.Details) < len(expectedErrors) {
			t.Fatalf("expected at least %d validation errors, got %d. Expected fields: %v, Got details: %v",
				len(expectedErrors), len(apiResp.Error.Details), expectedErrors, apiResp.Error.Details)
		}
	})
}

// --- Unit Tests for CustomerHandler ---

// mockCustomerUsecase is a mock implementation of CustomerUsecase methods
// used for handler-level testing. It wraps a real CustomerUsecase with mock repos.
type testHandlerSetup struct {
	app          *fiber.App
	customerRepo *mockHandlerCustomerRepo
	auditLogRepo *mockHandlerAuditLogRepo
}

// mockHandlerCustomerRepo is a simplified in-memory customer repo for handler tests.
type mockHandlerCustomerRepo struct {
	customers   map[string]*domain.Customer
	seqByTenant map[string]int
}

func newMockHandlerCustomerRepo() *mockHandlerCustomerRepo {
	return &mockHandlerCustomerRepo{
		customers:   make(map[string]*domain.Customer),
		seqByTenant: make(map[string]int),
	}
}

func (m *mockHandlerCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	if c.ID == "" {
		c.ID = fmt.Sprintf("cust-%d", len(m.customers)+1)
	}
	copy := *c
	m.customers[copy.ID] = &copy
	seq := m.seqByTenant[c.TenantID]
	m.seqByTenant[c.TenantID] = seq + 1
	return &copy, nil
}

func (m *mockHandlerCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	return &copy, nil
}

func (m *mockHandlerCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	if _, ok := m.customers[c.ID]; !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}

func (m *mockHandlerCustomerRepo) SoftDelete(_ context.Context, id string) error {
	c, ok := m.customers[id]
	if !ok {
		return domain.ErrCustomerNotFound
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

func (m *mockHandlerCustomerRepo) List(_ context.Context, params domain.CustomerListParams) (*domain.CustomerListResult, error) {
	var filtered []*domain.Customer
	for _, c := range m.customers {
		if c.TenantID != params.TenantID || c.DeletedAt != nil {
			continue
		}
		filtered = append(filtered, c)
	}
	total := int64(len(filtered))
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 25
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	return &domain.CustomerListResult{
		Data: filtered[start:end],
		Pagination: domain.PaginationMeta{
			Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages,
		},
	}, nil
}

func (m *mockHandlerCustomerRepo) UpdateStatus(_ context.Context, id string, status domain.CustomerStatus) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	c.Status = status
	copy := *c
	return &copy, nil
}

func (m *mockHandlerCustomerRepo) UpdatePackage(_ context.Context, id string, packageID string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	c.PackageID = packageID
	copy := *c
	return &copy, nil
}

func (m *mockHandlerCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	result := make(map[domain.CustomerStatus]int64)
	for _, c := range m.customers {
		if c.DeletedAt == nil {
			result[c.Status]++
		}
	}
	return result, nil
}

func (m *mockHandlerCustomerRepo) GetMaxSeq(_ context.Context, tenantID string) (int, error) {
	return m.seqByTenant[tenantID], nil
}

func (m *mockHandlerCustomerRepo) PhoneExists(_ context.Context, tenantID, phone, excludeID string) (bool, error) {
	for _, c := range m.customers {
		if c.TenantID == tenantID && c.Phone == phone && c.ID != excludeID && c.DeletedAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockHandlerCustomerRepo) BulkUpdateStatus(_ context.Context, ids []string, status domain.CustomerStatus) ([]domain.BulkResult, error) {
	var results []domain.BulkResult
	for _, id := range ids {
		if c, ok := m.customers[id]; ok {
			c.Status = status
			results = append(results, domain.BulkResult{ID: id, Success: true})
		} else {
			results = append(results, domain.BulkResult{ID: id, Success: false, Error: domain.ErrCustomerNotFound})
		}
	}
	return results, nil
}

func (m *mockHandlerCustomerRepo) BulkUpdateFields(_ context.Context, ids []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	var results []domain.BulkResult
	for _, id := range ids {
		if _, ok := m.customers[id]; ok {
			results = append(results, domain.BulkResult{ID: id, Success: true})
		} else {
			results = append(results, domain.BulkResult{ID: id, Success: false, Error: domain.ErrCustomerNotFound})
		}
	}
	return results, nil
}

func (m *mockHandlerCustomerRepo) BulkSoftDelete(_ context.Context, ids []string) ([]domain.BulkResult, error) {
	var results []domain.BulkResult
	for _, id := range ids {
		if _, ok := m.customers[id]; ok {
			results = append(results, domain.BulkResult{ID: id, Success: true})
		} else {
			results = append(results, domain.BulkResult{ID: id, Success: false, Error: domain.ErrCustomerNotFound})
		}
	}
	return results, nil
}

func (m *mockHandlerCustomerRepo) GetByIDs(_ context.Context, ids []string) ([]*domain.Customer, error) {
	var result []*domain.Customer
	for _, id := range ids {
		if c, ok := m.customers[id]; ok {
			copy := *c
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockHandlerCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}

// mockHandlerAuditLogRepo is a simplified in-memory audit log repo for handler tests.
type mockHandlerAuditLogRepo struct {
	logs []*domain.AuditLog
}

func newMockHandlerAuditLogRepo() *mockHandlerAuditLogRepo {
	return &mockHandlerAuditLogRepo{logs: make([]*domain.AuditLog, 0)}
}

func (m *mockHandlerAuditLogRepo) Create(_ context.Context, log *domain.AuditLog) error {
	copy := *log
	m.logs = append(m.logs, &copy)
	return nil
}

func (m *mockHandlerAuditLogRepo) ListByEntity(_ context.Context, entityType, entityID string) ([]*domain.AuditLog, error) {
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.EntityType == entityType && l.EntityID == entityID {
			copy := *l
			result = append(result, &copy)
		}
	}
	return result, nil
}

// setupTestApp creates a Fiber app with a real CustomerUsecase backed by mock repos.
func setupTestApp() *testHandlerSetup {
	customerRepo := newMockHandlerCustomerRepo()
	auditLogRepo := newMockHandlerAuditLogRepo()
	logger := zerolog.New(io.Discard)

	uc := usecase.NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)
	handler := NewCustomerHandler(uc, logger)

	app := fiber.New()

	// Set up routes matching the real router (without auth/tenant middleware)
	// We'll set tenant_id/user_id via middleware for testing
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	customers := app.Group("/api/v1/customers", setLocals)
	// Static routes harus didaftarkan sebelum parameterized routes (/:id)
	// agar Fiber tidak mencocokkan "bulk" atau "import" sebagai :id
	customers.Get("/", handler.List)
	customers.Get("/stats", handler.Stats)
	customers.Post("/", handler.Create)
	customers.Get("/import/template", handler.ImportTemplate)
	customers.Post("/bulk/isolir", handler.BulkIsolir)
	customers.Post("/bulk/activate", handler.BulkActivate)
	customers.Post("/bulk/notification", handler.BulkNotify)
	customers.Post("/bulk/change-package", handler.BulkChangePackage)
	customers.Post("/bulk/edit", handler.BulkEdit)
	customers.Delete("/bulk", handler.BulkDelete)
	customers.Get("/:id", handler.Get)
	customers.Put("/:id", handler.Update)
	customers.Delete("/:id", handler.Delete)
	customers.Post("/:id/isolir", handler.Isolir)
	customers.Post("/:id/activate", handler.Activate)
	customers.Post("/:id/change-package", handler.ChangePackage)

	return &testHandlerSetup{
		app:          app,
		customerRepo: customerRepo,
		auditLogRepo: auditLogRepo,
	}
}

// validCreateBody returns a valid JSON body for creating a customer.
func validCreateBody() []byte {
	body, _ := json.Marshal(domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "pppoe",
	})
	return body
}

// --- Create endpoint tests ---

func TestCustomerHandler_Create_Success(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestCustomerHandler_Create_ValidationError(t *testing.T) {
	setup := setupTestApp()

	// Missing required fields
	body, _ := json.Marshal(map[string]interface{}{
		"name": "AB", // too short
	})

	req := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestCustomerHandler_Create_PhoneDuplicate(t *testing.T) {
	setup := setupTestApp()

	// Create first customer
	req1 := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	req1.Header.Set("Content-Type", "application/json")
	setup.app.Test(req1, -1)

	// Create second customer with same phone
	req2 := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	req2.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req2, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "PHONE_DUPLICATE" {
		t.Fatalf("expected PHONE_DUPLICATE, got %v", apiResp.Error)
	}
}

func TestCustomerHandler_Create_InvalidBody(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Get endpoint tests ---

func TestCustomerHandler_Get_NotFound(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/customers/nonexistent-id", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "CUSTOMER_NOT_FOUND" {
		t.Fatalf("expected CUSTOMER_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestCustomerHandler_Get_Success(t *testing.T) {
	setup := setupTestApp()

	// Create a customer first
	createReq := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Get the customer
	getReq := httptest.NewRequest("GET", "/api/v1/customers/"+createApiResp.Data.ID, nil)
	resp, err := setup.app.Test(getReq, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- List endpoint tests ---

func TestCustomerHandler_List_Success(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/customers", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestCustomerHandler_List_InvalidPageSize(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/customers?page_size=99", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Update endpoint tests ---

func TestCustomerHandler_Update_NotFound(t *testing.T) {
	setup := setupTestApp()

	body, _ := json.Marshal(domain.UpdateCustomerRequest{Name: "New Name"})
	req := httptest.NewRequest("PUT", "/api/v1/customers/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Delete endpoint tests ---

func TestCustomerHandler_Delete_ConfirmationMismatch(t *testing.T) {
	setup := setupTestApp()

	// Create a customer first
	createReq := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Delete with wrong confirmation name
	body, _ := json.Marshal(domain.DeleteCustomerRequest{ConfirmationName: "Wrong Name"})
	req := httptest.NewRequest("DELETE", "/api/v1/customers/"+createApiResp.Data.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "CONFIRMATION_MISMATCH" {
		t.Fatalf("expected CONFIRMATION_MISMATCH, got %v", apiResp.Error)
	}
}

func TestCustomerHandler_Delete_NotFound(t *testing.T) {
	setup := setupTestApp()

	body, _ := json.Marshal(domain.DeleteCustomerRequest{ConfirmationName: "Test"})
	req := httptest.NewRequest("DELETE", "/api/v1/customers/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// --- Stats endpoint tests ---

func TestCustomerHandler_Stats_Success(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/customers/stats", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Action endpoint tests ---

func TestCustomerHandler_Isolir_InvalidTransition(t *testing.T) {
	setup := setupTestApp()

	// Create a customer (status: pending)
	createReq := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Try to isolir a pending customer (invalid: pending → isolir not allowed)
	req := httptest.NewRequest("POST", "/api/v1/customers/"+createApiResp.Data.ID+"/isolir", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %v", apiResp.Error)
	}
}

func TestCustomerHandler_Activate_Success(t *testing.T) {
	setup := setupTestApp()

	// Create a customer (status: pending)
	createReq := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Activate (pending → aktif)
	req := httptest.NewRequest("POST", "/api/v1/customers/"+createApiResp.Data.ID+"/activate", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestCustomerHandler_ChangePackage_SamePackage(t *testing.T) {
	setup := setupTestApp()

	// Create a customer
	createReq := httptest.NewRequest("POST", "/api/v1/customers", bytes.NewReader(validCreateBody()))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)

	var createApiResp struct {
		Data struct {
			ID        string `json:"id"`
			PackageID string `json:"package_id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	// Try to change to the same package
	body, _ := json.Marshal(domain.ChangePackageRequest{
		PackageID: "00000000-0000-0000-0000-000000000001",
	})
	req := httptest.NewRequest("POST", "/api/v1/customers/"+createApiResp.Data.ID+"/change-package", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "SAME_PACKAGE" {
		t.Fatalf("expected SAME_PACKAGE, got %v", apiResp.Error)
	}
}

// --- Bulk endpoint tests ---

func TestCustomerHandler_BulkIsolir_InvalidBody(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/customers/bulk/isolir", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCustomerHandler_BulkActivate_ValidationError(t *testing.T) {
	setup := setupTestApp()

	// Empty customer_ids
	body, _ := json.Marshal(domain.BulkIDsRequest{CustomerIDs: []string{}})
	req := httptest.NewRequest("POST", "/api/v1/customers/bulk/activate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Import Template endpoint test ---

func TestCustomerHandler_ImportTemplate_Success(t *testing.T) {
	setup := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/customers/import/template", nil)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/csv" {
		t.Fatalf("expected Content-Type text/csv, got %s", contentType)
	}
}
