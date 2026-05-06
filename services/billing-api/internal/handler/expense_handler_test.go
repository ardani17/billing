// expense_handler_test.go berisi integration tests untuk expense endpoints.
// Test: CRUD operations, soft delete, category constraint, category name duplicate.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock ExpenseUsecase untuk expense handler tests ---

// mockExpenseUsecase mengimplementasikan domain.ExpenseUsecase untuk testing.
type mockExpenseUsecase struct {
	expenses   map[string]*domain.Expense
	categories map[string]*domain.ExpenseCategory
	// expensesByCategory melacak jumlah expense per kategori
	expensesByCategory map[string]int
	seqID              int
}

func newMockExpenseUsecase() *mockExpenseUsecase {
	return &mockExpenseUsecase{
		expenses:           make(map[string]*domain.Expense),
		categories:         make(map[string]*domain.ExpenseCategory),
		expensesByCategory: make(map[string]int),
		seqID:              0,
	}
}

func (m *mockExpenseUsecase) Create(_ context.Context, tenantID string, req domain.CreateExpenseRequest, actor domain.ActorInfo) (*domain.Expense, error) {
	// Cek apakah kategori ada
	if _, ok := m.categories[req.CategoryID]; !ok {
		return nil, domain.ErrExpenseCategoryNotFound
	}

	m.seqID++
	id := fmt.Sprintf("exp-%d", m.seqID)
	expDate, _ := time.Parse("2006-01-02", req.ExpenseDate)
	expense := &domain.Expense{
		ID:           id,
		TenantID:     tenantID,
		CategoryID:   req.CategoryID,
		Amount:       req.Amount,
		Description:  req.Description,
		ExpenseDate:  expDate,
		IsRecurring:  req.IsRecurring,
		RecurringDay: req.RecurringDay,
		CreatedByID:  actor.ActorID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.expenses[id] = expense
	m.expensesByCategory[req.CategoryID]++
	return expense, nil
}

func (m *mockExpenseUsecase) GetByID(_ context.Context, id string) (*domain.Expense, error) {
	e, ok := m.expenses[id]
	if !ok {
		return nil, domain.ErrExpenseNotFound
	}
	copy := *e
	return &copy, nil
}

func (m *mockExpenseUsecase) Update(_ context.Context, id string, req domain.UpdateExpenseRequest, _ domain.ActorInfo) (*domain.Expense, error) {
	e, ok := m.expenses[id]
	if !ok {
		return nil, domain.ErrExpenseNotFound
	}
	if req.Amount != nil {
		e.Amount = *req.Amount
	}
	if req.Description != nil {
		e.Description = *req.Description
	}
	e.UpdatedAt = time.Now()
	copy := *e
	return &copy, nil
}

func (m *mockExpenseUsecase) Delete(_ context.Context, id string, _ domain.ActorInfo) error {
	e, ok := m.expenses[id]
	if !ok {
		return domain.ErrExpenseNotFound
	}
	now := time.Now()
	e.DeletedAt = &now
	return nil
}

func (m *mockExpenseUsecase) List(_ context.Context, tenantID string, _, _ time.Time, _ string) ([]*domain.Expense, error) {
	var result []*domain.Expense
	for _, e := range m.expenses {
		if e.TenantID == tenantID && e.DeletedAt == nil {
			copy := *e
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockExpenseUsecase) ListCategories(_ context.Context, tenantID string) ([]*domain.ExpenseCategory, error) {
	var result []*domain.ExpenseCategory
	for _, c := range m.categories {
		if c.TenantID == tenantID && c.DeletedAt == nil {
			copy := *c
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockExpenseUsecase) CreateCategory(_ context.Context, tenantID, name string) (*domain.ExpenseCategory, error) {
	// Cek duplikasi nama
	for _, c := range m.categories {
		if c.TenantID == tenantID && c.Name == name && c.DeletedAt == nil {
			return nil, domain.ErrCategoryNameDuplicate
		}
	}
	m.seqID++
	id := fmt.Sprintf("cat-%d", m.seqID)
	cat := &domain.ExpenseCategory{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.categories[id] = cat
	return cat, nil
}

func (m *mockExpenseUsecase) UpdateCategory(_ context.Context, id, name string) (*domain.ExpenseCategory, error) {
	c, ok := m.categories[id]
	if !ok {
		return nil, domain.ErrExpenseCategoryNotFound
	}
	c.Name = name
	c.UpdatedAt = time.Now()
	copy := *c
	return &copy, nil
}

func (m *mockExpenseUsecase) DeleteCategory(_ context.Context, id string) error {
	c, ok := m.categories[id]
	if !ok {
		return domain.ErrExpenseCategoryNotFound
	}
	// Cek apakah masih ada expense terkait
	if m.expensesByCategory[id] > 0 {
		return domain.ErrCategoryHasExpenses
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

// --- Setup helper ---

func setupExpenseTestApp(mock *mockExpenseUsecase) *fiber.App {
	logger := zerolog.New(io.Discard)
	handler := NewExpenseHandler(mock, logger)

	app := fiber.New()

	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	expenses := app.Group("/api/v1/expenses", setLocals)
	expenses.Get("/", handler.List)
	expenses.Post("/", handler.Create)
	expenses.Put("/:id", handler.Update)
	expenses.Delete("/:id", handler.Delete)

	categories := app.Group("/api/v1/expenses/categories", setLocals)
	categories.Get("/", handler.ListCategories)
	categories.Post("/", handler.CreateCategory)
	categories.Put("/:id", handler.UpdateCategory)
	categories.Delete("/:id", handler.DeleteCategory)

	return app
}

// --- Test: Expense CRUD ---

func TestExpenseHandler_Create_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	catID := "00000000-0000-0000-0000-000000000001"
	// Buat kategori dulu
	mock.categories[catID] = &domain.ExpenseCategory{
		ID: catID, TenantID: "test-tenant-id", Name: "Bandwidth",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	body, _ := json.Marshal(domain.CreateExpenseRequest{
		CategoryID:  catID,
		Amount:      500000,
		Description: "Biaya bandwidth bulan Januari",
		ExpenseDate: "2024-01-15",
	})

	req := httptest.NewRequest("POST", "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestExpenseHandler_Create_InvalidBody(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("POST", "/api/v1/expenses", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestExpenseHandler_Create_ValidationError(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	// Amount missing (required)
	body, _ := json.Marshal(map[string]interface{}{
		"category_id":  "not-a-uuid",
		"expense_date": "2024-01-15",
	})

	req := httptest.NewRequest("POST", "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestExpenseHandler_Create_CategoryNotFound(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	body, _ := json.Marshal(domain.CreateExpenseRequest{
		CategoryID:  "00000000-0000-0000-0000-000000000001",
		Amount:      500000,
		ExpenseDate: "2024-01-15",
	})

	req := httptest.NewRequest("POST", "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "CATEGORY_NOT_FOUND" {
		t.Fatalf("expected CATEGORY_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestExpenseHandler_Delete_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	mock.expenses["00000000-0000-0000-0000-000000000010"] = &domain.Expense{
		ID: "00000000-0000-0000-0000-000000000010", TenantID: "test-tenant-id", Amount: 500000,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/expenses/00000000-0000-0000-0000-000000000010", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Verifikasi soft delete
	if mock.expenses["00000000-0000-0000-0000-000000000010"].DeletedAt == nil {
		t.Fatal("expected expense to be soft deleted")
	}
}

func TestExpenseHandler_Delete_NotFound(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/expenses/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestExpenseHandler_Update_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	expID := "00000000-0000-0000-0000-000000000011"
	mock.expenses[expID] = &domain.Expense{
		ID: expID, TenantID: "test-tenant-id", Amount: 500000,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	newAmount := int64(750000)
	body, _ := json.Marshal(domain.UpdateExpenseRequest{
		Amount: &newAmount,
	})

	req := httptest.NewRequest("PUT", "/api/v1/expenses/"+expID, bytes.NewReader(body))
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

func TestExpenseHandler_Update_NotFound(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	newAmount := int64(750000)
	body, _ := json.Marshal(domain.UpdateExpenseRequest{
		Amount: &newAmount,
	})

	req := httptest.NewRequest("PUT", "/api/v1/expenses/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestExpenseHandler_List_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	mock.expenses["00000000-0000-0000-0000-000000000012"] = &domain.Expense{
		ID: "00000000-0000-0000-0000-000000000012", TenantID: "test-tenant-id", Amount: 500000,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/expenses?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestExpenseHandler_List_MissingPeriod(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/expenses", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Test: Category CRUD ---

func TestExpenseHandler_CreateCategory_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	body, _ := json.Marshal(domain.CreateCategoryRequest{Name: "Bandwidth"})
	req := httptest.NewRequest("POST", "/api/v1/expenses/categories", bytes.NewReader(body))
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

func TestExpenseHandler_CreateCategory_Duplicate(t *testing.T) {
	mock := newMockExpenseUsecase()
	catID := "00000000-0000-0000-0000-000000000002"
	mock.categories[catID] = &domain.ExpenseCategory{
		ID: catID, TenantID: "test-tenant-id", Name: "Bandwidth",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	body, _ := json.Marshal(domain.CreateCategoryRequest{Name: "Bandwidth"})
	req := httptest.NewRequest("POST", "/api/v1/expenses/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "CATEGORY_NAME_DUPLICATE" {
		t.Fatalf("expected CATEGORY_NAME_DUPLICATE, got %v", apiResp.Error)
	}
}

func TestExpenseHandler_DeleteCategory_HasExpenses(t *testing.T) {
	mock := newMockExpenseUsecase()
	catID := "00000000-0000-0000-0000-000000000003"
	mock.categories[catID] = &domain.ExpenseCategory{
		ID: catID, TenantID: "test-tenant-id", Name: "Bandwidth",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	// Simulasi ada expense terkait
	mock.expensesByCategory[catID] = 3
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/expenses/categories/"+catID, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "CATEGORY_HAS_EXPENSES" {
		t.Fatalf("expected CATEGORY_HAS_EXPENSES, got %v", apiResp.Error)
	}
}

func TestExpenseHandler_DeleteCategory_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	catID := "00000000-0000-0000-0000-000000000004"
	mock.categories[catID] = &domain.ExpenseCategory{
		ID: catID, TenantID: "test-tenant-id", Name: "Bandwidth",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/expenses/categories/"+catID, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestExpenseHandler_DeleteCategory_NotFound(t *testing.T) {
	mock := newMockExpenseUsecase()
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("DELETE", "/api/v1/expenses/categories/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestExpenseHandler_ListCategories_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	catID := "00000000-0000-0000-0000-000000000005"
	mock.categories[catID] = &domain.ExpenseCategory{
		ID: catID, TenantID: "test-tenant-id", Name: "Bandwidth",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/expenses/categories", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExpenseHandler_UpdateCategory_Success(t *testing.T) {
	mock := newMockExpenseUsecase()
	catID := "00000000-0000-0000-0000-000000000006"
	mock.categories[catID] = &domain.ExpenseCategory{
		ID: catID, TenantID: "test-tenant-id", Name: "Bandwidth",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	app := setupExpenseTestApp(mock)

	body, _ := json.Marshal(domain.UpdateCategoryRequest{Name: "Bandwidth/Upstream"})
	req := httptest.NewRequest("PUT", "/api/v1/expenses/categories/"+catID, bytes.NewReader(body))
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
