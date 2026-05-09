// invoice_usecase_test.go berisi unit test untuk InvoiceUsecase.
// Menguji business logic: pembuatan manual dengan pajak/kredit, prepaid dengan diskon,
package usecase

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"io"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
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
		Data:       filtered[start:end],
		Pagination: domain.PaginationMeta{Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages},
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
	return &domain.InvoiceSummary{Total: domain.InvoiceSummaryStat{}, ByStatus: make(map[domain.InvoiceStatus]domain.InvoiceSummaryStat)}, nil
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

// mockItemRepo adalah implementasi in-memory dari domain.InvoiceItemRepository.
type mockItemRepo struct {
	items map[string][]*domain.InvoiceItem
}

func newMockItemRepo() *mockItemRepo {
	return &mockItemRepo{items: make(map[string][]*domain.InvoiceItem)}
}

func (m *mockItemRepo) BulkCreate(_ context.Context, items []*domain.InvoiceItem) ([]*domain.InvoiceItem, error) {
	for _, item := range items {
		m.items[item.InvoiceID] = append(m.items[item.InvoiceID], item)
	}
	return items, nil
}
func (m *mockItemRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.InvoiceItem, error) {
	return m.items[invoiceID], nil
}
func (m *mockItemRepo) DeleteByInvoice(_ context.Context, invoiceID string) error {
	delete(m.items, invoiceID)
	return nil
}

// mockPaymentRepo adalah implementasi in-memory dari domain.InvoicePaymentRepository.
type mockPaymentRepo struct{}

func (m *mockPaymentRepo) Create(_ context.Context, p *domain.InvoicePayment) (*domain.InvoicePayment, error) {
	return p, nil
}
func (m *mockPaymentRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoicePayment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) VoidPayment(_ context.Context, _, _, _ string) error { return nil }
func (m *mockPaymentRepo) GetByID(_ context.Context, _ string) (*domain.InvoicePayment, error) {
	return nil, domain.ErrPaymentNotFound
}
func (m *mockPaymentRepo) ListWithFilters(_ context.Context, _ domain.PaymentListParams) (*domain.PaymentListResult, error) {
	return &domain.PaymentListResult{Data: []domain.PaymentListItem{}, Pagination: domain.PaginationMeta{}}, nil
}
func (m *mockPaymentRepo) GetSummary(_ context.Context, _ string, _ string, _, _ *int) (*domain.PaymentSummary, error) {
	return &domain.PaymentSummary{ByMethod: make(map[string]domain.PaymentSummaryStat)}, nil
}
func (m *mockPaymentRepo) FindDuplicate(_ context.Context, _ string, _ int64, _ string, _ time.Time) (bool, error) {
	return false, nil
}

// mockAuditRepo adalah implementasi in-memory dari domain.InvoiceAuditLogRepository.
type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(_ context.Context, _ *domain.InvoiceAuditLog) error { return nil }
func (m *mockAuditRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceAuditLog, error) {
	return nil, nil
}

// mockSequenceRepo adalah implementasi in-memory dari domain.InvoiceSequenceRepository.
type mockSequenceRepo struct {
	seq int
}

func (m *mockSequenceRepo) NextSequence(_ context.Context, _ string, _, _ int) (int, error) {
	m.seq++
	return m.seq, nil
}

// mockSettingsRepo adalah implementasi in-memory dari domain.BillingSettingsRepository.
type mockSettingsRepo struct {
	settings map[string]*domain.BillingSettings
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{settings: make(map[string]*domain.BillingSettings)}
}

func (m *mockSettingsRepo) GetByTenantID(_ context.Context, tenantID string) (*domain.BillingSettings, error) {
	s, ok := m.settings[tenantID]
	if !ok {
		return nil, domain.ErrBillingSettingsNotFound
	}
	return s, nil
}
func (m *mockSettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	m.settings[s.TenantID] = s
	return s, nil
}
func (m *mockSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	var result []*domain.BillingSettings
	for _, s := range m.settings {
		result = append(result, s)
	}
	return result, nil
}

