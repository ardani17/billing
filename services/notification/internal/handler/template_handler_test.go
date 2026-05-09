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
	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// =============================================================================

type mockTemplateRepoForHandler struct {
	templates     []*domain.NotificationTemplate
	byID          map[string]*domain.NotificationTemplate
	slugExists    bool
	slugExistsErr error
	createErr     error
	updateErr     error
	softDeleteErr error
	softDeleted   map[string]bool
}

func newMockTemplateRepoForHandler() *mockTemplateRepoForHandler {
	return &mockTemplateRepoForHandler{
		byID:        make(map[string]*domain.NotificationTemplate),
		softDeleted: make(map[string]bool),
	}
}

func (m *mockTemplateRepoForHandler) Create(_ context.Context, t *domain.NotificationTemplate) (*domain.NotificationTemplate, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	copy := *t
	copy.ID = "tpl-new"
	copy.CreatedAt = time.Now()
	copy.UpdatedAt = time.Now()
	return &copy, nil
}

func (m *mockTemplateRepoForHandler) GetByID(_ context.Context, id string) (*domain.NotificationTemplate, error) {
	t, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrTemplateNotFound
	}
	copy := *t
	return &copy, nil
}

func (m *mockTemplateRepoForHandler) GetBySlug(_ context.Context, _, _ string) (*domain.NotificationTemplate, error) {
	return nil, domain.ErrTemplateNotFound
}

func (m *mockTemplateRepoForHandler) GetByEventType(_ context.Context, _, _ string) (*domain.NotificationTemplate, error) {
	return nil, domain.ErrTemplateNotFound
}

func (m *mockTemplateRepoForHandler) Update(_ context.Context, t *domain.NotificationTemplate) (*domain.NotificationTemplate, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	copy := *t
	copy.UpdatedAt = time.Now()
	return &copy, nil
}

func (m *mockTemplateRepoForHandler) SoftDelete(_ context.Context, id string) error {
	if m.softDeleteErr != nil {
		return m.softDeleteErr
	}
	m.softDeleted[id] = true
	return nil
}

func (m *mockTemplateRepoForHandler) ListByTenant(_ context.Context, _ string) ([]*domain.NotificationTemplate, error) {
	return m.templates, nil
}

func (m *mockTemplateRepoForHandler) BulkCreate(_ context.Context, _ []*domain.NotificationTemplate) error {
	return nil
}

