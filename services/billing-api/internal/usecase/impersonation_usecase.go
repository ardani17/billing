package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/pkg/auth"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ImpersonationUsecaseConfig berisi semua dependensi yang dibutuhkan ImpersonationUsecase.
type ImpersonationUsecaseConfig struct {
	UserRepo  domain.UserRepository
	JWTSecret string
	JWTExpiry time.Duration
}

// ImpersonationUsecase mengimplementasikan business logic impersonasi super admin.
type ImpersonationUsecase struct {
	userRepo  domain.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewImpersonationUsecase membuat instance baru ImpersonationUsecase.
func NewImpersonationUsecase(cfg ImpersonationUsecaseConfig) *ImpersonationUsecase {
	return &ImpersonationUsecase{
		userRepo:  cfg.UserRepo,
		jwtSecret: cfg.JWTSecret,
		jwtExpiry: cfg.JWTExpiry,
	}
}

// StartImpersonation membuat JWT dengan claims target user + impersonator_id.
// Hanya user dengan role tenant_admin yang boleh di-impersonate.
// Impersonasi super_admin lain ditolak dengan error forbidden.
func (uc *ImpersonationUsecase) StartImpersonation(ctx context.Context, impersonatorID string, req domain.ImpersonateRequest) (*domain.TokenPair, error) {
	// Ambil data target user
	targetUser, err := uc.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil target user: %w", err)
	}

	// Tolak impersonasi super_admin
	if targetUser.Role == domain.RoleSuperAdmin {
		return nil, domain.ErrForbidden
	}

	// Hanya boleh impersonate tenant_admin
	if targetUser.Role != domain.RoleTenantAdmin {
		return nil, domain.ErrForbidden
	}

	// Generate JWT dengan claims target user + impersonator_id
	tokenCfg := auth.TokenConfig{
		Secret: uc.jwtSecret,
		Expiry: uc.jwtExpiry,
		Issuer: "ispboss",
	}

	claims := auth.Claims{
		TenantID:       targetUser.TenantID,
		UserID:         targetUser.ID,
		Role:           string(targetUser.Role),
		ImpersonatorID: impersonatorID,
	}

	accessToken, err := auth.GenerateToken(tokenCfg, claims)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT impersonation: %w", err)
	}

	return &domain.TokenPair{
		AccessToken: accessToken,
		ExpiresIn:   int64(uc.jwtExpiry.Seconds()),
	}, nil
}

// StopImpersonation mengembalikan JWT ke claims super admin asli.
// Mengambil data super admin berdasarkan impersonatorID dan generate JWT baru.
func (uc *ImpersonationUsecase) StopImpersonation(ctx context.Context, impersonatorID string) (*domain.TokenPair, error) {
	// Ambil data super admin berdasarkan impersonatorID
	superAdmin, err := uc.userRepo.GetByID(ctx, impersonatorID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data super admin: %w", err)
	}

	// Generate JWT dengan claims super admin asli (tanpa impersonator_id)
	tokenCfg := auth.TokenConfig{
		Secret: uc.jwtSecret,
		Expiry: uc.jwtExpiry,
		Issuer: "ispboss",
	}

	claims := auth.Claims{
		TenantID: superAdmin.TenantID,
		UserID:   superAdmin.ID,
		Role:     string(superAdmin.Role),
	}

	accessToken, err := auth.GenerateToken(tokenCfg, claims)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT stop impersonation: %w", err)
	}

	return &domain.TokenPair{
		AccessToken: accessToken,
		ExpiresIn:   int64(uc.jwtExpiry.Seconds()),
	}, nil
}
