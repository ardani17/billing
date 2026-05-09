// report_manager.go berisi ReportManager struct dan constructor.
// ReportManager mengimplementasikan sebagian dari domain.ReportUsecase
// untuk laporan keuangan: revenue, aging, payment, voucher, profit-loss,
// dan revenue by area.
package usecase

import (
	"context"
	"math"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ReportManager mengimplementasikan business logic untuk laporan.
// Menangani laporan keuangan, pelanggan, jaringan, dan operasional.
type ReportManager struct {
	aggregationRepo domain.ReportAggregationRepository
	expenseRepo     domain.ExpenseRepository
	kpiTargetRepo   domain.KPITargetRepository
	networkClient   domain.NetworkServiceClient
	redisClient     *redis.Client
	exportManager   *ReportExportManager
	dashboardCache  *DashboardCache
	logger          zerolog.Logger
}

// NewReportManager membuat instance baru ReportManager.
func NewReportManager(
	aggregationRepo domain.ReportAggregationRepository,
	expenseRepo domain.ExpenseRepository,
	kpiTargetRepo domain.KPITargetRepository,
	networkClient domain.NetworkServiceClient,
	redisClient *redis.Client,
	logger zerolog.Logger,
) *ReportManager {
	return &ReportManager{
		aggregationRepo: aggregationRepo,
		expenseRepo:     expenseRepo,
		kpiTargetRepo:   kpiTargetRepo,
		networkClient:   networkClient,
		redisClient:     redisClient,
		logger:          logger.With().Str("component", "report_manager").Logger(),
	}
}

// SetExportManager mengkonfigurasi export manager untuk ReportManager.
// Dipanggil setelah konstruksi untuk menghubungkan job repo dan asynq client.
func (rm *ReportManager) SetExportManager(jobRepo domain.ReportJobRepository, queueClient *asynq.Client) {
	rm.exportManager = &ReportExportManager{
		jobRepo:     jobRepo,
		queueClient: queueClient,
	}
}

// SetDashboardCache mengkonfigurasi dashboard cache untuk ReportManager.
// Dipanggil setelah konstruksi untuk menghubungkan DashboardCache.
func (rm *ReportManager) SetDashboardCache(dc *DashboardCache) {
	rm.dashboardCache = dc
}

// GetDashboardData mengambil data ringkasan untuk dashboard widget.
// Delegasi ke DashboardCache untuk caching dan aggregasi.
func (rm *ReportManager) GetDashboardData(ctx context.Context, tenantID string) (*domain.DashboardData, error) {
	if rm.dashboardCache != nil {
		return rm.dashboardCache.GetDashboardData(ctx, tenantID)
	}
	// Cadangan: ambil langsung dari aggregation repo tanpa cache
	return rm.aggregationRepo.GetDashboardData(ctx, tenantID)
}

// GetRevenueReport mengambil laporan ringkasan pendapatan per sumber.
// Mendukung perbandingan periode (jika compare_start/end disediakan) dan KPI target.
func (rm *ReportManager) GetRevenueReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.RevenueReport, error) {
	// Ambil data pendapatan periode saat ini
	current, err := rm.aggregationRepo.GetRevenueSummary(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd, filter.AreaID, filter.PackageID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil revenue summary")
		return nil, err
	}

	report := &domain.RevenueReport{
		Current: *current,
	}

	// Ambil data perbandingan jika parameter compare disediakan
	if filter.CompareStart != nil && filter.CompareEnd != nil {
		comparison, err := rm.aggregationRepo.GetRevenueSummary(ctx, tenantID, *filter.CompareStart, *filter.CompareEnd, filter.AreaID, filter.PackageID)
		if err != nil {
			rm.logger.Warn().Err(err).Msg("gagal mengambil comparison revenue, lanjut tanpa perbandingan")
		} else {
			report.Comparison = comparison
			report.Delta = rm.calculateRevenueDelta(current, comparison)
		}
	}

	// Ambil trend 12 bulan terakhir
	trend, err := rm.aggregationRepo.GetMonthlyRevenueTrend(ctx, tenantID, 12)
	if err != nil {
		rm.logger.Warn().Err(err).Msg("gagal mengambil revenue trend")
	} else {
		report.Trend = trend
	}

	// Tambahkan KPI target jika tersedia
	rm.attachRevenueKPI(ctx, tenantID, report)

	return report, nil
}

// attachRevenueKPI menambahkan KPI target dan progress ke revenue report.
func (rm *ReportManager) attachRevenueKPI(ctx context.Context, tenantID string, report *domain.RevenueReport) {
	kpi, err := rm.kpiTargetRepo.GetByTenant(ctx, tenantID)
	if err != nil || kpi == nil || kpi.MonthlyRevenueTarget == nil {
		return
	}
	report.KPITarget = kpi.MonthlyRevenueTarget
	if *kpi.MonthlyRevenueTarget > 0 {
		progress := float64(report.Current.Total) / float64(*kpi.MonthlyRevenueTarget) * 100
		report.KPIProgress = &progress
	}
}

// calculateRevenueDelta menghitung delta antara dua periode pendapatan.
func (rm *ReportManager) calculateRevenueDelta(current, comparison *domain.RevenueSource) map[string]domain.RevenueDelta {
	delta := make(map[string]domain.RevenueDelta)
	delta["total"] = calcDelta(current.Total, comparison.Total)
	delta["monthly_subscription"] = calcDelta(current.MonthlySubscription, comparison.MonthlySubscription)
	delta["voucher_sales"] = calcDelta(current.VoucherSales, comparison.VoucherSales)
	delta["installation_fees"] = calcDelta(current.InstallationFees, comparison.InstallationFees)
	delta["late_fees"] = calcDelta(current.LateFees, comparison.LateFees)
	delta["other"] = calcDelta(current.Other, comparison.Other)
	return delta
}

// calcDelta menghitung delta absolut dan persentase antara dua nilai.
func calcDelta(current, compare int64) domain.RevenueDelta {
	abs := current - compare
	var pct float64
	if compare != 0 {
		pct = float64(abs) / math.Abs(float64(compare)) * 100
	}
	return domain.RevenueDelta{Absolute: abs, Percentage: pct}
}

// GetAgingReport mengambil laporan piutang/aging dengan bucket, collection rate,
// top debtors, trend, dan KPI target.
func (rm *ReportManager) GetAgingReport(ctx context.Context, tenantID string, periodEnd time.Time, areaID, packageID string) (*domain.AgingReport, error) {
	report, err := rm.aggregationRepo.GetAgingReport(ctx, tenantID, periodEnd, areaID, packageID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil aging report")
		return nil, err
	}

	// Tambahkan KPI target collection rate jika tersedia
	kpi, err := rm.kpiTargetRepo.GetByTenant(ctx, tenantID)
	if err == nil && kpi != nil && kpi.CollectionRateTarget != nil {
		report.KPITarget = kpi.CollectionRateTarget
	}

	return report, nil
}

// GetPaymentReport mengambil laporan distribusi pembayaran per metode,
// pembayaran harian, dan peak payment date.
func (rm *ReportManager) GetPaymentReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.PaymentReport, error) {
	report, err := rm.aggregationRepo.GetPaymentDistribution(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd, filter.AreaID, filter.PackageID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil payment report")
		return nil, err
	}
	return report, nil
}

// GetVoucherRevenueReport mengambil laporan pendapatan voucher per paket dan reseller.
func (rm *ReportManager) GetVoucherRevenueReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.VoucherRevenueReport, error) {
	report, err := rm.aggregationRepo.GetVoucherRevenue(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil voucher revenue report")
		return nil, err
	}
	return report, nil
}