func (m *mockTemplateRepoForHandler) SlugExists(_ context.Context, _, _, _ string) (bool, error) {
	if m.slugExistsErr != nil {
		return false, m.slugExistsErr
	}
	return m.slugExists, nil
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_List_Success(t *testing.T) {
	repo := newMockTemplateRepoForHandler()
	now := time.Now()

	repo.templates = []*domain.NotificationTemplate{
		{
			ID:           "tpl-1",
			TenantID:     "tenant-1",
			Slug:         "invoice_new",
			Name:         "Invoice Baru",
			Category:     domain.CategoryTransactional,
			EventType:    "invoice.created",
			Channels:     []domain.Channel{domain.ChannelWhatsApp, domain.ChannelSMS},
			BodyWhatsApp: "Halo {nama}, invoice baru Anda.",
			BodySMS:      "Invoice baru untuk {nama}.",
			IsActive:     true,
			IsDefault:    true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "tpl-2",
			TenantID:     "tenant-1",
			Slug:         "promo_diskon",
			Name:         "Promo Diskon",
			Category:     domain.CategoryPromotion,
			Channels:     []domain.Channel{domain.ChannelWhatsApp},
			BodyWhatsApp: "Promo spesial untuk Anda!",
			IsActive:     true,
			IsDefault:    false,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Get("/api/v1/notifications/templates", testTenantMiddleware("tenant-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/notifications/templates", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Fatalf("expected success=true, got false")
	}

	// Verifikasi data berupa array template
	items, ok := apiResp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", apiResp.Data)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(items))
	}
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_Create_Success(t *testing.T) {
	repo := newMockTemplateRepoForHandler()
	repo.slugExists = false

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Post("/api/v1/notifications/templates", testTenantMiddleware("tenant-1"), handler.Create)

	reqBody := domain.CreateTemplateRequest{
		Slug:         "custom_reminder",
		Name:         "Pengingat Custom",
		Category:     domain.CategoryReminder,
		EventType:    "invoice.reminder",
		Channels:     []domain.Channel{domain.ChannelWhatsApp, domain.ChannelSMS},
		BodyWhatsApp: "Halo {nama}, jangan lupa bayar tagihan Anda.",
		BodySMS:      "Tagihan Anda menunggu, {nama}.",
		Variables:    []string{"nama"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/notifications/templates", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Fatalf("expected success=true, got false")
	}

	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", apiResp.Data)
	}

	if slug, ok := dataMap["slug"].(string); !ok || slug != "custom_reminder" {
		t.Fatalf("expected slug=custom_reminder, got %v", dataMap["slug"])
	}

	if isDefault, ok := dataMap["is_default"].(bool); !ok || isDefault {
		t.Fatalf("expected is_default=false, got %v", dataMap["is_default"])
	}
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_Create_SlugExists(t *testing.T) {
	repo := newMockTemplateRepoForHandler()
	repo.slugExists = true

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Post("/api/v1/notifications/templates", testTenantMiddleware("tenant-1"), handler.Create)

	reqBody := domain.CreateTemplateRequest{
		Slug:         "invoice_new",
		Name:         "Invoice Baru Duplikat",
		Category:     domain.CategoryTransactional,
		Channels:     []domain.Channel{domain.ChannelWhatsApp},
		BodyWhatsApp: "Halo {nama}",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/notifications/templates", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if apiResp.Success {
		t.Fatalf("expected success=false, got true")
	}

	if apiResp.Error == nil || apiResp.Error.Code != "TEMPLATE_SLUG_EXISTS" {
		t.Fatalf("expected error code TEMPLATE_SLUG_EXISTS, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_Create_NoBody(t *testing.T) {
	repo := newMockTemplateRepoForHandler()

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Post("/api/v1/notifications/templates", testTenantMiddleware("tenant-1"), handler.Create)

	reqBody := domain.CreateTemplateRequest{
		Slug:     "empty_body",
		Name:     "Template Tanpa Body",
		Category: domain.CategoryInformation,
		Channels: []domain.Channel{domain.ChannelWhatsApp},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/notifications/templates", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if apiResp.Success {
		t.Fatalf("expected success=false, got true")
	}

	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected error code VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_Update_Success(t *testing.T) {
	repo := newMockTemplateRepoForHandler()
	now := time.Now()

	// Siapkan template yang sudah ada
	repo.byID["tpl-1"] = &domain.NotificationTemplate{
		ID:           "tpl-1",
		TenantID:     "tenant-1",
		Slug:         "invoice_new",
		Name:         "Invoice Baru",
		Category:     domain.CategoryTransactional,
		Channels:     []domain.Channel{domain.ChannelWhatsApp},
		BodyWhatsApp: "Halo {nama}",
		IsActive:     true,
		IsDefault:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Put("/api/v1/notifications/templates/:id", testTenantMiddleware("tenant-1"), handler.Update)

	reqBody := domain.UpdateTemplateRequest{
		Name:         "Invoice Baru (Updated)",
		BodyWhatsApp: "Halo {nama}, invoice baru telah dibuat.",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/v1/notifications/templates/tpl-1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Fatalf("expected success=true, got false")
	}

	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", apiResp.Data)
	}

	if name, ok := dataMap["name"].(string); !ok || name != "Invoice Baru (Updated)" {
		t.Fatalf("expected name='Invoice Baru (Updated)', got %v", dataMap["name"])
	}
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_Delete_Success(t *testing.T) {
	repo := newMockTemplateRepoForHandler()
	now := time.Now()

	// Template kustom (is_default=false) yang bisa dihapus
	repo.byID["tpl-custom"] = &domain.NotificationTemplate{
		ID:           "tpl-custom",
		TenantID:     "tenant-1",
		Slug:         "promo_diskon",
		Name:         "Promo Diskon",
		Category:     domain.CategoryPromotion,
		Channels:     []domain.Channel{domain.ChannelWhatsApp},
		BodyWhatsApp: "Promo spesial!",
		IsActive:     true,
		IsDefault:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Delete("/api/v1/notifications/templates/:id", testTenantMiddleware("tenant-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/notifications/templates/tpl-custom", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Fatalf("expected success=true, got false")
	}

	// Verifikasi SoftDelete dipanggil
	if !repo.softDeleted["tpl-custom"] {
		t.Fatalf("expected SoftDelete to be called for tpl-custom")
	}
}

// =============================================================================
// =============================================================================

func TestTemplateHandler_Delete_DefaultNotDeletable(t *testing.T) {
	repo := newMockTemplateRepoForHandler()
	now := time.Now()

	// Template bawaan (is_default=true) yang tidak bisa dihapus
	repo.byID["tpl-default"] = &domain.NotificationTemplate{
		ID:           "tpl-default",
		TenantID:     "tenant-1",
		Slug:         "invoice_new",
		Name:         "Invoice Baru",
		Category:     domain.CategoryTransactional,
		Channels:     []domain.Channel{domain.ChannelWhatsApp},
		BodyWhatsApp: "Halo {nama}",
		IsActive:     true,
		IsDefault:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	handler := NewTemplateHandler(repo)
	app := fiber.New()
	app.Delete("/api/v1/notifications/templates/:id", testTenantMiddleware("tenant-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/notifications/templates/tpl-default", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if apiResp.Success {
		t.Fatalf("expected success=false, got true")
	}

	if apiResp.Error == nil || apiResp.Error.Code != "TEMPLATE_NOT_DELETABLE" {
		t.Fatalf("expected error code TEMPLATE_NOT_DELETABLE, got %v", apiResp.Error)
	}

	// Verifikasi SoftDelete TIDAK dipanggil
	if repo.softDeleted["tpl-default"] {
		t.Fatalf("expected SoftDelete NOT to be called for default template")
	}
}
