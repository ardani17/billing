package domain

import "time"

// =============================================================================
// CreditNote Entitas - nota kredit untuk penyesuaian invoice
// =============================================================================

// CreditNote merepresentasikan nota kredit untuk penyesuaian invoice.
type CreditNote struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	CreditNoteNumber string    `json:"credit_note_number"`
	InvoiceID        string    `json:"invoice_id"`
	Amount           int64     `json:"amount"`
	Reason           string    `json:"reason"`
	ApplyToCredit    bool      `json:"apply_to_credit"`
	CreatedByID      string    `json:"created_by_id"`
	CreatedByName    string    `json:"created_by_name"`
	CreatedAt        time.Time `json:"created_at"`
}

// =============================================================================
// DebitNote Entitas - nota debit untuk tagihan tambahan
// =============================================================================

// DebitNote merepresentasikan nota debit untuk tagihan tambahan.
type DebitNote struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	DebitNoteNumber string          `json:"debit_note_number"`
	CustomerID      string          `json:"customer_id"`
	DueDate         time.Time       `json:"due_date"`
	Items           []DebitNoteItem `json:"items"`
	TotalAmount     int64           `json:"total_amount"`
	InvoiceID       *string         `json:"invoice_id,omitempty"`
	CreatedByID     string          `json:"created_by_id"`
	CreatedByName   string          `json:"created_by_name"`
	CreatedAt       time.Time       `json:"created_at"`
}

// =============================================================================
// DebitNoteItem Entitas - satu item dalam debit note
// =============================================================================

// DebitNoteItem merepresentasikan satu item dalam debit note.
type DebitNoteItem struct {
	ID          string `json:"id"`
	DebitNoteID string `json:"debit_note_id"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
}

// =============================================================================
// CustomerRecurringItem Entitas - item berulang per pelanggan
// =============================================================================

// CustomerRecurringItem merepresentasikan item berulang per pelanggan.
type CustomerRecurringItem struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	CustomerID  string     `json:"customer_id"`
	Description string     `json:"description"`
	Amount      int64      `json:"amount"`
	IsActive    bool       `json:"is_active"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
