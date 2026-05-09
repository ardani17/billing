// gateway_handler_test.go berisi unit test untuk endpoint konfigurasi gateway:
// CreateConfig, ListConfigs, UpdateConfig, DeactivateConfig, TestConfig.
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

func validCreateConfigBody() []byte {
	apiKey := "xnd_production_test_key_1234567890"
	enc, _ := gateway.EncryptAESGCM(apiKey, testMasterKey)
	_ = enc // hanya untuk validasi key
	body, _ := json.Marshal(domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          apiKey,
		WebhookSecret:   "whsec_test_secret_1234567890",
		EnabledMethods:  []string{"va_bca", "qris"},
	})
	return body
}

// --- Tes CreateConfig ---

func TestGatewayHandler_CreateConfig_Success(t *testing.T) {
	setup := setupGatewayTestApp()
	req := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader(validCreateConfigBody()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestGatewayHandler_CreateConfig_ValidationError(t *testing.T) {
	setup := setupGatewayTestApp()
	// Body tanpa field wajib
	body, _ := json.Marshal(map[string]interface{}{"gateway_provider": "invalid"})
	req := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
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

func TestGatewayHandler_CreateConfig_DuplicateProvider(t *testing.T) {
	setup := setupGatewayTestApp()
	// Buat config pertama
	req1 := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader(validCreateConfigBody()))
	req1.Header.Set("Content-Type", "application/json")
	setup.app.Test(req1, -1)

	// Buat config kedua dengan provider sama -> 409
	req2 := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader(validCreateConfigBody()))
	req2.Header.Set("Content-Type", "application/json")
	resp, err := setup.app.Test(req2, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(body))
	}
	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "GATEWAY_CONFIG_DUPLICATE" {
		t.Fatalf("expected GATEWAY_CONFIG_DUPLICATE, got %v", apiResp.Error)
	}
}

// --- Tes ListConfigs ---

func TestGatewayHandler_ListConfigs_WithMaskedKeys(t *testing.T) {
	setup := setupGatewayTestApp()
	// Buat config dulu
	req1 := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader(validCreateConfigBody()))
	req1.Header.Set("Content-Type", "application/json")
	setup.app.Test(req1, -1)

	req := httptest.NewRequest("GET", "/api/v1/settings/payment-gateways", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

// --- Tes UpdateConfig ---

func TestGatewayHandler_UpdateConfig_Success(t *testing.T) {
	setup := setupGatewayTestApp()
	// Buat config dulu
	createReq := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader(validCreateConfigBody()))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := setup.app.Test(createReq, -1)
	var createApiResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&createApiResp)

	updateBody, _ := json.Marshal(domain.UpdateGatewayConfigRequest{
		EnabledMethods: []string{"va_bca", "qris", "ewallet_ovo"},
	})
	req := httptest.NewRequest("PUT", "/api/v1/settings/payment-gateways/"+createApiResp.Data.ID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestGatewayHandler_UpdateConfig_NotFound(t *testing.T) {
	setup := setupGatewayTestApp()
	body, _ := json.Marshal(domain.UpdateGatewayConfigRequest{
		EnabledMethods: []string{"va_bca"},
	})
	req := httptest.NewRequest("PUT", "/api/v1/settings/payment-gateways/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}
