package domain

// UpdateBillingSettingsRequest adalah payload untuk menyimpan konfigurasi billing tenant.
type UpdateBillingSettingsRequest struct {
	GenerateDays       int         `json:"generate_days" validate:"min=0,max=31"`
	GracePeriodDays    int         `json:"grace_period_days" validate:"min=0,max=90"`
	SuspendDays        int         `json:"suspend_days" validate:"min=0,max=180"`
	TaxEnabled         bool        `json:"tax_enabled"`
	TaxRate            float64     `json:"tax_rate" validate:"min=0,max=100"`
	PenaltyEnabled     bool        `json:"penalty_enabled"`
	PenaltyType        PenaltyType `json:"penalty_type" validate:"omitempty,oneof=fixed percentage daily"`
	PenaltyAmount      int64       `json:"penalty_amount" validate:"min=0"`
	PenaltyPercentage  float64     `json:"penalty_percentage" validate:"min=0,max=100"`
	PenaltyDailyAmount int64       `json:"penalty_daily_amount" validate:"min=0"`
	PenaltyMaxAmount   int64       `json:"penalty_max_amount" validate:"min=0"`
	InvoicePrefix      string      `json:"invoice_prefix" validate:"required,min=2,max=12"`
	NewCustomerBilling string      `json:"new_customer_billing" validate:"required,oneof=prorate full_next_cycle immediate"`
	Timezone           string      `json:"timezone" validate:"required,min=3,max=64"`
	AutoIsolir         bool        `json:"auto_isolir"`
	AutoOpenIsolir     bool        `json:"auto_open_isolir"`
}
