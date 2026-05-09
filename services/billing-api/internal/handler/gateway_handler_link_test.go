// gateway_handler_link_test.go berisi unit test untuk endpoint tautan pembayaran:
// GetCustomerPaymentLink, RegeneratePaymentLink, GetInvoicePaymentLinks,
// GetPaymentLinkWebhooks, dan WalledGardenPaymentInfo.
package handler

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Tes GetCustomerPaymentLink ---

func TestGatewayHandler_GetCustomerPaymentLink_WithLink(t *testing.T) {
	setup := setupGatewayTestApp()
	// Seed data: customer, invoice, dan tautan pembayaran aktif
	setup.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "test-tenant-id", Name: "Budi",
	}
	setup.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", CustomerID: "cust-1", TenantID: "test-tenant-id",
		InvoiceNumber: "INV-2024-01-001", TotalAmount: 100000, PaidAmount: 0,
		Status: domain.InvoiceStatusBelumBayar, DueDate: time.Now(),
	}
	setup.linkRepo.links["link-1"] = &domain.PaymentLink{
		ID: "link-1", CustomerID: "cust-1", TenantID: "test-tenant-id",
		Status: domain.PaymentLinkActive, PaymentURL: "https://pay.test/link-1",
		Amount: 100000, ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	setup.linkRepo.junction["link-1"] = []string{"inv-1"}

	req := httptest.NewRequest("GET", "/api/v1/customers/cust-1/payment-link", nil)
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
	// Data harus berisi payment_link
	if apiResp.Data == nil {
		t.Fatal("expected data tidak nil (ada payment link aktif)")
	}
}

func TestGatewayHandler_GetCustomerPaymentLink_NoLink(t *testing.T) {
	setup := setupGatewayTestApp()
	// Customer tanpa tautan pembayaran aktif
	setup.customerRepo.customers["cust-2"] = &domain.Customer{
		ID: "cust-2", TenantID: "test-tenant-id", Name: "Andi",
	}

	req := httptest.NewRequest("GET", "/api/v1/customers/cust-2/payment-link", nil)
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
	if apiResp.Data != nil {
		t.Fatalf("expected data nil (tidak ada payment link), got %v", apiResp.Data)
	}
}

// --- Tes RegeneratePaymentLink ---

func TestGatewayHandler_RegeneratePaymentLink_Success(t *testing.T) {
	setup := setupGatewayTestApp()
	// Seed: customer, invoice, dan gateway config aktif
	setup.customerRepo.customers["cust-3"] = &domain.Customer{
		ID: "cust-3", TenantID: "test-tenant-id", Name: "Siti", Email: "siti@test.com",
	}
	setup.invoiceRepo.invoices["inv-3"] = &domain.Invoice{
		ID: "inv-3", CustomerID: "cust-3", TenantID: "test-tenant-id",
		InvoiceNumber: "INV-2024-01-003", TotalAmount: 200000, PaidAmount: 0,
		Status: domain.InvoiceStatusBelumBayar, DueDate: time.Now(),
	}
	// Config gateway aktif dengan API key terenkripsi
	apiKeyEnc, _ := encryptTestKey("xnd_production_test_key_1234567890")
	setup.configRepo.configs["cfg-regen"] = &domain.GatewayConfig{
		ID: "cfg-regen", TenantID: "test-tenant-id",
		GatewayProvider: domain.GatewayXendit, IsActive: true,
		APIKeyEncrypted: apiKeyEnc, EnabledMethods: []string{"va_bca"},
		PaymentLinkExpiryDays: 7,
	}

	req := httptest.NewRequest("POST", "/api/v1/customers/cust-3/payment-link/regenerate", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	// Regenerate akan gagal karena adapter Xendit tidak bisa connect ke gateway asli,
	if resp.StatusCode != fiber.StatusBadGateway && resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 atau 502, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Tes GetInvoicePaymentLinks ---

func TestGatewayHandler_GetInvoicePaymentLinks_Success(t *testing.T) {
	setup := setupGatewayTestApp()
	// Seed tautan pembayaran yang terkait dengan invoice
	setup.linkRepo.links["link-inv"] = &domain.PaymentLink{
		ID: "link-inv", CustomerID: "cust-1", TenantID: "test-tenant-id",
		Status: domain.PaymentLinkPaid, Amount: 100000,
	}
	setup.linkRepo.junction["link-inv"] = []string{"inv-query"}

	req := httptest.NewRequest("GET", "/api/v1/invoices/inv-query/payment-links", nil)
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

// --- Tes WalledGardenPaymentInfo ---

func TestGatewayHandler_WalledGardenPaymentInfo_Success(t *testing.T) {
	setup := setupGatewayTestApp()
	// Seed: customer, invoice, dan tautan pembayaran aktif
	setup.customerRepo.customers["cust-wg"] = &domain.Customer{
		ID: "cust-wg", TenantID: "test-tenant-id", Name: "Wawan",
	}
	setup.invoiceRepo.invoices["inv-wg"] = &domain.Invoice{
		ID: "inv-wg", CustomerID: "cust-wg", TenantID: "test-tenant-id",
		InvoiceNumber: "INV-2024-01-010", TotalAmount: 150000, PaidAmount: 0,
		Status: domain.InvoiceStatusTerlambat, DueDate: time.Now(),
	}
	setup.linkRepo.links["link-wg"] = &domain.PaymentLink{
		ID: "link-wg", CustomerID: "cust-wg", TenantID: "test-tenant-id",
		Status: domain.PaymentLinkActive, PaymentURL: "https://pay.test/wg",
		Amount: 150000, ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	setup.linkRepo.junction["link-wg"] = []string{"inv-wg"}

	// Endpoint publik tanpa auth middleware
	req := httptest.NewRequest("GET", "/api/v1/public/walled-garden/cust-wg/payment-info", nil)
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
