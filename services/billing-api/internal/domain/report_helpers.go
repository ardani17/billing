package domain

// =============================================================================
// Report Helpers — pure functions untuk kalkulasi dan validasi laporan
// =============================================================================

// BuildAgingReport mengelompokkan outstanding invoices ke aging buckets
// dan menghitung total outstanding.
// Invarian: sum(bucket.TotalAmount) == TotalOutstanding
func BuildAgingReport(buckets []AgingBucket) AgingReport {
	var total int64
	for _, b := range buckets {
		total += b.TotalAmount
	}
	return AgingReport{
		Buckets:          buckets,
		TotalOutstanding: total,
	}
}

// BuildRevenueSource menghitung total dari breakdown pendapatan per sumber.
// Invarian: Total == MonthlySubscription + VoucherSales + InstallationFees + LateFees + Other
func BuildRevenueSource(monthly, voucher, installation, late, other int64) RevenueSource {
	return RevenueSource{
		MonthlySubscription: monthly,
		VoucherSales:        voucher,
		InstallationFees:    installation,
		LateFees:            late,
		Other:               other,
		Total:               monthly + voucher + installation + late + other,
	}
}

// BuildRevenueByAreaTotal menghitung total row dari daftar area revenues.
// Invarian: Total.TotalRevenue == sum(area.TotalRevenue)
// Invarian: Total.TotalOutstanding == sum(area.TotalOutstanding)
// Invarian: Total.CustomerCount == sum(area.CustomerCount)
func BuildRevenueByAreaTotal(areas []AreaRevenue) AreaRevenue {
	var total AreaRevenue
	total.AreaID = "total"
	total.AreaName = "Total"
	for _, a := range areas {
		total.CustomerCount += a.CustomerCount
		total.TotalRevenue += a.TotalRevenue
		total.TotalOutstanding += a.TotalOutstanding
	}
	if total.CustomerCount > 0 {
		total.ARPU = total.TotalRevenue / int64(total.CustomerCount)
	}
	return total
}

// CalculateDistribution menghitung persentase distribusi dari sekumpulan amounts.
// Mengembalikan slice float64 dengan panjang sama dengan input, di mana setiap
// elemen adalah persentase dari total (item_amount / total_amount * 100).
// Kasus khusus:
//   - Input kosong → return nil
//   - Semua amount nol → distribusi merata (100 / jumlah item)
//   - Satu item → return [100.0]
func CalculateDistribution(amounts []int64) []float64 {
	if len(amounts) == 0 {
		return nil
	}

	var total int64
	for _, a := range amounts {
		total += a
	}

	result := make([]float64, len(amounts))

	// Jika total nol, distribusi merata
	if total == 0 {
		equal := 100.0 / float64(len(amounts))
		for i := range result {
			result[i] = equal
		}
		return result
	}

	for i, a := range amounts {
		result[i] = float64(a) / float64(total) * 100.0
	}
	return result
}

// ClassifyAgingBucket mengklasifikasikan jumlah hari tunggakan ke bucket aging
// yang sesuai. Bucket yang tersedia:
//   - "1-7 hari"   : overdue 1 sampai 7 hari
//   - "8-14 hari"  : overdue 8 sampai 14 hari
//   - "15-30 hari" : overdue 15 sampai 30 hari
//   - "30+ hari"   : overdue lebih dari 30 hari
//
// Invarian: setiap overdueDays >= 1 pasti masuk tepat satu bucket.
func ClassifyAgingBucket(overdueDays int) string {
	switch {
	case overdueDays >= 1 && overdueDays <= 7:
		return "1-7 hari"
	case overdueDays >= 8 && overdueDays <= 14:
		return "8-14 hari"
	case overdueDays >= 15 && overdueDays <= 30:
		return "15-30 hari"
	default:
		return "30+ hari"
	}
}

// CalculateProfitLoss menghitung laba rugi bersih dan margin keuntungan.
// Invarian: netProfit == totalRevenue - totalExpenses
// Invarian: profitMargin == netProfit / totalRevenue * 100 (atau 0 jika totalRevenue == 0)
func CalculateProfitLoss(totalRevenue, totalExpenses int64) (netProfit int64, profitMargin float64) {
	netProfit = totalRevenue - totalExpenses
	if totalRevenue == 0 {
		profitMargin = 0
		return
	}
	profitMargin = float64(netProfit) / float64(totalRevenue) * 100.0
	return
}

