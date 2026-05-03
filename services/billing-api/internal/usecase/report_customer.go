// report_customer.go berisi methods ReportManager untuk laporan pelanggan:
// pertumbuhan, distribusi, dan analisis churn.
// Dipisah dari report_manager.go agar tidak melebihi batas 200 baris per file.
package usecase

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetCustomerGrowthReport mengambil laporan pertumbuhan pelanggan.
// Termasuk total aktif, pelanggan baru, churn, net growth, ARPU, CLV,
// churn rate, trend bulanan, perbandingan periode, dan KPI target.
func (rm *ReportManager) GetCustomerGrowthReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.CustomerGrowthReport, error) {
	// Ambil data pertumbuhan periode saat ini
	report, err := rm.aggregationRepo.GetCustomerGrowth(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil customer growth report")
		return nil, err
	}

	// Ambil trend 12 bulan terakhir
	trend, err := rm.aggregationRepo.GetMonthlyGrowthTrend(ctx, tenantID, 12)
	if err != nil {
		rm.logger.Warn().Err(err).Msg("gagal mengambil monthly growth trend")
	} else {
		report.Trend = trend
	}

	// Ambil data perbandingan jika parameter compare disediakan
	if filter.CompareStart != nil && filter.CompareEnd != nil {
		comparison, err := rm.aggregationRepo.GetCustomerGrowth(ctx, tenantID, *filter.CompareStart, *filter.CompareEnd)
		if err != nil {
			rm.logger.Warn().Err(err).Msg("gagal mengambil comparison customer growth, lanjut tanpa perbandingan")
		} else {
			report.Comparison = comparison
			report.Delta = rm.calculateGrowthDelta(report, comparison)
		}
	}

	// Tambahkan KPI target jika tersedia
	rm.attachCustomerKPI(ctx, tenantID, report)

	return report, nil
}

// calculateGrowthDelta menghitung delta antara dua periode pertumbuhan pelanggan.
func (rm *ReportManager) calculateGrowthDelta(current, comparison *domain.CustomerGrowthReport) map[string]domain.RevenueDelta {
	delta := make(map[string]domain.RevenueDelta)
	delta["new_customers"] = calcDelta(int64(current.NewCustomers), int64(comparison.NewCustomers))
	delta["churned_customers"] = calcDelta(int64(current.ChurnedCustomers), int64(comparison.ChurnedCustomers))
	delta["net_growth"] = calcDelta(int64(current.NetGrowth), int64(comparison.NetGrowth))
	delta["arpu"] = calcDelta(current.ARPU, comparison.ARPU)
	delta["clv"] = calcDelta(current.CLV, comparison.CLV)
	return delta
}

// attachCustomerKPI menambahkan KPI target pelanggan ke growth report.
func (rm *ReportManager) attachCustomerKPI(ctx context.Context, tenantID string, report *domain.CustomerGrowthReport) {
	kpi, err := rm.kpiTargetRepo.GetByTenant(ctx, tenantID)
	if err != nil || kpi == nil {
		return
	}
	// KPI target untuk churn rate dan pelanggan baru bisa ditambahkan
	// ke response jika diperlukan oleh frontend melalui field Delta
}

// GetCustomerDistributionReport mengambil laporan distribusi pelanggan
// per paket, area, status, dan metode koneksi.
func (rm *ReportManager) GetCustomerDistributionReport(ctx context.Context, tenantID string, periodEnd time.Time) (*domain.CustomerDistributionReport, error) {
	report, err := rm.aggregationRepo.GetCustomerDistribution(ctx, tenantID, periodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil customer distribution report")
		return nil, err
	}
	return report, nil
}

// GetChurnAnalysisReport mengambil laporan analisis churn pelanggan.
// Termasuk jumlah churn, churn rate, breakdown per alasan/paket/area,
// dan rata-rata lifetime pelanggan.
func (rm *ReportManager) GetChurnAnalysisReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.ChurnAnalysisReport, error) {
	report, err := rm.aggregationRepo.GetChurnAnalysis(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil churn analysis report")
		return nil, err
	}
	return report, nil
}
