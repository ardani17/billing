package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

type handlerBillingSettingsRepo struct {
	settings *domain.BillingSettings
}

func (r *handlerBillingSettingsRepo) GetByTenantID(_ context.Context, tenantID string) (*domain.BillingSettings, error) {
	if r.settings == nil {
		return nil, domain.ErrBillingSettingsNotFound
	}
	r.settings.TenantID = tenantID
	return r.settings, nil
}

func (r *handlerBillingSettingsRepo) Upsert(_ context.Context, settings *domain.BillingSettings) (*domain.BillingSettings, error) {
	r.settings = settings
	return settings, nil
}

func (r *handlerBillingSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

func setupBillingSettingsHandlerTestApp(withTenant bool) *fiber.App {
	app := fiber.New()
	repo := &handlerBillingSettingsRepo{}
	uc := usecase.NewBillingSettingsUsecase(repo, zerolog.New(io.Discard))
	h := NewBillingSettingsHandler(uc, zerolog.New(io.Discard))

	if withTenant {
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("tenant_id", "tenant-1")
			return c.Next()
		})
	}
	app.Get("/api/v1/settings/billing", h.Get)
	app.Put("/api/v1/settings/billing", h.Update)
	return app
}

func TestBillingSettingsHandlerGetUnauthorizedWithoutTenant(t *testing.T) {
	app := setupBillingSettingsHandlerTestApp(false)
	req := httptest.NewRequest("GET", "/api/v1/settings/billing", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestBillingSettingsHandlerGetReturnsDefaults(t *testing.T) {
	app := setupBillingSettingsHandlerTestApp(true)
	req := httptest.NewRequest("GET", "/api/v1/settings/billing", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Success bool                   `json:"success"`
		Data    domain.BillingSettings `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !apiResp.Success || apiResp.Data.InvoicePrefix != "INV" {
		t.Fatalf("unexpected response: %+v", apiResp)
	}
}

func TestBillingSettingsHandlerUpdateValidationError(t *testing.T) {
	app := setupBillingSettingsHandlerTestApp(true)
	body, _ := json.Marshal(domain.UpdateBillingSettingsRequest{
		GenerateDays:       1,
		GracePeriodDays:    3,
		SuspendDays:        30,
		PenaltyEnabled:     true,
		PenaltyType:        domain.PenaltyFixed,
		InvoicePrefix:      "INV",
		NewCustomerBilling: "prorate",
		Timezone:           "Asia/Jakarta",
	})
	req := httptest.NewRequest("PUT", "/api/v1/settings/billing", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}
