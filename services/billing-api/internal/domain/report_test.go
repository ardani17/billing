package domain

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 2.7, 1.2, 10.3**
//
// Untuk atur aging buckets yang dihasilkan, sum total_amount semua bucket
// HARUS sama dengan total_outstanding. Berlaku juga untuk revenue breakdown
// (sum sources == total) dan area revenue (sum area == total row).

// TestProperty_AgingBucketSumInvariant memverifikasi bahwa sum total_amount
// dari semua aging bucket sama dengan total_outstanding pada AgingReport.
func TestProperty_AgingBucketSumInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 4 aging buckets sesuai spesifikasi (1-7, 8-14, 15-30, 30+)
		labels := []string{"1-7 hari", "8-14 hari", "15-30 hari", "30+ hari"}
		buckets := make([]AgingBucket, len(labels))
		var expectedTotal int64

		for i, label := range labels {
			// Gunakan nominal non-negatif (piutang tidak bisa negatif)
			amount := rapid.Int64Range(0, 1_000_000_000).Draw(t, label+"_amount")
			count := rapid.IntRange(0, 10000).Draw(t, label+"_count")
			buckets[i] = AgingBucket{
				Label:         label,
				TotalAmount:   amount,
				CustomerCount: count,
			}
			expectedTotal += amount
		}

		report := BuildAgingReport(buckets)

		// Verifikasi invariant: sum bucket amounts == total outstanding
		if report.TotalOutstanding != expectedTotal {
			t.Fatalf(
				"sum aging buckets (%d) != total_outstanding (%d)",
				expectedTotal, report.TotalOutstanding,
			)
		}

		// Verifikasi jumlah bucket tetap 4
		if len(report.Buckets) != 4 {
			t.Fatalf("expected 4 buckets, got %d", len(report.Buckets))
		}
	})
}

// TestProperty_RevenueSourceSumInvariant memverifikasi bahwa sum semua
// sumber pendapatan sama dengan Total pada RevenueSource.
func TestProperty_RevenueSourceSumInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat nilai pendapatan per sumber (bisa negatif untuk koreksi/refund)
		monthly := rapid.Int64Range(0, 1_000_000_000).Draw(t, "monthly")
		voucher := rapid.Int64Range(0, 1_000_000_000).Draw(t, "voucher")
		installation := rapid.Int64Range(0, 1_000_000_000).Draw(t, "installation")
		late := rapid.Int64Range(0, 1_000_000_000).Draw(t, "late")
		other := rapid.Int64Range(0, 1_000_000_000).Draw(t, "other")

		source := BuildRevenueSource(monthly, voucher, installation, late, other)

		expectedTotal := monthly + voucher + installation + late + other

		// Verifikasi invariant: sum sources == total
		if source.Total != expectedTotal {
			t.Fatalf(
				"sum revenue sources (%d) != total (%d)",
				expectedTotal, source.Total,
			)
		}

		// Verifikasi setiap field tersimpan dengan benar
		if source.MonthlySubscription != monthly {
			t.Fatalf("MonthlySubscription mismatch: got %d, want %d", source.MonthlySubscription, monthly)
		}
		if source.VoucherSales != voucher {
			t.Fatalf("VoucherSales mismatch: got %d, want %d", source.VoucherSales, voucher)
		}
		if source.InstallationFees != installation {
			t.Fatalf("InstallationFees mismatch: got %d, want %d", source.InstallationFees, installation)
		}
		if source.LateFees != late {
			t.Fatalf("LateFees mismatch: got %d, want %d", source.LateFees, late)
		}
		if source.Other != other {
			t.Fatalf("Other mismatch: got %d, want %d", source.Other, other)
		}
	})
}

