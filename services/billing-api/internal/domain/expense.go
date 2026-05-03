package domain

import "time"

// =============================================================================
// Expense — entitas pengeluaran bisnis
// =============================================================================

// Expense merepresentasikan satu pengeluaran bisnis.
type Expense struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	CategoryID    string     `json:"category_id"`
	CategoryName  string     `json:"category_name,omitempty"`
	Amount        int64      `json:"amount"`
	Description   string     `json:"description"`
	ExpenseDate   time.Time  `json:"expense_date"`
	IsRecurring   bool       `json:"is_recurring"`
	RecurringDay  *int       `json:"recurring_day,omitempty"`
	CreatedByID   string     `json:"created_by_id"`
	CreatedByName string     `json:"created_by_name,omitempty"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// =============================================================================
// ExpenseCategory — kategori pengeluaran per tenant
// =============================================================================

// ExpenseCategory merepresentasikan kategori pengeluaran per tenant.
type ExpenseCategory struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	Name         string     `json:"name"`
	IsDefault    bool       `json:"is_default"`
	ExpenseCount int        `json:"expense_count,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// DefaultExpenseCategories berisi kategori default untuk tenant baru.
var DefaultExpenseCategories = []string{
	"Bandwidth/Upstream",
	"Gaji Karyawan",
	"Sewa Tiang/Infrastruktur",
	"Listrik & Operasional",
	"Perangkat",
	"Notifikasi",
	"Lainnya",
}

// =============================================================================
// KPITarget — target KPI per tenant
// =============================================================================

// KPITarget merepresentasikan target KPI per tenant.
type KPITarget struct {
	ID                        string    `json:"id"`
	TenantID                  string    `json:"tenant_id"`
	MonthlyRevenueTarget      *int64    `json:"monthly_revenue_target,omitempty"`
	CollectionRateTarget      *float64  `json:"collection_rate_target,omitempty"`
	MaxReceivables            *int64    `json:"max_receivables,omitempty"`
	NewCustomersMonthlyTarget *int      `json:"new_customers_monthly_target,omitempty"`
	MaxChurnRate              *float64  `json:"max_churn_rate,omitempty"`
	TotalCustomersTarget      *int      `json:"total_customers_target,omitempty"`
	SLAUptimeTarget           *float64  `json:"sla_uptime_target,omitempty"`
	MaxActiveAlarms           *int      `json:"max_active_alarms,omitempty"`
	MinSignalQualityPct       *float64  `json:"min_signal_quality_percentage,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// =============================================================================
// ReportSchedule — jadwal laporan otomatis
// =============================================================================

// ScheduleType mendefinisikan tipe jadwal laporan.
type ScheduleType string

const (
	// ScheduleDaily — jadwal harian.
	ScheduleDaily ScheduleType = "daily"
	// ScheduleWeekly — jadwal mingguan.
	ScheduleWeekly ScheduleType = "weekly"
	// ScheduleMonthly — jadwal bulanan.
	ScheduleMonthly ScheduleType = "monthly"
)

// ReportSchedule merepresentasikan jadwal laporan otomatis.
type ReportSchedule struct {
	ID           string       `json:"id"`
	TenantID     string       `json:"tenant_id"`
	ReportType   string       `json:"report_type"`
	ScheduleType ScheduleType `json:"schedule_type"`
	Format       string       `json:"format"`
	Recipients   []Recipient  `json:"recipients"`
	Filters      ReportFilter `json:"filters"`
	IsActive     bool         `json:"is_active"`
	CreatedByID  string       `json:"created_by_id"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Recipient merepresentasikan penerima laporan.
type Recipient struct {
	Type    string `json:"type"`    // "email" atau "whatsapp"
	Address string `json:"address"`
}

// =============================================================================
// ReportJob — job export laporan (async)
// =============================================================================

// ReportJobStatus mendefinisikan status job export.
type ReportJobStatus string

const (
	// JobPending — job menunggu diproses.
	JobPending ReportJobStatus = "pending"
	// JobProcessing — job sedang diproses.
	JobProcessing ReportJobStatus = "processing"
	// JobCompleted — job selesai diproses.
	JobCompleted ReportJobStatus = "completed"
	// JobFailed — job gagal diproses.
	JobFailed ReportJobStatus = "failed"
)

// ReportJob merepresentasikan job export laporan.
type ReportJob struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	ReportType  string          `json:"report_type"`
	Format      string          `json:"format"`
	Filters     ReportFilter    `json:"filters"`
	Status      ReportJobStatus `json:"status"`
	DownloadURL string          `json:"download_url,omitempty"`
	Error       string          `json:"error,omitempty"`
	RequestedBy string          `json:"requested_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// =============================================================================
// CustomReportTemplate — template laporan custom
// =============================================================================

// CustomReportTemplate merepresentasikan template laporan custom.
type CustomReportTemplate struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	Name               string    `json:"name"`
	Metrics            []string  `json:"metrics"`
	GroupBy            string    `json:"group_by"`
	SubGroupBy         string    `json:"sub_group_by,omitempty"`
	DisplayType        string    `json:"display_type"`
	DefaultPeriodRange string    `json:"default_period_range,omitempty"`
	CreatedByID        string    `json:"created_by_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
