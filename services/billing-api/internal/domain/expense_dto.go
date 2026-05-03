package domain

// =============================================================================
// DTO: Expense — Request untuk CRUD pengeluaran
// =============================================================================

// CreateExpenseRequest adalah payload untuk POST /v1/expenses.
// Menerima data pengeluaran baru dengan validasi field wajib.
type CreateExpenseRequest struct {
	// CategoryID adalah UUID kategori pengeluaran.
	CategoryID string `json:"category_id" validate:"required,uuid"`

	// Amount adalah jumlah pengeluaran dalam Rupiah (harus > 0).
	Amount int64 `json:"amount" validate:"required,gt=0"`

	// Description adalah keterangan pengeluaran.
	Description string `json:"description" validate:"omitempty"`

	// ExpenseDate adalah tanggal pengeluaran (format: YYYY-MM-DD).
	ExpenseDate string `json:"expense_date" validate:"required"`

	// IsRecurring menandakan apakah pengeluaran berulang setiap bulan.
	IsRecurring bool `json:"is_recurring"`

	// RecurringDay adalah tanggal auto-repeat bulanan (1-28, wajib jika is_recurring=true).
	RecurringDay *int `json:"recurring_day,omitempty" validate:"omitempty,min=1,max=28"`
}

// UpdateExpenseRequest adalah payload untuk PUT /v1/expenses/:id.
// Semua field opsional, hanya field yang dikirim yang diupdate.
type UpdateExpenseRequest struct {
	// CategoryID adalah UUID kategori pengeluaran.
	CategoryID string `json:"category_id" validate:"omitempty,uuid"`

	// Amount adalah jumlah pengeluaran dalam Rupiah (harus > 0).
	Amount *int64 `json:"amount,omitempty" validate:"omitempty,gt=0"`

	// Description adalah keterangan pengeluaran.
	Description *string `json:"description,omitempty"`

	// ExpenseDate adalah tanggal pengeluaran (format: YYYY-MM-DD).
	ExpenseDate *string `json:"expense_date,omitempty"`

	// IsRecurring menandakan apakah pengeluaran berulang setiap bulan.
	IsRecurring *bool `json:"is_recurring,omitempty"`

	// RecurringDay adalah tanggal auto-repeat bulanan (1-28).
	RecurringDay *int `json:"recurring_day,omitempty" validate:"omitempty,min=1,max=28"`
}

// =============================================================================
// DTO: Expense Category — Request untuk CRUD kategori pengeluaran
// =============================================================================