// invMockCustomerRepo adalah implementasi in-memory dari domain.CustomerRepository untuk invoice tests.
type invMockCustomerRepo struct {
	customers map[string]*domain.Customer
}

func newInvMockCustomerRepo() *invMockCustomerRepo {
	return &invMockCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *invMockCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}
func (m *invMockCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	return &copy, nil
}
func (m *invMockCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	if _, ok := m.customers[c.ID]; !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	m.customers[copy.ID] = &copy
	return &copy, nil
}
func (m *invMockCustomerRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *invMockCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return &domain.CustomerListResult{Data: []*domain.Customer{}, Pagination: domain.PaginationMeta{Total: 0, Page: 1, PageSize: 25, TotalPages: 1}}, nil
}
func (m *invMockCustomerRepo) UpdateStatus(_ context.Context, _ string, _ domain.CustomerStatus) (*domain.Customer, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *invMockCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *invMockCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) {
	return nil, nil
}
func (m *invMockCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}

// invMockPackageRepo adalah implementasi in-memory dari domain.PackageRepository untuk invoice tests.
type invMockPackageRepo struct {
	packages map[string]*domain.Package
}

func newInvMockPackageRepo() *invMockPackageRepo {
	return &invMockPackageRepo{packages: make(map[string]*domain.Package)}
}

func (m *invMockPackageRepo) Create(_ context.Context, _ *domain.Package) (*domain.Package, error) {
	return nil, nil
}
func (m *invMockPackageRepo) GetByID(_ context.Context, id string) (*domain.Package, error) {
	p, ok := m.packages[id]
	if !ok {
		return nil, domain.ErrPackageNotFound
	}
	return p, nil
}
func (m *invMockPackageRepo) Update(_ context.Context, _ *domain.Package) (*domain.Package, error) {
	return nil, nil
}
func (m *invMockPackageRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *invMockPackageRepo) List(_ context.Context, _ domain.PackageListParams) (*domain.PackageListResult, error) {
	return nil, nil
}
func (m *invMockPackageRepo) UpdateIsActive(_ context.Context, _ string, _ bool) (*domain.Package, error) {
	return nil, nil
}
func (m *invMockPackageRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *invMockPackageRepo) CustomerCount(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *invMockPackageRepo) ListNamesByPrefix(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

// =============================================================================
// =============================================================================

type invoiceUsecaseSetup struct {
	uc           *InvoiceUsecase
	invoiceRepo  *mockInvoiceRepo
	itemRepo     *mockItemRepo
	customerRepo *invMockCustomerRepo
	packageRepo  *invMockPackageRepo
	settingsRepo *mockSettingsRepo
	sequenceRepo *mockSequenceRepo
}

func setupInvoiceUsecase() *invoiceUsecaseSetup {
	invoiceRepo := newMockInvoiceRepo()
	itemRepo := newMockItemRepo()
	paymentRepo := &mockPaymentRepo{}
	auditRepo := &mockAuditRepo{}
	sequenceRepo := &mockSequenceRepo{}
	settingsRepo := newMockSettingsRepo()
	customerRepo := newInvMockCustomerRepo()
	packageRepo := newInvMockPackageRepo()
	logger := zerolog.New(io.Discard)

	uc := NewInvoiceUsecase(
		invoiceRepo, itemRepo, paymentRepo, auditRepo,
		sequenceRepo, settingsRepo, customerRepo, packageRepo,
		nil, nil, logger,
	)

	return &invoiceUsecaseSetup{
		uc:           uc,
		invoiceRepo:  invoiceRepo,
		itemRepo:     itemRepo,
		customerRepo: customerRepo,
		packageRepo:  packageRepo,
		settingsRepo: settingsRepo,
		sequenceRepo: sequenceRepo,
	}
}

// =============================================================================
// Unit Tests - InvoiceUsecase
// =============================================================================

// TestInvoiceUsecase_Create_Success menguji pembuatan invoice manual berhasil.
func TestInvoiceUsecase_Create_Success(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	// Setup pelanggan aktif
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID:       "cust-1",
		TenantID: "tenant-1",
		Name:     "Test",
		Status:   domain.CustomerStatusAktif,
	}

	req := domain.CreateInvoiceRequest{
		CustomerID: "cust-1",
		DueDate:    "2024-06-15",
		Items: []domain.CreateInvoiceItemRequest{
			{Description: "Tagihan bulanan", Quantity: 1, UnitPrice: 200000},
			{Description: "Biaya tambahan", Quantity: 2, UnitPrice: 50000},
		},
	}

	inv, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{ActorID: "user-1"})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	// Verifikasi subtotal = 200000 + (2*50000) = 300000
	if inv.Subtotal != 300000 {
		t.Fatalf("expected subtotal 300000, got %d", inv.Subtotal)
	}
	if inv.Status != domain.InvoiceStatusBelumBayar {
		t.Fatalf("expected status belum_bayar, got %s", inv.Status)
	}
	if inv.Version != 1 {
		t.Fatalf("expected version 1, got %d", inv.Version)
	}
}

