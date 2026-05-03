package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// Invoice Status State Machine
// =============================================================================

// InvoiceStatus mendefinisikan status invoice dalam sistem.
type InvoiceStatus string

const (
	InvoiceStatusBelumBayar    InvoiceStatus = "belum_bayar"
	InvoiceStatusTerlambat     InvoiceStatus = "terlambat"
	InvoiceStatusLunas         InvoiceStatus = "lunas"
	InvoiceStatusBayarSebagian InvoiceStatus = "bayar_sebagian"
	InvoiceStatusBatal         InvoiceStatus = "batal"
	InvoiceStatusProrate       InvoiceStatus = "prorate"
)

// ValidInvoiceTransitions mendefinisikan transisi status invoice yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidInvoiceTransitions = map[InvoiceStatus][]InvoiceStatus{
	InvoiceStatusBelumBayar:    {InvoiceStatusTerlambat, InvoiceStatusLunas, InvoiceStatusBayarSebagian, InvoiceStatusBatal},
	InvoiceStatusTerlambat:     {InvoiceStatusLunas, InvoiceStatusBayarSebagian, InvoiceStatusBatal},
	InvoiceStatusBayarSebagian: {InvoiceStatusLunas, InvoiceStatusBatal},
	InvoiceStatusProrate:       {InvoiceStatusLunas, InvoiceStatusBayarSebagian, InvoiceStatusBatal},
	InvoiceStatusLunas:         {}, // terminal state
	InvoiceStatusBatal:         {}, // terminal state
}

