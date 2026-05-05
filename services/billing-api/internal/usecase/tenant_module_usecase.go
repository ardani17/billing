package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type TenantModuleRepository interface {
	Capabilities(ctx context.Context, tenantID string) (domain.TenantModuleCapabilities, error)
}

type TenantModuleUsecase struct {
	repo TenantModuleRepository
}

func NewTenantModuleUsecase(repo TenantModuleRepository) *TenantModuleUsecase {
	return &TenantModuleUsecase{repo: repo}
}

func (u *TenantModuleUsecase) Capabilities(ctx context.Context, tenantID string) (domain.TenantModuleCapabilities, error) {
	return u.repo.Capabilities(ctx, tenantID)
}
