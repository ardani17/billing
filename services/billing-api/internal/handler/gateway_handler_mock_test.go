package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

type mockGatewayConfigRepo struct {
	configs map[string]*domain.GatewayConfig
}

func newMockGatewayConfigRepo() *mockGatewayConfigRepo {
	return &mockGatewayConfigRepo{configs: make(map[string]*domain.GatewayConfig)}
}

func (m *mockGatewayConfigRepo) Create(_ context.Context, c *domain.GatewayConfig) (*domain.GatewayConfig, error) {
	if c.ID == "" {
		c.ID = fmt.Sprintf("cfg-%d", len(m.configs)+1)
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	cp := *c
	m.configs[cp.ID] = &cp
	return &cp, nil
}

func (m *mockGatewayConfigRepo) GetByID(_ context.Context, id string) (*domain.GatewayConfig, error) {
	c, ok := m.configs[id]
	if !ok {
		return nil, domain.ErrGatewayConfigNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *mockGatewayConfigRepo) Update(_ context.Context, c *domain.GatewayConfig) (*domain.GatewayConfig, error) {
	if _, ok := m.configs[c.ID]; !ok {
		return nil, domain.ErrGatewayConfigNotFound
	}
	c.UpdatedAt = time.Now()
	cp := *c
	m.configs[cp.ID] = &cp
	return &cp, nil
}

func (m *mockGatewayConfigRepo) Deactivate(_ context.Context, id string) error {
	c, ok := m.configs[id]
	if !ok {
		return domain.ErrGatewayConfigNotFound
	}
	c.IsActive = false
	return nil
}

func (m *mockGatewayConfigRepo) ListByTenant(_ context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	var result []*domain.GatewayConfig
	for _, c := range m.configs {
		if c.TenantID == tenantID {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockGatewayConfigRepo) GetActiveByTenant(_ context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	var result []*domain.GatewayConfig
	for _, c := range m.configs {
		if c.TenantID == tenantID && c.IsActive {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockGatewayConfigRepo) GetActiveByProvider(_ context.Context, tenantID string, provider domain.GatewayProvider) (*domain.GatewayConfig, error) {
	for _, c := range m.configs {
		if c.TenantID == tenantID && c.GatewayProvider == provider && c.IsActive {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrGatewayConfigNotFound
}

func (m *mockGatewayConfigRepo) ExistsByProvider(_ context.Context, tenantID string, provider domain.GatewayProvider) (bool, error) {
	for _, c := range m.configs {
		if c.TenantID == tenantID && c.GatewayProvider == provider && c.IsActive {
			return true, nil
		}
	}
	return false, nil
}

// encryptTestKey mengenkripsi key menggunakan testMasterKey untuk test.
func encryptTestKey(key string) (string, error) {
	return gateway.EncryptAESGCM(key, testMasterKey)
}
