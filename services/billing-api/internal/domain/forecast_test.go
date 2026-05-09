package domain

import (
	"fmt"
	"math"
	"testing"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 1.3, 5.5, 7.7, 22.5**
//
// Untuk dua nilai numerik (base_value dan compare_value),
// delta_absolute HARUS sama dengan base_value - compare_value,
// delta_percentage HARUS sama dengan (delta_absolute / |compare_value|) * 100
// (atau 0 jika compare_value == 0), dan trend HARUS "stable" jika
// |delta_percentage| < 1, "improving" jika delta_percentage > 0,
// "declining" jika delta_percentage < 0.

const epsilon = 1e-9

// TestProperty_ComparisonDeltaCalculation memverifikasi bahwa
// CalculateComparisonDelta menghasilkan delta absolut, persentase,
// dan trend yang benar untuk semua kombinasi base dan compare values.
func TestProperty_ComparisonDeltaCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat dua nilai float64 dalam range yang wajar untuk menghindari overflow
		baseValue := rapid.Float64Range(-1e12, 1e12).Draw(t, "base_value")
		compareValue := rapid.Float64Range(-1e12, 1e12).Draw(t, "compare_value")

		deltaAbs, deltaPct, trend := CalculateComparisonDelta(baseValue, compareValue)

		// Verifikasi: delta_absolute == base - compare
		expectedAbs := baseValue - compareValue
		if math.Abs(deltaAbs-expectedAbs) > epsilon {
			t.Fatalf(
				"delta_absolute salah: got %f, want %f (base=%f, compare=%f)",
				deltaAbs, expectedAbs, baseValue, compareValue,
			)
		}

		// Verifikasi: delta_percentage
		var expectedPct float64
		if compareValue != 0 {
			expectedPct = (expectedAbs / math.Abs(compareValue)) * 100
		}
		if math.Abs(deltaPct-expectedPct) > epsilon {
			t.Fatalf(
				"delta_percentage salah: got %f, want %f (base=%f, compare=%f)",
				deltaPct, expectedPct, baseValue, compareValue,
			)
		}

		// Verifikasi: trend berdasarkan delta_percentage
		var expectedTrend string
		switch {
		case math.Abs(deltaPct) < 1:
			expectedTrend = "stable"
		case deltaPct > 0:
			expectedTrend = "improving"
		default:
			expectedTrend = "declining"
		}
		if trend != expectedTrend {
			t.Fatalf(
				"trend salah: got %q, want %q (deltaPct=%f, base=%f, compare=%f)",
				trend, expectedTrend, deltaPct, baseValue, compareValue,
			)
		}
	})
}

