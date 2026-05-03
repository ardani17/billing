// invoice_handler_test.go berisi unit test untuk InvoiceHandler.
// Menguji HTTP status codes, parsing request, format response untuk semua endpoint CRUD.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// =============================================================================
// Mock repositories untuk InvoiceHandler tests
// =============================================================================

// mockInvoiceRepo adalah implementasi in-memory dari domain.InvoiceRepository.
type mockInvoiceRepo struct {
	invoices map[string]*domain.Invoice
	counter  int
}

func newMockInvoiceRepo() *mockInvoiceRepo {
	return &mockInvoiceRepo{invoices: make(map[string]*domain.Invoice)}
}

func (m *mockInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	m.counter++
	inv.ID = fmt.Sprintf("inv-%d", m.counter)
	inv.CreatedAt = time.Now()
	inv.UpdatedAt = time.Now()
	copy := *inv
	m.invoices[copy.ID] = &copy
	return &copy, nil
}

func (m *mockInvoiceRepo) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	copy := *inv
	return &copy, nil
}

func (m *mockInvoiceRepo) Update(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	if _, ok := m.invoices[inv.ID]; !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.UpdatedAt = time.Now()
	copy := *inv
	m.invoices[copy.ID] = &copy
	return &copy, nil
}

func (m *mockInvoiceRepo) UpdateStatus(_ context.Context, id string, status domain.InvoiceStatus, version int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	if inv.Version != version {
		return nil, fmt.Errorf("version conflict")
	}
	inv.Status = status
	inv.Version++
	copy := *inv
	return &copy, nil
}

func (m *mockInvoiceRepo) UpdatePaidAmount(_ context.Context, id string, paidAmount int64, version int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	if inv.Version != version {
		return nil, fmt.Errorf("version conflict")
	}
	inv.PaidAmount = paidAmount
	inv.Version++
	copy := *inv
	return &copy, nil
}

func (m *mockInvoiceRepo) List(_ context.Context, params domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	var filtered []*domain.Invoice
	for _, inv := range m.invoices {
		if params.TenantID != "" && inv.TenantID != params.TenantID {
			continue
		}
		filtered = append(filtered, inv)
	}
	total := int64(len(filtered))
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 25
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	return &domain.InvoiceListResult{
		Data: filtered[start:end],
		Pagination: domain.PaginationMeta{
			Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages,
		},
	}, nil
}

func (m *mockInvoiceRepo) ExistsForPeriod(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *mockInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *mockInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *mockInvoiceRepo) GetSummary(_ context.Context, _ string, _, _ *int) (*domain.InvoiceSummary, error) {
	return &domain.InvoiceSummary{
		Total:    domain.InvoiceSummaryStat{Count: 0, TotalAmount: 0},
		ByStatus: make(map[domain.InvoiceStatus]domain.InvoiceSummaryStat),
	}, nil
}

