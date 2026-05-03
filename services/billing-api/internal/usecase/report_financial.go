// report_financial.go berisi methods ReportManager untuk laporan laba rugi
// dan pendapatan per area. Dipisah dari report_manager.go agar tidak
// melebihi batas 200 baris per file.
package usecase

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetProfitLossReport mengambil laporan laba rugi sederhana.
// Revenue diambil dari aggregation repo, expenses dari expense repo (SumByCategory).
// Mendukung perbandingan periode jika compare_start/end disediakan.
func (rm *ReportManager) GetProfitLossReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.ProfitLossReport, error) {
	report, err := rm.buildProfitLoss(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd, filter.AreaID, filter.PackageID)
	if err != nil {
		return nil, err
	}

	// Ambil data perbandingan jika parameter compare disediakan
	if filter.CompareStart != nil && filter.CompareEnd != nil {
		comparison, err := rm.buildProfitLoss(ctx, tenantID, *filter.CompareStart, *filter.CompareEnd, filter.AreaID, filter.PackageID)
		if err != nil {
			rm.logger.Warn().Err(err).Msg("gagal mengambil comparison profit-loss, lanjut tanpa perbandingan")
		} else {
			report.Comparison = comparison
		}
	}

	return report, nil
}

// buildProfitLoss membangun laporan laba rugi untuk satu periode.
// Menggabungkan revenue dari billing dan expenses dari input manual.
func (rm *ReportManager) buildProfitLoss(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, areaID, packageID string) (*domain.ProfitLossReport, error) {
	// Ambil revenue dari aggregation repo
	revenue, err := rm.aggregationRepo.GetRevenueSummary(ctx, tenantID, periodStart, periodEnd, areaID, packageID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil revenue untuk profit-loss")
		return nil, err
	}

	// Bangun revenue line items
	revenueItems := []domain.ProfitLossLineItem{
		{Label: "Tagihan Bulanan", Amount: revenue.MonthlySubscription},
		{Label: "Penjualan Voucher", Amount: revenue.VoucherSales},
		{Label: "Biaya Pasang", Amount: revenue.InstallationFees},
		{Label: "Denda Keterlambatan", Amount: revenue.LateFees},
		{Label: "Lainnya", Amount: revenue.Other},
	}

	// Ambil expenses dari expense repo (grouped by category)
	expenseItems, err := rm.expenseRepo.SumByCategory(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		rm.logger.Warn().Err(err).Msg("gagal mengambil expenses, lanjut dengan expenses kosong")
		expenseItems = nil
	}

	// Hitung total expenses
	var totalExpenses int64
	for _, item := range expenseItems {
		totalExpenses += item.Amount
	}

	// Hitung net profit dan profit margin
	netProfit := revenue.Total - totalExpenses
	var profitMargin float64
	if revenue.Total > 0 {
		profitMargin = float64(netProfit) / float64(revenue.Total) * 100
	}

	return &domain.ProfitLossReport{
		RevenueItems:  revenueItems,
		TotalRevenue:  revenue.Total,
		ExpenseItems:  expenseItems,
		TotalExpenses: totalExpenses,
		NetProfit:     netProfit,
		ProfitMargin:  profitMargin,
	}, nil
}

// GetRevenueByAreaReport mengambil laporan pendapatan per area.
// Mengidentifikasi area paling menguntungkan dan area yang perlu perhatian.
func (rm *ReportManager) GetRevenueByAreaReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.RevenueByAreaReport, error) {
	report, err := rm.aggregationRepo.GetRevenueByArea(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil revenue by area report")
		return nil, err
	}
	return report, nil
}
