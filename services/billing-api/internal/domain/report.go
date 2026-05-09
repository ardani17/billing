package domain

import "time"

// =============================================================================
// Filter laporan - parameter filter global untuk semua laporan
// =============================================================================

// ReportFilter berisi parameter filter global untuk semua laporan.
type ReportFilter struct {
	PeriodStart  time.Time  `json:"period_start"`
	PeriodEnd    time.Time  `json:"period_end"`
	CompareStart *time.Time `json:"compare_start,omitempty"`
	CompareEnd   *time.Time `json:"compare_end,omitempty"`
	AreaID       string     `json:"area_id,omitempty"`
	PackageID    string     `json:"package_id,omitempty"`
	RouterID     string     `json:"router_id,omitempty"`
}

// =============================================================================
// Laporan pendapatan - laporan ringkasan pendapatan
// =============================================================================

// RevenueSource berisi breakdown pendapatan per sumber.
type RevenueSource struct {
	MonthlySubscription int64 `json:"monthly_subscription"`
	VoucherSales        int64 `json:"voucher_sales"`
	InstallationFees    int64 `json:"installation_fees"`
	LateFees            int64 `json:"late_fees"`
	Other               int64 `json:"other"`
	Total               int64 `json:"total"`
}

// RevenueDelta berisi delta antara dua periode.
type RevenueDelta struct {
	Absolute   int64   `json:"absolute"`
	Percentage float64 `json:"percentage"`
}

// MonthlyRevenueTrend berisi data trend pendapatan per bulan.
type MonthlyRevenueTrend struct {
	Month               string `json:"month"` // format: "2006-01"
	TotalRevenue        int64  `json:"total_revenue"`
	MonthlySubscription int64  `json:"monthly_subscription"`
	VoucherSales        int64  `json:"voucher_sales"`
	OtherRevenue        int64  `json:"other_revenue"`
}

// RevenueReport berisi laporan ringkasan pendapatan.
type RevenueReport struct {
	Current     RevenueSource           `json:"current"`
	Comparison  *RevenueSource          `json:"comparison,omitempty"`
	Delta       map[string]RevenueDelta `json:"delta,omitempty"`
	Trend       []MonthlyRevenueTrend   `json:"trend"`
	KPITarget   *int64                  `json:"kpi_target,omitempty"`
	KPIProgress *float64                `json:"kpi_progress,omitempty"`
}

// =============================================================================
// Laporan aging - laporan piutang / aging
// =============================================================================

// AgingBucket berisi data per bucket umur piutang.
type AgingBucket struct {
	Label         string `json:"label"`
	TotalAmount   int64  `json:"total_amount"`
	CustomerCount int    `json:"customer_count"`
}

// TopDebtor berisi data debitur terbesar.
type TopDebtor struct {
	CustomerID       string `json:"customer_id"`
	CustomerName     string `json:"customer_name"`
	TotalOutstanding int64  `json:"total_outstanding"`
	MonthsOverdue    int    `json:"months_overdue"`
}

// ReceivablesTrend berisi data trend piutang per bulan.
type ReceivablesTrend struct {
	Month            string `json:"month"`
	TotalOutstanding int64  `json:"total_outstanding"`
}

// AgingReport berisi laporan piutang/aging.
type AgingReport struct {
	Buckets          []AgingBucket      `json:"buckets"`
	TotalOutstanding int64              `json:"total_outstanding"`
	CollectionRate   float64            `json:"collection_rate"`
	AvgDaysToPay     float64            `json:"avg_days_to_pay"`
	TopDebtors       []TopDebtor        `json:"top_debtors"`
	Trend            []ReceivablesTrend `json:"trend"`
	KPITarget        *float64           `json:"kpi_target,omitempty"`
}

// =============================================================================
// Laporan pembayaran - laporan distribusi pembayaran
// =============================================================================

// PaymentMethodBreakdown berisi distribusi per metode pembayaran.
type PaymentMethodBreakdown struct {
	MethodName       string  `json:"method_name"`
	TotalAmount      int64   `json:"total_amount"`
	TransactionCount int     `json:"transaction_count"`
	Percentage       float64 `json:"percentage"`
}

// DailyPayment berisi data pembayaran harian.
type DailyPayment struct {
	Date             string `json:"date"` // format: "2006-01-02"
	TotalAmount      int64  `json:"total_amount"`
	TransactionCount int    `json:"transaction_count"`
}

// PaymentReport berisi laporan distribusi pembayaran.
type PaymentReport struct {
	Methods         []PaymentMethodBreakdown `json:"methods"`
	DailyPayments   []DailyPayment           `json:"daily_payments"`
	PeakPaymentDate string                   `json:"peak_payment_date"`
	PeakAmount      int64                    `json:"peak_amount"`
}

// =============================================================================
// Voucher Laporan pendapatan - laporan pendapatan voucher
// =============================================================================

// VoucherByPackage berisi penjualan voucher per paket.
type VoucherByPackage struct {
	PackageName  string  `json:"package_name"`
	TotalRevenue int64   `json:"total_revenue"`
	VoucherCount int     `json:"voucher_count"`
	Percentage   float64 `json:"percentage"`
}

// VoucherByReseller berisi penjualan voucher per reseller.
type VoucherByReseller struct {
	ResellerName   string `json:"reseller_name"`
	TotalRevenue   int64  `json:"total_revenue"`
	VoucherCount   int    `json:"voucher_count"`
	ResellerMargin int64  `json:"reseller_margin"`
}

// VoucherRevenueReport berisi laporan pendapatan voucher.
type VoucherRevenueReport struct {
	TotalRevenue        int64               `json:"total_revenue"`
	TotalVoucherCount   int                 `json:"total_voucher_count"`
	ByPackage           []VoucherByPackage  `json:"by_package"`
	ByReseller          []VoucherByReseller `json:"by_reseller"`
	TotalResellerMargin int64               `json:"total_reseller_margin"`
}

// =============================================================================
// Laporan laba rugi - laporan laba rugi sederhana
// =============================================================================

// ProfitLossLineItem berisi satu baris item laba rugi.
type ProfitLossLineItem struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// ProfitLossReport berisi laporan laba rugi sederhana.
type ProfitLossReport struct {
	RevenueItems  []ProfitLossLineItem `json:"revenue_items"`
	TotalRevenue  int64                `json:"total_revenue"`
	ExpenseItems  []ProfitLossLineItem `json:"expense_items"`
	TotalExpenses int64                `json:"total_expenses"`
	NetProfit     int64                `json:"net_profit"`
	ProfitMargin  float64              `json:"profit_margin"`
	Comparison    *ProfitLossReport    `json:"comparison,omitempty"`
}