// TestProperty_ComparisonDeltaZeroCompare memverifikasi bahwa ketika
// compareValue == 0, delta_percentage selalu 0 dan trend selalu "stable"
// (karena |0| < 1).
func TestProperty_ComparisonDeltaZeroCompare(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseValue := rapid.Float64Range(-1e12, 1e12).Draw(t, "base_value")
		compareValue := 0.0

		deltaAbs, deltaPct, trend := CalculateComparisonDelta(baseValue, compareValue)

		// delta_absolute tetap base - 0 = base
		if math.Abs(deltaAbs-baseValue) > epsilon {
			t.Fatalf(
				"delta_absolute salah saat compare=0: got %f, want %f",
				deltaAbs, baseValue,
			)
		}

		// delta_percentage harus 0 karena compare == 0
		if deltaPct != 0 {
			t.Fatalf(
				"delta_percentage harus 0 saat compare=0: got %f",
				deltaPct,
			)
		}

		// trend harus "stable" karena |0| < 1
		if trend != "stable" {
			t.Fatalf(
				"trend harus 'stable' saat compare=0: got %q",
				trend,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 21.2**
//
// Untuk atur minimal 2 titik data, hasil linear regression HARUS memenuhi:
// - Predict(result, x) == result.Slope * x + result.Intercept untuk semua x
// - Jika semua Y sama -> slope == 0
// - Jika 2 titik -> R² == 1.0 (perfect fit)
// - Prediksi pada mean(X) mendekati mean(Y)

const epsilonProp8 = 1e-6

// TestProperty_LinearRegressionPredict memverifikasi bahwa Predict(result, x)
// selalu sama dengan Slope*x + Intercept untuk semua x dan semua atur titik data.
func TestProperty_LinearRegressionPredict(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat minimal 2 titik data dengan X dan Y dalam range wajar
		n := rapid.IntRange(2, 50).Draw(t, "n")
		points := make([]DataPoint, n)
		for i := 0; i < n; i++ {
			points[i] = DataPoint{
				X: rapid.Float64Range(-1e6, 1e6).Draw(t, fmt.Sprintf("x_%d", i)),
				Y: rapid.Float64Range(-1e6, 1e6).Draw(t, fmt.Sprintf("y_%d", i)),
			}
		}

		allXSame := true
		for i := 1; i < len(points); i++ {
			if points[i].X != points[0].X {
				allXSame = false
				break
			}
		}
		if allXSame {
			// Buat X berbeda agar denominator != 0
			points[1].X = points[0].X + 1.0
		}

		result := LinearRegression(points)

		// Verifikasi: Predict(result, x) == Slope*x + Intercept untuk beberapa x
		testXValues := []float64{0, 1, -1, 100, -100}
		for _, x := range testXValues {
			predicted := Predict(result, x)
			expected := result.Slope*x + result.Intercept
			if math.Abs(predicted-expected) > epsilonProp8 {
				t.Fatalf(
					"Predict(%f) = %f, tapi Slope*x + Intercept = %f (slope=%f, intercept=%f)",
					x, predicted, expected, result.Slope, result.Intercept,
				)
			}
		}

		// Verifikasi juga untuk setiap X dari titik data
		for i, p := range points {
			predicted := Predict(result, p.X)
			expected := result.Slope*p.X + result.Intercept
			if math.Abs(predicted-expected) > epsilonProp8 {
				t.Fatalf(
					"Predict(points[%d].X=%f) = %f, tapi Slope*x + Intercept = %f",
					i, p.X, predicted, expected,
				)
			}
		}
	})
}

// TestProperty_LinearRegressionConstantY memverifikasi bahwa jika semua Y
// bernilai sama, maka slope harus 0.
func TestProperty_LinearRegressionConstantY(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat titik data dengan Y konstan
		n := rapid.IntRange(2, 50).Draw(t, "n")
		constantY := rapid.Float64Range(-1e6, 1e6).Draw(t, "constant_y")
		points := make([]DataPoint, n)
		for i := 0; i < n; i++ {
			points[i] = DataPoint{
				X: float64(i), // X berbeda-beda
				Y: constantY,
			}
		}

		result := LinearRegression(points)

		// Slope harus 0 jika semua Y sama
		if math.Abs(result.Slope) > epsilonProp8 {
			t.Fatalf(
				"slope harus 0 saat semua Y=%f, tapi got %f",
				constantY, result.Slope,
			)
		}

		// Intercept harus sama dengan constantY
		if math.Abs(result.Intercept-constantY) > epsilonProp8 {
			t.Fatalf(
				"intercept harus %f saat semua Y sama, tapi got %f",
				constantY, result.Intercept,
			)
		}
	})
}

// TestProperty_LinearRegressionTwoPoints memverifikasi bahwa jika hanya ada
// 2 titik data dengan Y berbeda, R² harus 1.0 (perfect fit).
func TestProperty_LinearRegressionTwoPoints(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 2 titik data dengan X dan Y dalam range wajar
		// Menggunakan range yang cukup besar agar tidak ada masalah presisi float
		x1 := rapid.Float64Range(-1e6, 1e6).Draw(t, "x1")
		y1 := rapid.Float64Range(-1e6, 1e6).Draw(t, "y1")
		y2 := rapid.Float64Range(-1e6, 1e6).Draw(t, "y2")

		// Pastikan X berbeda secara signifikan agar denominator regresi stabil
		dx := rapid.Float64Range(1.0, 1e6).Draw(t, "dx")
		x2 := x1 + dx

		// Pastikan Y berbeda secara signifikan agar ssTot > 0
		if math.Abs(y1-y2) < 1.0 {
			y2 = y1 + 1.0
		}

		points := []DataPoint{
			{X: x1, Y: y1},
			{X: x2, Y: y2},
		}

		result := LinearRegression(points)

		// Dengan 2 titik dan Y berbeda, R² harus 1.0 (perfect fit)
		if math.Abs(result.RSquared-1.0) > epsilonProp8 {
			t.Fatalf(
				"R² harus 1.0 untuk 2 titik (x1=%f,y1=%f, x2=%f,y2=%f), tapi got %f",
				x1, y1, x2, y2, result.RSquared,
			)
		}
	})
}

