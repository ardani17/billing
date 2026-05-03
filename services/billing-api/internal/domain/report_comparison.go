package domain

// =============================================================================
// Comparison Report — laporan perbandingan antar periode
// =============================================================================

// ComparisonType mendefinisikan tipe perbandingan.
type ComparisonType string

const (
	// ComparisonMoM — perbandingan bulan ke bulan (Month over Month).
	ComparisonMoM ComparisonType = "mom"
	// ComparisonYoY — perbandingan tahun ke tahun (Year over Year).
	ComparisonYoY ComparisonType = "yoy"
	// ComparisonQoQ — perbandingan kuartal ke kuartal (Quarter over Quarter).
	ComparisonQoQ ComparisonType = "qoq"
	// ComparisonCustom — perbandingan periode kustom.
	ComparisonCustom ComparisonType = "custom"
)

// ComparisonMetric berisi satu metrik perbandingan.
type ComparisonMetric struct {
	MetricName      string  `json:"metric_name"`
	BaseValue       float64 `json:"base_value"`
	CompareValue    float64 `json:"compare_value"`
	DeltaAbsolute   float64 `json:"delta_absolute"`
	DeltaPercentage float64 `json:"delta_percentage"`
	Trend           string  `json:"trend"` // "improving", "declining", "stable"
}

// ComparisonReport berisi laporan perbandingan antar periode.
type ComparisonReport struct {
	ComparisonType ComparisonType   `json:"comparison_type"`
	BasePeriod     string           `json:"base_period"`
	ComparePeriod  string           `json:"compare_period"`
	Metrics        []ComparisonMetric `json:"metrics"`
	Insights       []string         `json:"insights"`
}

// =============================================================================
// Forecast Report — laporan proyeksi / forecasting
// =============================================================================

// ForecastMonth berisi proyeksi per bulan.
type ForecastMonth struct {
	Month                string `json:"month"`
	ProjectedRevenue     int64  `json:"projected_revenue"`
	ProjectedCustomers   int    `json:"projected_customers"`
	ProjectedReceivables int64  `json:"projected_receivables"`
}

// ForecastReport berisi laporan proyeksi.
type ForecastReport struct {
	Projections         []ForecastMonth   `json:"projections"`
	EstimatedTargetDate map[string]string `json:"estimated_target_date,omitempty"`
	InsufficientData    bool              `json:"insufficient_data"`
	Disclaimer          string            `json:"disclaimer,omitempty"`
}

// =============================================================================
// Dashboard Widget — data untuk dashboard widget di halaman utama
// =============================================================================

// DashboardData berisi data untuk dashboard widget.
type DashboardData struct {
	TotalActiveCustomers int              `json:"total_active_customers"`
	CustomersTrend       float64          `json:"customers_trend"`
	MonthlyRevenue       int64            `json:"monthly_revenue"`
	RevenueTarget        *int64           `json:"revenue_target,omitempty"`
	RevenueProgress      *float64         `json:"revenue_progress,omitempty"`
	TotalReceivables     int64            `json:"total_receivables"`
	ReceivablesCount     int              `json:"receivables_count"`
	RoutersOnline        int              `json:"routers_online"`
	RoutersOffline       int              `json:"routers_offline"`
	CollectionRate       float64          `json:"collection_rate"`
	CollectionTarget     *float64         `json:"collection_target,omitempty"`
	ChurnRate            float64          `json:"churn_rate"`
	ChurnTarget          *float64         `json:"churn_target,omitempty"`
	ARPU                 int64            `json:"arpu"`
	ModuleInactive       map[string]bool  `json:"module_inactive,omitempty"`
}
