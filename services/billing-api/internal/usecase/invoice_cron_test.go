// invoice_cron_test.go berisi unit test untuk InvoiceCronUsecase.
// Menguji auto-generate idempotency, skip prepaid periods, include recurring items,
// tax calculation, credit application, overdue status update.
package usecase

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Mock repositories khusus untuk InvoiceCronUsecase tests
// =============================================================================

// mockCronInvoiceRepo extends mockInvoiceRepo dengan kontrol idempotency.
type mockCronInvoiceRepo struct {
	*mockInvoiceRepo
	existingPeriods map[string]bool // key: "customerID-month-year"
	prepaidPeriods  map[string]bool
	overdueList     []*domain.Invoice
}

func newMockCronInvoiceRepo() *mockCronInvoiceRepo {
	return &mockCronInvoiceRepo{
		mockInvoiceRepo: newMockInvoiceRepo(),
		existingPeriods: make(map[string]bool),
		prepaidPeriods:  make(map[string]bool),
	}
}

func (m *mockCronInvoiceRepo) ExistsForPeriod(_ context.Context, customerID string, month, year int) (bool, error) {
	key := fmt.Sprintf("%s-%d-%d", customerID, month, year)
	return m.existingPeriods[key], nil
}

func (m *mockCronInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, customerID string, month, year int) (bool, error) {
	key := fmt.Sprintf("%s-%d-%d", customerID, month, year)
	return m.prepaidPeriods[key], nil
}

func (m *mockCronInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return m.overdueList, nil
}

// mockCronCustomerRepo extends invMockCustomerRepo dengan list yang mengembalikan pelanggan.
type mockCronCustomerRepo struct {
	*invMockCustomerRepo
}

func newMockCronCustomerRepo() *mockCronCustomerRepo {
	return &mockCronCustomerRepo{invMockCustomerRepo: newInvMockCustomerRepo()}
}

func (m *mockCronCustomerRepo) List(_ context.Context, params domain.CustomerListParams) (*domain.CustomerListResult, error) {
	var filtered []*domain.Customer
	for _, c := range m.customers {
		if params.TenantID != "" && c.TenantID != params.TenantID {
			continue
		}
		if params.Status != "" && string(c.Status) != params.Status {
			continue
		}
		filtered = append(filtered, c)
	}
	return &domain.CustomerListResult{
		Data: filtered,
		Pagination: domain.PaginationMeta{
			Total: int64(len(filtered)), Page: 1, PageSize: 50, TotalPages: 1,
		},
	}, nil
}

// mockRecurringItemRepo adalah implementasi in-memory dari domain.CustomerRecurringItemRepository.
type mockRecurringItemRepo struct {
	items map[string][]*domain.CustomerRecurringItem // key: customerID
}

func newMockRecurringItemRepo() *mockRecurringItemRepo {
	return &mockRecurringItemRepo{items: make(map[string][]*domain.CustomerRecurringItem)}
}

func (m *mockRecurringItemRepo) Create(_ context.Context, item *domain.CustomerRecurringItem) (*domain.CustomerRecurringItem, error) {
	if item.ID == "" {
		item.ID = fmt.Sprintf("ri-%d", len(m.items)+1)
	}
	copy := *item
	m.items[item.CustomerID] = append(m.items[item.CustomerID], &copy)
	return &copy, nil
}
func (m *mockRecurringItemRepo) GetByID(_ context.Context, id string) (*domain.CustomerRecurringItem, error) {
	for _, items := range m.items {
		for _, item := range items {
			if item.ID == id {
				copy := *item
				return &copy, nil
			}
		}
	}
	return nil, domain.ErrRecurringItemNotFound
}
func (m *mockRecurringItemRepo) Update(_ context.Context, item *domain.CustomerRecurringItem) (*domain.CustomerRecurringItem, error) {
	copy := *item
	return &copy, nil
}
func (m *mockRecurringItemRepo) Deactivate(_ context.Context, _ string) error { return nil }
func (m *mockRecurringItemRepo) ListByCustomer(_ context.Context, customerID string) ([]*domain.CustomerRecurringItem, error) {
	return m.items[customerID], nil
}
func (m *mockRecurringItemRepo) ListActiveByCustomer(_ context.Context, customerID string, _ time.Time) ([]*domain.CustomerRecurringItem, error) {
	var active []*domain.CustomerRecurringItem
	for _, item := range m.items[customerID] {
		if item.IsActive {
			active = append(active, item)
		}
	}
	return active, nil
}

// =============================================================================
// Helper untuk membuat InvoiceCronUsecase dengan mock repos
// =============================================================================

type cronUsecaseSetup struct {
	uc              *InvoiceCronUsecase
	invoiceRepo     *mockCronInvoiceRepo
	itemRepo        *mockItemRepo
	customerRepo    *mockCronCustomerRepo
	packageRepo     *invMockPackageRepo
	settingsRepo    *mockSettingsRepo
	recurringRepo   *mockRecurringItemRepo
}

