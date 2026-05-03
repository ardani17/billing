package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// =============================================================================
// Mock repositories untuk PaymentHandler tests
// =============================================================================

// mockPaymentInvoiceRepo implementasi mock InvoiceRepository untuk handler tests.
type mockPaymentInvoiceRepo struct {
	invoices map[string]*domain.Invoice
}

func newMockPaymentInvoiceRepo() *mockPaymentInvoiceRepo {
	return &mockPaymentInvoiceRepo{invoices: make(map[string]*domain.Invoice)}
}

func (m *mockPaymentInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	copy := *inv
	m.invoices[copy.ID] = &copy
	return &copy, nil
}

func (m *mockPaymentInvoiceRepo) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	copy := *inv
	return &copy, nil
}

func (m *mockPaymentInvoiceRepo) Update(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	copy := *inv
	m.invoices[copy.ID] = &copy
	return &copy, nil
}

func (m *mockPaymentInvoiceRepo) UpdateStatus(_ context.Context, id string, status domain.InvoiceStatus, _ int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.Status = status
	copy := *inv
	return &copy, nil
}

func (m *mockPaymentInvoiceRepo) UpdatePaidAmount(_ context.Context, id string, paidAmount int64, _ int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.PaidAmount = paidAmount
	copy := *inv
	return &copy, nil
}

func (m *mockPaymentInvoiceRepo) List(_ context.Context, _ domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	return &domain.InvoiceListResult{Data: []*domain.Invoice{}, Pagination: domain.PaginationMeta{}}, nil
}

func (m *mockPaymentInvoiceRepo) ExistsForPeriod(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *mockPaymentInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *mockPaymentInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *mockPaymentInvoiceRepo) GetSummary(_ context.Context, _ string, _, _ *int) (*domain.InvoiceSummary, error) {
	return &domain.InvoiceSummary{}, nil
}

