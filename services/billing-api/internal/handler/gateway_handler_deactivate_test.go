// gateway_handler_deactivate_test.go berisi unit test untuk DeactivateConfig dan TestConfig.
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Test DeactivateConfig ---

func TestGatewayHandler_DeactivateConfig_Success(t *testing.T) {
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

	// Deactivate config
	req := httptest.NewRequest("DELETE", "/api/v1/settings/payment-gateways/"+createApiResp.Data.ID, nil)
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

func TestGatewayHandler_DeactivateConfig_NotFound(t *testing.T) {
	setup := setupGatewayTestApp()
	req := httptest.NewRequest("DELETE", "/api/v1/settings/payment-gateways/nonexistent-id", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "GATEWAY_CONFIG_NOT_FOUND" {
		t.Fatalf("expected GATEWAY_CONFIG_NOT_FOUND, got %v", apiResp.Error)
	}
}

// --- Test TestConfig ---

func TestGatewayHandler_TestConfig_Success(t *testing.T) {
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

	// Test config — akan mengembalikan 200 dengan hasil test (success atau failure)
	// Karena mock tidak punya gateway asli, adapter.TestConnection akan gagal,
	// tapi handler tetap mengembalikan 200 dengan GatewayTestResult
	req := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways/"+createApiResp.Data.ID+"/test", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	// TestConfig selalu mengembalikan 200 dengan GatewayTestResult (success atau failure)
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true (API call success, test result may vary)")
	}
}

func TestGatewayHandler_TestConfig_NotFound(t *testing.T) {
	setup := setupGatewayTestApp()
	req := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways/nonexistent-id/test", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestGatewayHandler_CreateConfig_InvalidBody(t *testing.T) {
	setup := setupGatewayTestApp()
	req := httptest.NewRequest("POST", "/api/v1/settings/payment-gateways", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