func setupCronUsecase() *cronUsecaseSetup {
	invoiceRepo := newMockCronInvoiceRepo()
	itemRepo := newMockItemRepo()
	auditRepo := &mockAuditRepo{}
	sequenceRepo := &mockSequenceRepo{}
	settingsRepo := newMockSettingsRepo()
	customerRepo := newMockCronCustomerRepo()
	packageRepo := newInvMockPackageRepo()
	recurringRepo := newMockRecurringItemRepo()
	logger := zerolog.New(io.Discard)

	uc := NewInvoiceCronUsecase(
		invoiceRepo, itemRepo, auditRepo, sequenceRepo,
		settingsRepo, customerRepo, packageRepo, recurringRepo,
		nil, nil, logger,
	)

	return &cronUsecaseSetup{
		uc:            uc,
		invoiceRepo:   invoiceRepo,
		itemRepo:      itemRepo,
		customerRepo:  customerRepo,
		packageRepo:   packageRepo,
		settingsRepo:  settingsRepo,
		recurringRepo: recurringRepo,
	}
}

// =============================================================================
// Unit Tests — InvoiceCronUsecase
// =============================================================================

// TestCron_AutoGenerate_Idempotency menguji bahwa invoice tidak di-generate ulang untuk periode yang sama.
func TestCron_AutoGenerate_Idempotency(t *testing.T) {
	s := setupCronUsecase()
	ctx := context.Background()

	monthlyPrice := int64(200000)
	s.settingsRepo.settings["tenant-1"] = &domain.BillingSettings{
		TenantID:     "tenant-1",
		GenerateDays: 5,
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		PackageID: "pkg-1", DueDate: 15,
	}
	s.packageRepo.packages["pkg-1"] = &domain.Package{
		ID: "pkg-1", Name: "Test", MonthlyPrice: &monthlyPrice,
	}

	// Tandai periode sudah ada
	s.invoiceRepo.existingPeriods["cust-1-6-2024"] = true

	// Simulasi generate pada tanggal 10 Juni (due_date 15 - generate_days 5 = 10)
	now := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	err := s.uc.processAutoGenerateForTenant(ctx, s.settingsRepo.settings["tenant-1"], now)
	if err != nil {
		t.Fatalf("processAutoGenerateForTenant gagal: %v", err)
	}

	// Tidak boleh ada invoice baru karena sudah ada
	if len(s.invoiceRepo.invoices) != 0 {
		t.Fatalf("expected 0 invoices (idempotent), got %d", len(s.invoiceRepo.invoices))
	}
}

// TestCron_AutoGenerate_SkipPrepaid menguji bahwa periode yang di-cover prepaid di-skip.
func TestCron_AutoGenerate_SkipPrepaid(t *testing.T) {
	s := setupCronUsecase()
	ctx := context.Background()

	monthlyPrice := int64(200000)
	s.settingsRepo.settings["tenant-1"] = &domain.BillingSettings{
		TenantID:     "tenant-1",
		GenerateDays: 5,
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		PackageID: "pkg-1", DueDate: 15,
	}
	s.packageRepo.packages["pkg-1"] = &domain.Package{
		ID: "pkg-1", Name: "Test", MonthlyPrice: &monthlyPrice,
	}

	// Tandai periode di-cover prepaid
	s.invoiceRepo.prepaidPeriods["cust-1-6-2024"] = true

	now := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	err := s.uc.processAutoGenerateForTenant(ctx, s.settingsRepo.settings["tenant-1"], now)
	if err != nil {
		t.Fatalf("processAutoGenerateForTenant gagal: %v", err)
	}

	// Tidak boleh ada invoice baru karena prepaid sudah cover
	if len(s.invoiceRepo.invoices) != 0 {
		t.Fatalf("expected 0 invoices (prepaid covers), got %d", len(s.invoiceRepo.invoices))
	}
}

// TestCron_AutoGenerate_IncludeRecurringItems menguji bahwa recurring items disertakan.
func TestCron_AutoGenerate_IncludeRecurringItems(t *testing.T) {
	s := setupCronUsecase()
	ctx := context.Background()

	monthlyPrice := int64(200000)
	s.settingsRepo.settings["tenant-1"] = &domain.BillingSettings{
		TenantID:     "tenant-1",
		GenerateDays: 5,
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		PackageID: "pkg-1", DueDate: 15,
	}
	s.packageRepo.packages["pkg-1"] = &domain.Package{
		ID: "pkg-1", Name: "Test", MonthlyPrice: &monthlyPrice,
	}

	// Tambah recurring item aktif
	s.recurringRepo.items["cust-1"] = []*domain.CustomerRecurringItem{
		{ID: "ri-1", CustomerID: "cust-1", Description: "Sewa router", Amount: 25000, IsActive: true},
	}

	now := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	err := s.uc.processAutoGenerateForTenant(ctx, s.settingsRepo.settings["tenant-1"], now)
	if err != nil {
		t.Fatalf("processAutoGenerateForTenant gagal: %v", err)
	}

	// Harus ada 1 invoice
	if len(s.invoiceRepo.invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(s.invoiceRepo.invoices))
	}

	// Subtotal harus = monthly + recurring = 200000 + 25000 = 225000
	for _, inv := range s.invoiceRepo.invoices {
		if inv.Subtotal != 225000 {
			t.Fatalf("expected subtotal 225000, got %d", inv.Subtotal)
		}
	}
}