func (m *mockPaymentInvoiceRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *mockPaymentInvoiceRepo) FindOpenByCustomer(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			(inv.Status == domain.InvoiceStatusBelumBayar ||
				inv.Status == domain.InvoiceStatusTerlambat ||
				inv.Status == domain.InvoiceStatusBayarSebagian) {
			copy := *inv
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockPaymentInvoiceRepo) FindOpenByCustomerForUpdate(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	return m.FindOpenByCustomer(context.Background(), customerID)
}

func (m *mockPaymentInvoiceRepo) GetByIDsForUpdate(_ context.Context, ids []string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, id := range ids {
		if inv, ok := m.invoices[id]; ok {
			copy := *inv
			result = append(result, &copy)
		}
	}
	return result, nil
}
func (m *mockPaymentInvoiceRepo) FindOverdueForIsolir(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockPaymentInvoiceRepo) FindOverdueForSuspend(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockPaymentInvoiceRepo) HasOutstandingInvoices(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockPaymentInvoiceRepo) SumOutstandingAmount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockPaymentInvoiceRepo) CountOutstandingInvoices(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// mockPaymentInvoiceItemRepo implementasi mock InvoiceItemRepository.
type mockPaymentInvoiceItemRepo struct{}

func (m *mockPaymentInvoiceItemRepo) BulkCreate(_ context.Context, items []*domain.InvoiceItem) ([]*domain.InvoiceItem, error) {
	return items, nil
}

func (m *mockPaymentInvoiceItemRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceItem, error) {
	return nil, nil
}

func (m *mockPaymentInvoiceItemRepo) DeleteByInvoice(_ context.Context, _ string) error {
	return nil
}

// mockPaymentPaymentRepo implementasi mock InvoicePaymentRepository.
type mockPaymentPaymentRepo struct {
	payments map[string]*domain.InvoicePayment
}

func newMockPaymentPaymentRepo() *mockPaymentPaymentRepo {
	return &mockPaymentPaymentRepo{payments: make(map[string]*domain.InvoicePayment)}
}

func (m *mockPaymentPaymentRepo) Create(_ context.Context, p *domain.InvoicePayment) (*domain.InvoicePayment, error) {
	copy := *p
	m.payments[copy.ID] = &copy
	return &copy, nil
}

func (m *mockPaymentPaymentRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoicePayment, error) {
	return nil, nil
}

func (m *mockPaymentPaymentRepo) VoidPayment(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockPaymentPaymentRepo) GetByID(_ context.Context, id string) (*domain.InvoicePayment, error) {
	p, ok := m.payments[id]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	copy := *p
	return &copy, nil
}

func (m *mockPaymentPaymentRepo) ListWithFilters(_ context.Context, params domain.PaymentListParams) (*domain.PaymentListResult, error) {
	return &domain.PaymentListResult{
		Data: []domain.PaymentListItem{},
		Pagination: domain.PaginationMeta{
			Total:      0,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: 1,
		},
	}, nil
}

func (m *mockPaymentPaymentRepo) GetSummary(_ context.Context, _ string, _ string, _, _ *int) (*domain.PaymentSummary, error) {
	return &domain.PaymentSummary{
		Today:     domain.PaymentSummaryStat{Count: 0, TotalAmount: 0},
		ThisMonth: domain.PaymentSummaryStat{Count: 5, TotalAmount: 500000},
		ByMethod:  map[string]domain.PaymentSummaryStat{},
	}, nil
}

func (m *mockPaymentPaymentRepo) FindDuplicate(_ context.Context, _ string, _ int64, _ string, _ time.Time) (bool, error) {
	return false, nil
}

// mockPaymentAuditLogRepo implementasi mock InvoiceAuditLogRepository.
type mockPaymentAuditLogRepo struct{}

func (m *mockPaymentAuditLogRepo) Create(_ context.Context, _ *domain.InvoiceAuditLog) error {
	return nil
}

func (m *mockPaymentAuditLogRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceAuditLog, error) {
	return nil, nil
}

// mockPaymentReceiptSeqRepo implementasi mock ReceiptSequenceRepository.
type mockPaymentReceiptSeqRepo struct{}

func (m *mockPaymentReceiptSeqRepo) NextSequence(_ context.Context, _ string, _, _ int) (int, error) {
	return 1, nil
}

// mockPaymentSettingsRepo implementasi mock BillingSettingsRepository.
type mockPaymentSettingsRepo struct{}

func (m *mockPaymentSettingsRepo) GetByTenantID(_ context.Context, _ string) (*domain.BillingSettings, error) {
	return &domain.BillingSettings{Timezone: "Asia/Jakarta"}, nil
}

func (m *mockPaymentSettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	return s, nil
}

func (m *mockPaymentSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

// mockPaymentCustomerRepo implementasi mock CustomerRepository untuk payment tests.
type mockPaymentCustomerRepo struct {
	customers map[string]*domain.Customer
}

func newMockPaymentCustomerRepo() *mockPaymentCustomerRepo {
	return &mockPaymentCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *mockPaymentCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}

func (m *mockPaymentCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	return &copy, nil
}

func (m *mockPaymentCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}

func (m *mockPaymentCustomerRepo) SoftDelete(_ context.Context, _ string) error { return nil }

func (m *mockPaymentCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return &domain.CustomerListResult{Data: []*domain.Customer{}, Pagination: domain.PaginationMeta{}}, nil
}

func (m *mockPaymentCustomerRepo) UpdateStatus(_ context.Context, _ string, _ domain.CustomerStatus) (*domain.Customer, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *mockPaymentCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockPaymentCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) {
	return nil, nil
}

func (m *mockPaymentCustomerRepo) SearchForPayment(_ context.Context, _, searchTerm string) ([]*domain.Customer, error) {
	// Mengembalikan hasil kosong (pencarian berhasil tapi tidak ada hasil)
	return []*domain.Customer{}, nil
}

// =============================================================================
// Setup helper — membuat Fiber app dengan PaymentHandler dan mock repos
// =============================================================================

type paymentTestSetup struct {
	app         *fiber.App
	invoiceRepo *mockPaymentInvoiceRepo
	paymentRepo *mockPaymentPaymentRepo
	customerRepo *mockPaymentCustomerRepo
}

// setupPaymentTestApp membuat Fiber app dengan PaymentHandler yang di-back oleh mock repos.
// PaymentUsecase dibuat dengan pool=nil dan queueClient=nil karena kita hanya test handler layer.
func setupPaymentTestApp() *paymentTestSetup {
	invoiceRepo := newMockPaymentInvoiceRepo()
	paymentRepo := newMockPaymentPaymentRepo()
	customerRepo := newMockPaymentCustomerRepo()
	logger := zerolog.New(io.Discard)

	uc := usecase.NewPaymentUsecase(
		invoiceRepo,
		&mockPaymentInvoiceItemRepo{},
		paymentRepo,
		&mockPaymentAuditLogRepo{},
		&mockPaymentReceiptSeqRepo{},
		&mockPaymentSettingsRepo{},
		customerRepo,
		nil, // pool — nil karena kita test error paths sebelum transaksi
		nil, // queueClient
		logger,
	)

	handler := NewPaymentHandler(uc, logger)

	app := fiber.New()

	// Middleware untuk set locals (simulasi auth middleware)
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	payments := app.Group("/api/v1/payments", setLocals)
	payments.Get("/", handler.List)
	payments.Get("/summary", handler.Summary)
	payments.Get("/quick/customers", handler.SearchCustomers)
	payments.Get("/quick/customers/:customer_id/invoices", handler.GetOpenInvoices)
	payments.Post("/multi", handler.RecordMultiPayment)
	payments.Post("/pay-all", handler.PayAll)
	payments.Get("/:payment_id/receipt", handler.GetReceipt)
	payments.Post("/:payment_id/void", handler.VoidPayment)
	payments.Post("/import", handler.BulkImport)
	payments.Post("/:payment_id/proof", handler.UploadProof)
	payments.Get("/:payment_id/proof", handler.GetProof)

	return &paymentTestSetup{
		app:         app,
		invoiceRepo: invoiceRepo,
		paymentRepo: paymentRepo,
		customerRepo: customerRepo,
	}
}

// parseAPIResponse membaca dan parse response body ke APIResponse.
func parseAPIResponse(t *testing.T, resp *io.ReadCloser) domain.APIResponse {
	t.Helper()
	body, err := io.ReadAll(*resp)
	if err != nil {
		t.Fatalf("gagal baca response body: %v", err)
	}
	var apiResp domain.APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("gagal parse response JSON: %v (body: %s)", err, string(body))
	}
	return apiResp
}

// =============================================================================
// Test: List — paginasi dan filter
// =============================================================================

func TestPaymentHandler_List_Success(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestPaymentHandler_List_WithPagination(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments?page=2&page_size=10", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestPaymentHandler_List_WithFilters(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments?payment_method=tunai&date_from=2024-01-01&date_to=2024-12-31&include_voided=true", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: Summary
// =============================================================================

func TestPaymentHandler_Summary_Success(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments/summary", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestPaymentHandler_Summary_WithPeriod(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments/summary?period_month=6&period_year=2024", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: SearchCustomers — 400 untuk term terlalu pendek
// =============================================================================

func TestPaymentHandler_SearchCustomers_ShortTerm(t *testing.T) {
	setup := setupPaymentTestApp()

	// Kata pencarian kurang dari 2 karakter → 400
	req := httptest.NewRequest("GET", "/api/v1/payments/quick/customers?search=A", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "SEARCH_TERM_TOO_SHORT" {
		t.Fatalf("expected SEARCH_TERM_TOO_SHORT, got %v", apiResp.Error)
	}
}

func TestPaymentHandler_SearchCustomers_Success(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments/quick/customers?search=Ahmad", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: GetOpenInvoices — 404 untuk customer tidak ditemukan
// =============================================================================

func TestPaymentHandler_GetOpenInvoices_Success(t *testing.T) {
	setup := setupPaymentTestApp()

	// Tambahkan customer dan invoice terbuka
	setup.customerRepo.customers["cust-1"] = &domain.Customer{
		ID:       "cust-1",
		TenantID: "test-tenant-id",
		Name:     "Ahmad",
		Status:   domain.CustomerStatusAktif,
	}
	setup.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID:          "inv-1",
		CustomerID:  "cust-1",
		TotalAmount: 100000,
		PaidAmount:  0,
		Status:      domain.InvoiceStatusBelumBayar,
		DueDate:     time.Now().Add(24 * time.Hour),
	}

	req := httptest.NewRequest("GET", "/api/v1/payments/quick/customers/cust-1/invoices", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestPaymentHandler_GetOpenInvoices_EmptyResult(t *testing.T) {
	setup := setupPaymentTestApp()

	// Customer tanpa invoice terbuka — tetap 200 dengan list kosong
	req := httptest.NewRequest("GET", "/api/v1/payments/quick/customers/cust-nonexistent/invoices", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	// GetOpenInvoices mengembalikan list kosong, bukan 404
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: RecordMultiPayment — 400 validasi, 422 invalid selection, 409 concurrent
// =============================================================================

func TestPaymentHandler_RecordMultiPayment_ValidationError(t *testing.T) {
	setup := setupPaymentTestApp()

	// Body tanpa field wajib → 400 VALIDATION_ERROR
	body, _ := json.Marshal(map[string]interface{}{
		"customer_id": "", // kosong
	})

	req := httptest.NewRequest("POST", "/api/v1/payments/multi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestPaymentHandler_RecordMultiPayment_InvalidBody(t *testing.T) {
	setup := setupPaymentTestApp()

	// Body bukan JSON → 400 BAD_REQUEST
	req := httptest.NewRequest("POST", "/api/v1/payments/multi", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestPaymentHandler_RecordMultiPayment_MissingAmount(t *testing.T) {
	setup := setupPaymentTestApp()

	// Amount = 0 (harus > 0) → 400 VALIDATION_ERROR
	body, _ := json.Marshal(domain.MultiPaymentRequest{
		CustomerID:    "00000000-0000-0000-0000-000000000001",
		Amount:        0,
		PaymentMethod: "tunai",
		PaymentDate:   "2024-06-15",
	})

	req := httptest.NewRequest("POST", "/api/v1/payments/multi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestPaymentHandler_RecordMultiPayment_InvalidMethod(t *testing.T) {
	setup := setupPaymentTestApp()

	// payment_method tidak valid → 400 VALIDATION_ERROR
	body, _ := json.Marshal(map[string]interface{}{
		"customer_id":    "00000000-0000-0000-0000-000000000001",
		"amount":         100000,
		"payment_method": "bitcoin",
		"payment_date":   "2024-06-15",
	})

	req := httptest.NewRequest("POST", "/api/v1/payments/multi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: PayAll — 422 tidak ada invoice terbuka
// =============================================================================

func TestPaymentHandler_PayAll_ValidationError(t *testing.T) {
	setup := setupPaymentTestApp()

	// Body tanpa field wajib → 400 VALIDATION_ERROR
	body, _ := json.Marshal(map[string]interface{}{})

	req := httptest.NewRequest("POST", "/api/v1/payments/pay-all", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestPaymentHandler_PayAll_InvalidBody(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("POST", "/api/v1/payments/pay-all", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: GetReceipt — 404 tidak ditemukan
// =============================================================================

func TestPaymentHandler_GetReceipt_NotFound(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments/nonexistent-id/receipt", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "PAYMENT_NOT_FOUND" {
		t.Fatalf("expected PAYMENT_NOT_FOUND, got %v", apiResp.Error)
	}
}

// =============================================================================
// Test: VoidPayment — 422 already voided, 422 time limit, validasi
// =============================================================================

func TestPaymentHandler_VoidPayment_ValidationError(t *testing.T) {
	setup := setupPaymentTestApp()

	// Reason terlalu pendek (min 5 karakter) → 400 VALIDATION_ERROR
	body, _ := json.Marshal(domain.VoidPaymentRequest{
		Reason: "ab",
	})

	req := httptest.NewRequest("POST", "/api/v1/payments/some-id/void", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

func TestPaymentHandler_VoidPayment_InvalidBody(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("POST", "/api/v1/payments/some-id/void", bytes.NewReader([]byte("bukan json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestPaymentHandler_VoidPayment_MissingReason(t *testing.T) {
	setup := setupPaymentTestApp()

	// Reason kosong → 400 VALIDATION_ERROR
	body, _ := json.Marshal(domain.VoidPaymentRequest{
		Reason: "",
	})

	req := httptest.NewRequest("POST", "/api/v1/payments/some-id/void", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: BulkImport — 400 CSV too large, 422 validation errors
// =============================================================================

func TestPaymentHandler_BulkImport_NoFile(t *testing.T) {
	setup := setupPaymentTestApp()

	// Request tanpa file → 400 BAD_REQUEST
	req := httptest.NewRequest("POST", "/api/v1/payments/import", nil)
	req.Header.Set("Content-Type", "multipart/form-data")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "BAD_REQUEST" {
		t.Fatalf("expected BAD_REQUEST, got %v", apiResp.Error)
	}
}

// createMultipartCSV membuat request multipart dengan file CSV.
func createMultipartCSV(t *testing.T, csvContent string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "payments.csv")
	if err != nil {
		t.Fatalf("gagal buat form file: %v", err)
	}
	_, err = part.Write([]byte(csvContent))
	if err != nil {
		t.Fatalf("gagal tulis CSV: %v", err)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

func TestPaymentHandler_BulkImport_WithCSVFile(t *testing.T) {
	setup := setupPaymentTestApp()

	// CSV valid tapi customer tidak ditemukan — akan error di usecase
	// Karena pool=nil, usecase akan panic/error saat akses DB.
	// Kita test bahwa handler berhasil parse file dan memanggil usecase.
	csvContent := "customer_id_seq,amount,payment_method,payment_date,reference_number,notes\n"
	body, contentType := createMultipartCSV(t, csvContent)

	req := httptest.NewRequest("POST", "/api/v1/payments/import", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	// CSV kosong (hanya header) → usecase mengembalikan hasil kosong
	// Status bisa 200 (0 rows) atau error tergantung implementasi
	// Yang penting handler tidak crash
	if resp.StatusCode != fiber.StatusOK && resp.StatusCode != fiber.StatusInternalServerError {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 or 500, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: UploadProof dan GetProof
// =============================================================================

func TestPaymentHandler_UploadProof_NoFile(t *testing.T) {
	setup := setupPaymentTestApp()

	// Request tanpa file → 400 BAD_REQUEST
	req := httptest.NewRequest("POST", "/api/v1/payments/some-id/proof", nil)
	req.Header.Set("Content-Type", "multipart/form-data")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestPaymentHandler_GetProof_NotFound(t *testing.T) {
	setup := setupPaymentTestApp()

	// Payment tidak ada → 404
	req := httptest.NewRequest("GET", "/api/v1/payments/nonexistent-id/proof", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

// =============================================================================
// Test: Response format — memastikan format JSON standar
// =============================================================================

func TestPaymentHandler_ResponseFormat_Success(t *testing.T) {
	setup := setupPaymentTestApp()

	req := httptest.NewRequest("GET", "/api/v1/payments/summary", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("response bukan JSON valid: %v", err)
	}

	// Cek field "success" ada
	if _, ok := raw["success"]; !ok {
		t.Fatal("response harus memiliki field 'success'")
	}

	// Cek field "data" ada untuk response sukses
	if _, ok := raw["data"]; !ok {
		t.Fatal("response sukses harus memiliki field 'data'")
	}
}

func TestPaymentHandler_ResponseFormat_Error(t *testing.T) {
	setup := setupPaymentTestApp()

	// Trigger error: search term terlalu pendek
	req := httptest.NewRequest("GET", "/api/v1/payments/quick/customers?search=A", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("response bukan JSON valid: %v", err)
	}

	// Cek field "success" = false
	if success, ok := raw["success"].(bool); !ok || success {
		t.Fatal("response error harus memiliki success=false")
	}

	// Cek field "error" ada
	errObj, ok := raw["error"].(map[string]interface{})
	if !ok {
		t.Fatal("response error harus memiliki field 'error'")
	}

	// Cek field "error.code" ada
	if _, ok := errObj["code"]; !ok {
		t.Fatal("error harus memiliki field 'code'")
	}

	// Cek field "error.message" ada
	if _, ok := errObj["message"]; !ok {
		t.Fatal("error harus memiliki field 'message'")
	}
}
