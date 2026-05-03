package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KPITargetRepo mengimplementasikan domain.KPITargetRepository
// dengan membungkus sqlc-generated Queries dan pgxpool.Pool.
type KPITargetRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi kpi_targets.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewKPITargetRepo membuat instance baru KPITargetRepo.
func NewKPITargetRepo(queries *Queries, pool *pgxpool.Pool) *KPITargetRepo {
	return &KPITargetRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper mapping sqlc ↔ domain ---

// mapKPITargetRow memetakan sqlc KpiTarget ke domain.KPITarget.
func mapKPITargetRow(row KpiTarget) *domain.KPITarget {
	return &domain.KPITarget{
		ID:                        uuidToString(row.ID),
		TenantID:                  uuidToString(row.TenantID),
		MonthlyRevenueTarget:      int8ToInt64Ptr(row.MonthlyRevenueTarget),
		CollectionRateTarget:      numericToFloat64Ptr(row.CollectionRateTarget),
		MaxReceivables:            int8ToInt64Ptr(row.MaxReceivables),
		NewCustomersMonthlyTarget: int4ToIntPtr(row.NewCustomersMonthlyTarget),
		MaxChurnRate:              numericToFloat64Ptr(row.MaxChurnRate),
		TotalCustomersTarget:      int4ToIntPtr(row.TotalCustomersTarget),
		SLAUptimeTarget:           numericToFloat64Ptr(row.SlaUptimeTarget),
		MaxActiveAlarms:           int4ToIntPtr(row.MaxActiveAlarms),
		MinSignalQualityPct:       numericToFloat64Ptr(row.MinSignalQualityPercentage),
		CreatedAt:                 timestamptzToTime(row.CreatedAt),
		UpdatedAt:                 timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.KPITargetRepository ---

// GetByTenant mengambil target KPI berdasarkan tenant ID.
func (r *KPITargetRepo) GetByTenant(ctx context.Context, tenantID string) (*domain.KPITarget, error) {
	row, err := r.queries.GetKPITargetByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrKPITargetNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil KPI target: %w", err)
	}
	return mapKPITargetRow(row), nil
}

// Upsert membuat atau memperbarui target KPI untuk tenant (INSERT ON CONFLICT DO UPDATE).
func (r *KPITargetRepo) Upsert(ctx context.Context, target *domain.KPITarget) (*domain.KPITarget, error) {
	row, err := r.queries.UpsertKPITarget(ctx, UpsertKPITargetParams{
		TenantID:                   stringToUUID(target.TenantID),
		MonthlyRevenueTarget:       int64PtrToInt8(target.MonthlyRevenueTarget),
		CollectionRateTarget:       float64PtrToNumeric(target.CollectionRateTarget),
		MaxReceivables:             int64PtrToInt8(target.MaxReceivables),
		NewCustomersMonthlyTarget:  intPtrToInt4(target.NewCustomersMonthlyTarget),
		MaxChurnRate:               float64PtrToNumeric(target.MaxChurnRate),
		TotalCustomersTarget:       intPtrToInt4(target.TotalCustomersTarget),
		SlaUptimeTarget:            float64PtrToNumeric(target.SLAUptimeTarget),
		MaxActiveAlarms:            intPtrToInt4(target.MaxActiveAlarms),
		MinSignalQualityPercentage: float64PtrToNumeric(target.MinSignalQualityPct),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal upsert KPI target: %w", err)
	}
	return mapKPITargetRow(row), nil
}
