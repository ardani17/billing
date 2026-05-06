package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// BillingSettingsUsecase membungkus repository billing settings dengan default dan validasi bisnis.
type BillingSettingsUsecase struct {
	repo   domain.BillingSettingsRepository
	logger zerolog.Logger
}

// NewBillingSettingsUsecase membuat instance BillingSettingsUsecase.
func NewBillingSettingsUsecase(repo domain.BillingSettingsRepository, logger zerolog.Logger) *BillingSettingsUsecase {
	return &BillingSettingsUsecase{
		repo:   repo,
		logger: logger.With().Str("component", "billing_settings_usecase").Logger(),
	}
}

// Get mengambil settings tenant, atau mengembalikan default jika belum pernah disimpan.
func (u *BillingSettingsUsecase) Get(ctx context.Context, tenantID string) (*domain.BillingSettings, error) {
	settings, err := u.repo.GetByTenantID(ctx, tenantID)
	if err == nil {
		return settings, nil
	}
	if errors.Is(err, domain.ErrBillingSettingsNotFound) {
		return defaultBillingSettings(tenantID), nil
	}
	return nil, err
}

// Update menyimpan settings billing tenant.
func (u *BillingSettingsUsecase) Update(ctx context.Context, tenantID string, req domain.UpdateBillingSettingsRequest) (*domain.BillingSettings, error) {
	if err := validateBillingSettingsRequest(req); err != nil {
		return nil, err
	}

	settings := &domain.BillingSettings{
		TenantID:           tenantID,
		GenerateDays:       req.GenerateDays,
		GracePeriodDays:    req.GracePeriodDays,
		SuspendDays:        req.SuspendDays,
		TaxEnabled:         req.TaxEnabled,
		TaxRate:            req.TaxRate,
		PenaltyEnabled:     req.PenaltyEnabled,
		PenaltyType:        req.PenaltyType,
		PenaltyAmount:      req.PenaltyAmount,
		PenaltyPercentage:  req.PenaltyPercentage,
		PenaltyDailyAmount: req.PenaltyDailyAmount,
		PenaltyMaxAmount:   req.PenaltyMaxAmount,
		InvoicePrefix:      strings.ToUpper(strings.TrimSpace(req.InvoicePrefix)),
		NewCustomerBilling: req.NewCustomerBilling,
		Timezone:           strings.TrimSpace(req.Timezone),
		AutoIsolir:         req.AutoIsolir,
		AutoOpenIsolir:     req.AutoOpenIsolir,
	}

	saved, err := u.repo.Upsert(ctx, settings)
	if err != nil {
		u.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal menyimpan billing settings")
		return nil, err
	}
	return saved, nil
}

func defaultBillingSettings(tenantID string) *domain.BillingSettings {
	return &domain.BillingSettings{
		TenantID:           tenantID,
		GenerateDays:       1,
		GracePeriodDays:    3,
		SuspendDays:        30,
		TaxEnabled:         false,
		TaxRate:            0,
		PenaltyEnabled:     false,
		PenaltyType:        domain.PenaltyFixed,
		PenaltyAmount:      0,
		PenaltyPercentage:  0,
		PenaltyDailyAmount: 0,
		PenaltyMaxAmount:   0,
		InvoicePrefix:      "INV",
		NewCustomerBilling: "prorate",
		Timezone:           "Asia/Jakarta",
		AutoIsolir:         false,
		AutoOpenIsolir:     true,
	}
}

func validateBillingSettingsRequest(req domain.UpdateBillingSettingsRequest) error {
	prefix := strings.TrimSpace(req.InvoicePrefix)
	if prefix == "" {
		return fmt.Errorf("invoice prefix wajib diisi")
	}
	if req.PenaltyEnabled {
		switch req.PenaltyType {
		case domain.PenaltyFixed:
			if req.PenaltyAmount <= 0 {
				return fmt.Errorf("penalty_amount wajib lebih dari 0 untuk tipe fixed")
			}
		case domain.PenaltyPercentage:
			if req.PenaltyPercentage <= 0 {
				return fmt.Errorf("penalty_percentage wajib lebih dari 0 untuk tipe percentage")
			}
		case domain.PenaltyDaily:
			if req.PenaltyDailyAmount <= 0 {
				return fmt.Errorf("penalty_daily_amount wajib lebih dari 0 untuk tipe daily")
			}
		default:
			return fmt.Errorf("penalty_type tidak valid")
		}
	}
	return nil
}