// TestInvoiceUsecase_Create_WithTax menguji pembuatan invoice dengan pajak.
func TestInvoiceUsecase_Create_WithTax(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
	}
	s.settingsRepo.settings["tenant-1"] = &domain.BillingSettings{
		TenantID:   "tenant-1",
		TaxEnabled: true,
		TaxRate:    11,
	}

	applyTax := true
	req := domain.CreateInvoiceRequest{
		CustomerID: "cust-1",
		DueDate:    "2024-06-15",
		Items:      []domain.CreateInvoiceItemRequest{{Description: "Test", Quantity: 1, UnitPrice: 100000}},
		ApplyTax:   &applyTax,
	}

	inv, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	// Pajak = 100000 * 11 / 100 = 11000
	if inv.TaxAmount != 11000 {
		t.Fatalf("expected tax 11000, got %d", inv.TaxAmount)
	}
	// Total = 100000 + 11000 = 111000
	if inv.TotalAmount != 111000 {
		t.Fatalf("expected total 111000, got %d", inv.TotalAmount)
	}
}

// TestInvoiceUsecase_Create_WithCredit menguji pembuatan invoice dengan kredit pelanggan.
func TestInvoiceUsecase_Create_WithCredit(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		CreditBalance: 30000,
	}

	req := domain.CreateInvoiceRequest{
		CustomerID: "cust-1",
		DueDate:    "2024-06-15",
		Items:      []domain.CreateInvoiceItemRequest{{Description: "Test", Quantity: 1, UnitPrice: 100000}},
	}

	inv, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	// Kredit diterapkan = min(30000, 100000) = 30000
	if inv.CreditApplied != 30000 {
		t.Fatalf("expected credit_applied 30000, got %d", inv.CreditApplied)
	}
	// Total = 100000 - 30000 = 70000
	if inv.TotalAmount != 70000 {
		t.Fatalf("expected total 70000, got %d", inv.TotalAmount)
	}

	// Verifikasi saldo kredit pelanggan berkurang
	// Catatan: credit_balance diupdate secara atomik via SQL langsung (pool.Exec),
	// Verifikasi ini hanya berlaku di environment dengan database nyata.
}

