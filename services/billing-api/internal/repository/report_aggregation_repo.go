package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportAggregationRepo mengimplementasikan domain.ReportAggregationRepository
// dengan membungkus semua aggregation sqlc queries dan merakit DTOs dari raw query results.
// Dipecah ke beberapa file karena melebihi 200 baris:
//   - report_aggregation_repo.go - struct, constructor, financial methods
//   - report_aggregation_customer_repo.go - customer methods
//   - report_aggregation_operational_repo.go - operational, dashboard, kustom, forecast methods
type ReportAggregationRepo struct {
	// queries adalah sqlc-generated Queries untuk aggregation queries.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewReportAggregationRepo membuat instance baru ReportAggregationRepo.
func NewReportAggregationRepo(queries *Queries, pool *pgxpool.Pool) *ReportAggregationRepo {
	return &ReportAggregationRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Fungsi bantu konversi waktu ke pgtype ---

// timeToPgTimestamptz mengkonversi time.Time ke pgtype.Timestamptz.
func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// timeToPgDate mengkonversi time.Time ke pgtype.Date.
func timeToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()),
		Valid: !t.IsZero(),
	}
}

// --- Financial Methods ---

// GetRevenueSummary mengambil ringkasan pendapatan per sumber untuk periode tertentu.
func (r *ReportAggregationRepo) GetRevenueSummary(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, areaID, packageID string) (*domain.RevenueSource, error) {
	row, err := r.queries.GetRevenueSummary(ctx, GetRevenueSummaryParams{
		TenantID:      stringToUUID(tenantID),
		PurchasedAt:   timeToPgTimestamptz(periodStart),
		PurchasedAt_2: timeToPgTimestamptz(periodEnd),
		AreaID:        stringToUUID(areaID),
		PackageID:     stringToUUID(packageID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil revenue summary: %w", err)
	}
	return &domain.RevenueSource{
		MonthlySubscription: row.MonthlySubscription,
		VoucherSales:        row.VoucherSales,
		InstallationFees:    row.InstallationFees,
		LateFees:            row.LateFees,
		Other:               row.Other,
		Total:               row.Total,
	}, nil
}

// GetMonthlyRevenueTrend mengambil trend pendapatan bulanan untuk N bulan terakhir.
func (r *ReportAggregationRepo) GetMonthlyRevenueTrend(ctx context.Context, tenantID string, months int) ([]domain.MonthlyRevenueTrend, error) {
	rows, err := r.queries.GetMonthlyRevenueTrend(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil monthly revenue trend: %w", err)
	}
	trends := make([]domain.MonthlyRevenueTrend, 0, len(rows))
	for _, row := range rows {
		trends = append(trends, domain.MonthlyRevenueTrend{
			Month:               row.Month,
			TotalRevenue:        row.TotalRevenue,
			MonthlySubscription: row.MonthlySubscription,
			VoucherSales:        row.VoucherSales,
			OtherRevenue:        row.OtherRevenue,
		})
	}
	return trends, nil
}

// GetAgingReport mengambil laporan piutang/aging dengan bucket, collection rate, dan top debtors.
func (r *ReportAggregationRepo) GetAgingReport(ctx context.Context, tenantID string, periodEnd time.Time, areaID, packageID string) (*domain.AgingReport, error) {
	tID := stringToUUID(tenantID)
	aID := stringToUUID(areaID)
	pID := stringToUUID(packageID)

	// Ambil aging buckets.
	bucketRows, err := r.queries.GetAgingBuckets(ctx, GetAgingBucketsParams{
		TenantID: tID, AreaID: aID, PackageID: pID,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil aging buckets: %w", err)
	}
	buckets := make([]domain.AgingBucket, 0, len(bucketRows))
	var totalOutstanding int64
	for _, row := range bucketRows {
		label := fmt.Sprintf("%v", row.Label)
		buckets = append(buckets, domain.AgingBucket{
			Label: label, TotalAmount: row.TotalAmount, CustomerCount: int(row.CustomerCount),
		})
		totalOutstanding += row.TotalAmount
	}

	// Ambil collection rate.
	periodStart := time.Date(periodEnd.Year(), periodEnd.Month(), 1, 0, 0, 0, 0, periodEnd.Location())
	crRow, err := r.queries.GetCollectionRate(ctx, GetCollectionRateParams{
		TenantID: tID, DueDate: timeToPgDate(periodStart), DueDate_2: timeToPgDate(periodEnd),
		AreaID: aID, PackageID: pID,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil collection rate: %w", err)
	}

	// Ambil avg days to pay.
	avgDays, err := r.queries.GetAvgDaysToPay(ctx, GetAvgDaysToPayParams{
		TenantID: tID, PaymentDate: timeToPgDate(periodStart), PaymentDate_2: timeToPgDate(periodEnd),
		AreaID: aID, PackageID: pID,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil avg days to pay: %w", err)
	}

	// Ambil top debtors.
	debtorRows, err := r.queries.GetTopDebtors(ctx, GetTopDebtorsParams{
		TenantID: tID, AreaID: aID, PackageID: pID,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil top debtors: %w", err)
	}
	debtors := make([]domain.TopDebtor, 0, len(debtorRows))
	for _, row := range debtorRows {
		debtors = append(debtors, domain.TopDebtor{
			CustomerID: uuidToString(row.CustomerID), CustomerName: row.CustomerName,
			TotalOutstanding: row.TotalOutstanding, MonthsOverdue: int(row.MonthsOverdue),
		})
	}

	// Ambil receivables trend.
	trendRows, err := r.queries.GetReceivablesTrend(ctx, tID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil receivables trend: %w", err)
	}
	trend := make([]domain.ReceivablesTrend, 0, len(trendRows))
	for _, row := range trendRows {
		trend = append(trend, domain.ReceivablesTrend{
			Month: row.Month, TotalOutstanding: row.TotalOutstanding,
		})
	}

	return &domain.AgingReport{
		Buckets: buckets, TotalOutstanding: totalOutstanding,
		CollectionRate: crRow.CollectionRate, AvgDaysToPay: avgDays,
		TopDebtors: debtors, Trend: trend,
	}, nil
}
