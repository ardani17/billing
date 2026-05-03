// payment_usecase_test.go berisi unit test untuk PaymentUsecase — list, summary, search, open invoices.
package usecase

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// tenantCtxKey adalah key context untuk tenant_id (sama dengan pkg/tenant).
type tenantCtxKey string

const tenantCtxKeyID tenantCtxKey = "tenant_id"

// =============================================================================
// Mock repositories untuk PaymentUsecase tests
// =============================================================================

// paymentMockInvoiceRepo implementasi mock InvoiceRepository.
type paymentMockInvoiceRepo struct {
	invoices map[string]*domain.Invoice
}

func newPaymentMockInvoiceRepo() *paymentMockInvoiceRepo {
	return &paymentMockInvoiceRepo{invoices: make(map[string]*domain.Invoice)}
}

func (m *paymentMockInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	c := *inv
	m.invoices[c.ID] = &c
	return &c, nil
}

func (m *paymentMockInvoiceRepo) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	c := *inv
	return &c, nil
}

func (m *paymentMockInvoiceRepo) Update(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	c := *inv
	m.invoices[c.ID] = &c
	return &c, nil
}

func (m *paymentMockInvoiceRepo) UpdateStatus(_ context.Context, id string, status domain.InvoiceStatus, _ int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.Status = status
	c := *inv
	return &c, nil
}

func (m *paymentMockInvoiceRepo) UpdatePaidAmount(_ context.Context, id string, paidAmount int64, _ int) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	inv.PaidAmount = paidAmount
	c := *inv
	return &c, nil
}

func (m *paymentMockInvoiceRepo) List(_ context.Context, _ domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	return &domain.InvoiceListResult{Data: []*domain.Invoice{}, Pagination: domain.PaginationMeta{}}, nil
}

func (m *paymentMockInvoiceRepo) ExistsForPeriod(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *paymentMockInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *paymentMockInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *paymentMockInvoiceRepo) GetSummary(_ context.Context, _ string, _, _ *int) (*domain.InvoiceSummary, error) {
	return &domain.InvoiceSummary{}, nil
}

func (m *paymentMockInvoiceRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *paymentMockInvoiceRepo) FindOpenByCustomer(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			(inv.Status == domain.InvoiceStatusBelumBayar ||
				inv.Status == domain.InvoiceStatusTerlambat ||
				inv.Status == domain.InvoiceStatusBayarSebagian) {
			c := *inv
			result = append(result, &c)
		}
	}
	return result, nil
}

func (m *paymentMockInvoiceRepo) FindOpenByCustomerForUpdate(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	return m.FindOpenByCustomer(context.Background(), customerID)
}

func (m *paymentMockInvoiceRepo) GetByIDsForUpdate(_ context.Context, ids []string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, id := range ids {
		if inv, ok := m.invoices[id]; ok {
			c := *inv
			result = append(result, &c)
		}
	}
	return result, nil
}