func TestInvoiceUsecase_Create_CustomerNotFound(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	req := domain.CreateInvoiceRequest{
		CustomerID: "nonexistent",
		DueDate:    "2024-06-15",
		Items:      []domain.CreateInvoiceItemRequest{{Description: "Test", Quantity: 1, UnitPrice: 100000}},
	}

	_, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestInvoiceUsecase_CreatePrepaid_WithDiscount menguji pembuatan invoice prepaid dengan diskon.
func TestInvoiceUsecase_CreatePrepaid_WithDiscount(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	monthlyPrice := int64(200000)
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		PackageID: "pkg-1", DueDate: 10,
	}
	s.packageRepo.packages["pkg-1"] = &domain.Package{
		ID: "pkg-1", Name: "Paket 50M", MonthlyPrice: &monthlyPrice,
	}

	req := domain.CreatePrepaidInvoiceRequest{
		CustomerID:       "cust-1",
		Months:           6,
		StartPeriodMonth: 1,
		StartPeriodYear:  2024,
		DiscountMonths:   1,
	}

	inv, err := s.uc.CreatePrepaid(ctx, "tenant-1", req, domain.ActorInfo{})
	if err != nil {
		t.Fatalf("CreatePrepaid gagal: %v", err)
	}

	// Subtotal = 6 * 200000 = 1200000
	if inv.Subtotal != 1200000 {
		t.Fatalf("expected subtotal 1200000, got %d", inv.Subtotal)
	}
	// Diskon = 1 * 200000 = 200000
	if inv.DiscountAmount != 200000 {
		t.Fatalf("expected discount 200000, got %d", inv.DiscountAmount)
	}
	// Total = 1200000 - 200000 = 1000000
	if inv.TotalAmount != 1000000 {
		t.Fatalf("expected total 1000000, got %d", inv.TotalAmount)
	}
	if !inv.IsPrepaid {
		t.Fatal("expected is_prepaid true")
	}
}

// TestInvoiceUsecase_Edit_OnlyBelumBayar menguji bahwa hanya invoice belum_bayar yang bisa diedit.
func TestInvoiceUsecase_Edit_OnlyBelumBayar(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	// Buat invoice dengan status terlambat
	s.invoiceRepo.invoices["inv-late"] = &domain.Invoice{
		ID:       "inv-late",
		TenantID: "tenant-1",
		Status:   domain.InvoiceStatusTerlambat,
		Version:  1,
	}

	_, err := s.uc.Edit(ctx, "inv-late", domain.EditInvoiceRequest{Notes: "test"}, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected ErrInvoiceNotEditable, got nil")
	}
	if err != domain.ErrInvoiceNotEditable {
		t.Fatalf("expected ErrInvoiceNotEditable, got %v", err)
	}
}

// TestInvoiceUsecase_Edit_Success menguji edit invoice belum_bayar berhasil.
func TestInvoiceUsecase_Edit_Success(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	s.invoiceRepo.invoices["inv-edit"] = &domain.Invoice{
		ID:       "inv-edit",
		TenantID: "tenant-1",
		Status:   domain.InvoiceStatusBelumBayar,
		DueDate:  time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		Version:  1,
	}

	updated, err := s.uc.Edit(ctx, "inv-edit", domain.EditInvoiceRequest{Notes: "catatan baru"}, domain.ActorInfo{})
	if err != nil {
		t.Fatalf("Edit gagal: %v", err)
	}
	if updated.Notes != "catatan baru" {
		t.Fatalf("expected notes 'catatan baru', got '%s'", updated.Notes)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
}

func TestInvoiceUsecase_List_DefaultPagination(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	// Tambah beberapa invoice
	for i := 0; i < 3; i++ {
		s.invoiceRepo.invoices[fmt.Sprintf("inv-%d", i)] = &domain.Invoice{
			ID:       fmt.Sprintf("inv-%d", i),
			TenantID: "tenant-1",
			Status:   domain.InvoiceStatusBelumBayar,
		}
	}

	result, err := s.uc.List(ctx, domain.InvoiceListParams{TenantID: "tenant-1"})
	if err != nil {
		t.Fatalf("List gagal: %v", err)
	}
	if result.Pagination.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Pagination.Total)
	}
	if result.Pagination.Page != 1 {
		t.Fatalf("expected page 1, got %d", result.Pagination.Page)
	}
}

func TestInvoiceUsecase_GetByID_NotFound(t *testing.T) {
	s := setupInvoiceUsecase()
	ctx := context.Background()

	_, err := s.uc.GetByID(ctx, "nonexistent", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