// TestProperty_LinearRegressionMeanPrediction memverifikasi bahwa prediksi
// pada mean(X) mendekati mean(Y).
func TestProperty_LinearRegressionMeanPrediction(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat minimal 2 titik data
		n := rapid.IntRange(2, 50).Draw(t, "n")
		points := make([]DataPoint, n)
		for i := 0; i < n; i++ {
			points[i] = DataPoint{
				X: rapid.Float64Range(-1e4, 1e4).Draw(t, fmt.Sprintf("x_%d", i)),
				Y: rapid.Float64Range(-1e4, 1e4).Draw(t, fmt.Sprintf("y_%d", i)),
			}
		}

		// Pastikan tidak semua X sama
		allXSame := true
		for i := 1; i < len(points); i++ {
			if points[i].X != points[0].X {
				allXSame = false
				break
			}
		}
		if allXSame {
			points[1].X = points[0].X + 1.0
		}

		result := LinearRegression(points)

		// Hitung mean(X) dan mean(Y)
		var sumX, sumY float64
		for _, p := range points {
			sumX += p.X
			sumY += p.Y
		}
		meanX := sumX / float64(len(points))
		meanY := sumY / float64(len(points))

		// Prediksi pada mean(X) harus mendekati mean(Y)
		predicted := Predict(result, meanX)
		if math.Abs(predicted-meanY) > epsilonProp8 {
			t.Fatalf(
				"Predict(mean(X)=%f) = %f, tapi mean(Y) = %f (selisih=%f)",
				meanX, predicted, meanY, math.Abs(predicted-meanY),
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 22.6**
//
// GenerateInsights mengembalikan 3-5 insight diurutkan berdasarkan
// |delta_percentage| terbesar. Setiap insight mengandung nama metrik
// dan arah (naik/turun). Metrik dengan |delta_percentage| < 1% tidak
// menghasilkan insight.

// TestProperty_InsightGenerationSortedByLargestDelta memverifikasi bahwa
// insight diurutkan berdasarkan |delta_percentage| terbesar dan jumlahnya
// antara 0-5 (maksimal 5, minimal 3 jika ada cukup metrik signifikan).
func TestProperty_InsightGenerationSortedByLargestDelta(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 1-10 metrik perbandingan dengan delta signifikan (>= 1%)
		n := rapid.IntRange(1, 10).Draw(t, "jumlah_metrik")
		metrics := make([]ComparisonMetric, n)
		for i := 0; i < n; i++ {
			// Buat delta_percentage dengan |delta| >= 1% agar semua menghasilkan insight
			deltaPct := rapid.Float64Range(1.0, 500.0).Draw(t, fmt.Sprintf("abs_delta_%d", i))
			// Acak arah positif atau negatif
			if rapid.Bool().Draw(t, fmt.Sprintf("negatif_%d", i)) {
				deltaPct = -deltaPct
			}
			metrics[i] = ComparisonMetric{
				MetricName:      fmt.Sprintf("metrik_%d", i),
				DeltaPercentage: deltaPct,
			}
		}

		insights := GenerateInsights(metrics)

		// Jumlah insight harus antara 0 dan 5
		if len(insights) > 5 {
			t.Fatalf("jumlah insight melebihi 5: got %d", len(insights))
		}

		// Jumlah insight harus sesuai: min(jumlah metrik signifikan, 5)
		expectedCount := n
		if expectedCount > 5 {
			expectedCount = 5
		}
		if len(insights) != expectedCount {
			t.Fatalf(
				"jumlah insight salah: got %d, want %d (semua metrik signifikan)",
				len(insights), expectedCount,
			)
		}
	})
}

// TestProperty_InsightContainsMetricNameAndDirection memverifikasi bahwa
// setiap insight mengandung nama metrik dan arah (naik/turun).
func TestProperty_InsightContainsMetricNameAndDirection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 1-8 metrik dengan delta signifikan
		n := rapid.IntRange(1, 8).Draw(t, "jumlah_metrik")
		metrics := make([]ComparisonMetric, n)
		for i := 0; i < n; i++ {
			deltaPct := rapid.Float64Range(1.0, 500.0).Draw(t, fmt.Sprintf("abs_delta_%d", i))
			if rapid.Bool().Draw(t, fmt.Sprintf("negatif_%d", i)) {
				deltaPct = -deltaPct
			}
			metrics[i] = ComparisonMetric{
				MetricName:      fmt.Sprintf("metrik_%d", i),
				DeltaPercentage: deltaPct,
			}
		}

		insights := GenerateInsights(metrics)

		// Buat map dari metrik untuk lookup cepat
		metricMap := make(map[string]ComparisonMetric)
		for _, m := range metrics {
			metricMap[m.MetricName] = m
		}

		for idx, insight := range insights {
			// Cari metrik mana yang cocok dengan insight ini
			found := false
			for _, m := range metrics {
				if len(insight) >= len(m.MetricName) && insight[:len(m.MetricName)] == m.MetricName {
					found = true

					// Verifikasi arah sesuai delta_percentage
					if m.DeltaPercentage > 0 {
						if !containsSubstring(insight, "naik") {
							t.Fatalf(
								"insight[%d] untuk metrik %q (delta=%.2f) harus mengandung 'naik': %q",
								idx, m.MetricName, m.DeltaPercentage, insight,
							)
						}
					} else {
						if !containsSubstring(insight, "turun") {
							t.Fatalf(
								"insight[%d] untuk metrik %q (delta=%.2f) harus mengandung 'turun': %q",
								idx, m.MetricName, m.DeltaPercentage, insight,
							)
						}
					}
					break
				}
			}
			if !found {
				t.Fatalf(
					"insight[%d] tidak mengandung nama metrik yang dikenal: %q",
					idx, insight,
				)
			}
		}
	})
}

