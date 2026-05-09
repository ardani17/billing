package handler

import (
	"context"
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
// =============================================================================

type mockIsolirCustomerRepo struct {
	customers map[string]*domain.Customer
}

func newMockIsolirCustomerRepo() *mockIsolirCustomerRepo {
	return &mockIsolirCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *mockIsolirCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}
func (m *mockIsolirCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	return &copy, nil
}
func (m *mockIsolirCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}
func (m *mockIsolirCustomerRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *mockIsolirCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return &domain.CustomerListResult{Data: []*domain.Customer{}, Pagination: domain.PaginationMeta{}}, nil
}
func (m *mockIsolirCustomerRepo) UpdateStatus(_ context.Context, id string, status domain.CustomerStatus) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	c.Status = status
	copy := *c
	return &copy, nil
}
func (m *mockIsolirCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) {
	return nil, nil
}
func (m *mockIsolirCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	result := make(map[domain.CustomerStatus]int64)
	for _, c := range m.customers {
		result[c.Status]++
	}
	return result, nil
}
func (m *mockIsolirCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *mockIsolirCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockIsolirCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *mockIsolirCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *mockIsolirCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *mockIsolirCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) {
	return nil, nil
}
func (m *mockIsolirCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}

type mockIsolirInvoiceRepo struct {
	invoices map[string]*domain.Invoice
}

func newMockIsolirInvoiceRepo() *mockIsolirInvoiceRepo {
	return &mockIsolirInvoiceRepo{invoices: make(map[string]*domain.Invoice)}
}

func (m *mockIsolirInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	copy := *inv
	m.invoices[copy.ID] = &copy
	return &copy, nil
}
func (m *mockIsolirInvoiceRepo) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	copy := *inv
	return &copy, nil
}
func (m *mockIsolirInvoiceRepo) Update(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	copy := *inv
	m.invoices[copy.ID] = &copy
	return &copy, nil
}
func (m *mockIsolirInvoiceRepo) UpdateStatus(_ context.Context, id string, status domain.InvoiceStatus, _ int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.Status = status
	copy := *inv
	return &copy, nil
}
func (m *mockIsolirInvoiceRepo) UpdatePaidAmount(_ context.Context, id string, paidAmount int64, _ int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.PaidAmount = paidAmount
	copy := *inv
	return &copy, nil
}
func (m *mockIsolirInvoiceRepo) List(_ context.Context, _ domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	return &domain.InvoiceListResult{Data: []*domain.Invoice{}, Pagination: domain.PaginationMeta{}}, nil
}
func (m *mockIsolirInvoiceRepo) ExistsForPeriod(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}
func (m *mockIsolirInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}
func (m *mockIsolirInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) GetSummary(_ context.Context, _ string, _, _ *int) (*domain.InvoiceSummary, error) {
	return &domain.InvoiceSummary{}, nil
}
func (m *mockIsolirInvoiceRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) FindOpenByCustomer(_ context.Context, _ string) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) FindOpenByCustomerForUpdate(_ context.Context, _ string) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) GetByIDsForUpdate(_ context.Context, _ []string) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) FindOverdueForIsolir(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) FindOverdueForSuspend(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockIsolirInvoiceRepo) HasOutstandingInvoices(_ context.Context, customerID string) (bool, error) {
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			inv.Status != domain.InvoiceStatusLunas && inv.Status != domain.InvoiceStatusBatal {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockIsolirInvoiceRepo) SumOutstandingAmount(_ context.Context, customerID string) (int64, error) {
	var total int64
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			inv.Status != domain.InvoiceStatusLunas && inv.Status != domain.InvoiceStatusBatal {
			total += inv.TotalAmount - inv.PaidAmount
		}
	}
	return total, nil
}
func (m *mockIsolirInvoiceRepo) CountOutstandingInvoices(_ context.Context, customerID string) (int, error) {
	count := 0
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			inv.Status != domain.InvoiceStatusLunas && inv.Status != domain.InvoiceStatusBatal {
			count++
		}
	}
	return count, nil
}

