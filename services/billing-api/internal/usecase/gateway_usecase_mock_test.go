// dan BillingSettingsRepository untuk unit test GatewayUsecase.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// testMasterKey adalah 32-byte key untuk test enkripsi.
var testMasterKey = []byte("01234567890123456789012345678901")

type gwMockConfigRepo struct {
	configs map[string]*domain.GatewayConfig
}

func newGwMockConfigRepo() *gwMockConfigRepo {
	return &gwMockConfigRepo{configs: make(map[string]*domain.GatewayConfig)}
}

func (m *gwMockConfigRepo) Create(_ context.Context, c *domain.GatewayConfig) (*domain.GatewayConfig, error) {
	if c.ID == "" {
		c.ID = fmt.Sprintf("cfg-%d", len(m.configs)+1)
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	cp := *c
	m.configs[cp.ID] = &cp
	return &cp, nil
}

func (m *gwMockConfigRepo) GetByID(_ context.Context, id string) (*domain.GatewayConfig, error) {
	c, ok := m.configs[id]
	if !ok {
		return nil, domain.ErrGatewayConfigNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *gwMockConfigRepo) Update(_ context.Context, c *domain.GatewayConfig) (*domain.GatewayConfig, error) {
	if _, ok := m.configs[c.ID]; !ok {
		return nil, domain.ErrGatewayConfigNotFound
	}
	c.UpdatedAt = time.Now()
	cp := *c
	m.configs[cp.ID] = &cp
	return &cp, nil
}

func (m *gwMockConfigRepo) Deactivate(_ context.Context, id string) error {
	c, ok := m.configs[id]
	if !ok {
		return domain.ErrGatewayConfigNotFound
	}
	c.IsActive = false
	return nil
}

func (m *gwMockConfigRepo) ListByTenant(_ context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	var result []*domain.GatewayConfig
	for _, c := range m.configs {
		if c.TenantID == tenantID {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *gwMockConfigRepo) GetActiveByTenant(_ context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	var result []*domain.GatewayConfig
	for _, c := range m.configs {
		if c.TenantID == tenantID && c.IsActive {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *gwMockConfigRepo) GetActiveByProvider(_ context.Context, tenantID string, provider domain.GatewayProvider) (*domain.GatewayConfig, error) {
	for _, c := range m.configs {
		if c.TenantID == tenantID && c.GatewayProvider == provider && c.IsActive {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrGatewayConfigNotFound
}

func (m *gwMockConfigRepo) ExistsByProvider(_ context.Context, tenantID string, provider domain.GatewayProvider) (bool, error) {
	for _, c := range m.configs {
		if c.TenantID == tenantID && c.GatewayProvider == provider && c.IsActive {
			return true, nil
		}
	}
	return false, nil
}

type gwMockLinkRepo struct {
	links    map[string]*domain.PaymentLink
	junction map[string][]string // linkID -> invoiceIDs
}

func newGwMockLinkRepo() *gwMockLinkRepo {
	return &gwMockLinkRepo{
		links:    make(map[string]*domain.PaymentLink),
		junction: make(map[string][]string),
	}
}

func (m *gwMockLinkRepo) Create(_ context.Context, l *domain.PaymentLink, invoiceIDs []string) (*domain.PaymentLink, error) {
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()
	cp := *l
	m.links[cp.ID] = &cp
	m.junction[cp.ID] = invoiceIDs
	return &cp, nil
}

func (m *gwMockLinkRepo) GetByID(_ context.Context, id string) (*domain.PaymentLink, error) {
	l, ok := m.links[id]
	if !ok {
		return nil, domain.ErrPaymentLinkNotFound
	}
	cp := *l
	return &cp, nil
}

func (m *gwMockLinkRepo) GetByExternalID(_ context.Context, extID string) (*domain.PaymentLink, error) {
	for _, l := range m.links {
		if l.ExternalID == extID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, domain.ErrPaymentLinkNotFound
}

func (m *gwMockLinkRepo) GetActiveByCustomer(_ context.Context, customerID string) (*domain.PaymentLink, error) {
	for _, l := range m.links {
		if l.CustomerID == customerID && l.Status == domain.PaymentLinkActive {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *gwMockLinkRepo) GetInvoiceIDsByLinkID(_ context.Context, linkID string) ([]string, error) {
	return m.junction[linkID], nil
}

func (m *gwMockLinkRepo) UpdateStatus(_ context.Context, id string, status domain.PaymentLinkStatus) error {
	l, ok := m.links[id]
	if !ok {
		return domain.ErrPaymentLinkNotFound
	}
	l.Status = status
	return nil
}

func (m *gwMockLinkRepo) UpdateStatusPaid(_ context.Context, id string, method string, paidAt time.Time) error {
	l, ok := m.links[id]
	if !ok {
		return domain.ErrPaymentLinkNotFound
	}
	l.Status = domain.PaymentLinkPaid
	l.PaidMethod = method
	l.PaidAt = &paidAt
	return nil
}

func (m *gwMockLinkRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.PaymentLink, error) {
	var result []*domain.PaymentLink
	for linkID, invIDs := range m.junction {
		for _, id := range invIDs {
			if id == invoiceID {
				if l, ok := m.links[linkID]; ok {
					cp := *l
					result = append(result, &cp)
				}
			}
		}
	}
	return result, nil
}

func (m *gwMockLinkRepo) FindExpired(_ context.Context, _ int) ([]*domain.PaymentLink, error) {
	return nil, nil
}

func (m *gwMockLinkRepo) ExpireByID(_ context.Context, id string) error {
	l, ok := m.links[id]
	if !ok {
		return domain.ErrPaymentLinkNotFound
	}
	l.Status = domain.PaymentLinkExpired
	return nil
}
