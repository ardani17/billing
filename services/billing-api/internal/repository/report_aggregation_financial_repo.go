package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Financial Methods (lanjutan) ---

// GetPaymentDistribution mengambil distribusi pembayaran per metode dan harian.
func (r *ReportAggregationRepo) GetPaymentDistribution(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, areaID, packageID string) (*domain.PaymentReport, error) {
	tID := stringToUUID(tenantID)
	aID := stringToUUID(areaID)
	pID := stringToUUID(packageID)
	pStart := timeToPgDate(periodStart)
	pEnd := timeToPgDate(periodEnd)

	// Ambil distribusi per metode pembayaran.
	distRows, err := r.queries.GetPaymentDistribution(ctx, GetPaymentDistributionParams{
		TenantID: tID, PaymentDate: pStart, PaymentDate_2: pEnd,
		AreaID: aID, PackageID: pID,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil payment distribution: %w", err)
	}
	methods := make([]domain.PaymentMethodBreakdown, 0, len(distRows))
	for _, row := range distRows {
		methods = append(methods, domain.PaymentMethodBreakdown{
			MethodName:       row.MethodName,
			TotalAmount:      row.TotalAmount,
			TransactionCount: int(row.TransactionCount),
			Percentage:       row.Percentage,
		})
	}

	// Ambil pembayaran harian.
	dailyRows, err := r.queries.GetDailyPayments(ctx, GetDailyPaymentsParams{
		TenantID: tID, PaymentDate: pStart, PaymentDate_2: pEnd,
		AreaID: aID, PackageID: pID,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daily payments: %w", err)
	}
	dailyPayments := make([]domain.DailyPayment, 0, len(dailyRows))
	var peakDate string
	var peakAmount int64
	for _, row := range dailyRows {
		dailyPayments = append(dailyPayments, domain.DailyPayment{
			Date:             row.Date,
			TotalAmount:      row.TotalAmount,
			TransactionCount: int(row.TransactionCount),
		})
		if row.TotalAmount > peakAmount {
			peakAmount = row.TotalAmount
			peakDate = row.Date
		}
	}

	return &domain.PaymentReport{
		Methods:         methods,
		DailyPayments:   dailyPayments,
		PeakPaymentDate: peakDate,
		PeakAmount:      peakAmount,
	}, nil
}

// GetVoucherRevenue mengambil laporan pendapatan voucher per paket dan reseller.
func (r *ReportAggregationRepo) GetVoucherRevenue(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.VoucherRevenueReport, error) {
	tID := stringToUUID(tenantID)
	pStart := timeToPgTimestamptz(periodStart)
	pEnd := timeToPgTimestamptz(periodEnd)

	// Ambil voucher revenue per paket.
	pkgRows, err := r.queries.GetVoucherRevenueByPackage(ctx, GetVoucherRevenueByPackageParams{
		TenantID: tID, PurchasedAt: pStart, PurchasedAt_2: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil voucher revenue by package: %w", err)
	}
	byPackage := make([]domain.VoucherByPackage, 0, len(pkgRows))
	var totalRevenue int64
	var totalCount int
	for _, row := range pkgRows {
		byPackage = append(byPackage, domain.VoucherByPackage{
			PackageName:  row.PackageName,
			TotalRevenue: row.TotalRevenue,
			VoucherCount: int(row.VoucherCount),
			Percentage:   row.Percentage,
		})
		totalRevenue += row.TotalRevenue
		totalCount += int(row.VoucherCount)
	}

	// Ambil voucher revenue per reseller.
	resRows, err := r.queries.GetVoucherRevenueByReseller(ctx, GetVoucherRevenueByResellerParams{
		TenantID: tID, PurchasedAt: pStart, PurchasedAt_2: pEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil voucher revenue by reseller: %w", err)
	}
	byReseller := make([]domain.VoucherByReseller, 0, len(resRows))
	var totalMargin int64
	for _, row := range resRows {
		byReseller = append(byReseller, domain.VoucherByReseller{
			ResellerName:   row.ResellerName,
			TotalRevenue:   row.TotalRevenue,
			VoucherCount:   int(row.VoucherCount),
			ResellerMargin: row.ResellerMargin,
		})
		totalMargin += row.ResellerMargin
	}

	return &domain.VoucherRevenueReport{
		TotalRevenue:        totalRevenue,
		TotalVoucherCount:   totalCount,
		ByPackage:           byPackage,
		ByReseller:          byReseller,
		TotalResellerMargin: totalMargin,
	}, nil
}

// GetRevenueByArea mengambil laporan pendapatan per area.
func (r *ReportAggregationRepo) GetRevenueByArea(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.RevenueByAreaReport, error) {
	rows, err := r.queries.GetRevenueByArea(ctx, GetRevenueByAreaParams{
		TenantID:    stringToUUID(tenantID),
		PaymentDate: timeToPgDate(periodStart), PaymentDate_2: timeToPgDate(periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil revenue by area: %w", err)
	}

	areas := make([]domain.AreaRevenue, 0, len(rows))
	var total domain.AreaRevenue
	var mostProfitable, attentionNeeded string
	var maxRevenue int64
	var maxOutstanding int64

	for _, row := range rows {
		area := domain.AreaRevenue{
			AreaID:           uuidToString(row.AreaID),
			AreaName:         row.AreaName,
			CustomerCount:    int(row.CustomerCount),
			TotalRevenue:     row.TotalRevenue,
			TotalOutstanding: row.TotalOutstanding,
			ARPU:             row.Arpu,
		}
		areas = append(areas, area)
		total.CustomerCount += int(row.CustomerCount)
		total.TotalRevenue += row.TotalRevenue
		total.TotalOutstanding += row.TotalOutstanding

		if row.TotalRevenue > maxRevenue {
			maxRevenue = row.TotalRevenue
			mostProfitable = row.AreaName
		}
		if row.TotalOutstanding > maxOutstanding {
			maxOutstanding = row.TotalOutstanding
			attentionNeeded = row.AreaName
		}
	}

	if total.CustomerCount > 0 {
		total.ARPU = total.TotalRevenue / int64(total.CustomerCount)
	}

	return &domain.RevenueByAreaReport{
		Areas:               areas,
		Total:               total,
		MostProfitableArea:  mostProfitable,
		AttentionNeededArea: attentionNeeded,
	}, nil
}