type mockIsolirInvoiceItemRepo struct {
	items map[string][]*domain.InvoiceItem // invoiceID -> items
}

func newMockIsolirInvoiceItemRepo() *mockIsolirInvoiceItemRepo {
	return &mockIsolirInvoiceItemRepo{items: make(map[string][]*domain.InvoiceItem)}
}

func (m *mockIsolirInvoiceItemRepo) BulkCreate(_ context.Context, items []*domain.InvoiceItem) ([]*domain.InvoiceItem, error) {
	for _, item := range items {
		m.items[item.InvoiceID] = append(m.items[item.InvoiceID], item)
	}
	return items, nil
}
func (m *mockIsolirInvoiceItemRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.InvoiceItem, error) {
	return m.items[invoiceID], nil
}
func (m *mockIsolirInvoiceItemRepo) DeleteByInvoice(_ context.Context, invoiceID string) error {
	delete(m.items, invoiceID)
	return nil
}

type mockIsolirPendingSyncRepo struct {
	syncs map[string]*domain.PendingSync
}

func newMockIsolirPendingSyncRepo() *mockIsolirPendingSyncRepo {
	return &mockIsolirPendingSyncRepo{syncs: make(map[string]*domain.PendingSync)}
}

func (m *mockIsolirPendingSyncRepo) Create(_ context.Context, ps *domain.PendingSync) (*domain.PendingSync, error) {
	if ps.ID == "" {
		ps.ID = "ps-" + ps.CustomerID
	}
	copy := *ps
	m.syncs[copy.ID] = &copy
	return &copy, nil
}
func (m *mockIsolirPendingSyncRepo) GetByID(_ context.Context, id string) (*domain.PendingSync, error) {
	ps, ok := m.syncs[id]
	if !ok {
		return nil, domain.ErrNoPendingSync
	}
	copy := *ps
	return &copy, nil
}
func (m *mockIsolirPendingSyncRepo) UpdateStatus(_ context.Context, id string, status domain.SyncStatus) error {
	if ps, ok := m.syncs[id]; ok {
		ps.Status = status
	}
	return nil
}
func (m *mockIsolirPendingSyncRepo) UpdateRetry(_ context.Context, id string, retryCount int, nextRetryAt time.Time, errMsg string) error {
	if ps, ok := m.syncs[id]; ok {
		ps.RetryCount = retryCount
		ps.NextRetryAt = &nextRetryAt
		ps.ErrorMessage = errMsg
	}
	return nil
}
func (m *mockIsolirPendingSyncRepo) MarkCompleted(_ context.Context, id string) error {
	if ps, ok := m.syncs[id]; ok {
		ps.Status = domain.SyncStatusCompleted
	}
	return nil
}
func (m *mockIsolirPendingSyncRepo) MarkFailed(_ context.Context, id string, errMsg string) error {
	if ps, ok := m.syncs[id]; ok {
		ps.Status = domain.SyncStatusFailed
		ps.ErrorMessage = errMsg
	}
	return nil
}
func (m *mockIsolirPendingSyncRepo) FindPendingForRetry(_ context.Context, _ int) ([]*domain.PendingSync, error) {
	return nil, nil
}
func (m *mockIsolirPendingSyncRepo) FindByCustomer(_ context.Context, customerID string) ([]*domain.PendingSync, error) {
	var result []*domain.PendingSync
	for _, ps := range m.syncs {
		if ps.CustomerID == customerID && (ps.Status == domain.SyncStatusPending || ps.Status == domain.SyncStatusFailed) {
			copy := *ps
			result = append(result, &copy)
		}
	}
	return result, nil
}
func (m *mockIsolirPendingSyncRepo) FindByTenantAndStatus(_ context.Context, tenantID string, status *domain.SyncStatus, page, pageSize int) (*domain.PendingSyncListResult, error) {
	var filtered []*domain.PendingSync
	for _, ps := range m.syncs {
		if ps.TenantID != tenantID {
			continue
		}
		if status != nil && ps.Status != *status {
			continue
		}
		copy := *ps
		filtered = append(filtered, &copy)
	}
	total := int64(len(filtered))
	totalPages := 1
	if total > 0 && pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return &domain.PendingSyncListResult{
		Items: filtered, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages,
	}, nil
}
func (m *mockIsolirPendingSyncRepo) ResetRetryForCustomer(_ context.Context, customerID string) error {
	for _, ps := range m.syncs {
		if ps.CustomerID == customerID {
			ps.RetryCount = 0
		}
	}
	return nil
}
func (m *mockIsolirPendingSyncRepo) ResetRetryAll(_ context.Context, tenantID string) (int, error) {
	count := 0
	for _, ps := range m.syncs {
		if ps.TenantID == tenantID && (ps.Status == domain.SyncStatusPending || ps.Status == domain.SyncStatusFailed) {
			ps.RetryCount = 0
			count++
		}
	}
	return count, nil
}
func (m *mockIsolirPendingSyncRepo) CountByTenantAndStatuses(_ context.Context, tenantID string, statuses []domain.SyncStatus) (int64, error) {
	var count int64
	statusSet := make(map[domain.SyncStatus]bool)
	for _, s := range statuses {
		statusSet[s] = true
	}
	for _, ps := range m.syncs {
		if ps.TenantID == tenantID && statusSet[ps.Status] {
			count++
		}
	}
	return count, nil
}

