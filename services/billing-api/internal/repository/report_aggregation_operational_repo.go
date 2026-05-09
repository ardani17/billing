package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Operational Methods ---

// GetAdminActivity mengambil laporan aktivitas admin/user dari audit logs.
func (r *ReportAggregationRepo) GetAdminActivity(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.ActivityReport, error) {
	tID := stringToUUID(tenantID)
	pStart := timeToPgTimestamptz(periodStart)
	pEnd := timeToPgTimestamptz(periodEnd)

	// Ambil aktivitas per user.
	actRows, err := r.queries.GetAdminActivity(ctx, GetAdminActivityParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil admin activity: %w", err)
	}
	perUser := make([]domain.UserActivity, 0, len(actRows))
	for _, row := range actRows {
		// Konversi last_active_at dari interface{} ke time.Time.
		var lastActive time.Time
		if t, ok := row.LastActiveAt.(time.Time); ok {
			lastActive = t
		}
		perUser = append(perUser, domain.UserActivity{
			UserID:       uuidToString(row.UserID),
			UserName:     row.UserName,
			Role:         row.Role,
			LoginDays:    int(row.LoginDays),
			ActionCount:  int(row.ActionCount),
			LastActiveAt: lastActive,
		})
	}

	// Ambil top actions.
	actionRows, err := r.queries.GetTopActions(ctx, GetTopActionsParams{
		TenantID: tID, PeriodStart: pStart, PeriodEnd: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil top actions: %w", err)
	}
	topActions := make([]domain.ActionSummary, 0, len(actionRows))
	for _, row := range actionRows {
		topActions = append(topActions, domain.ActionSummary{
			ActionType: row.ActionType,
			Count:      int(row.Count),
			Percentage: row.Percentage,
		})
	}

	return &domain.ActivityReport{
		PerUser:    perUser,
		TopActions: topActions,
	}, nil
}

// --- Dashboard Methods ---

// GetDashboardData mengambil data ringkasan untuk dashboard widget.
func (r *ReportAggregationRepo) GetDashboardData(ctx context.Context, tenantID string) (*domain.DashboardData, error) {
	row, err := r.queries.GetDashboardData(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil dashboard data: %w", err)
	}
	return &domain.DashboardData{
		TotalActiveCustomers: int(row.TotalActiveCustomers),
		CustomersTrend:       row.CustomersTrend,
		MonthlyRevenue:       row.MonthlyRevenue,
		TotalReceivables:     row.TotalReceivables,
		ReceivablesCount:     int(row.ReceivablesCount),
		CollectionRate:       row.CollectionRate,
		ChurnRate:            row.ChurnRate,
		ARPU:                 row.Arpu,
	}, nil
}

// --- Custom Report Methods ---

// GetCustomReportData mengambil data laporan kustom berdasarkan metrik dan dimensi.
// Implementasi sederhana: mengembalikan nil karena kustom report memerlukan
// dinamis kueri building yang akan dihandle di usecase layer.
func (r *ReportAggregationRepo) GetCustomReportData(ctx context.Context, tenantID string, metrics []string, groupBy, subGroupBy string, periodStart, periodEnd time.Time) (interface{}, error) {
	// Custom report data diassemble di usecase layer dari kueri-kueri yang sudah ada.
	// Repositori hanya menyediakan building blocks via method lain.
	return nil, nil
}

// --- Forecast Data Methods ---

// GetMonthlyRevenueHistory mengambil data historis pendapatan bulanan untuk linear regression.
func (r *ReportAggregationRepo) GetMonthlyRevenueHistory(ctx context.Context, tenantID string, months int) ([]domain.DataPoint, error) {
	rows, err := r.queries.GetMonthlyRevenueHistory(ctx, GetMonthlyRevenueHistoryParams{
		Column1:  int32(months),
		TenantID: stringToUUID(tenantID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil monthly revenue history: %w", err)
	}
	points := make([]domain.DataPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.DataPoint{
			X: float64(row.X),
			Y: row.Y,
		})
	}
	return points, nil
}

// GetMonthlyCustomerHistory mengambil data historis jumlah pelanggan bulanan untuk linear regression.
func (r *ReportAggregationRepo) GetMonthlyCustomerHistory(ctx context.Context, tenantID string, months int) ([]domain.DataPoint, error) {
	rows, err := r.queries.GetMonthlyCustomerHistory(ctx, GetMonthlyCustomerHistoryParams{
		Column1:  int32(months),
		TenantID: stringToUUID(tenantID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil monthly customer history: %w", err)
	}
	points := make([]domain.DataPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.DataPoint{
			X: float64(row.X),
			Y: row.Y,
		})
	}
	return points, nil
}

// GetMonthlyReceivablesHistory mengambil data historis piutang bulanan untuk linear regression.
func (r *ReportAggregationRepo) GetMonthlyReceivablesHistory(ctx context.Context, tenantID string, months int) ([]domain.DataPoint, error) {
	rows, err := r.queries.GetMonthlyReceivablesHistory(ctx, GetMonthlyReceivablesHistoryParams{
		Column1:  int32(months),
		TenantID: stringToUUID(tenantID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil monthly receivables history: %w", err)
	}
	points := make([]domain.DataPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.DataPoint{
			X: float64(row.X),
			Y: row.Y,
		})
	}
	return points, nil
}