func (m *paymentMockInvoiceRepo) FindOverdueForIsolir(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *paymentMockInvoiceRepo) FindOverdueForSuspend(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *paymentMockInvoiceRepo) HasOutstandingInvoices(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *paymentMockInvoiceRepo) SumOutstandingAmount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *paymentMockInvoiceRepo) CountOutstandingInvoices(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// paymentMockItemRepo implementasi mock InvoiceItemRepository.
type paymentMockItemRepo struct{}

func (m *paymentMockItemRepo) BulkCreate(_ context.Context, items []*domain.InvoiceItem) ([]*domain.InvoiceItem, error) {
	return items, nil
}

func (m *paymentMockItemRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceItem, error) {
	return nil, nil
}

func (m *paymentMockItemRepo) DeleteByInvoice(_ context.Context, _ string) error {
	return nil
}

// paymentMockPaymentRepo implementasi mock InvoicePaymentRepository.
type paymentMockPaymentRepo struct {
	payments    map[string]*domain.InvoicePayment
	listResult  *domain.PaymentListResult
	summaryResult *domain.PaymentSummary
	findDupResult bool
}

func newPaymentMockPaymentRepo() *paymentMockPaymentRepo {
	return &paymentMockPaymentRepo{payments: make(map[string]*domain.InvoicePayment)}
}

func (m *paymentMockPaymentRepo) Create(_ context.Context, p *domain.InvoicePayment) (*domain.InvoicePayment, error) {
	c := *p
	m.payments[c.ID] = &c
	return &c, nil
}

func (m *paymentMockPaymentRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoicePayment, error) {
	return nil, nil
}

func (m *paymentMockPaymentRepo) VoidPayment(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *paymentMockPaymentRepo) GetByID(_ context.Context, id string) (*domain.InvoicePayment, error) {
	p, ok := m.payments[id]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	c := *p
	return &c, nil
}

func (m *paymentMockPaymentRepo) ListWithFilters(_ context.Context, params domain.PaymentListParams) (*domain.PaymentListResult, error) {
	if m.listResult != nil {
		return m.listResult, nil
	}
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

func (m *paymentMockPaymentRepo) GetSummary(_ context.Context, _ string, _ string, _, _ *int) (*domain.PaymentSummary, error) {
	if m.summaryResult != nil {
		return m.summaryResult, nil
	}
	return &domain.PaymentSummary{
		Today:     domain.PaymentSummaryStat{Count: 3, TotalAmount: 300000},
		ThisMonth: domain.PaymentSummaryStat{Count: 10, TotalAmount: 1500000},
		ByMethod: map[string]domain.PaymentSummaryStat{
			"tunai":    {Count: 5, TotalAmount: 750000},
			"transfer": {Count: 5, TotalAmount: 750000},
		},
	}, nil
}

func (m *paymentMockPaymentRepo) FindDuplicate(_ context.Context, _ string, _ int64, _ string, _ time.Time) (bool, error) {
	return m.findDupResult, nil
}

// paymentMockAuditRepo implementasi mock InvoiceAuditLogRepository.
type paymentMockAuditRepo struct{}

func (m *paymentMockAuditRepo) Create(_ context.Context, _ *domain.InvoiceAuditLog) error {
	return nil
}

func (m *paymentMockAuditRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceAuditLog, error) {
	return nil, nil
}

// paymentMockReceiptSeqRepo implementasi mock ReceiptSequenceRepository.
type paymentMockReceiptSeqRepo struct {
	seq int
}

func (m *paymentMockReceiptSeqRepo) NextSequence(_ context.Context, _ string, _, _ int) (int, error) {
	m.seq++
	return m.seq, nil
}

// paymentMockSettingsRepo implementasi mock BillingSettingsRepository.
type paymentMockSettingsRepo struct{}

func (m *paymentMockSettingsRepo) GetByTenantID(_ context.Context, _ string) (*domain.BillingSettings, error) {
	return &domain.BillingSettings{Timezone: "Asia/Jakarta"}, nil
}

func (m *paymentMockSettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	return s, nil
}

func (m *paymentMockSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

// paymentMockCustomerRepo implementasi mock CustomerRepository.
type paymentMockCustomerRepo struct {
	customers    map[string]*domain.Customer
	searchResult []*domain.Customer
}

func newPaymentMockCustomerRepo() *paymentMockCustomerRepo {
	return &paymentMockCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *paymentMockCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	cp := *c
	m.customers[cp.ID] = &cp
	return &cp, nil
}

func (m *paymentMockCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *paymentMockCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	cp := *c
	m.customers[cp.ID] = &cp
	return &cp, nil
}

func (m *paymentMockCustomerRepo) SoftDelete(_ context.Context, _ string) error { return nil }

func (m *paymentMockCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return &domain.CustomerListResult{Data: []*domain.Customer{}, Pagination: domain.PaginationMeta{}}, nil
}

func (m *paymentMockCustomerRepo) UpdateStatus(_ context.Context, _ string, _ domain.CustomerStatus) (*domain.Customer, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *paymentMockCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *paymentMockCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) {
	return nil, nil
}

func (m *paymentMockCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	if m.searchResult != nil {
		return m.searchResult, nil
	}
	return []*domain.Customer{}, nil
}

// =============================================================================
// Helper — membuat PaymentUsecase dengan mock repos
// =============================================================================

type paymentUsecaseSetup struct {
	uc           *PaymentUsecase
	invoiceRepo  *paymentMockInvoiceRepo
	paymentRepo  *paymentMockPaymentRepo
	customerRepo *paymentMockCustomerRepo
}

// ctxWithTenant membuat context dengan tenant_id (menggunakan key yang sama dengan pkg/tenant).
func ctxWithTenant(tenantID string) context.Context {
	return context.WithValue(context.Background(), tenantCtxKeyID, tenantID)
}

func setupPaymentUsecase() *paymentUsecaseSetup {
	invoiceRepo := newPaymentMockInvoiceRepo()
	paymentRepo := newPaymentMockPaymentRepo()
	customerRepo := newPaymentMockCustomerRepo()
	logger := zerolog.New(io.Discard)

	uc := NewPaymentUsecase(
		invoiceRepo,
		&paymentMockItemRepo{},
		paymentRepo,
		&paymentMockAuditRepo{},
		&paymentMockReceiptSeqRepo{},
		&paymentMockSettingsRepo{},
		customerRepo,
		nil, // pool — nil karena kita test path yang tidak butuh transaksi
		nil, // queueClient
		logger,
	)

	return &paymentUsecaseSetup{
		uc:           uc,
		invoiceRepo:  invoiceRepo,
		paymentRepo:  paymentRepo,
		customerRepo: customerRepo,
	}
}

// =============================================================================
// Test: List — paginasi default
// =============================================================================

// TestPaymentUsecase_List_DefaultPagination menguji List dengan paginasi default.
func TestPaymentUsecase_List_DefaultPagination(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	// Panggil List tanpa page/page_size → default page=1, page_size=25
	params := domain.PaymentListParams{TenantID: "tenant-1"}
	result, err := s.uc.List(ctx, params)
	if err != nil {
		t.Fatalf("List gagal: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verifikasi paginasi default diterapkan
	if result.Pagination.Page != 1 {
		t.Fatalf("expected page 1, got %d", result.Pagination.Page)
	}
	if result.Pagination.PageSize != 25 {
		t.Fatalf("expected page_size 25, got %d", result.Pagination.PageSize)
	}
}

// TestPaymentUsecase_List_CustomPagination menguji List dengan paginasi kustom.
func TestPaymentUsecase_List_CustomPagination(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	params := domain.PaymentListParams{
		TenantID: "tenant-1",
		Page:     3,
		PageSize: 10,
	}
	result, err := s.uc.List(ctx, params)
	if err != nil {
		t.Fatalf("List gagal: %v", err)
	}

	if result.Pagination.Page != 3 {
		t.Fatalf("expected page 3, got %d", result.Pagination.Page)
	}
	if result.Pagination.PageSize != 10 {
		t.Fatalf("expected page_size 10, got %d", result.Pagination.PageSize)
	}
}

// =============================================================================
// Test: Summary — agregasi statistik
// =============================================================================

// TestPaymentUsecase_Summary_Aggregation menguji Summary mengembalikan statistik agregat.
func TestPaymentUsecase_Summary_Aggregation(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	result, err := s.uc.Summary(ctx, "tenant-1", nil, nil)
	if err != nil {
		t.Fatalf("Summary gagal: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verifikasi data dari mock
	if result.Today.Count != 3 {
		t.Fatalf("expected today count 3, got %d", result.Today.Count)
	}
	if result.Today.TotalAmount != 300000 {
		t.Fatalf("expected today total 300000, got %d", result.Today.TotalAmount)
	}
	if result.ThisMonth.Count != 10 {
		t.Fatalf("expected this_month count 10, got %d", result.ThisMonth.Count)
	}
	if result.ThisMonth.TotalAmount != 1500000 {
		t.Fatalf("expected this_month total 1500000, got %d", result.ThisMonth.TotalAmount)
	}

	// Verifikasi by_method
	if len(result.ByMethod) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(result.ByMethod))
	}
	tunai, ok := result.ByMethod["tunai"]
	if !ok {
		t.Fatal("expected tunai in by_method")
	}
	if tunai.Count != 5 || tunai.TotalAmount != 750000 {
		t.Fatalf("expected tunai count=5 total=750000, got count=%d total=%d", tunai.Count, tunai.TotalAmount)
	}
}

// =============================================================================
// Test: SearchCustomers — validasi term pendek
// =============================================================================

// TestPaymentUsecase_SearchCustomers_ShortTermError menguji error saat term < 2 karakter.
func TestPaymentUsecase_SearchCustomers_ShortTermError(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	_, err := s.uc.SearchCustomers(ctx, "tenant-1", "A")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrSearchTermTooShort) {
		t.Fatalf("expected ErrSearchTermTooShort, got %v", err)
	}
}

// TestPaymentUsecase_SearchCustomers_EmptyTermError menguji error saat term kosong.
func TestPaymentUsecase_SearchCustomers_EmptyTermError(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	_, err := s.uc.SearchCustomers(ctx, "tenant-1", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrSearchTermTooShort) {
		t.Fatalf("expected ErrSearchTermTooShort, got %v", err)
	}
}

// TestPaymentUsecase_SearchCustomers_ValidTerm menguji pencarian dengan term valid.
func TestPaymentUsecase_SearchCustomers_ValidTerm(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	// Setup mock search result
	s.customerRepo.searchResult = []*domain.Customer{
		{ID: "cust-1", Name: "Ahmad Rizki", CustomerIDSeq: "PLG-001"},
	}

	result, err := s.uc.SearchCustomers(ctx, "tenant-1", "Ahmad")
	if err != nil {
		t.Fatalf("SearchCustomers gagal: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Name != "Ahmad Rizki" {
		t.Fatalf("expected name Ahmad Rizki, got %s", result[0].Name)
	}
}

// =============================================================================
// Test: GetOpenInvoices — remaining_amount dan total_arrears
// =============================================================================

// TestPaymentUsecase_GetOpenInvoices_RemainingAndArrears menguji kalkulasi remaining_amount dan total_arrears.
func TestPaymentUsecase_GetOpenInvoices_RemainingAndArrears(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	// Setup 2 invoice terbuka dengan pembayaran parsial
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", CustomerID: "cust-1", InvoiceNumber: "INV-2024-06-001",
		TotalAmount: 100000, PaidAmount: 30000,
		Status: domain.InvoiceStatusBayarSebagian,
		DueDate: time.Now().Add(-48 * time.Hour), PeriodMonth: 6, PeriodYear: 2024,
	}
	s.invoiceRepo.invoices["inv-2"] = &domain.Invoice{
		ID: "inv-2", CustomerID: "cust-1", InvoiceNumber: "INV-2024-07-001",
		TotalAmount: 150000, PaidAmount: 0,
		Status: domain.InvoiceStatusBelumBayar,
		DueDate: time.Now().Add(24 * time.Hour), PeriodMonth: 7, PeriodYear: 2024,
	}

	result, err := s.uc.GetOpenInvoices(ctx, "cust-1")
	if err != nil {
		t.Fatalf("GetOpenInvoices gagal: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Invoices) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(result.Invoices))
	}

	// Verifikasi total_arrears = (100000-30000) + (150000-0) = 220000
	expectedArrears := int64(220000)
	if result.TotalArrears != expectedArrears {
		t.Fatalf("expected total_arrears %d, got %d", expectedArrears, result.TotalArrears)
	}

	// Verifikasi remaining_amount per invoice
	for _, inv := range result.Invoices {
		expectedRemaining := inv.TotalAmount - inv.PaidAmount
		if inv.RemainingAmount != expectedRemaining {
			t.Fatalf("invoice %s: expected remaining %d, got %d", inv.ID, expectedRemaining, inv.RemainingAmount)
		}
	}
}

// TestPaymentUsecase_GetOpenInvoices_EmptyResult menguji respons kosong saat tidak ada invoice terbuka.
func TestPaymentUsecase_GetOpenInvoices_EmptyResult(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	result, err := s.uc.GetOpenInvoices(ctx, "cust-nonexistent")
	if err != nil {
		t.Fatalf("GetOpenInvoices gagal: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Invoices) != 0 {
		t.Fatalf("expected 0 invoices, got %d", len(result.Invoices))
	}

	if result.TotalArrears != 0 {
		t.Fatalf("expected total_arrears 0, got %d", result.TotalArrears)
	}
}
