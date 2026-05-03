// gateway_handler_mock3_test.go berisi mock InvoiceRepo dan CustomerRepo untuk test GatewayHandler.
package handler

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock: InvoiceRepository (subset yang dibutuhkan) ---

type mockGatewayInvoiceRepo struct {
	invoices map[string]*domain.Invoice
}

func newMockGatewayInvoiceRepo() *mockGatewayInvoiceRepo {
	return &mockGatewayInvoiceRepo{invoices: make(map[string]*domain.Invoice)}
}

func (m *mockGatewayInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	cp := *inv
	m.invoices[cp.ID] = &cp
	return &cp, nil
}
func (m *mockGatewayInvoiceRepo) GetByID(_ context.Context, id string) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	cp := *inv
	return &cp, nil
}
func (m *mockGatewayInvoiceRepo) Update(_ context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	cp := *inv
	m.invoices[cp.ID] = &cp
	return &cp, nil
}
func (m *mockGatewayInvoiceRepo) UpdateStatus(_ context.Context, _ string, _ domain.InvoiceStatus, _ int) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) UpdatePaidAmount(_ context.Context, _ string, _ int64, _ int) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) List(_ context.Context, _ domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) ExistsForPeriod(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}
func (m *mockGatewayInvoiceRepo) ExistsForPeriodPrepaid(_ context.Context, _ string, _, _ int) (bool, error) {
	return false, nil
}
func (m *mockGatewayInvoiceRepo) FindOverdue(_ context.Context, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) GetSummary(_ context.Context, _ string, _, _ *int) (*domain.InvoiceSummary, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) FindOpenByCustomer(_ context.Context, customerID string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID && (inv.Status == domain.InvoiceStatusBelumBayar ||
			inv.Status == domain.InvoiceStatusTerlambat ||
			inv.Status == domain.InvoiceStatusBayarSebagian) {
			cp := *inv
			result = append(result, &cp)
		}
	}
	return result, nil
}
func (m *mockGatewayInvoiceRepo) FindOpenByCustomerForUpdate(ctx context.Context, customerID string) ([]*domain.Invoice, error) {
	return m.FindOpenByCustomer(ctx, customerID)
}
func (m *mockGatewayInvoiceRepo) GetByIDsForUpdate(_ context.Context, ids []string) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, id := range ids {
		if inv, ok := m.invoices[id]; ok {
			cp := *inv
			result = append(result, &cp)
		}
	}
	return result, nil
}
func (m *mockGatewayInvoiceRepo) FindOverdueForIsolir(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) FindOverdueForSuspend(_ context.Context, _ string, _ int, _ time.Time) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockGatewayInvoiceRepo) HasOutstandingInvoices(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockGatewayInvoiceRepo) SumOutstandingAmount(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockGatewayInvoiceRepo) CountOutstandingInvoices(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// --- Mock: CustomerRepository (subset yang dibutuhkan) ---

type mockGatewayCustomerRepo struct {
	customers map[string]*domain.Customer
}

func newMockGatewayCustomerRepo() *mockGatewayCustomerRepo {
	return &mockGatewayCustomerRepo{customers: make(map[string]*domain.Customer)}
}

func (m *mockGatewayCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	cp := *c
	return &cp, nil
}
func (m *mockGatewayCustomerRepo) Create(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	return c, nil
}
func (m *mockGatewayCustomerRepo) Update(_ context.Context, c *domain.Customer) (*domain.Customer, error) {
	return c, nil
}
func (m *mockGatewayCustomerRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *mockGatewayCustomerRepo) List(_ context.Context, _ domain.CustomerListParams) (*domain.CustomerListResult, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) UpdateStatus(_ context.Context, _ string, _ domain.CustomerStatus) (*domain.Customer, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) UpdatePackage(_ context.Context, _, _ string) (*domain.Customer, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) GetMaxSeq(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *mockGatewayCustomerRepo) PhoneExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockGatewayCustomerRepo) BulkUpdateStatus(_ context.Context, _ []string, _ domain.CustomerStatus) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) BulkUpdateFields(_ context.Context, _ []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) BulkSoftDelete(_ context.Context, _ []string) ([]domain.BulkResult, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) GetByIDs(_ context.Context, _ []string) ([]*domain.Customer, error) {
	return nil, nil
}
func (m *mockGatewayCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}