// CalculateCustomerMetrics menghitung metrik pelanggan untuk satu periode.
// Parameter:
//   - newCustomers: jumlah pelanggan baru dalam periode
//   - churnedCustomers: jumlah pelanggan yang berhenti dalam periode
//   - avgActive: rata-rata pelanggan aktif dalam periode
//   - revenue: total pendapatan dalam periode (dalam Rupiah)
//   - avgLifetimeMonths: rata-rata masa berlangganan pelanggan (dalam bulan)
//   - totalStart: total pelanggan aktif di awal periode
//
// Invarian:
//   - netGrowth == newCustomers - churnedCustomers
//   - arpu == revenue / avgActive (atau 0 jika avgActive == 0)
//   - clv == arpu * avgLifetimeMonths (dibulatkan ke int64)
//   - churnRate == churnedCustomers / totalStart * 100 (atau 0 jika totalStart == 0)
func CalculateCustomerMetrics(
	newCustomers, churnedCustomers, avgActive int,
	revenue int64,
	avgLifetimeMonths float64,
	totalStart int,
) (netGrowth int, arpu int64, clv int64, churnRate float64) {
	// Net growth = pelanggan baru dikurangi pelanggan yang churn
	netGrowth = newCustomers - churnedCustomers

	// ARPU = total revenue / rata-rata pelanggan aktif
	if avgActive > 0 {
		arpu = revenue / int64(avgActive)
	}

	// CLV = ARPU * rata-rata masa berlangganan
	clv = int64(float64(arpu) * avgLifetimeMonths)

	// Churn rate = pelanggan churn / total pelanggan awal periode * 100
	if totalStart > 0 {
		churnRate = float64(churnedCustomers) / float64(totalStart) * 100.0
	}

	return
}

// CalculateKPIProgress menghitung progress KPI dan menentukan status label.
// Parameter:
//   - current: nilai KPI saat ini
//   - target: nilai target KPI
//
// Invarian:
//   - progress == current / target * 100 (atau 0 jika target == 0)
//   - status == "tercapai" jika progress >= 100%
//   - status == "hampir" jika progress >= 80% dan < 100%
//   - status == "di_bawah_target" jika progress < 80%
func CalculateKPIProgress(current, target float64) (progress float64, status string) {
	// Jika target nol, progress adalah 0 dan status di bawah target
	if target == 0 {
		return 0, "di_bawah_target"
	}

	progress = current / target * 100.0

	switch {
	case progress >= 100:
		status = "tercapai"
	case progress >= 80:
		status = "hampir"
	default:
		status = "di_bawah_target"
	}

	return
}

// SortAndLimitTopDebtors mengurutkan daftar debitur berdasarkan TotalOutstanding
// secara descending dan membatasi hasilnya maksimal `limit` item.
// Invarian: hasil diurutkan descending berdasarkan TotalOutstanding, len(result) <= limit.
func SortAndLimitTopDebtors(debtors []TopDebtor, limit int) []TopDebtor {
	if len(debtors) == 0 {
		return nil
	}

	// Buat salinan agar tidak mengubah slice asli
	sorted := make([]TopDebtor, len(debtors))
	copy(sorted, debtors)

	// Urutkan berdasarkan TotalOutstanding descending (insertion sort sederhana)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j].TotalOutstanding < key.TotalOutstanding {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	// Batasi jumlah item
	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}

	return sorted
}

// FindPeakPaymentDate mencari tanggal dengan TotalAmount tertinggi dari
// daftar pembayaran harian. Mengembalikan tanggal dan jumlah tertinggi.
// Jika input kosong, mengembalikan string kosong dan 0.
func FindPeakPaymentDate(payments []DailyPayment) (date string, amount int64) {
	if len(payments) == 0 {
		return "", 0
	}

	date = payments[0].Date
	amount = payments[0].TotalAmount

	for _, p := range payments[1:] {
		if p.TotalAmount > amount {
			date = p.Date
			amount = p.TotalAmount
		}
	}

	return date, amount
}
