// invoice_action_handler_test.go berisi unit test untuk InvoiceActionHandler.
// Menguji HTTP status codes untuk cancel, record payment, bulk reminder, bulk cancel, export.
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// =============================================================================
// Setup helper untuk InvoiceActionHandler tests
// =============================================================================

// actionTestSetup berisi semua dependensi untuk testing InvoiceActionHandler.
type actionTestSetup struct {
	app          *fiber.App
	invoiceRepo  *mockInvoiceRepo
	customerRepo *mockInvCustomerRepo
}

// setupActionTestApp membuat Fiber app dengan InvoiceActionHandler yang di-back oleh mock repos.
func setupActionTestApp() *actionTestSetup {
	invoiceRepo := newMockInvoiceRepo()
	itemRepo := newMockInvoiceItemRepo()
	paymentRepo := newMockInvoicePaymentRepo()
	auditRepo := newMockInvoiceAuditRepo()
	settingsRepo := newMockBillingSettingsRepo()
	customerRepo := newMockInvCustomerRepo()
	logger := zerolog.New(io.Discard)

	// Tambah pelanggan default
	customerRepo.customers["cust-1"] = &domain.Customer{
		ID:       "cust-1",
		TenantID: "test-tenant",
		Name:     "Test Customer",
		Status:   domain.CustomerStatusAktif,
	}

	actionUC := usecase.NewInvoiceActionUsecase(
		invoiceRepo, itemRepo, paymentRepo, auditRepo,
		settingsRepo, customerRepo, nil, nil, logger,
	)
	handler := NewInvoiceActionHandler(actionUC, logger)

	app := fiber.New()

	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant")
		c.Locals("user_id", "test-user")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	invoices := app.Group("/api/v1/invoices", setLocals)
	invoices.Post("/bulk/reminder", handler.BulkReminder)
	invoices.Post("/bulk/cancel", handler.BulkCancel)
	invoices.Get("/export", handler.ExportCSV)
	invoices.Post("/:id/cancel", handler.Cancel)
	invoices.Post("/:id/payment", handler.RecordPayment)

	return &actionTestSetup{
		app:          app,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
	}
}

// =============================================================================
// Unit Tests — InvoiceActionHandler
// =============================================================================

// --- Cancel endpoint ---

