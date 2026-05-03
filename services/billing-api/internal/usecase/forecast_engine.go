// forecast_engine.go berisi ForecastEngine yang mengimplementasikan
// domain.ForecastUsecase untuk proyeksi bisnis menggunakan linear regression.
// Mengambil 6 bulan data historis → regresi linear → 3 bulan proyeksi.
package usecase

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// minHistoryMonths adalah jumlah minimum bulan data historis untuk proyeksi.
const minHistoryMonths = 3

// historyMonths adalah jumlah bulan data historis yang diambil.
const historyMonths = 6

// projectionMonths adalah jumlah bulan proyeksi ke depan.
const projectionMonths = 3

// ForecastEngine mengimplementasikan business logic untuk proyeksi bisnis.
type ForecastEngine struct {
	aggregationRepo domain.ReportAggregationRepository
	kpiTargetRepo   domain.KPITargetRepository
	logger          zerolog.Logger
}

// NewForecastEngine membuat instance baru ForecastEngine.
func NewForecastEngine(
	aggregationRepo domain.ReportAggregationRepository,
	kpiTargetRepo domain.KPITargetRepository,
	logger zerolog.Logger,
) *ForecastEngine {
	return &ForecastEngine{
		aggregationRepo: aggregationRepo,
		kpiTargetRepo:   kpiTargetRepo,
		logger:          logger.With().Str("component", "forecast_engine").Logger(),
	}
}

// GetForecastReport mengambil proyeksi 3 bulan ke depan berdasarkan data historis.
// Flow: ambil 6 bulan data → cek minimum 3 bulan → linear regression →
// generate proyeksi → hitung estimated target date → tambahkan disclaimer.
func (fe *ForecastEngine) GetForecastReport(ctx context.Context, tenantID string) (*domain.ForecastReport, error) {
	// Ambil data historis revenue, customers, dan receivables
	revenueHistory, err := fe.aggregationRepo.GetMonthlyRevenueHistory(ctx, tenantID, historyMonths)
	if err != nil {
		fe.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil revenue history")
		return nil, err
	}

	// Cek apakah data cukup untuk proyeksi
	if len(revenueHistory) < minHistoryMonths {
		return &domain.ForecastReport{
			InsufficientData: true,
			Disclaimer:       "Data historis belum cukup (minimal 3 bulan) untuk menghasilkan proyeksi.",
		}, nil
	}

	customerHistory, err := fe.aggregationRepo.GetMonthlyCustomerHistory(ctx, tenantID, historyMonths)
	if err != nil {
		fe.logger.Warn().Err(err).Msg("gagal mengambil customer history, lanjut tanpa proyeksi pelanggan")
	}

	receivablesHistory, err := fe.aggregationRepo.GetMonthlyReceivablesHistory(ctx, tenantID, historyMonths)
	if err != nil {
		fe.logger.Warn().Err(err).Msg("gagal mengambil receivables history, lanjut tanpa proyeksi piutang")
	}

	// Jalankan linear regression untuk setiap metrik
	revenueReg := domain.LinearRegression(revenueHistory)
	customerReg := domain.LinearRegression(customerHistory)
	receivablesReg := domain.LinearRegression(receivablesHistory)

	// Generate proyeksi 3 bulan ke depan
	now := time.Now()
	n := float64(len(revenueHistory))
	projections := make([]domain.ForecastMonth, projectionMonths)

	for i := 0; i < projectionMonths; i++ {
		futureMonth := now.AddDate(0, i+1, 0)
		x := n + float64(i)

		projectedRevenue := int64(math.Max(0, domain.Predict(revenueReg, x)))
		projectedCustomers := int(math.Max(0, domain.Predict(customerReg, x)))
		projectedReceivables := int64(math.Max(0, domain.Predict(receivablesReg, x)))

		projections[i] = domain.ForecastMonth{
			Month:                futureMonth.Format("2006-01"),
			ProjectedRevenue:     projectedRevenue,
			ProjectedCustomers:   projectedCustomers,
			ProjectedReceivables: projectedReceivables,
		}
	}

	report := &domain.ForecastReport{
		Projections: projections,
		Disclaimer:  "Proyeksi berdasarkan tren historis menggunakan regresi linear sederhana. Hasil aktual dapat berbeda.",
	}

	// Hitung estimated target date dari KPI jika tersedia
	fe.attachTargetEstimates(ctx, tenantID, report, revenueReg, customerReg, n)

	return report, nil
}

// attachTargetEstimates menghitung perkiraan tanggal pencapaian KPI target.
func (fe *ForecastEngine) attachTargetEstimates(ctx context.Context, tenantID string, report *domain.ForecastReport, revenueReg, customerReg domain.LinearRegressionResult, n float64) {
	kpi, err := fe.kpiTargetRepo.GetByTenant(ctx, tenantID)
	if err != nil || kpi == nil {
		return
	}

	estimates := make(map[string]string)
	now := time.Now()

	// Estimasi pencapaian target revenue bulanan
	if kpi.MonthlyRevenueTarget != nil && revenueReg.Slope > 0 {
		target := float64(*kpi.MonthlyRevenueTarget)
		// Solve: slope * x + intercept = target → x = (target - intercept) / slope
		x := (target - revenueReg.Intercept) / revenueReg.Slope
		monthsAhead := x - n
		if monthsAhead > 0 && monthsAhead <= 24 {
			estDate := now.AddDate(0, int(math.Ceil(monthsAhead)), 0)
			estimates["monthly_revenue"] = estDate.Format("2006-01")
		}
	}

	// Estimasi pencapaian target total pelanggan
	if kpi.TotalCustomersTarget != nil && customerReg.Slope > 0 {
		target := float64(*kpi.TotalCustomersTarget)
		x := (target - customerReg.Intercept) / customerReg.Slope
		monthsAhead := x - n
		if monthsAhead > 0 && monthsAhead <= 24 {
			estDate := now.AddDate(0, int(math.Ceil(monthsAhead)), 0)
			estimates["total_customers"] = estDate.Format("2006-01")
		}
	}

	if len(estimates) > 0 {
		report.EstimatedTargetDate = estimates
	}
}
