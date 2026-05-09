package domain

import (
	"fmt"
	"math"
	"sort"
)

// =============================================================================
// Linear Regression - pure function untuk forecasting
// =============================================================================

// DataPoint merepresentasikan satu titik data untuk regresi linear.
type DataPoint struct {
	X float64 // index bulan (0, 1, 2, ...)
	Y float64 // nilai (revenue, jumlah pelanggan, dll)
}

// LinearRegressionResult berisi hasil regresi linear.
type LinearRegressionResult struct {
	Slope     float64 // kemiringan garis (trend per bulan)
	Intercept float64 // titik potong sumbu Y
	RSquared  float64 // koefisien determinasi (0-1)
}

// LinearRegression menghitung regresi linear sederhana dari titik data.
// Mengembalikan slope, intercept, dan R-squared.
// Invarian: Predict(result, x) == result.Slope * x + result.Intercept
// Invarian: len(points) >= 2 (minimal 2 titik data)
// Invarian: jika semua Y sama -> slope == 0
// Invarian: jika 2 titik -> R² == 1.0
func LinearRegression(points []DataPoint) LinearRegressionResult {
	n := float64(len(points))
	if n < 2 {
		return LinearRegressionResult{}
	}

	var sumX, sumY, sumXY, sumX2 float64
	for _, p := range points {
		sumX += p.X
		sumY += p.Y
		sumXY += p.X * p.Y
		sumX2 += p.X * p.X
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		// Semua X sama - tidak bisa menghitung slope
		return LinearRegressionResult{Intercept: sumY / n}
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / n

	// Hitung R-squared (koefisien determinasi)
	meanY := sumY / n
	var ssRes, ssTot float64
	for _, p := range points {
		predicted := slope*p.X + intercept
		ssRes += (p.Y - predicted) * (p.Y - predicted)
		ssTot += (p.Y - meanY) * (p.Y - meanY)
	}

	var rSquared float64
	if ssTot > 0 {
		rSquared = 1 - ssRes/ssTot
	}

	return LinearRegressionResult{
		Slope:     slope,
		Intercept: intercept,
		RSquared:  rSquared,
	}
}

// Predict menghitung nilai prediksi untuk x menggunakan hasil regresi.
// Invarian: Predict(result, x) == result.Slope * x + result.Intercept
func Predict(result LinearRegressionResult, x float64) float64 {
	return result.Slope*x + result.Intercept
}

// =============================================================================
// Comparison Delta - kalkulasi delta perbandingan periode
// =============================================================================

// CalculateComparisonDelta menghitung delta antara dua nilai.
// Mengembalikan delta absolut, persentase, dan trend.
// delta_absolute == baseValue - compareValue
// delta_percentage == (delta_absolute / |compareValue|) * 100 (atau 0 jika compareValue == 0)
// trend == "stable" jika |pct| < 1, "improving" jika pct > 0, "declining" jika pct < 0
func CalculateComparisonDelta(baseValue, compareValue float64) (deltaAbs, deltaPct float64, trend string) {
	deltaAbs = baseValue - compareValue

	if compareValue != 0 {
		deltaPct = (deltaAbs / math.Abs(compareValue)) * 100
	}

	switch {
	case math.Abs(deltaPct) < 1:
		trend = "stable"
	case deltaPct > 0:
		trend = "improving"
	default:
		trend = "declining"
	}
	return
}

// =============================================================================
// Insight Generation - buat insight otomatis dari metrik perbandingan
// =============================================================================

// GenerateInsights menghasilkan insight otomatis dari metrik perbandingan.
// Mengembalikan 3-5 insight diurutkan berdasarkan |delta_percentage| terbesar.
// Metrik dengan |delta_percentage| < 1% tidak menghasilkan insight.
func GenerateInsights(metrics []ComparisonMetric) []string {
	if len(metrics) == 0 {
		return nil
	}

	// Urutkan berdasarkan |delta_percentage| descending
	sorted := make([]ComparisonMetric, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		return math.Abs(sorted[i].DeltaPercentage) > math.Abs(sorted[j].DeltaPercentage)
	})

	var insights []string
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}

	for i := 0; i < limit; i++ {
		m := sorted[i]
		// Lewati metrik dengan delta kecil (< 1%)
		if math.Abs(m.DeltaPercentage) < 1 {
			continue
		}

		var direction string
		if m.DeltaPercentage > 0 {
			direction = "naik"
		} else {
			direction = "turun"
		}

		insight := m.MetricName + " " + direction + " " +
			formatPercentage(math.Abs(m.DeltaPercentage)) +
			" dibanding periode sebelumnya"
		insights = append(insights, insight)
	}
	return insights
}

// formatPercentage memformat persentase untuk ditampilkan di insight.
// Jika bilangan bulat, tampilkan tanpa desimal. Jika tidak, tampilkan 1 desimal.
func formatPercentage(pct float64) string {
	if pct == math.Trunc(pct) {
		return fmt.Sprintf("%.0f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}