// TestInvoiceActionHandler_Cancel_Success menguji cancel invoice berhasil.
func TestInvoiceActionHandler_Cancel_Success(t *testing.T) {
	setup := setupActionTestApp()

	// Tambah invoice yang bisa dibatalkan
	setup.invoiceRepo.invoices["inv-cancel"] = &domain.Invoice{
		ID:            "inv-cancel",
		TenantID:      "test-tenant",
		CustomerID:    "cust-1",
		InvoiceNumber: "INV-2024-01-001",
		Status:        domain.InvoiceStatusBelumBayar,
		Version:       1,
	}

	body, _ := json.Marshal(domain.CancelInvoiceRequest{
		ConfirmationNumber: "INV-2024-01-001",
		Reason:             "Pelanggan membatalkan layanan",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/inv-cancel/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestInvoiceActionHandler_Cancel_ConfirmationMismatch menguji 400 saat konfirmasi tidak cocok.
func TestInvoiceActionHandler_Cancel_ConfirmationMismatch(t *testing.T) {
	setup := setupActionTestApp()

	setup.invoiceRepo.invoices["inv-mismatch"] = &domain.Invoice{
		ID:            "inv-mismatch",
		TenantID:      "test-tenant",
		CustomerID:    "cust-1",
		InvoiceNumber: "INV-2024-01-001",
		Status:        domain.InvoiceStatusBelumBayar,
		Version:       1,
	}

	body, _ := json.Marshal(domain.CancelInvoiceRequest{
		ConfirmationNumber: "WRONG-NUMBER",
		Reason:             "Alasan pembatalan yang valid",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/inv-mismatch/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
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

// TestInvoiceActionHandler_Cancel_NotCancellable menguji 422 saat invoice sudah lunas.
func TestInvoiceActionHandler_Cancel_NotCancellable(t *testing.T) {
	setup := setupActionTestApp()

	setup.invoiceRepo.invoices["inv-lunas"] = &domain.Invoice{
		ID:            "inv-lunas",
		TenantID:      "test-tenant",
		CustomerID:    "cust-1",
		InvoiceNumber: "INV-2024-01-002",
		Status:        domain.InvoiceStatusLunas,
		Version:       1,
	}

	body, _ := json.Marshal(domain.CancelInvoiceRequest{
		ConfirmationNumber: "INV-2024-01-002",
		Reason:             "Alasan pembatalan yang valid",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/inv-lunas/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "INVOICE_NOT_CANCELLABLE" {
		t.Fatalf("expected INVOICE_NOT_CANCELLABLE, got %v", apiResp.Error)
	}
}

// TestInvoiceActionHandler_Cancel_NotFound menguji 404 saat invoice tidak ditemukan.
func TestInvoiceActionHandler_Cancel_NotFound(t *testing.T) {
	setup := setupActionTestApp()

	body, _ := json.Marshal(domain.CancelInvoiceRequest{
		ConfirmationNumber: "INV-XXX",
		Reason:             "Alasan pembatalan yang valid",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/nonexistent/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestInvoiceActionHandler_Cancel_ValidationError menguji 400 saat body tidak valid.
func TestInvoiceActionHandler_Cancel_ValidationError(t *testing.T) {
	setup := setupActionTestApp()

	// Body tanpa reason (required, min=5)
	body, _ := json.Marshal(map[string]interface{}{
		"confirmation_number": "INV-001",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/inv-1/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// --- RecordPayment endpoint ---

// TestInvoiceActionHandler_RecordPayment_Success menguji pencatatan pembayaran berhasil.
func TestInvoiceActionHandler_RecordPayment_Success(t *testing.T) {
	setup := setupActionTestApp()

	setup.invoiceRepo.invoices["inv-pay"] = &domain.Invoice{
		ID:          "inv-pay",
		TenantID:    "test-tenant",
		CustomerID:  "cust-1",
		Status:      domain.InvoiceStatusBelumBayar,
		TotalAmount: 100000,
		PaidAmount:  0,
		DueDate:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		Version:     1,
	}

	body, _ := json.Marshal(domain.RecordPaymentRequest{
		Amount:        100000,
		PaymentMethod: "tunai",
		PaymentDate:   "2024-06-10",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/inv-pay/payment", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestInvoiceActionHandler_RecordPayment_ValidationError menguji validasi payment gagal.
func TestInvoiceActionHandler_RecordPayment_ValidationError(t *testing.T) {
	setup := setupActionTestApp()

	// Body tanpa amount dan payment_method
	body, _ := json.Marshal(map[string]interface{}{
		"payment_date": "2024-06-10",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/inv-1/payment", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// --- BulkReminder endpoint ---

// TestInvoiceActionHandler_BulkReminder_Success menguji bulk reminder berhasil.
func TestInvoiceActionHandler_BulkReminder_Success(t *testing.T) {
	setup := setupActionTestApp()

	setup.invoiceRepo.invoices["00000000-0000-0000-0000-000000000011"] = &domain.Invoice{
		ID:       "00000000-0000-0000-0000-000000000011",
		TenantID: "test-tenant",
		Status:   domain.InvoiceStatusBelumBayar,
		Version:  1,
	}

	body, _ := json.Marshal(domain.BulkInvoiceIDsRequest{
		InvoiceIDs: []string{"00000000-0000-0000-0000-000000000011"},
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/bulk/reminder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestInvoiceActionHandler_BulkReminder_ValidationError menguji validasi bulk reminder gagal.
func TestInvoiceActionHandler_BulkReminder_ValidationError(t *testing.T) {
	setup := setupActionTestApp()

	// Body dengan invoice_ids kosong
	body, _ := json.Marshal(domain.BulkInvoiceIDsRequest{
		InvoiceIDs: []string{},
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/bulk/reminder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- BulkCancel endpoint ---

// TestInvoiceActionHandler_BulkCancel_Success menguji bulk cancel berhasil.
func TestInvoiceActionHandler_BulkCancel_Success(t *testing.T) {
	setup := setupActionTestApp()

	setup.invoiceRepo.invoices["00000000-0000-0000-0000-000000000012"] = &domain.Invoice{
		ID:            "00000000-0000-0000-0000-000000000012",
		TenantID:      "test-tenant",
		CustomerID:    "cust-1",
		InvoiceNumber: "INV-2024-01-010",
		Status:        domain.InvoiceStatusBelumBayar,
		Version:       1,
	}

	body, _ := json.Marshal(domain.BulkCancelRequest{
		InvoiceIDs: []string{"00000000-0000-0000-0000-000000000012"},
		Reason:     "Pembatalan massal untuk testing",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices/bulk/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// --- ExportCSV endpoint ---

// TestInvoiceActionHandler_ExportCSV_Success menguji export CSV berhasil.
func TestInvoiceActionHandler_ExportCSV_Success(t *testing.T) {
	setup := setupActionTestApp()

	req := httptest.NewRequest("GET", "/api/v1/invoices/export", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/csv" {
		t.Fatalf("expected Content-Type text/csv, got %s", contentType)
	}
}