// CreateCategoryRequest adalah payload untuk POST /v1/expenses/categories.
type CreateCategoryRequest struct {
	// Name adalah nama kategori pengeluaran.
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// UpdateCategoryRequest adalah payload untuk PUT /v1/expenses/categories/:id.
type UpdateCategoryRequest struct {
	// Name adalah nama kategori pengeluaran.
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// =============================================================================
// DTO: Report Schedule — Request untuk CRUD jadwal laporan
// =============================================================================

// CreateScheduleRequest adalah payload untuk POST /v1/reports/schedules.
type CreateScheduleRequest struct {
	// ReportType adalah tipe laporan yang dijadwalkan.
	ReportType string `json:"report_type" validate:"required"`

	// ScheduleType adalah frekuensi jadwal (daily, weekly, monthly).
	ScheduleType string `json:"schedule_type" validate:"required,oneof=daily weekly monthly"`

	// Format adalah format output laporan (pdf, xlsx).
	Format string `json:"format" validate:"required,oneof=pdf xlsx"`

	// Recipients adalah daftar penerima laporan.
	Recipients []RecipientRequest `json:"recipients" validate:"required,min=1,dive"`

	// Filters adalah filter yang diterapkan pada laporan.
	Filters *ReportFilter `json:"filters,omitempty"`
}

// RecipientRequest adalah payload penerima laporan.
type RecipientRequest struct {
	// Type adalah tipe penerima (email atau whatsapp).
	Type string `json:"type" validate:"required,oneof=email whatsapp"`

	// Address adalah alamat penerima (email atau nomor WhatsApp).
	Address string `json:"address" validate:"required"`
}

// UpdateScheduleRequest adalah payload untuk PUT /v1/reports/schedules/:id.
type UpdateScheduleRequest struct {
	// ReportType adalah tipe laporan yang dijadwalkan.
	ReportType string `json:"report_type" validate:"omitempty"`

	// ScheduleType adalah frekuensi jadwal (daily, weekly, monthly).
	ScheduleType string `json:"schedule_type" validate:"omitempty,oneof=daily weekly monthly"`

	// Format adalah format output laporan (pdf, xlsx).
	Format string `json:"format" validate:"omitempty,oneof=pdf xlsx"`

	// Recipients adalah daftar penerima laporan.
	Recipients []RecipientRequest `json:"recipients,omitempty" validate:"omitempty,min=1,dive"`

	// Filters adalah filter yang diterapkan pada laporan.
	Filters *ReportFilter `json:"filters,omitempty"`
}

// =============================================================================
// DTO: KPI Target — Request untuk update target KPI
// =============================================================================

// UpdateKPITargetRequest adalah payload untuk PUT /v1/reports/kpi-targets.
// Semua field opsional — hanya field yang dikirim yang diupdate.
type UpdateKPITargetRequest struct {
	// MonthlyRevenueTarget adalah target pendapatan bulanan (Rupiah).
	MonthlyRevenueTarget *int64 `json:"monthly_revenue_target,omitempty" validate:"omitempty,gt=0"`

	// CollectionRateTarget adalah target collection rate (persentase, 0-100).
	CollectionRateTarget *float64 `json:"collection_rate_target,omitempty" validate:"omitempty,min=0,max=100"`

	// MaxReceivables adalah batas maksimal piutang (Rupiah).
	MaxReceivables *int64 `json:"max_receivables,omitempty" validate:"omitempty,gt=0"`

	// NewCustomersMonthlyTarget adalah target pelanggan baru per bulan.
	NewCustomersMonthlyTarget *int `json:"new_customers_monthly_target,omitempty" validate:"omitempty,gt=0"`

	// MaxChurnRate adalah batas maksimal churn rate (persentase, 0-100).
	MaxChurnRate *float64 `json:"max_churn_rate,omitempty" validate:"omitempty,min=0,max=100"`

	// TotalCustomersTarget adalah target total pelanggan akhir tahun.
	TotalCustomersTarget *int `json:"total_customers_target,omitempty" validate:"omitempty,gt=0"`

	// SLAUptimeTarget adalah target SLA uptime (persentase, 0-100).
	SLAUptimeTarget *float64 `json:"sla_uptime_target,omitempty" validate:"omitempty,min=0,max=100"`

	// MaxActiveAlarms adalah batas maksimal alarm aktif.
	MaxActiveAlarms *int `json:"max_active_alarms,omitempty" validate:"omitempty,min=0"`

	// MinSignalQualityPct adalah persentase minimum ONT dengan signal normal (0-100).
	MinSignalQualityPct *float64 `json:"min_signal_quality_percentage,omitempty" validate:"omitempty,min=0,max=100"`
}

// =============================================================================
// DTO: Custom Report Template — Request untuk buat template laporan custom
// =============================================================================

// CreateTemplateRequest adalah payload untuk POST /v1/reports/custom/templates.
type CreateTemplateRequest struct {
	// Name adalah nama template laporan.
	Name string `json:"name" validate:"required,min=1,max=255"`

	// Metrics adalah daftar metrik yang dipilih (maksimal 3).
	Metrics []string `json:"metrics" validate:"required,min=1,max=3,dive,required"`

	// GroupBy adalah dimensi utama pengelompokan data.
	GroupBy string `json:"group_by" validate:"required,oneof=area package month status connection_method router"`

	// SubGroupBy adalah dimensi sekunder pengelompokan (opsional).
	SubGroupBy string `json:"sub_group_by,omitempty" validate:"omitempty,oneof=area package month status connection_method router"`

	// DisplayType adalah tipe tampilan laporan.
	DisplayType string `json:"display_type" validate:"required,oneof=table bar_chart line_chart pie_chart"`

	// DefaultPeriodRange adalah rentang periode default (opsional).
	DefaultPeriodRange string `json:"default_period_range,omitempty"`
}

// =============================================================================
// DTO: Export — Request untuk export laporan
// =============================================================================

// ExportRequest adalah payload untuk POST /v1/reports/export.
type ExportRequest struct {
	// ReportType adalah tipe laporan yang diexport.
	ReportType string `json:"report_type" validate:"required"`

	// Format adalah format output (pdf, xlsx, csv).
	Format string `json:"format" validate:"required,oneof=pdf xlsx csv"`

	// Filters berisi parameter filter untuk laporan yang diexport.
	Filters *ReportFilter `json:"filters,omitempty"`
}
