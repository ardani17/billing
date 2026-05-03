package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Customer Methods ---

// GetCustomerGrowth mengambil data pertumbuhan pelanggan untuk periode tertentu.
func (r *ReportAggregationRepo) GetCustomerGrowth(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.CustomerGrowthReport, error) {
	tID := stringToUUID(tenantID)

	// Ambil data pertumbuhan.
	growthRow, err := r.queries.GetCustomerGrowthData(ctx, GetCustomerGrowthDataParams{
		TenantID:    tID,
		PeriodStart: timeToPgDate(periodStart),
		PeriodEnd:   timeToPgDate(periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil customer growth data: %w", err)
	}

	// Ambil ARPU.
	arpu, err := r.queries.GetARPU(ctx, GetARPUParams{
		TenantID:    tID,
		PeriodStart: timeToPgDate(periodStart),
		PeriodEnd:   timeToPgDate(periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil ARPU: %w", err)
	}

	// Ambil CLV.
	clv, err := r.queries.GetCLV(ctx, GetCLVParams{
		TenantID:    tID,
		PeriodStart: timeToPgDate(periodStart),
		PeriodEnd:   timeToPgDate(periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil CLV: %w", err)
	}

	// Ambil churn analysis untuk churn rate.
	churnRow, err := r.queries.GetChurnAnalysis(ctx, GetChurnAnalysisParams{
		TenantID:    tID,
		PeriodStart: timeToPgTimestamptz(periodStart),
		PeriodEnd:   timeToPgTimestamptz(periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil churn analysis: %w", err)
	}

	return &domain.CustomerGrowthReport{
		TotalActive:      int(growthRow.TotalActive),
		NewCustomers:     int(growthRow.NewCustomers),
		ChurnedCustomers: int(growthRow.ChurnedCustomers),
		NetGrowth:        int(growthRow.NetGrowth),
		ARPU:             arpu,
		CLV:              clv,
		ChurnRate:        churnRow.ChurnRate,
	}, nil
}

// GetMonthlyGrowthTrend mengambil trend pertumbuhan pelanggan bulanan.
func (r *ReportAggregationRepo) GetMonthlyGrowthTrend(ctx context.Context, tenantID string, months int) ([]domain.MonthlyGrowthTrend, error) {
	rows, err := r.queries.GetMonthlyGrowthTrend(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil monthly growth trend: %w", err)
	}
	trends := make([]domain.MonthlyGrowthTrend, 0, len(rows))
	for _, row := range rows {
		trends = append(trends, domain.MonthlyGrowthTrend{
			Month:            row.Month,
			TotalActive:      int(row.TotalActive),
			NewCustomers:     int(row.NewCustomers),
			ChurnedCustomers: int(row.ChurnedCustomers),
		})
	}
	return trends, nil
}

// GetCustomerDistribution mengambil distribusi pelanggan per paket, area, status, dan metode koneksi.
func (r *ReportAggregationRepo) GetCustomerDistribution(ctx context.Context, tenantID string, periodEnd time.Time) (*domain.CustomerDistributionReport, error) {
	tID := stringToUUID(tenantID)

	// Distribusi per paket.
	pkgRows, err := r.queries.GetCustomerDistributionByPackage(ctx, tID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil distribution by package: %w", err)
	}
	byPackage := make([]domain.DistributionItem, 0, len(pkgRows))
	for _, row := range pkgRows {
		byPackage = append(byPackage, domain.DistributionItem{
			ID: row.ID, Name: row.Name, Count: int(row.Count), Percentage: row.Percentage,
		})
	}

	// Distribusi per area.
	areaRows, err := r.queries.GetCustomerDistributionByArea(ctx, tID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil distribution by area: %w", err)
	}
	byArea := make([]domain.DistributionItem, 0, len(areaRows))
	for _, row := range areaRows {
		byArea = append(byArea, domain.DistributionItem{
			ID: row.ID, Name: row.Name, Count: int(row.Count), Percentage: row.Percentage,
		})
	}

	// Distribusi per status.
	statusRows, err := r.queries.GetCustomerDistributionByStatus(ctx, tID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil distribution by status: %w", err)
	}
	byStatus := make(map[domain.CustomerStatus]int, len(statusRows))
	for _, row := range statusRows {
		byStatus[domain.CustomerStatus(row.Name)] = int(row.Count)
	}

	// Distribusi per metode koneksi.
	connRows, err := r.queries.GetCustomerDistributionByConnectionMethod(ctx, tID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil distribution by connection method: %w", err)
	}
	byConn := make([]domain.DistributionItem, 0, len(connRows))
	for _, row := range connRows {
		byConn = append(byConn, domain.DistributionItem{
			Name: row.Name, Count: int(row.Count), Percentage: row.Percentage,
		})
	}

	return &domain.CustomerDistributionReport{
		ByPackage:          byPackage,
		ByArea:             byArea,
		ByStatus:           byStatus,
		ByConnectionMethod: byConn,
	}, nil
}

// GetChurnAnalysis mengambil analisis churn pelanggan per alasan, paket, dan area.
func (r *ReportAggregationRepo) GetChurnAnalysis(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.ChurnAnalysisReport, error) {
	tID := stringToUUID(tenantID)
	pStart := timeToPgTimestamptz(periodStart)
	pEnd := timeToPgTimestamptz(periodEnd)

	// Ambil churn summary.
	churnRow, err := r.queries.GetChurnAnalysis(ctx, GetChurnAnalysisParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil churn analysis: %w", err)
	}

	// Ambil churn per alasan.
	reasonRows, err := r.queries.GetChurnByReason(ctx, GetChurnByReasonParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil churn by reason: %w", err)
	}
	byReason := make([]domain.ChurnByReason, 0, len(reasonRows))
	for _, row := range reasonRows {
		byReason = append(byReason, domain.ChurnByReason{
			Reason: fmt.Sprintf("%v", row.Reason), Count: int(row.Count), Percentage: row.Percentage,
		})
	}

	// Ambil churn per paket.
	pkgRows, err := r.queries.GetChurnByPackage(ctx, GetChurnByPackageParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil churn by package: %w", err)
	}
	byPackage := make([]domain.DistributionItem, 0, len(pkgRows))
	for _, row := range pkgRows {
		byPackage = append(byPackage, domain.DistributionItem{
			Name: row.Name, Count: int(row.Count), Percentage: row.Percentage,
		})
	}

	// Ambil churn per area.
	areaRows, err := r.queries.GetChurnByArea(ctx, GetChurnByAreaParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil churn by area: %w", err)
	}
	byArea := make([]domain.DistributionItem, 0, len(areaRows))
	for _, row := range areaRows {
		byArea = append(byArea, domain.DistributionItem{
			Name: row.Name, Count: int(row.Count), Percentage: row.Percentage,
		})
	}

	// Ambil avg customer lifetime.
	avgLifetime, err := r.queries.GetAvgCustomerLifetime(ctx, GetAvgCustomerLifetimeParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil avg customer lifetime: %w", err)
	}

	return &domain.ChurnAnalysisReport{
		ChurnedCount:          int(churnRow.ChurnedCount),
		ChurnRate:             churnRow.ChurnRate,
		ByReason:              byReason,
		ByPackage:             byPackage,
		ByArea:                byArea,
		AverageLifetimeMonths: avgLifetime,
	}, nil
}