// TestCron_AutoGenerate_WithTax menguji bahwa pajak dihitung dengan benar.
func TestCron_AutoGenerate_WithTax(t *testing.T) {
	s := setupCronUsecase()
	ctx := context.Background()

	monthlyPrice := int64(200000)
	s.settingsRepo.settings["tenant-1"] = &domain.BillingSettings{
		TenantID:     "tenant-1",
		GenerateDays: 5,
		TaxEnabled:   true,
		TaxRate:      11,
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		PackageID: "pkg-1", DueDate: 15,
	}
	s.packageRepo.packages["pkg-1"] = &domain.Package{
		ID: "pkg-1", Name: "Test", MonthlyPrice: &monthlyPrice,
	}

	now := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	err := s.uc.processAutoGenerateForTenant(ctx, s.settingsRepo.settings["tenant-1"], now)
	if err != nil {
		t.Fatalf("processAutoGenerateForTenant gagal: %v", err)
	}

	for _, inv := range s.invoiceRepo.invoices {
		// Pajak = 200000 * 11 / 100 = 22000
		if inv.TaxAmount != 22000 {
			t.Fatalf("expected tax 22000, got %d", inv.TaxAmount)
		}
		// Total = 200000 + 22000 = 222000
		if inv.TotalAmount != 222000 {
			t.Fatalf("expected total 222000, got %d", inv.TotalAmount)
		}
	}
}

// TestCron_AutoGenerate_WithCredit menguji bahwa kredit pelanggan diterapkan.
func TestCron_AutoGenerate_WithCredit(t *testing.T) {
	s := setupCronUsecase()
	ctx := context.Background()

	monthlyPrice := int64(200000)
	s.settingsRepo.settings["tenant-1"] = &domain.BillingSettings{
		TenantID:     "tenant-1",
		GenerateDays: 5,
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
		PackageID: "pkg-1", DueDate: 15, CreditBalance: 50000,
	}
	s.packageRepo.packages["pkg-1"] = &domain.Package{
		ID: "pkg-1", Name: "Test", MonthlyPrice: &monthlyPrice,
	}

	now := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	err := s.uc.processAutoGenerateForTenant(ctx, s.settingsRepo.settings["tenant-1"], now)
	if err != nil {
		t.Fatalf("processAutoGenerateForTenant gagal: %v", err)
	}

	for _, inv := range s.invoiceRepo.invoices {
		if inv.CreditApplied != 50000 {
			t.Fatalf("expected credit_applied 50000, got %d", inv.CreditApplied)
		}
		// Total = 200000 - 50000 = 150000
		if inv.TotalAmount != 150000 {
			t.Fatalf("expected total 150000, got %d", inv.TotalAmount)
		}
	}

	// Verifikasi saldo kredit pelanggan berkurang
	// Catatan: credit_balance diupdate secara atomik via SQL langsung (pool.Exec),
	// bukan melalui customerRepo.Update, sehingga mock tidak terpengaruh.
	// Verifikasi ini hanya berlaku di environment dengan database nyata.
}

// TestCron_OverdueUpdate menguji update status overdue.
func TestCron_OverdueUpdate(t *testing.T) {
	s := setupCronUsecase()
	ctx := context.Background()

	// Tambah invoice overdue
	inv := &domain.Invoice{
		ID:            "inv-overdue",
		TenantID:      "tenant-1",
		CustomerID:    "cust-1",
		InvoiceNumber: "INV-2024-05-001",
		Status:        domain.InvoiceStatusBelumBayar,
		DueDate:       time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
		Version:       1,
	}
	s.invoiceRepo.invoices["inv-overdue"] = inv
	s.invoiceRepo.overdueList = []*domain.Invoice{inv}

	err := s.uc.ProcessOverdueUpdate(ctx)
	if err != nil {
		t.Fatalf("ProcessOverdueUpdate gagal: %v", err)
	}

	// Verifikasi status berubah ke terlambat
	updated := s.invoiceRepo.invoices["inv-overdue"]
	if updated.Status != domain.InvoiceStatusTerlambat {
		t.Fatalf("expected status terlambat, got %s", updated.Status)
	}
}