type mockIsolirSettingsRepo struct{}

func (m *mockIsolirSettingsRepo) GetByTenantID(_ context.Context, _ string) (*domain.BillingSettings, error) {
	return &domain.BillingSettings{Timezone: "Asia/Jakarta"}, nil
}
func (m *mockIsolirSettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	return s, nil
}
func (m *mockIsolirSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

type mockIsolirAuditLogRepo struct{}

func (m *mockIsolirAuditLogRepo) Create(_ context.Context, _ *domain.InvoiceAuditLog) error {
	return nil
}
func (m *mockIsolirAuditLogRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceAuditLog, error) {
	return nil, nil
}

// =============================================================================
// =============================================================================

type isolirTestSetup struct {
	app             *fiber.App
	customerRepo    *mockIsolirCustomerRepo
	invoiceRepo     *mockIsolirInvoiceRepo
	invoiceItemRepo *mockIsolirInvoiceItemRepo
	pendingSyncRepo *mockIsolirPendingSyncRepo
}

func setupIsolirTestApp() *isolirTestSetup {
	customerRepo := newMockIsolirCustomerRepo()
	invoiceRepo := newMockIsolirInvoiceRepo()
	invoiceItemRepo := newMockIsolirInvoiceItemRepo()
	pendingSyncRepo := newMockIsolirPendingSyncRepo()
	logger := zerolog.New(io.Discard)

	uc := usecase.NewIsolirUsecase(
		customerRepo,
		invoiceRepo,
		invoiceItemRepo,
		pendingSyncRepo,
		&mockIsolirSettingsRepo{},
		&mockIsolirAuditLogRepo{},
		nil, // pool - nil karena kita test handler layer
		nil, // queueClient
		logger,
	)

	handler := NewIsolirHandler(uc, logger)

	app := fiber.New()

	// Middleware untuk atur locals (simulasi auth middleware)
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	// Registrasi route sesuai router.go
	isolir := app.Group("/api/v1/isolir", setLocals)
	isolir.Post("/sync/:customer_id", handler.ManualSync)
	isolir.Post("/sync-all", handler.ManualSyncAll)
	isolir.Get("/pending-syncs", handler.ListPendingSyncs)
	isolir.Get("/summary", handler.Summary)

	invoices := app.Group("/api/v1/invoices", setLocals)
	invoices.Post("/:id/waive-penalty", handler.WaivePenalty)

	customers := app.Group("/api/v1/customers", setLocals)
	customers.Post("/:id/reactivate", handler.Reactivate)

	return &isolirTestSetup{
		app:             app,
		customerRepo:    customerRepo,
		invoiceRepo:     invoiceRepo,
		invoiceItemRepo: invoiceItemRepo,
		pendingSyncRepo: pendingSyncRepo,
	}
}

func parseIsolirAPIResponse(t *testing.T, resp *io.ReadCloser) domain.APIResponse {
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
// =============================================================================

func TestIsolirHandler_ManualSync_Success(t *testing.T) {
	setup := setupIsolirTestApp()

	custID := "00000000-0000-0000-0000-000000000001"
	// Tambahkan customer dan pending sync
	setup.customerRepo.customers[custID] = &domain.Customer{
		ID: custID, TenantID: "test-tenant-id", Name: "Ahmad", Status: domain.CustomerStatusIsolir,
	}
	setup.pendingSyncRepo.syncs["ps-1"] = &domain.PendingSync{
		ID: "ps-1", TenantID: "test-tenant-id", CustomerID: custID,
		OperationType: domain.SyncOpIsolir, Status: domain.SyncStatusPending, MaxRetries: 5,
	}

	req := httptest.NewRequest("POST", "/api/v1/isolir/sync/"+custID, nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestIsolirHandler_ManualSync_NoPendingSync(t *testing.T) {
	setup := setupIsolirTestApp()

	custID := "00000000-0000-0000-0000-000000000002"
	// Customer ada tapi tidak ada pending sync
	setup.customerRepo.customers[custID] = &domain.Customer{
		ID: custID, TenantID: "test-tenant-id", Name: "Budi", Status: domain.CustomerStatusAktif,
	}

	req := httptest.NewRequest("POST", "/api/v1/isolir/sync/"+custID, nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "NO_PENDING_SYNC" {
		t.Fatalf("expected NO_PENDING_SYNC, got %v", apiResp.Error)
	}
}

func TestIsolirHandler_ManualSync_InvalidUUID(t *testing.T) {
	setup := setupIsolirTestApp()

	req := httptest.NewRequest("POST", "/api/v1/isolir/sync/bukan-uuid", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "BAD_REQUEST" {
		t.Fatalf("expected BAD_REQUEST, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestIsolirHandler_ManualSyncAll_Success(t *testing.T) {
	setup := setupIsolirTestApp()

	// Tambahkan beberapa pending sync
	setup.pendingSyncRepo.syncs["ps-1"] = &domain.PendingSync{
		ID: "ps-1", TenantID: "test-tenant-id", CustomerID: "cust-1",
		OperationType: domain.SyncOpIsolir, Status: domain.SyncStatusPending, MaxRetries: 5,
	}
	setup.pendingSyncRepo.syncs["ps-2"] = &domain.PendingSync{
		ID: "ps-2", TenantID: "test-tenant-id", CustomerID: "cust-2",
		OperationType: domain.SyncOpUnIsolir, Status: domain.SyncStatusFailed, MaxRetries: 5,
	}

	req := httptest.NewRequest("POST", "/api/v1/isolir/sync-all", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
	// Verifikasi data mengandung count
	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	count, ok := dataMap["count"]
	if !ok {
		t.Fatal("expected 'count' field in response data")
	}
	if count.(float64) != 2 {
		t.Fatalf("expected count=2, got %v", count)
	}
}

// =============================================================================
// =============================================================================

func TestIsolirHandler_ListPendingSyncs_Success(t *testing.T) {
	setup := setupIsolirTestApp()

	setup.pendingSyncRepo.syncs["ps-1"] = &domain.PendingSync{
		ID: "ps-1", TenantID: "test-tenant-id", CustomerID: "cust-1",
		OperationType: domain.SyncOpIsolir, Status: domain.SyncStatusPending, MaxRetries: 5,
	}

	req := httptest.NewRequest("GET", "/api/v1/isolir/pending-syncs", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestIsolirHandler_ListPendingSyncs_WithPagination(t *testing.T) {
	setup := setupIsolirTestApp()

	req := httptest.NewRequest("GET", "/api/v1/isolir/pending-syncs?page=1&page_size=10", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestIsolirHandler_ListPendingSyncs_FilterByStatus(t *testing.T) {
	setup := setupIsolirTestApp()

	setup.pendingSyncRepo.syncs["ps-1"] = &domain.PendingSync{
		ID: "ps-1", TenantID: "test-tenant-id", CustomerID: "cust-1",
		OperationType: domain.SyncOpIsolir, Status: domain.SyncStatusPending, MaxRetries: 5,
	}
	setup.pendingSyncRepo.syncs["ps-2"] = &domain.PendingSync{
		ID: "ps-2", TenantID: "test-tenant-id", CustomerID: "cust-2",
		OperationType: domain.SyncOpIsolir, Status: domain.SyncStatusFailed, MaxRetries: 5,
	}

	req := httptest.NewRequest("GET", "/api/v1/isolir/pending-syncs?status=pending", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

// =============================================================================
// =============================================================================

func TestIsolirHandler_Summary_Success(t *testing.T) {
	setup := setupIsolirTestApp()

	// Tambahkan customer dengan berbagai status
	setup.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "test-tenant-id", Name: "Ahmad", Status: domain.CustomerStatusIsolir,
	}
	setup.customerRepo.customers["cust-2"] = &domain.Customer{
		ID: "cust-2", TenantID: "test-tenant-id", Name: "Budi", Status: domain.CustomerStatusSuspend,
	}
	setup.pendingSyncRepo.syncs["ps-1"] = &domain.PendingSync{
		ID: "ps-1", TenantID: "test-tenant-id", CustomerID: "cust-1",
		OperationType: domain.SyncOpIsolir, Status: domain.SyncStatusPending, MaxRetries: 5,
	}

	req := httptest.NewRequest("GET", "/api/v1/isolir/summary", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
	// Verifikasi struktur summary
	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	for _, key := range []string{"total_isolir", "total_suspend", "total_pending_sync", "revenue_at_risk"} {
		if _, exists := dataMap[key]; !exists {
			t.Fatalf("expected '%s' field in summary response", key)
		}
	}
}

// =============================================================================
// =============================================================================

func TestIsolirHandler_WaivePenalty_Success(t *testing.T) {
	setup := setupIsolirTestApp()

	invID := "00000000-0000-0000-0000-000000000010"
	setup.invoiceRepo.invoices[invID] = &domain.Invoice{
		ID: invID, TenantID: "test-tenant-id", CustomerID: "cust-1",
		InvoiceNumber: "INV-001", Subtotal: 100000, PenaltyAmount: 10000,
		TotalAmount: 110000, Status: domain.InvoiceStatusTerlambat,
	}
	// Tambahkan item denda
	setup.invoiceItemRepo.items[invID] = []*domain.InvoiceItem{
		{ID: "item-1", InvoiceID: invID, ItemType: domain.ItemTypeMonthly, Amount: 100000},
		{ID: "item-2", InvoiceID: invID, ItemType: domain.ItemTypePenalty, Amount: 10000},
	}

	req := httptest.NewRequest("POST", "/api/v1/invoices/"+invID+"/waive-penalty", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestIsolirHandler_WaivePenalty_NotFound(t *testing.T) {
	setup := setupIsolirTestApp()

	invID := "00000000-0000-0000-0000-000000000099"
	req := httptest.NewRequest("POST", "/api/v1/invoices/"+invID+"/waive-penalty", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "INVOICE_NOT_FOUND" {
		t.Fatalf("expected INVOICE_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestIsolirHandler_WaivePenalty_NoPenalty(t *testing.T) {
	setup := setupIsolirTestApp()

	invID := "00000000-0000-0000-0000-000000000011"
	setup.invoiceRepo.invoices[invID] = &domain.Invoice{
		ID: invID, TenantID: "test-tenant-id", CustomerID: "cust-1",
		InvoiceNumber: "INV-002", Subtotal: 100000, PenaltyAmount: 0,
		TotalAmount: 100000, Status: domain.InvoiceStatusTerlambat,
	}
	// Hanya item biasa, tanpa denda
	setup.invoiceItemRepo.items[invID] = []*domain.InvoiceItem{
		{ID: "item-1", InvoiceID: invID, ItemType: domain.ItemTypeMonthly, Amount: 100000},
	}

	req := httptest.NewRequest("POST", "/api/v1/invoices/"+invID+"/waive-penalty", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "NO_PENALTY_TO_WAIVE" {
		t.Fatalf("expected NO_PENALTY_TO_WAIVE, got %v", apiResp.Error)
	}
}

func TestIsolirHandler_WaivePenalty_NotEditable(t *testing.T) {
	setup := setupIsolirTestApp()

	invID := "00000000-0000-0000-0000-000000000012"
	setup.invoiceRepo.invoices[invID] = &domain.Invoice{
		ID: invID, TenantID: "test-tenant-id", CustomerID: "cust-1",
		InvoiceNumber: "INV-003", Subtotal: 100000, PenaltyAmount: 10000,
		TotalAmount: 110000, Status: domain.InvoiceStatusLunas, // sudah lunas
	}

	req := httptest.NewRequest("POST", "/api/v1/invoices/"+invID+"/waive-penalty", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "INVOICE_NOT_EDITABLE" {
		t.Fatalf("expected INVOICE_NOT_EDITABLE, got %v", apiResp.Error)
	}
}

// =============================================================================
// =============================================================================

func TestIsolirHandler_Reactivate_Success(t *testing.T) {
	setup := setupIsolirTestApp()

	custID := "00000000-0000-0000-0000-000000000020"
	setup.customerRepo.customers[custID] = &domain.Customer{
		ID: custID, TenantID: "test-tenant-id", Name: "Ahmad",
		Status: domain.CustomerStatusSuspend,
	}
	// Semua invoice sudah lunas
	setup.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", CustomerID: custID, TotalAmount: 100000,
		PaidAmount: 100000, Status: domain.InvoiceStatusLunas,
	}

	req := httptest.NewRequest("POST", "/api/v1/customers/"+custID+"/reactivate", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestIsolirHandler_Reactivate_NotFound(t *testing.T) {
	setup := setupIsolirTestApp()

	custID := "00000000-0000-0000-0000-000000000099"
	req := httptest.NewRequest("POST", "/api/v1/customers/"+custID+"/reactivate", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseIsolirAPIResponse(t, &resp.Body)
	if apiResp.Error == nil || apiResp.Error.Code != "CUSTOMER_NOT_FOUND" {
		t.Fatalf("expected CUSTOMER_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestIsolirHandler_Reactivate_OutstandingInvoices(t *testing.T) {
	setup := setupIsolirTestApp()

	custID := "00000000-0000-0000-0000-000000000021"
	setup.customerRepo.customers[custID] = &domain.Customer{
		ID: custID, TenantID: "test-tenant-id", Name: "Budi",
		Status: domain.CustomerStatusSuspend,
	}
	// Invoice belum lunas
	setup.invoiceRepo.invoices["inv-2"] = &domain.Invoice{
		ID: "inv-2", CustomerID: custID, TotalAmount: 200000,
		PaidAmount: 0, Status: domain.InvoiceStatusTerlambat,
	}

	req := httptest.NewRequest("POST", "/api/v1/customers/"+custID+"/reactivate", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	var apiResp domain.APIResponse
	json.Unmarshal(body, &apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "OUTSTANDING_INVOICES_EXIST" {
		t.Fatalf("expected OUTSTANDING_INVOICES_EXIST, got %v", apiResp.Error)
	}
}

func TestIsolirHandler_Reactivate_InvalidStatus(t *testing.T) {
	setup := setupIsolirTestApp()

	custID := "00000000-0000-0000-0000-000000000022"
	setup.customerRepo.customers[custID] = &domain.Customer{
		ID: custID, TenantID: "test-tenant-id", Name: "Citra",
		Status: domain.CustomerStatusAktif, // bukan suspend
	}

	req := httptest.NewRequest("POST", "/api/v1/customers/"+custID+"/reactivate", nil)
	resp, err := setup.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	var apiResp domain.APIResponse
	json.Unmarshal(body, &apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("expected INVALID_STATUS_TRANSITION, got %v", apiResp.Error)
	}
}
