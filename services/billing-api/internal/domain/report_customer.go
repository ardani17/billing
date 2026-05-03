package domain

// =============================================================================
// Customer Growth Report — laporan pertumbuhan pelanggan
// =============================================================================

// CustomerGrowthReport berisi laporan pertumbuhan pelanggan.
type CustomerGrowthReport struct {
	TotalActive      int                       `json:"total_active"`
	NewCustomers     int                       `json:"new_customers"`
	ChurnedCustomers int                       `json:"churned_customers"`
	NetGrowth        int                       `json:"net_growth"`
	ARPU             int64                     `json:"arpu"`
	CLV              int64                     `json:"clv"`
	ChurnRate        float64                   `json:"churn_rate"`
	Trend            []MonthlyGrowthTrend      `json:"trend"`
	Comparison       *CustomerGrowthReport     `json:"comparison,omitempty"`
	Delta            map[string]RevenueDelta   `json:"delta,omitempty"`
}

// MonthlyGrowthTrend berisi data trend pertumbuhan per bulan.
type MonthlyGrowthTrend struct {
	Month            string `json:"month"`
	TotalActive      int    `json:"total_active"`
	NewCustomers     int    `json:"new_customers"`
	ChurnedCustomers int    `json:"churned_customers"`
}

// =============================================================================
// Customer Distribution Report — laporan distribusi pelanggan
// =============================================================================

// DistributionItem berisi satu item distribusi.
type DistributionItem struct {
	ID         string  `json:"id,omitempty"`
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// CustomerDistributionReport berisi laporan distribusi pelanggan.
type CustomerDistributionReport struct {
	ByPackage          []DistributionItem     `json:"by_package"`
	ByArea             []DistributionItem     `json:"by_area"`
	ByStatus           map[CustomerStatus]int `json:"by_status"`
	ByConnectionMethod []DistributionItem     `json:"by_connection_method"`
}

// =============================================================================
// Churn Analysis Report — laporan analisis churn pelanggan
// =============================================================================

// ChurnByReason berisi churn per alasan.
type ChurnByReason struct {
	Reason     string  `json:"reason"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ChurnAnalysisReport berisi laporan analisis churn.
type ChurnAnalysisReport struct {
	ChurnedCount          int                `json:"churned_count"`
	ChurnRate             float64            `json:"churn_rate"`
	ByReason              []ChurnByReason    `json:"by_reason"`
	ByPackage             []DistributionItem `json:"by_package"`
	ByArea                []DistributionItem `json:"by_area"`
	AverageLifetimeMonths float64            `json:"average_lifetime_months"`
}

// =============================================================================
// Revenue by Area Report — laporan pendapatan per area
// =============================================================================

// AreaRevenue berisi pendapatan per area.
type AreaRevenue struct {
	AreaID           string `json:"area_id"`
	AreaName         string `json:"area_name"`
	CustomerCount    int    `json:"customer_count"`
	TotalRevenue     int64  `json:"total_revenue"`
	TotalOutstanding int64  `json:"total_outstanding"`
	ARPU             int64  `json:"arpu"`
}

// RevenueByAreaReport berisi laporan pendapatan per area.
type RevenueByAreaReport struct {
	Areas               []AreaRevenue `json:"areas"`
	Total               AreaRevenue   `json:"total"`
	MostProfitableArea  string        `json:"most_profitable_area"`
	AttentionNeededArea string        `json:"attention_needed_area"`
}
