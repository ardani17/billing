// comparison_engine.go berisi ComparisonEngine yang mengimplementasikan
// domain.ComparisonUsecase untuk perbandingan metrik antar periode.
// Mendukung MoM, YoY, QoQ, dan kustom period comparison.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ComparisonEngine mengimplementasikan business logic untuk perbandingan periode.
type ComparisonEngine struct {
	aggregationRepo domain.ReportAggregationRepository
	kpiTargetRepo   domain.KPITargetRepository
	logger          zerolog.Logger
}

// NewComparisonEngine membuat instance baru ComparisonEngine.
func NewComparisonEngine(
	aggregationRepo domain.ReportAggregationRepository,
	kpiTargetRepo domain.KPITargetRepository,
	logger zerolog.Logger,
) *ComparisonEngine {
	return &ComparisonEngine{
		aggregationRepo: aggregationRepo,
		kpiTargetRepo:   kpiTargetRepo,
		logger:          logger.With().Str("component", "comparison_engine").Logger(),
	}
}

// GetComparisonReport mengambil laporan perbandingan metrik antara dua periode.
// Alur: tentukan comparison period -> kueri metrik -> hitung delta -> buat insights.
func (ce *ComparisonEngine) GetComparisonReport(ctx context.Context, tenantID string, compType domain.ComparisonType, basePeriodStart, basePeriodEnd time.Time, comparePeriodStart, comparePeriodEnd *time.Time) (*domain.ComparisonReport, error) {
	// Tentukan periode perbandingan berdasarkan tipe
	compStart, compEnd := ce.resolveComparisonPeriod(compType, basePeriodStart, basePeriodEnd, comparePeriodStart, comparePeriodEnd)

	// Kueri metrik untuk periode base
	baseRevenue, err := ce.aggregationRepo.GetRevenueSummary(ctx, tenantID, basePeriodStart, basePeriodEnd, "", "")
	if err != nil {
		ce.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil base revenue")
		return nil, err
	}

	baseGrowth, err := ce.aggregationRepo.GetCustomerGrowth(ctx, tenantID, basePeriodStart, basePeriodEnd)
	if err != nil {
		ce.logger.Warn().Err(err).Msg("gagal mengambil base customer growth")
	}

	// Kueri metrik untuk periode comparison
	compRevenue, err := ce.aggregationRepo.GetRevenueSummary(ctx, tenantID, compStart, compEnd, "", "")
	if err != nil {
		ce.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil comparison revenue")
		return nil, err
	}

	compGrowth, err := ce.aggregationRepo.GetCustomerGrowth(ctx, tenantID, compStart, compEnd)
	if err != nil {
		ce.logger.Warn().Err(err).Msg("gagal mengambil comparison customer growth")
	}

	// Hitung delta untuk setiap metrik menggunakan domain.CalculateComparisonDelta
	var metrics []domain.ComparisonMetric

	// Revenue metrics
	metrics = append(metrics, ce.buildMetric("Total Pendapatan", float64(baseRevenue.Total), float64(compRevenue.Total)))
	metrics = append(metrics, ce.buildMetric("Tagihan Bulanan", float64(baseRevenue.MonthlySubscription), float64(compRevenue.MonthlySubscription)))
	metrics = append(metrics, ce.buildMetric("Penjualan Voucher", float64(baseRevenue.VoucherSales), float64(compRevenue.VoucherSales)))

	// Customer metrics (jika tersedia)
	if baseGrowth != nil && compGrowth != nil {
		metrics = append(metrics, ce.buildMetric("Pelanggan Baru", float64(baseGrowth.NewCustomers), float64(compGrowth.NewCustomers)))
		metrics = append(metrics, ce.buildMetric("Pelanggan Churn", float64(baseGrowth.ChurnedCustomers), float64(compGrowth.ChurnedCustomers)))
		metrics = append(metrics, ce.buildMetric("Net Growth", float64(baseGrowth.NetGrowth), float64(compGrowth.NetGrowth)))
		metrics = append(metrics, ce.buildMetric("ARPU", float64(baseGrowth.ARPU), float64(compGrowth.ARPU)))
	}

	// Buat insights otomatis
	insights := domain.GenerateInsights(metrics)

	return &domain.ComparisonReport{
		ComparisonType: compType,
		BasePeriod:     basePeriodStart.Format("2006-01-02") + " s/d " + basePeriodEnd.Format("2006-01-02"),
		ComparePeriod:  compStart.Format("2006-01-02") + " s/d " + compEnd.Format("2006-01-02"),
		Metrics:        metrics,
		Insights:       insights,
	}, nil
}

// resolveComparisonPeriod menentukan periode perbandingan berdasarkan tipe.
func (ce *ComparisonEngine) resolveComparisonPeriod(compType domain.ComparisonType, baseStart, baseEnd time.Time, customStart, customEnd *time.Time) (time.Time, time.Time) {
	switch compType {
	case domain.ComparisonMoM:
		// Bulan sebelumnya
		return baseStart.AddDate(0, -1, 0), baseEnd.AddDate(0, -1, 0)
	case domain.ComparisonYoY:
		// Tahun sebelumnya
		return baseStart.AddDate(-1, 0, 0), baseEnd.AddDate(-1, 0, 0)
	case domain.ComparisonQoQ:
		// Kuartal sebelumnya (3 bulan)
		return baseStart.AddDate(0, -3, 0), baseEnd.AddDate(0, -3, 0)
	case domain.ComparisonCustom:
		// Periode kustom dari parameter
		if customStart != nil && customEnd != nil {
			return *customStart, *customEnd
		}
		// Cadangan ke bulan sebelumnya jika kustom period tidak disediakan
		return baseStart.AddDate(0, -1, 0), baseEnd.AddDate(0, -1, 0)
	default:
		return baseStart.AddDate(0, -1, 0), baseEnd.AddDate(0, -1, 0)
	}
}

// buildMetric membangun ComparisonMetric dari dua nilai menggunakan domain helper.
func (ce *ComparisonEngine) buildMetric(name string, baseValue, compareValue float64) domain.ComparisonMetric {
	deltaAbs, deltaPct, trend := domain.CalculateComparisonDelta(baseValue, compareValue)
	return domain.ComparisonMetric{
		MetricName:      name,
		BaseValue:       baseValue,
		CompareValue:    compareValue,
		DeltaAbsolute:   deltaAbs,
		DeltaPercentage: deltaPct,
		Trend:           trend,
	}
}
