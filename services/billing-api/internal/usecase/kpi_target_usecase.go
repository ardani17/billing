// kpi_target_usecase.go mengimplementasikan domain.KPITargetUsecase.
// Membungkus KPITargetRepository dengan logika bisnis sederhana.
package usecase

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// KPITargetUsecase mengimplementasikan domain.KPITargetUsecase.
type KPITargetUsecase struct {
	kpiTargetRepo domain.KPITargetRepository
	logger        zerolog.Logger
}

// NewKPITargetUsecase membuat instance baru KPITargetUsecase.
func NewKPITargetUsecase(
	kpiTargetRepo domain.KPITargetRepository,
	logger zerolog.Logger,
) *KPITargetUsecase {
	return &KPITargetUsecase{
		kpiTargetRepo: kpiTargetRepo,
		logger:        logger.With().Str("component", "kpi_target_usecase").Logger(),
	}
}

// Get mengambil target KPI untuk tenant.
func (u *KPITargetUsecase) Get(ctx context.Context, tenantID string) (*domain.KPITarget, error) {
	return u.kpiTargetRepo.GetByTenant(ctx, tenantID)
}

// Upsert membuat atau memperbarui target KPI untuk tenant.
func (u *KPITargetUsecase) Upsert(ctx context.Context, tenantID string, req domain.UpdateKPITargetRequest) (*domain.KPITarget, error) {
	target := &domain.KPITarget{
		TenantID:                  tenantID,
		MonthlyRevenueTarget:      req.MonthlyRevenueTarget,
		CollectionRateTarget:      req.CollectionRateTarget,
		MaxReceivables:            req.MaxReceivables,
		NewCustomersMonthlyTarget: req.NewCustomersMonthlyTarget,
		MaxChurnRate:              req.MaxChurnRate,
		TotalCustomersTarget:      req.TotalCustomersTarget,
		SLAUptimeTarget:           req.SLAUptimeTarget,
		MaxActiveAlarms:           req.MaxActiveAlarms,
		MinSignalQualityPct:       req.MinSignalQualityPct,
	}

	return u.kpiTargetRepo.Upsert(ctx, target)
}