// TestProperty_AreaRevenueSumInvariant memverifikasi bahwa sum pendapatan
// semua area sama dengan total row pada RevenueByAreaReport.
func TestProperty_AreaRevenueSumInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 1-20 area revenues
		numAreas := rapid.IntRange(1, 20).Draw(t, "num_areas")
		areas := make([]AreaRevenue, numAreas)

		var expectedRevenue, expectedOutstanding int64
		var expectedCustomers int

		for i := 0; i < numAreas; i++ {
			revenue := rapid.Int64Range(0, 1_000_000_000).Draw(t, "revenue")
			outstanding := rapid.Int64Range(0, 1_000_000_000).Draw(t, "outstanding")
			customers := rapid.IntRange(0, 10000).Draw(t, "customers")

			areas[i] = AreaRevenue{
				AreaID:           rapid.StringMatching(`^area-[0-9]+$`).Draw(t, "area_id"),
				AreaName:         rapid.StringMatching(`^Area [A-Z]$`).Draw(t, "area_name"),
				CustomerCount:    customers,
				TotalRevenue:     revenue,
				TotalOutstanding: outstanding,
			}
			// Hitung ARPU per area
			if customers > 0 {
				areas[i].ARPU = revenue / int64(customers)
			}

			expectedRevenue += revenue
			expectedOutstanding += outstanding
			expectedCustomers += customers
		}

		total := BuildRevenueByAreaTotal(areas)

		// Verifikasi invariant: sum area revenues == total revenue
		if total.TotalRevenue != expectedRevenue {
			t.Fatalf(
				"sum area revenues (%d) != total revenue (%d)",
				expectedRevenue, total.TotalRevenue,
			)
		}

		// Verifikasi invariant: sum area outstanding == total outstanding
		if total.TotalOutstanding != expectedOutstanding {
			t.Fatalf(
				"sum area outstanding (%d) != total outstanding (%d)",
				expectedOutstanding, total.TotalOutstanding,
			)
		}

		// Verifikasi invariant: sum area customers == total customers
		if total.CustomerCount != expectedCustomers {
			t.Fatalf(
				"sum area customers (%d) != total customers (%d)",
				expectedCustomers, total.CustomerCount,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 2.2**
//
// Untuk setiap invoice yang belum lunas dengan jumlah hari terlambat tertentu,
// invoice tersebut HARUS masuk ke bucket yang benar:
//   - 1-7 hari   jika terlambat 1-7
//   - 8-14 hari  jika terlambat 8-14
//   - 15-30 hari jika terlambat 15-30
//   - 30+ hari   jika terlambat > 30

// TestProperty_AgingBucketClassification memverifikasi bahwa ClassifyAgingBucket
// mengembalikan label bucket yang benar berdasarkan jumlah hari tunggakan.
func TestProperty_AgingBucketClassification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat terlambat days antara 1 dan 365
		overdueDays := rapid.IntRange(1, 365).Draw(t, "overdue_days")

		// Klasifikasikan menggunakan fungsi yang diuji
		bucket := ClassifyAgingBucket(overdueDays)

		// Tentukan bucket yang diharapkan berdasarkan rentang hari
		var expected string
		switch {
		case overdueDays >= 1 && overdueDays <= 7:
			expected = "1-7 hari"
		case overdueDays >= 8 && overdueDays <= 14:
			expected = "8-14 hari"
		case overdueDays >= 15 && overdueDays <= 30:
			expected = "15-30 hari"
		case overdueDays > 30:
			expected = "30+ hari"
		}

		// Verifikasi klasifikasi sesuai dengan yang diharapkan
		if bucket != expected {
			t.Fatalf(
				"ClassifyAgingBucket(%d) = %q, expected %q",
				overdueDays, bucket, expected,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 3.2, 4.2, 8.2, 8.3, 8.5**
//
// Untuk atur items yang didistribusikan berdasarkan kategori (metode pembayaran,
// paket voucher, paket pelanggan, area, status, metode koneksi), jumlah semua
// percentage HARUS mendekati 100% (toleransi rounding ±0.1%). Setiap percentage
// HARUS sama dengan item_amount / total_amount * 100.

// TestProperty_DistributionPercentageSumTo100 memverifikasi bahwa CalculateDistribution
func TestProperty_DistributionPercentageSumTo100(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 1-20 items dengan nominal non-negatif
		numItems := rapid.IntRange(1, 20).Draw(t, "num_items")
		amounts := make([]int64, numItems)
		for i := 0; i < numItems; i++ {
			amounts[i] = rapid.Int64Range(0, 1_000_000_000).Draw(t, "amount")
		}

		// Hitung distribusi menggunakan fungsi yang diuji
		percentages := CalculateDistribution(amounts)

		if len(percentages) != numItems {
			t.Fatalf("expected %d percentages, got %d", numItems, len(percentages))
		}

		// Hitung total persentase
		var totalPct float64
		for _, pct := range percentages {
			totalPct += pct
		}

		// Verifikasi invariant: sum persentase mendekati 100% (toleransi ±0.1%)
		if totalPct < 99.9 || totalPct > 100.1 {
			t.Fatalf(
				"sum persentase distribusi (%.4f%%) di luar toleransi ±0.1%% dari 100%%",
				totalPct,
			)
		}

		// Verifikasi setiap persentase sesuai formula: item_amount / total_amount * 100
		var totalAmount int64
		for _, a := range amounts {
			totalAmount += a
		}

		for i, pct := range percentages {
			var expected float64
			if totalAmount == 0 {
				// Distribusi merata jika total nol
				expected = 100.0 / float64(numItems)
			} else {
				expected = float64(amounts[i]) / float64(totalAmount) * 100.0
			}

			// Toleransi floating point
			diff := pct - expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				t.Fatalf(
					"persentase item[%d] = %.6f, expected %.6f (diff %.6f)",
					i, pct, expected, diff,
				)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 5.4, 4.4**
//
// Untuk setiap total_revenue dan total_expenses, net_profit HARUS sama dengan
// total_revenue - total_expenses, dan profit_margin HARUS sama dengan
// net_profit / total_revenue * 100 (atau 0 jika total_revenue == 0).

// TestProperty_ProfitLossCalculation memverifikasi bahwa CalculateProfitLoss
func TestProperty_ProfitLossCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat total revenue dan total expenses (bisa nol atau positif)
		totalRevenue := rapid.Int64Range(0, 10_000_000_000).Draw(t, "total_revenue")
		totalExpenses := rapid.Int64Range(0, 10_000_000_000).Draw(t, "total_expenses")

		// Hitung menggunakan fungsi yang diuji
		netProfit, profitMargin := CalculateProfitLoss(totalRevenue, totalExpenses)

		// Verifikasi invariant: net_profit == total_revenue - total_expenses
		expectedNetProfit := totalRevenue - totalExpenses
		if netProfit != expectedNetProfit {
			t.Fatalf(
				"CalculateProfitLoss(%d, %d): net_profit = %d, expected %d",
				totalRevenue, totalExpenses, netProfit, expectedNetProfit,
			)
		}

		// Verifikasi invariant: profit_margin
		if totalRevenue == 0 {
			// Jika revenue nol, margin harus 0
			if profitMargin != 0 {
				t.Fatalf(
					"CalculateProfitLoss(%d, %d): profit_margin = %f, expected 0 (revenue == 0)",
					totalRevenue, totalExpenses, profitMargin,
				)
			}
		} else {
			// profit_margin == net_profit / total_revenue * 100
			expectedMargin := float64(expectedNetProfit) / float64(totalRevenue) * 100.0
			diff := profitMargin - expectedMargin
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Fatalf(
					"CalculateProfitLoss(%d, %d): profit_margin = %.6f, expected %.6f (diff %.6f)",
					totalRevenue, totalExpenses, profitMargin, expectedMargin, diff,
				)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 7.2, 7.4, 7.5, 7.6**
//
// Untuk data pelanggan dalam satu periode:
//   - net_growth HARUS sama dengan new_customers - churned_customers
//   - arpu HARUS sama dengan total_revenue / avg_active_customers (atau 0 jika avg_active == 0)
//   - clv HARUS sama dengan arpu * avg_lifetime_months
//   - churn_rate HARUS sama dengan churned / total_start * 100 (atau 0 jika total_start == 0)

// TestProperty_CustomerMetricsCalculation memverifikasi bahwa CalculateCustomerMetrics
func TestProperty_CustomerMetricsCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat data pelanggan
		newCustomers := rapid.IntRange(0, 10000).Draw(t, "new_customers")
		churnedCustomers := rapid.IntRange(0, 10000).Draw(t, "churned_customers")
		avgActive := rapid.IntRange(0, 50000).Draw(t, "avg_active")
		revenue := rapid.Int64Range(0, 100_000_000_000).Draw(t, "revenue")
		avgLifetimeMonths := rapid.Float64Range(0, 120).Draw(t, "avg_lifetime_months")
		totalStart := rapid.IntRange(0, 50000).Draw(t, "total_start")

		// Hitung menggunakan fungsi yang diuji
		netGrowth, arpu, clv, churnRate := CalculateCustomerMetrics(
			newCustomers, churnedCustomers, avgActive,
			revenue, avgLifetimeMonths, totalStart,
		)

		// Verifikasi invariant: net_growth == new - churned
		expectedNetGrowth := newCustomers - churnedCustomers
		if netGrowth != expectedNetGrowth {
			t.Fatalf(
				"net_growth = %d, expected %d (new=%d, churned=%d)",
				netGrowth, expectedNetGrowth, newCustomers, churnedCustomers,
			)
		}

		// Verifikasi invariant: arpu == revenue / avg_active (atau 0)
		var expectedARPU int64
		if avgActive > 0 {
			expectedARPU = revenue / int64(avgActive)
		}
		if arpu != expectedARPU {
			t.Fatalf(
				"arpu = %d, expected %d (revenue=%d, avg_active=%d)",
				arpu, expectedARPU, revenue, avgActive,
			)
		}

		// Verifikasi invariant: clv == arpu * avg_lifetime_months
		expectedCLV := int64(float64(expectedARPU) * avgLifetimeMonths)
		if clv != expectedCLV {
			t.Fatalf(
				"clv = %d, expected %d (arpu=%d, avg_lifetime=%.2f)",
				clv, expectedCLV, expectedARPU, avgLifetimeMonths,
			)
		}

		// Verifikasi invariant: churn_rate == churned / total_start * 100 (atau 0)
		if totalStart == 0 {
			if churnRate != 0 {
				t.Fatalf(
					"churn_rate = %f, expected 0 (total_start == 0)",
					churnRate,
				)
			}
		} else {
			expectedChurnRate := float64(churnedCustomers) / float64(totalStart) * 100.0
			diff := churnRate - expectedChurnRate
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Fatalf(
					"churn_rate = %.6f, expected %.6f (churned=%d, total_start=%d)",
					churnRate, expectedChurnRate, churnedCustomers, totalStart,
				)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 20.4**
//
// Untuk setiap current_value dan target_value, progress_percentage HARUS sama
// Status label HARUS "tercapai" jika progress >= 100%, "hampir" jika >= 80%,
// dan "di_bawah_target" jika < 80%.

// TestProperty_KPIProgressAndStatusLabel memverifikasi bahwa CalculateKPIProgress
func TestProperty_KPIProgressAndStatusLabel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		current := rapid.Float64Range(0, 10_000_000_000).Draw(t, "current")
		target := rapid.Float64Range(0, 10_000_000_000).Draw(t, "target")

		// Hitung menggunakan fungsi yang diuji
		progress, status := CalculateKPIProgress(current, target)

		if target == 0 {
			if progress != 0 {
				t.Fatalf(
					"CalculateKPIProgress(%f, %f): progress = %f, expected 0 (target == 0)",
					current, target, progress,
				)
			}
		} else {
			expectedProgress := current / target * 100.0
			diff := progress - expectedProgress
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.0001 {
				t.Fatalf(
					"CalculateKPIProgress(%f, %f): progress = %.6f, expected %.6f (diff %.6f)",
					current, target, progress, expectedProgress, diff,
				)
			}
		}

		// Verifikasi invariant: status label berdasarkan progress
		switch {
		case progress >= 100:
			if status != "tercapai" {
				t.Fatalf(
					"CalculateKPIProgress(%f, %f): progress = %.2f%%, status = %q, expected %q",
					current, target, progress, status, "tercapai",
				)
			}
		case progress >= 80:
			if status != "hampir" {
				t.Fatalf(
					"CalculateKPIProgress(%f, %f): progress = %.2f%%, status = %q, expected %q",
					current, target, progress, status, "hampir",
				)
			}
		default:
			if status != "di_bawah_target" {
				t.Fatalf(
					"CalculateKPIProgress(%f, %f): progress = %.2f%%, status = %q, expected %q",
					current, target, progress, status, "di_bawah_target",
				)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 2.5, 3.4**
//
// Untuk atur debitur apapun, top_debtors HARUS diurutkan berdasarkan
// total_outstanding descending dan maksimal 10 item. Untuk daily payments,
// peak_payment_date HARUS merupakan tanggal dengan total_amount tertinggi.

// TestProperty_TopDebtorsOrderingAndLimit memverifikasi bahwa SortAndLimitTopDebtors
// mengurutkan debitur berdasarkan TotalOutstanding descending dan membatasi ke 10 item.
func TestProperty_TopDebtorsOrderingAndLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 0-25 debitur (bisa lebih dari 10 untuk menguji limiting)
		numDebtors := rapid.IntRange(0, 25).Draw(t, "num_debtors")
		debtors := make([]TopDebtor, numDebtors)

		for i := 0; i < numDebtors; i++ {
			debtors[i] = TopDebtor{
				CustomerID:       rapid.StringMatching(`^cust-[0-9]{1,5}$`).Draw(t, "customer_id"),
				CustomerName:     rapid.StringMatching(`^Pelanggan [A-Z][a-z]{2,8}$`).Draw(t, "customer_name"),
				TotalOutstanding: rapid.Int64Range(0, 10_000_000_000).Draw(t, "total_outstanding"),
				MonthsOverdue:    rapid.IntRange(0, 36).Draw(t, "months_overdue"),
			}
		}

		// Jalankan fungsi yang diuji dengan limit 10
		result := SortAndLimitTopDebtors(debtors, 10)

		if numDebtors == 0 {
			if result != nil {
				t.Fatalf("expected nil for empty input, got %d items", len(result))
			}
			return
		}

		// Verifikasi invariant: maksimal 10 item
		expectedLen := numDebtors
		if expectedLen > 10 {
			expectedLen = 10
		}
		if len(result) != expectedLen {
			t.Fatalf("expected %d items, got %d", expectedLen, len(result))
		}

		// Verifikasi invariant: diurutkan descending berdasarkan TotalOutstanding
		for i := 1; i < len(result); i++ {
			if result[i].TotalOutstanding > result[i-1].TotalOutstanding {
				t.Fatalf(
					"top_debtors tidak terurut descending: item[%d].TotalOutstanding (%d) > item[%d].TotalOutstanding (%d)",
					i, result[i].TotalOutstanding, i-1, result[i-1].TotalOutstanding,
				)
			}
		}

		if len(result) > 0 {
			maxOutstanding := result[0].TotalOutstanding
			for _, d := range debtors {
				if d.TotalOutstanding > maxOutstanding {
					t.Fatalf(
						"debitur dengan TotalOutstanding %d tidak masuk top_debtors (max di result: %d)",
						d.TotalOutstanding, maxOutstanding,
					)
				}
			}
		}
	})
}

// TestProperty_PeakPaymentDate memverifikasi bahwa FindPeakPaymentDate
// mengembalikan tanggal dengan TotalAmount tertinggi dari daftar pembayaran harian.
func TestProperty_PeakPaymentDate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 0-60 pembayaran harian (simulasi 1-2 bulan)
		numPayments := rapid.IntRange(0, 60).Draw(t, "num_payments")
		payments := make([]DailyPayment, numPayments)

		for i := 0; i < numPayments; i++ {
			// Buat tanggal unik dalam format YYYY-MM-DD
			day := rapid.IntRange(1, 28).Draw(t, "day")
			month := rapid.IntRange(1, 12).Draw(t, "month")
			dateStr := rapid.Just(
				fmt.Sprintf("2024-%02d-%02d", month, day),
			).Draw(t, "date")

			payments[i] = DailyPayment{
				Date:             dateStr,
				TotalAmount:      rapid.Int64Range(0, 10_000_000_000).Draw(t, "total_amount"),
				TransactionCount: rapid.IntRange(0, 1000).Draw(t, "tx_count"),
			}
		}

		// Jalankan fungsi yang diuji
		peakDate, peakAmount := FindPeakPaymentDate(payments)

		if numPayments == 0 {
			if peakDate != "" || peakAmount != 0 {
				t.Fatalf("expected empty result for empty input, got date=%q amount=%d", peakDate, peakAmount)
			}
			return
		}

		// Cari TotalAmount tertinggi secara manual untuk verifikasi
		var maxAmount int64
		for _, p := range payments {
			if p.TotalAmount > maxAmount {
				maxAmount = p.TotalAmount
			}
		}

		// Verifikasi invariant: peakAmount harus sama dengan TotalAmount tertinggi
		if peakAmount != maxAmount {
			t.Fatalf(
				"peak_amount (%d) != max TotalAmount (%d)",
				peakAmount, maxAmount,
			)
		}

		// Verifikasi invariant: peakDate harus merupakan tanggal yang memiliki TotalAmount == maxAmount
		found := false
		for _, p := range payments {
			if p.Date == peakDate && p.TotalAmount == peakAmount {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf(
				"peak_payment_date %q dengan amount %d tidak ditemukan di input payments",
				peakDate, peakAmount,
			)
		}
	})
}
