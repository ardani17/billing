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

type mockConfigRepo struct {
	configs     []*domain.NotificationConfig
	settings    *domain.ConfigSettings
	upsertErr   error
	getErr      error
	settingsErr error
}

func newMockConfigRepo() *mockConfigRepo {
	return &mockConfigRepo{}
}

func (m *mockConfigRepo) GetByTenant(_ context.Context, _ string) ([]*domain.NotificationConfig, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.configs, nil
}

func (m *mockConfigRepo) GetByTenantAndChannel(_ context.Context, _ string, _ domain.Channel) (*domain.NotificationConfig, error) {
	return nil, nil
}

func (m *mockConfigRepo) Upsert(_ context.Context, cfg *domain.NotificationConfig) (*domain.NotificationConfig, error) {
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	result := *cfg
	result.ID = "config-new"
	result.CreatedAt = time.Now()
	result.UpdatedAt = time.Now()
	return &result, nil
}

func (m *mockConfigRepo) GetSettings(_ context.Context, _ string) (*domain.ConfigSettings, error) {
	if m.settingsErr != nil {
		return nil, m.settingsErr
	}
	if m.settings != nil {
		return m.settings, nil
	}
	return &domain.ConfigSettings{
		ChannelPriority:   []domain.Channel{domain.ChannelWhatsApp, domain.ChannelSMS, domain.ChannelEmail},
		QuietHoursStart:   "07:00",
		QuietHoursEnd:     "21:00",
		Timezone:          "Asia/Jakarta",
		DailyLimitPerCust: 5,
		CooldownMinutes:   30,
	}, nil
}

func (m *mockConfigRepo) UpdateSettings(_ context.Context, _ string, _ domain.ConfigSettings) error {
	return nil
}

// =============================================================================
// =============================================================================

type mockTemplateRepo struct {
	templates  []*domain.NotificationTemplate
	bulkErr    error
	bulkCalled bool
}

func newMockTemplateRepo() *mockTemplateRepo {
	return &mockTemplateRepo{}
}

func (m *mockTemplateRepo) Create(_ context.Context, t *domain.NotificationTemplate) (*domain.NotificationTemplate, error) {
	copy := *t
	return &copy, nil
}

func (m *mockTemplateRepo) GetByID(_ context.Context, _ string) (*domain.NotificationTemplate, error) {
	return nil, domain.ErrTemplateNotFound
}

func (m *mockTemplateRepo) GetBySlug(_ context.Context, _, _ string) (*domain.NotificationTemplate, error) {
	return nil, domain.ErrTemplateNotFound
}

func (m *mockTemplateRepo) GetByEventType(_ context.Context, _, _ string) (*domain.NotificationTemplate, error) {
	return nil, domain.ErrTemplateNotFound
}

func (m *mockTemplateRepo) Update(_ context.Context, t *domain.NotificationTemplate) (*domain.NotificationTemplate, error) {
	copy := *t
	return &copy, nil
}

func (m *mockTemplateRepo) SoftDelete(_ context.Context, _ string) error {
	return nil
}

func (m *mockTemplateRepo) ListByTenant(_ context.Context, _ string) ([]*domain.NotificationTemplate, error) {
	return m.templates, nil
}

func (m *mockTemplateRepo) BulkCreate(_ context.Context, _ []*domain.NotificationTemplate) error {
	m.bulkCalled = true
	return m.bulkErr
}

func (m *mockTemplateRepo) SlugExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

// =============================================================================
// =============================================================================

func TestConfigHandler_Get_Success(t *testing.T) {
	configRepo := newMockConfigRepo()
	templateRepo := newMockTemplateRepo()

	waCreds, _ := json.Marshal(domain.WhatsAppCredentials{
		APIToken:     "token-secret-12345678",
		SenderNumber: "08123456789",
	})
	configRepo.configs = []*domain.NotificationConfig{
		{
			ID:          "cfg-1",
			TenantID:    "tenant-1",
			Channel:     domain.ChannelWhatsApp,
			Provider:    "fonnte",
			Credentials: waCreds,
			IsEnabled:   true,
			Priority:    1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	handler := NewConfigHandler(configRepo, templateRepo)
	app := fiber.New()
	app.Get("/api/v1/notifications/config", testTenantMiddleware("tenant-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/notifications/config", nil)
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

	// Verifikasi data respons
	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", apiResp.Data)
	}

	// Verifikasi configs ada
	configs, ok := dataMap["configs"].([]interface{})
	if !ok {
		t.Fatalf("expected configs to be an array, got %T", dataMap["configs"])
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	// Verifikasi credential di-mask
	cfg := configs[0].(map[string]interface{})
	credsRaw, ok := cfg["credentials"]
	if !ok {
		t.Fatalf("expected credentials field in config")
	}

	// Credential harus berupa map dengan nilai yang di-mask
	credsMap, ok := credsRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected credentials to be a map, got %T", credsRaw)
	}

	// api_token harus di-mask, hanya 4 karakter terakhir yang terlihat
	maskedToken, ok := credsMap["api_token"].(string)
	if !ok {
		t.Fatalf("expected api_token to be a string")
	}
	// Token asli: "token-secret-12345678", 4 karakter terakhir: "5678"
	if len(maskedToken) == 0 {
		t.Fatalf("expected masked token to not be empty")
	}
	last4 := maskedToken[len(maskedToken)-4:]
	if last4 != "5678" {
		t.Fatalf("expected masked token to end with '5678', got '%s'", last4)
	}

	// Verifikasi settings ada
	if _, ok := dataMap["settings"]; !ok {
		t.Fatalf("expected settings field in response")
	}
}

// =============================================================================
// =============================================================================

func TestConfigHandler_Update_Success(t *testing.T) {
	configRepo := newMockConfigRepo()
	templateRepo := newMockTemplateRepo()

	// Sudah ada konfigurasi sebelumnya (bukan konfigurasi pertama)
	configRepo.configs = []*domain.NotificationConfig{
		{ID: "cfg-existing", TenantID: "tenant-1", Channel: domain.ChannelSMS},
	}

	handler := NewConfigHandler(configRepo, templateRepo)
	app := fiber.New()
	app.Put("/api/v1/notifications/config", testTenantMiddleware("tenant-1"), handler.Update)

	creds, _ := json.Marshal(domain.WhatsAppCredentials{
		APIToken:     "my-api-token-1234",
		SenderNumber: "08123456789",
	})
	reqBody := domain.UpdateConfigRequest{
		Channel:     domain.ChannelWhatsApp,
		Provider:    "fonnte",
		Credentials: creds,
		IsEnabled:   true,
		Priority:    1,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/v1/notifications/config", bytes.NewReader(bodyBytes))
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
}

// =============================================================================
// =============================================================================

func TestConfigHandler_Update_ValidationError(t *testing.T) {
	configRepo := newMockConfigRepo()
	templateRepo := newMockTemplateRepo()

	handler := NewConfigHandler(configRepo, templateRepo)
	app := fiber.New()
	app.Put("/api/v1/notifications/config", testTenantMiddleware("tenant-1"), handler.Update)

	creds, _ := json.Marshal(domain.WhatsAppCredentials{
		APIToken:     "",
		SenderNumber: "",
	})
	reqBody := domain.UpdateConfigRequest{
		Channel:     domain.ChannelWhatsApp,
		Provider:    "fonnte",
		Credentials: creds,
		IsEnabled:   true,
		Priority:    1,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/v1/notifications/config", bytes.NewReader(bodyBytes))
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