func (m *mockInvoiceRepo) GetByIDs(_ context.Context, ids []string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, id := range ids {
		if inv, ok := m.invoices[id]; ok {
			copy := *inv
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockInvoiceRepo) FindOpenByCustomer(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			(inv.Status == domain.InvoiceStatusBelumBayar || inv.Status == domain.InvoiceStatusTerlambat || inv.Status == domain.InvoiceStatusBayarSebagian) {
			copy := *inv
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockInvoiceRepo) FindOpenByCustomerForUpdate(ctx context.Context, customerID string) ([]*domain.Invoice, error) {
	return m.FindOpenByCustomer(ctx, customerID)
}

func (m *mockInvoiceRepo) GetByIDsForUpdate(_ context.Context, ids []string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, id := range ids {
		if inv, ok := m.invoices[id]; ok {
			copy := *inv
			result = append(result, &copy)
		}
	}
	return result, nil
}
func (m *mockInvoiceRepo) FindOverdueForIsolir(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) FindOverdueForSuspend(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) HasOutstandingInvoices(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockInvoiceRepo) SumOutstandingAmount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockInvoiceRepo) CountOutstandingInvoices(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// mockInvoiceItemRepo adalah implementasi in-memory dari domain.InvoiceItemRepository.
type mockInvoiceItemRepo struct {
	items map[string][]*domain.InvoiceItem
}

func newMockInvoiceItemRepo() *mockInvoiceItemRepo {
	return &mockInvoiceItemRepo{items: make(map[string][]*domain.InvoiceItem)}
}

func (m *mockInvoiceItemRepo) BulkCreate(_ context.Context, items []*domain.InvoiceItem) ([]*domain.InvoiceItem, error) {
	for _, item := range items {
		m.items[item.InvoiceID] = append(m.items[item.InvoiceID], item)
	}
	return items, nil
}

func (m *mockInvoiceItemRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.InvoiceItem, error) {
	return m.items[invoiceID], nil
}

func (m *mockInvoiceItemRepo) DeleteByInvoice(_ context.Context, invoiceID string) error {
	delete(m.items, invoiceID)
	return nil
}

// mockInvoicePaymentRepo adalah implementasi in-memory dari domain.InvoicePaymentRepository.
type mockInvoicePaymentRepo struct {
	payments map[string][]*domain.InvoicePayment
}

func newMockInvoicePaymentRepo() *mockInvoicePaymentRepo {
	return &mockInvoicePaymentRepo{payments: make(map[string][]*domain.InvoicePayment)}
}

func (m *mockInvoicePaymentRepo) Create(_ context.Context, p *domain.InvoicePayment) (*domain.InvoicePayment, error) {
	m.payments[p.InvoiceID] = append(m.payments[p.InvoiceID], p)
	return p, nil
}

func (m *mockInvoicePaymentRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.InvoicePayment, error) {
	return m.payments[invoiceID], nil
}

func (m *mockInvoicePaymentRepo) VoidPayment(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockInvoicePaymentRepo) GetByID(_ context.Context, _ string) (*domain.InvoicePayment, error) {
	return nil, domain.ErrPaymentNotFound
}

func (m *mockInvoicePaymentRepo) ListWithFilters(_ context.Context, _ domain.PaymentListParams) (*domain.PaymentListResult, error) {
	return &domain.PaymentListResult{Data: []domain.PaymentListItem{}, Pagination: domain.PaginationMeta{}}, nil
}

func (m *mockInvoicePaymentRepo) GetSummary(_ context.Context, _ string, _ string, _, _ *int) (*domain.PaymentSummary, error) {
	return &domain.PaymentSummary{ByMethod: make(map[string]domain.PaymentSummaryStat)}, nil
}

func (m *mockInvoicePaymentRepo) FindDuplicate(_ context.Context, _ string, _ int64, _ string, _ time.Time) (bool, error) {
	return false, nil
}

// mockInvoiceAuditRepo adalah implementasi in-memory dari domain.InvoiceAuditLogRepository.
type mockInvoiceAuditRepo struct {
	logs []*domain.InvoiceAuditLog
}

func newMockInvoiceAuditRepo() *mockInvoiceAuditRepo {
	return &mockInvoiceAuditRepo{logs: make([]*domain.InvoiceAuditLog, 0)}
}

func (m *mockInvoiceAuditRepo) Create(_ context.Context, log *domain.InvoiceAuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockInvoiceAuditRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.InvoiceAuditLog, error) {
	var result []*domain.InvoiceAuditLog
	for _, l := range m.logs {
		if l.InvoiceID == invoiceID {
			result = append(result, l)
		}
	}
	return result, nil
}

// mockInvoiceSequenceRepo adalah implementasi in-memory dari domain.InvoiceSequenceRepository.
type mockInvoiceSequenceRepo struct {
	seq int
}

func newMockInvoiceSequenceRepo() *mockInvoiceSequenceRepo {
	return &mockInvoiceSequenceRepo{}
}

func (m *mockInvoiceSequenceRepo) NextSequence(_ context.Context, _ string, _, _ int) (int, error) {
	m.seq++
	return m.seq, nil
}

// mockBillingSettingsRepo adalah implementasi in-memory dari domain.BillingSettingsRepository.
type mockBillingSettingsRepo struct {
	settings map[string]*domain.BillingSettings
}

func newMockBillingSettingsRepo() *mockBillingSettingsRepo {
	return &mockBillingSettingsRepo{settings: make(map[string]*domain.BillingSettings)}
}

func (m *mockBillingSettingsRepo) GetByTenantID(_ context.Context, tenantID string) (*domain.BillingSettings, error) {
	s, ok := m.settings[tenantID]
	if !ok {
		return nil, domain.ErrBillingSettingsNotFound
	}
	return s, nil
}

func (m *mockBillingSettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	m.settings[s.TenantID] = s
	return s, nil
}

func (m *mockBillingSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	var result []*domain.BillingSettings
	for _, s := range m.settings {
		result = append(result, s)
	}
	return result, nil
}

// mockInvCustomerRepo adalah implementasi in-memory dari domain.CustomerRepository untuk invoice tests.
type mockInvCustomerRepo struct {
	customers map[string]*domain.Customer
}

func newMockInvCustomerRepo() *mockInvCustomerRepo {
	return &mockInvCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *mockInvCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	if c.ID == "" {
		c.ID = fmt.Sprintf("cust-%d", len(m.customers)+1)
	}
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}

func (m *mockInvCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	return &copy, nil
}

func (m *mockInvCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	if _, ok := m.customers[c.ID]; !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}

func (m *mockInvCustomerRepo) SoftDelete(_ context.Context, _ string) error                                  { return nil }
func (m *mockInvCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return &domain.CustomerListResult{Data: []*domain.Customer{}, Pagination: domain.PaginationMeta{Total: 0, Page: 1, PageSize: 25, TotalPages: 1}}, nil
}
func (m *mockInvCustomerRepo) UpdateStatus(_ context.Context, _ string, _ domain.CustomerStatus) (*domain.Customer, error) { return nil, nil }
func (m *mockInvCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) { return nil, nil }
func (m *mockInvCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) { return nil, nil }
func (m *mockInvCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *mockInvCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) { return false, nil }
func (m *mockInvCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) { return nil, nil }
func (m *mockInvCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) { return nil, nil }
func (m *mockInvCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) { return nil, nil }
func (m *mockInvCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) { return nil, nil }
func (m *mockInvCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}

// mockInvPackageRepo adalah implementasi in-memory dari domain.PackageRepository untuk invoice tests.
type mockInvPackageRepo struct {
	packages map[string]*domain.Package
}

func newMockInvPackageRepo() *mockInvPackageRepo {
	return &mockInvPackageRepo{packages: make(map[string]*domain.Package)}
}

func (m *mockInvPackageRepo) Create(_ context.Context, _ *domain.Package) (*domain.Package, error) { return nil, nil }
func (m *mockInvPackageRepo) GetByID(_ context.Context, id string) (*domain.Package, error) {
	p, ok := m.packages[id]
	if !ok {
		return nil, domain.ErrPackageNotFound
	}
	return p, nil
}
func (m *mockInvPackageRepo) Update(_ context.Context, _ *domain.Package) (*domain.Package, error) { return nil, nil }
func (m *mockInvPackageRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockInvPackageRepo) List(_ context.Context, _ domain.PackageListParams) (*domain.PackageListResult, error) { return nil, nil }
func (m *mockInvPackageRepo) UpdateIsActive(_ context.Context, _ string, _ bool) (*domain.Package, error) { return nil, nil }
func (m *mockInvPackageRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) { return false, nil }
func (m *mockInvPackageRepo) CustomerCount(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *mockInvPackageRepo) ListNamesByPrefix(_ context.Context, _, _ string) ([]string, error) { return nil, nil }

// =============================================================================
// Setup helper untuk InvoiceHandler tests
// =============================================================================

// invoiceTestSetup berisi semua dependensi untuk testing InvoiceHandler.
type invoiceTestSetup struct {
	app          *fiber.App
	invoiceRepo  *mockInvoiceRepo
	customerRepo *mockInvCustomerRepo
	packageRepo  *mockInvPackageRepo
	settingsRepo *mockBillingSettingsRepo
	sequenceRepo *mockInvoiceSequenceRepo
}

// setupInvoiceTestApp membuat Fiber app dengan InvoiceHandler yang di-back oleh mock repos.
func setupInvoiceTestApp() *invoiceTestSetup {
	invoiceRepo := newMockInvoiceRepo()
	itemRepo := newMockInvoiceItemRepo()
	paymentRepo := newMockInvoicePaymentRepo()
	auditRepo := newMockInvoiceAuditRepo()
	sequenceRepo := newMockInvoiceSequenceRepo()
	settingsRepo := newMockBillingSettingsRepo()
	customerRepo := newMockInvCustomerRepo()
	packageRepo := newMockInvPackageRepo()
	logger := zerolog.New(io.Discard)

	// Tambah pelanggan aktif default untuk testing (UUID untuk validasi)
	customerRepo.customers["00000000-0000-0000-0000-000000000001"] = &domain.Customer{
		ID:       "00000000-0000-0000-0000-000000000001",
		TenantID: "test-tenant",
		Name:     "Test Customer",
		Status:   domain.CustomerStatusAktif,
		DueDate:  10,
	}

	uc := usecase.NewInvoiceUsecase(
		invoiceRepo, itemRepo, paymentRepo, auditRepo,
		sequenceRepo, settingsRepo, customerRepo, packageRepo,
		nil, nil, logger,
	)
	handler := NewInvoiceHandler(uc, logger)

	app := fiber.New()

	// Middleware untuk set locals (simulasi auth middleware)
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant")
		c.Locals("user_id", "test-user")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	invoices := app.Group("/api/v1/invoices", setLocals)
	invoices.Get("/", handler.List)
	invoices.Get("/summary", handler.Summary)
	invoices.Get("/:id", handler.Get)
	invoices.Post("/", handler.Create)
	invoices.Post("/prepaid", handler.CreatePrepaid)
	invoices.Put("/:id", handler.Edit)

	return &invoiceTestSetup{
		app:          app,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
		packageRepo:  packageRepo,
		settingsRepo: settingsRepo,
		sequenceRepo: sequenceRepo,
	}
}

// =============================================================================
// Unit Tests — InvoiceHandler
// =============================================================================

// --- List endpoint ---

// TestInvoiceHandler_List_Success menguji list invoice berhasil dengan status 200.
func TestInvoiceHandler_List_Success(t *testing.T) {
	setup := setupInvoiceTestApp()

	req := httptest.NewRequest("GET", "/api/v1/invoices", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestInvoiceHandler_List_InvalidPageSize menguji validasi page_size yang tidak valid.
func TestInvoiceHandler_List_InvalidPageSize(t *testing.T) {
	setup := setupInvoiceTestApp()

	req := httptest.NewRequest("GET", "/api/v1/invoices?page_size=99", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Get endpoint ---

// TestInvoiceHandler_Get_NotFound menguji 404 saat invoice tidak ditemukan.
func TestInvoiceHandler_Get_NotFound(t *testing.T) {
	setup := setupInvoiceTestApp()

	req := httptest.NewRequest("GET", "/api/v1/invoices/nonexistent-id", nil)
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
	if apiResp.Error == nil || apiResp.Error.Code != "INVOICE_NOT_FOUND" {
		t.Fatalf("expected INVOICE_NOT_FOUND, got %v", apiResp.Error)
	}
}

// TestInvoiceHandler_Get_Success menguji get invoice berhasil.
func TestInvoiceHandler_Get_Success(t *testing.T) {
	setup := setupInvoiceTestApp()

	// Tambah invoice langsung ke repo
	setup.invoiceRepo.invoices["inv-test"] = &domain.Invoice{
		ID:            "inv-test",
		TenantID:      "test-tenant",
		CustomerID:    "cust-1",
		InvoiceNumber: "INV-2024-01-001",
		Status:        domain.InvoiceStatusBelumBayar,
		Version:       1,
	}

	req := httptest.NewRequest("GET", "/api/v1/invoices/inv-test", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Create endpoint ---

// TestInvoiceHandler_Create_Success menguji pembuatan invoice manual berhasil.
func TestInvoiceHandler_Create_Success(t *testing.T) {
	setup := setupInvoiceTestApp()

	body, _ := json.Marshal(domain.CreateInvoiceRequest{
		CustomerID: "00000000-0000-0000-0000-000000000001",
		DueDate:    "2024-06-15",
		Items: []domain.CreateInvoiceItemRequest{
			{Description: "Tagihan bulanan", Quantity: 1, UnitPrice: 100000},
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestInvoiceHandler_Create_ValidationError menguji validasi gagal saat body tidak valid.
func TestInvoiceHandler_Create_ValidationError(t *testing.T) {
	setup := setupInvoiceTestApp()

	// Body tanpa customer_id dan items
	body, _ := json.Marshal(map[string]interface{}{
		"due_date": "2024-06-15",
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
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
	if apiResp.Error == nil || apiResp.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", apiResp.Error)
	}
}

// TestInvoiceHandler_Create_InvalidBody menguji body JSON yang tidak valid.
func TestInvoiceHandler_Create_InvalidBody(t *testing.T) {
	setup := setupInvoiceTestApp()

	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// TestInvoiceHandler_Create_CustomerNotFound menguji error saat customer tidak ditemukan.
func TestInvoiceHandler_Create_CustomerNotFound(t *testing.T) {
	setup := setupInvoiceTestApp()

	body, _ := json.Marshal(domain.CreateInvoiceRequest{
		CustomerID: "00000000-0000-0000-0000-000000000099",
		DueDate:    "2024-06-15",
		Items: []domain.CreateInvoiceItemRequest{
			{Description: "Test", Quantity: 1, UnitPrice: 50000},
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
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

// --- Edit endpoint ---

// TestInvoiceHandler_Edit_NotEditable menguji 422 saat invoice bukan belum_bayar.
func TestInvoiceHandler_Edit_NotEditable(t *testing.T) {
	setup := setupInvoiceTestApp()

	// Tambah invoice dengan status lunas
	setup.invoiceRepo.invoices["inv-lunas"] = &domain.Invoice{
		ID:       "inv-lunas",
		TenantID: "test-tenant",
		Status:   domain.InvoiceStatusLunas,
		Version:  1,
	}

	body, _ := json.Marshal(domain.EditInvoiceRequest{
		Notes: "update notes",
	})

	req := httptest.NewRequest("PUT", "/api/v1/invoices/inv-lunas", bytes.NewReader(body))
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
	if apiResp.Error == nil || apiResp.Error.Code != "INVOICE_NOT_EDITABLE" {
		t.Fatalf("expected INVOICE_NOT_EDITABLE, got %v", apiResp.Error)
	}
}

// TestInvoiceHandler_Edit_Success menguji edit invoice berhasil.
func TestInvoiceHandler_Edit_Success(t *testing.T) {
	setup := setupInvoiceTestApp()

	// Tambah invoice dengan status belum_bayar
	setup.invoiceRepo.invoices["inv-edit"] = &domain.Invoice{
		ID:       "inv-edit",
		TenantID: "test-tenant",
		Status:   domain.InvoiceStatusBelumBayar,
		DueDate:  time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		Version:  1,
	}

	body, _ := json.Marshal(domain.EditInvoiceRequest{
		Notes: "catatan baru",
	})

	req := httptest.NewRequest("PUT", "/api/v1/invoices/inv-edit", bytes.NewReader(body))
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

// --- Summary endpoint ---

// TestInvoiceHandler_Summary_Success menguji summary berhasil.
func TestInvoiceHandler_Summary_Success(t *testing.T) {
	setup := setupInvoiceTestApp()

	req := httptest.NewRequest("GET", "/api/v1/invoices/summary", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}
