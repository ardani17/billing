package domain

import "time"

// =============================================================================
// BillingSettings Entity — konfigurasi billing per tenant
// =============================================================================

// BillingSettings merepresentasikan konfigurasi billing per tenant.
type BillingSettings struct {
	ID                 string      `json:"id"`
	TenantID           string      `json:"tenant_id"`
	GenerateDays       int         `json:"generate_days"`
	GracePeriodDays    int         `json:"grace_period_days"`
	SuspendDays        int         `json:"suspend_days"`
	TaxEnabled         bool        `json:"tax_enabled"`
	TaxRate            float64     `json:"tax_rate"`
	PenaltyEnabled     bool        `json:"penalty_enabled"`
	PenaltyType        PenaltyType `json:"penalty_type"`
	PenaltyAmount      int64       `json:"penalty_amount"`
	PenaltyPercentage  float64     `json:"penalty_percentage"`
	PenaltyDailyAmount int64       `json:"penalty_daily_amount"`
	PenaltyMaxAmount   int64       `json:"penalty_max_amount"`
	InvoicePrefix      string      `json:"invoice_prefix"`
	NewCustomerBilling string      `json:"new_customer_billing"`
	Timezone           string      `json:"timezone"`
	AutoIsolir         bool        `json:"auto_isolir"`
	AutoOpenIsolir     bool        `json:"auto_open_isolir"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}