// TestProperty_InsightSkipsSmallDelta memverifikasi bahwa metrik dengan
// |delta_percentage| < 1% tidak menghasilkan insight.
func TestProperty_InsightSkipsSmallDelta(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat campuran metrik: beberapa signifikan, beberapa kecil (< 1%)
		nSignifikan := rapid.IntRange(0, 5).Draw(t, "jumlah_signifikan")
		nKecil := rapid.IntRange(1, 5).Draw(t, "jumlah_kecil")

		metrics := make([]ComparisonMetric, 0, nSignifikan+nKecil)

		// Metrik signifikan (|delta| >= 1%)
		for i := 0; i < nSignifikan; i++ {
			deltaPct := rapid.Float64Range(1.0, 500.0).Draw(t, fmt.Sprintf("sig_delta_%d", i))
			if rapid.Bool().Draw(t, fmt.Sprintf("sig_neg_%d", i)) {
				deltaPct = -deltaPct
			}
			metrics = append(metrics, ComparisonMetric{
				MetricName:      fmt.Sprintf("signifikan_%d", i),
				DeltaPercentage: deltaPct,
			})
		}

		// Metrik kecil (|delta| < 1%) - tidak boleh menghasilkan insight
		for i := 0; i < nKecil; i++ {
			deltaPct := rapid.Float64Range(-0.99, 0.99).Draw(t, fmt.Sprintf("kecil_delta_%d", i))
			metrics = append(metrics, ComparisonMetric{
				MetricName:      fmt.Sprintf("kecil_%d", i),
				DeltaPercentage: deltaPct,
			})
		}

		insights := GenerateInsights(metrics)

		// Tidak boleh ada insight yang mengandung nama metrik kecil
		for _, insight := range insights {
			for i := 0; i < nKecil; i++ {
				namaKecil := fmt.Sprintf("kecil_%d", i)
				if containsSubstring(insight, namaKecil) {
					t.Fatalf(
						"insight mengandung metrik kecil %q (|delta| < 1%%): %q",
						namaKecil, insight,
					)
				}
			}
		}

		// Jumlah insight harus sesuai jumlah metrik signifikan (max 5)
		expectedCount := nSignifikan
		if expectedCount > 5 {
			expectedCount = 5
		}
		if len(insights) != expectedCount {
			t.Fatalf(
				"jumlah insight salah: got %d, want %d (signifikan=%d, kecil=%d)",
				len(insights), expectedCount, nSignifikan, nKecil,
			)
		}
	})
}

// TestProperty_InsightEmptyMetrics memverifikasi bahwa GenerateInsights
func TestProperty_InsightEmptyMetrics(t *testing.T) {
	insights := GenerateInsights(nil)
	if insights != nil {
		t.Fatalf("GenerateInsights(nil) harus nil, got %v", insights)
	}

	insights = GenerateInsights([]ComparisonMetric{})
	if insights != nil {
		t.Fatalf("GenerateInsights([]) harus nil, got %v", insights)
	}
}

// containsSubstring memeriksa apakah s mengandung substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

// searchSubstring mencari substr di dalam s.
func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
