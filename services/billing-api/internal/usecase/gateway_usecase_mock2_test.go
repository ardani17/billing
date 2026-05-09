package usecase

import (
	"context"
	"io"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type gwMockInvoiceRepo struct {
	invoices map[string]*domain.Invoice
}

func newGwMockInvoiceRepo() *gwMockInvoiceRepo {
	return &gwMockInvoiceRepo{invoices: make(map[string]*domain.Invoice)}
}

func (m *gwMockInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	cp := *inv
	m.invoices[cp.ID] = &cp
	return &cp, nil
}

func (m *gwMockInvoiceRepo) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	cp := *inv
	return &cp, nil
}

func (m *gwMockInvoiceRepo) Update(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	cp := *inv
	m.invoices[cp.ID] = &cp
	return &cp, nil
}

func (m *gwMockInvoiceRepo) UpdateStatus(_ context.Context, _ string, _ domain.InvoiceStatus, _ int) (*domain.Invoice, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) UpdatePaidAmount(_ context.Context, _ string, _ int64, _ int) (*domain.Invoice, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) List(_ context.Context, _ domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) ExistsForPeriod(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *gwMockInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}

func (m *gwMockInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) GetSummary(_ context.Context, _ string, _, _ *int) (*domain.InvoiceSummary, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) FindOpenByCustomer(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID &&
			(inv.Status == domain.InvoiceStatusBelumBayar ||
				inv.Status == domain.InvoiceStatusTerlambat ||
				inv.Status == domain.InvoiceStatusBayarSebagian) {
			cp := *inv
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *gwMockInvoiceRepo) FindOpenByCustomerForUpdate(ctx context.Context, customerID string) ([]*domain.Invoice, error) {
	return m.FindOpenByCustomer(ctx, customerID)
}

func (m *gwMockInvoiceRepo) GetByIDsForUpdate(_ context.Context, ids []string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, id := range ids {
		if inv, ok := m.invoices[id]; ok {
			cp := *inv
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *gwMockInvoiceRepo) FindOverdueForIsolir(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) FindOverdueForSuspend(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *gwMockInvoiceRepo) HasOutstandingInvoices(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *gwMockInvoiceRepo) SumOutstandingAmount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *gwMockInvoiceRepo) CountOutstandingInvoices(_ context.Context, _ string) (int, error) {
	return 0, nil
}

type gwMockCustomerRepo struct {
	customers map[string]*domain.Customer
}

func newGwMockCustomerRepo() *gwMockCustomerRepo {
	return &gwMockCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *gwMockCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *gwMockCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	return c, nil
}
func (m *gwMockCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	return c, nil
}
func (m *gwMockCustomerRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *gwMockCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) UpdateStatus(_ context.Context, _ string, _ domain.CustomerStatus) (*domain.Customer, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *gwMockCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *gwMockCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) {
	return nil, nil
}
func (m *gwMockCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}

type gwMockSettingsRepo struct{}

func (m *gwMockSettingsRepo) GetByTenantID(_ context.Context, _ string) (*domain.BillingSettings, error) {
	return &domain.BillingSettings{}, nil
}
func (m *gwMockSettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	return s, nil
}
func (m *gwMockSettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

// gwTestSetup berisi semua komponen yang dibutuhkan untuk test.
type gwTestSetup struct {
	uc           *GatewayUsecase
	configRepo   *gwMockConfigRepo
	linkRepo     *gwMockLinkRepo
	invoiceRepo  *gwMockInvoiceRepo
	customerRepo *gwMockCustomerRepo
}

func setupGatewayUsecase() *gwTestSetup {
	configRepo := newGwMockConfigRepo()
	linkRepo := newGwMockLinkRepo()
	invoiceRepo := newGwMockInvoiceRepo()
	customerRepo := newGwMockCustomerRepo()
	settingsRepo := &gwMockSettingsRepo{}
	logger := zerolog.New(io.Discard)

	uc := NewGatewayUsecase(
		configRepo, linkRepo, invoiceRepo, customerRepo,
		settingsRepo, nil, nil, testMasterKey, logger,
	)

	return &gwTestSetup{
		uc:           uc,
		configRepo:   configRepo,
		linkRepo:     linkRepo,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
	}
}