// CanInvoiceTransition memeriksa apakah transisi dari current ke target valid.
func CanInvoiceTransition(current, target InvoiceStatus) bool {
	targets, ok := ValidInvoiceTransitions[current]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// InvoiceTransition melakukan transisi status dan mengembalikan status baru.
// Mengembalikan error jika transisi tidak valid.
func InvoiceTransition(current, target InvoiceStatus) (InvoiceStatus, error) {
	if CanInvoiceTransition(current, target) {
		return target, nil
	}
	allowed := AllowedInvoiceTargets(current)
	allowedStrs := make([]string, len(allowed))
	for i, a := range allowed {
		allowedStrs[i] = string(a)
	}
	var msg string
	if len(allowedStrs) == 0 {
		msg = fmt.Sprintf("transisi dari %s ke %s tidak diizinkan, transisi yang diizinkan: (tidak ada)", current, target)
	} else {
		msg = fmt.Sprintf("transisi dari %s ke %s tidak diizinkan, transisi yang diizinkan: %s", current, target, strings.Join(allowedStrs, ", "))
	}
	return current, fmt.Errorf("%w: %s", ErrInvalidInvoiceStatusTransition, msg)
}

// AllowedInvoiceTargets mengembalikan daftar status tujuan yang valid dari status saat ini.
func AllowedInvoiceTargets(current InvoiceStatus) []InvoiceStatus {
	targets, ok := ValidInvoiceTransitions[current]
	if !ok {
		return nil
	}
	return targets
}

// =============================================================================
// Invoice Item Type — tipe item dalam invoice
// =============================================================================

// InvoiceItemType mendefinisikan tipe item invoice.
type InvoiceItemType string

const (
	ItemTypeMonthly       InvoiceItemType = "monthly"
	ItemTypeInstallation  InvoiceItemType = "installation"
	ItemTypeProrateCharge InvoiceItemType = "prorate_charge"
	ItemTypeProrateCredit InvoiceItemType = "prorate_credit"
	ItemTypePenalty       InvoiceItemType = "penalty"
	ItemTypeTax           InvoiceItemType = "tax"
	ItemTypeDiscount      InvoiceItemType = "discount"
	ItemTypeRecurring     InvoiceItemType = "recurring"
	ItemTypeCustom        InvoiceItemType = "custom"
	ItemTypeCreditApplied InvoiceItemType = "credit_applied"
)

// =============================================================================
// Penalty Type — tipe denda keterlambatan
// =============================================================================

// PenaltyType mendefinisikan tipe denda keterlambatan.
type PenaltyType string

const (
	PenaltyFixed      PenaltyType = "fixed"
	PenaltyPercentage PenaltyType = "percentage"
	PenaltyDaily      PenaltyType = "daily"
)

// =============================================================================
// Invoice Entity
// =============================================================================

// Invoice merepresentasikan invoice pelanggan ISP.
type Invoice struct {
	ID             string        `json:"id"`
	TenantID       string        `json:"tenant_id"`
	CustomerID     string        `json:"customer_id"`
	InvoiceNumber  string        `json:"invoice_number"`
	PeriodMonth    int           `json:"period_month"`
	PeriodYear     int           `json:"period_year"`
	DueDate        time.Time     `json:"due_date"`
	Subtotal       int64         `json:"subtotal"`
	TaxAmount      int64         `json:"tax_amount"`
	PenaltyAmount  int64         `json:"penalty_amount"`
	DiscountAmount int64         `json:"discount_amount"`
	CreditApplied  int64         `json:"credit_applied"`
	TotalAmount    int64         `json:"total_amount"`
	PaidAmount     int64         `json:"paid_amount"`
	Status         InvoiceStatus `json:"status"`
	Notes          string        `json:"notes,omitempty"`
	IsPrepaid      bool          `json:"is_prepaid"`
	PrepaidMonths  *int          `json:"prepaid_months,omitempty"`
	Version        int           `json:"version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	// Field gabungan (dari query JOIN)
	CustomerName    string `json:"customer_name,omitempty"`
	CustomerIDSeq   string `json:"customer_id_seq,omitempty"`
	CustomerPhone   string `json:"customer_phone,omitempty"`
	CustomerAddress string `json:"customer_address,omitempty"`
	PackageName     string `json:"package_name,omitempty"`
}

// =============================================================================
// InvoiceItem Entity — satu baris item dalam invoice
// =============================================================================

// InvoiceItem merepresentasikan satu baris item dalam invoice.
type InvoiceItem struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	InvoiceID   string                 `json:"invoice_id"`
	ItemType    InvoiceItemType        `json:"item_type"`
	Description string                 `json:"description"`
	Quantity    int                    `json:"quantity"`
	UnitPrice   int64                  `json:"unit_price"`
	Amount      int64                  `json:"amount"`
	SortOrder   int                    `json:"sort_order"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// =============================================================================
// InvoicePayment Entity — catatan pembayaran terhadap invoice
// =============================================================================

// InvoicePayment merepresentasikan catatan pembayaran terhadap invoice.
type InvoicePayment struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	InvoiceID       string     `json:"invoice_id"`
	Amount          int64      `json:"amount"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentDate     time.Time  `json:"payment_date"`
	ReferenceNumber string     `json:"reference_number,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	RecordedByID    string     `json:"recorded_by_id"`
	RecordedByName  string     `json:"recorded_by_name"`
	ReceiptNumber   string     `json:"receipt_number,omitempty"`
	ReceiptGroupID  string     `json:"receipt_group_id,omitempty"`
	ProofImageURL   string     `json:"proof_image_url,omitempty"`
	Voided          bool       `json:"voided"`
	VoidedAt        *time.Time `json:"voided_at,omitempty"`
	VoidedBy        string     `json:"voided_by,omitempty"`
	VoidReason      string     `json:"void_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// =============================================================================
// InvoiceAuditLog Entity — catatan lifecycle invoice (append-only)
// =============================================================================

// InvoiceAuditLog merepresentasikan catatan lifecycle invoice (append-only).
type InvoiceAuditLog struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	InvoiceID string                 `json:"invoice_id"`
	Action    string                 `json:"action"`
	ActorID   string                 `json:"actor_id"`
	ActorName string                 `json:"actor_name"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// =============================================================================
// CalculateLateFee — menghitung denda keterlambatan
// =============================================================================

// CalculateLateFee menghitung denda keterlambatan berdasarkan konfigurasi billing.
// Mengembalikan jumlah denda yang sudah di-cap oleh penalty_max_amount (jika > 0).
// Jika penalty tidak diaktifkan, mengembalikan 0.
func CalculateLateFee(settings *BillingSettings, subtotal int64, daysOverdue int) int64 {
	if settings == nil || !settings.PenaltyEnabled {
		return 0
	}

	var fee int64

	switch settings.PenaltyType {
	case PenaltyFixed:
		fee = settings.PenaltyAmount
	case PenaltyPercentage:
		fee = subtotal * int64(settings.PenaltyPercentage) / 100
	case PenaltyDaily:
		fee = settings.PenaltyDailyAmount * int64(daysOverdue)
	default:
		return 0
	}

	// Cap denda berdasarkan penalty_max_amount jika dikonfigurasi (> 0)
	if settings.PenaltyMaxAmount > 0 && fee > settings.PenaltyMaxAmount {
		fee = settings.PenaltyMaxAmount
	}

	return fee
}

// =============================================================================
// Domain Error Variables — error khusus domain invoice
// =============================================================================

var (
	// ErrInvoiceNotFound dikembalikan saat invoice tidak ditemukan atau milik tenant lain
	ErrInvoiceNotFound = errors.New("invoice tidak ditemukan")

	// ErrInvalidInvoiceStatusTransition dikembalikan saat transisi status invoice tidak valid
	ErrInvalidInvoiceStatusTransition = errors.New("transisi status invoice tidak valid")

	// ErrInvoiceNotEditable dikembalikan saat invoice tidak bisa diedit (status bukan belum_bayar)
	ErrInvoiceNotEditable = errors.New("invoice hanya bisa diedit saat status belum_bayar")

	// ErrInvoiceNotCancellable dikembalikan saat invoice tidak bisa dibatalkan (status lunas atau batal)
	ErrInvoiceNotCancellable = errors.New("invoice tidak bisa dibatalkan")

	// ErrInvoiceConfirmationMismatch dikembalikan saat nomor konfirmasi tidak cocok dengan nomor invoice
	ErrInvoiceConfirmationMismatch = errors.New("nomor konfirmasi tidak cocok")

	// ErrInvoiceDuplicate dikembalikan saat invoice untuk periode yang sama sudah ada
	ErrInvoiceDuplicate = errors.New("invoice untuk periode ini sudah ada")

	// ErrCreditNoteNotFound dikembalikan saat credit note tidak ditemukan
	ErrCreditNoteNotFound = errors.New("credit note tidak ditemukan")

	// ErrDebitNoteNotFound dikembalikan saat debit note tidak ditemukan
	ErrDebitNoteNotFound = errors.New("debit note tidak ditemukan")

	// ErrRecurringItemNotFound dikembalikan saat recurring item tidak ditemukan
	ErrRecurringItemNotFound = errors.New("recurring item tidak ditemukan")

	// ErrBillingSettingsNotFound dikembalikan saat billing settings tidak ditemukan untuk tenant
	ErrBillingSettingsNotFound = errors.New("billing settings tidak ditemukan")
)
