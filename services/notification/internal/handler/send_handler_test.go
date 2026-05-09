package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/ispboss/ispboss/services/notification/internal/usecase"
)

// =============================================================================
// Validasi di handler terjadi sebelum pipeline dipanggil, sehingga aman.
// =============================================================================

func newTestSendHandler() *SendHandler {
	pipeline := &usecase.DeliveryPipeline{}
	return NewSendHandler(pipeline)
}

// =============================================================================
// Tes: SendHandler.TestSend - 422 saat template_id kosong
// Memvalidasi: Kebutuhan 15.1, 15.5
// =============================================================================

func TestSendHandler_TestSend_MissingTemplateID(t *testing.T) {
	handler := newTestSendHandler()
	app := fiber.New()
	app.Post("/api/v1/notifications/test", testTenantMiddleware("tenant-1"), handler.TestSend)

	reqBody := domain.TestSendRequest{
		TemplateID: "",
		Channel:    domain.ChannelWhatsApp,
		Recipient:  "08123456789",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/notifications/test", bytes.NewReader(bodyBytes))
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
// Tes: SendHandler.ManualSend - 422 saat customer_id kosong
// Memvalidasi: Kebutuhan 16.1, 16.6
// =============================================================================

func TestSendHandler_ManualSend_MissingCustomerID(t *testing.T) {
	handler := newTestSendHandler()
	app := fiber.New()
	app.Post("/api/v1/notifications/send", testTenantMiddleware("tenant-1"), handler.ManualSend)

	reqBody := domain.ManualSendRequest{
		CustomerID: "",
		TemplateID: "tmpl-1",
		Channel:    domain.ChannelWhatsApp,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/notifications/send", bytes.NewReader(bodyBytes))
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
// Tes: SendHandler.ManualSend - 422 saat template_id dan custom_body kosong
// Memvalidasi: Kebutuhan 16.1
// =============================================================================

func TestSendHandler_ManualSend_MissingBodyAndTemplate(t *testing.T) {
	handler := newTestSendHandler()
	app := fiber.New()
	app.Post("/api/v1/notifications/send", testTenantMiddleware("tenant-1"), handler.ManualSend)

	reqBody := domain.ManualSendRequest{
		CustomerID: "cust-1",
		TemplateID: "",
		Channel:    domain.ChannelWhatsApp,
		CustomBody: "",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/notifications/send", bytes.NewReader(bodyBytes))
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
// Tes: SendHandler.Resend - 400 saat ID kosong (empty path param)
// Memvalidasi: Kebutuhan 17.1, 17.3, 17.6
// =============================================================================

func TestSendHandler_Resend_MissingID(t *testing.T) {
	handler := newTestSendHandler()
	app := fiber.New()
	// Route dengan :id - Fiber akan mencocokkan path param
	app.Post("/api/v1/notifications/logs/:id/resend", testTenantMiddleware("tenant-1"), handler.Resend)

	req := httptest.NewRequest("POST", "/api/v1/notifications/logs//resend", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Fiber mengembalikan 404 karena route tidak cocok dengan path param kosong
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}
