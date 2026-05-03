package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// --- Enum: Provider Payment Gateway ---

// GatewayProvider mendefinisikan provider payment gateway yang didukung.
type GatewayProvider string

const (
	GatewayXendit   GatewayProvider = "xendit"
	GatewayMidtrans GatewayProvider = "midtrans"
)

// --- Enum: Status Payment Link ---

// PaymentLinkStatus mendefinisikan status payment link.
type PaymentLinkStatus string

const (
	PaymentLinkActive  PaymentLinkStatus = "active"
	PaymentLinkExpired PaymentLinkStatus = "expired"
	PaymentLinkPaid    PaymentLinkStatus = "paid"
	PaymentLinkFailed  PaymentLinkStatus = "failed"
)

// --- Enum: Status Pemrosesan Webhook ---

// WebhookProcessingStatus mendefinisikan status pemrosesan webhook.
type WebhookProcessingStatus string

const (
	WebhookReceived  WebhookProcessingStatus = "received"
	WebhookVerified  WebhookProcessingStatus = "verified"
	WebhookProcessed WebhookProcessingStatus = "processed"
	WebhookFailed    WebhookProcessingStatus = "failed"
	WebhookDuplicate WebhookProcessingStatus = "duplicate"
)

// --- Entity: Konfigurasi Gateway per Tenant ---

// GatewayConfig merepresentasikan konfigurasi payment gateway per tenant.
type GatewayConfig struct {
	ID                     string          `json:"id"`
	TenantID               string          `json:"tenant_id"`
	GatewayProvider        GatewayProvider `json:"gateway_provider"`
	IsActive               bool            `json:"is_active"`
	APIKeyEncrypted        string          `json:"-"`
	WebhookSecretEncrypted string          `json:"-"`
	APIKeyMasked           string          `json:"api_key_masked,omitempty"`
	EnabledMethods         []string        `json:"enabled_methods"`
	PaymentLinkExpiryDays  int             `json:"payment_link_expiry_days"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

// --- Entity: Payment Link ---

// PaymentLink merepresentasikan payment link yang digenerate via gateway.
type PaymentLink struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	CustomerID      string            `json:"customer_id"`
	GatewayProvider GatewayProvider   `json:"gateway_provider"`
	GatewayConfigID string            `json:"gateway_config_id"`
	ExternalID      string            `json:"external_id"`
	PaymentURL      string            `json:"payment_url"`
	Amount          int64             `json:"amount"`
	Status          PaymentLinkStatus `json:"status"`
	ExpiresAt       time.Time         `json:"expires_at"`
	PaidAt          *time.Time        `json:"paid_at,omitempty"`
	PaidMethod      string            `json:"paid_method,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// --- Entity: Junction Payment Link ↔ Invoice ---

// PaymentLinkInvoice merepresentasikan junction antara payment link dan invoice.
type PaymentLinkInvoice struct {
	ID            string `json:"id"`
	PaymentLinkID string `json:"payment_link_id"`
	InvoiceID     string `json:"invoice_id"`
}

// --- Entity: Log Webhook (append-only) ---

// WebhookLog merepresentasikan log webhook request (append-only).
type WebhookLog struct {
	ID               string                  `json:"id"`
	TenantID         *string                 `json:"tenant_id,omitempty"`
	GatewayProvider  GatewayProvider         `json:"gateway_provider"`
	EventType        string                  `json:"event_type"`
	ExternalID       string                  `json:"external_id"`
	RequestBody      json.RawMessage         `json:"request_body"`
	SourceIP         string                  `json:"source_ip"`
	SignatureValid   *bool                   `json:"signature_valid,omitempty"`
	ProcessingStatus WebhookProcessingStatus `json:"processing_status"`
	ErrorMessage     string                  `json:"error_message,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
}

// --- Error Domain: error khusus domain payment gateway ---

var (
	// ErrGatewayConfigNotFound dikembalikan saat konfigurasi gateway tidak ditemukan
	ErrGatewayConfigNotFound = errors.New("konfigurasi gateway tidak ditemukan")

	// ErrGatewayConfigDuplicate dikembalikan saat konfigurasi gateway untuk provider sudah ada
	ErrGatewayConfigDuplicate = errors.New("konfigurasi gateway untuk provider ini sudah ada")

	// ErrPaymentLinkNotFound dikembalikan saat payment link tidak ditemukan
	ErrPaymentLinkNotFound = errors.New("payment link tidak ditemukan")

	// ErrPaymentLinkAlreadyActive dikembalikan saat payment link aktif sudah ada
	ErrPaymentLinkAlreadyActive = errors.New("payment link aktif sudah ada untuk customer ini")

	// ErrPaymentLinkExpired dikembalikan saat payment link sudah expired
	ErrPaymentLinkExpired = errors.New("payment link sudah expired")

	// ErrWebhookSignatureInvalid dikembalikan saat signature webhook tidak valid
	ErrWebhookSignatureInvalid = errors.New("signature webhook tidak valid")

	// ErrWebhookDuplicate dikembalikan saat webhook sudah diproses sebelumnya
	ErrWebhookDuplicate = errors.New("webhook sudah diproses sebelumnya")

	// ErrWebhookIPNotWhitelisted dikembalikan saat IP webhook tidak ada dalam whitelist
	ErrWebhookIPNotWhitelisted = errors.New("IP webhook tidak ada dalam whitelist")

	// ErrGatewayUnavailable dikembalikan saat payment gateway tidak tersedia
	ErrGatewayUnavailable = errors.New("payment gateway tidak tersedia")

	// ErrGatewayInvalidAPIKey dikembalikan saat API key gateway tidak valid
	ErrGatewayInvalidAPIKey = errors.New("API key gateway tidak valid")

	// ErrNoActiveGateway dikembalikan saat tidak ada gateway aktif untuk tenant
	ErrNoActiveGateway = errors.New("tidak ada gateway aktif untuk tenant ini")

	// ErrInvalidEnabledMethods dikembalikan saat enabled_methods mengandung nilai tidak valid
	ErrInvalidEnabledMethods = errors.New("enabled_methods mengandung nilai tidak valid")

	// ErrEncryptionFailed dikembalikan saat gagal mengenkripsi data
	ErrEncryptionFailed = errors.New("gagal mengenkripsi data")

	// ErrDecryptionFailed dikembalikan saat gagal mendekripsi data
	ErrDecryptionFailed = errors.New("gagal mendekripsi data")
)

// --- Validasi: Metode Pembayaran yang Valid per Provider ---

// ValidXenditMethods berisi metode pembayaran yang valid untuk Xendit.
var ValidXenditMethods = map[string]bool{
	"va_bca": true, "va_bni": true, "va_bri": true,
	"va_mandiri": true, "va_permata": true, "qris": true,
	"ewallet_ovo": true, "ewallet_gopay": true,
	"ewallet_dana": true, "ewallet_shopeepay": true,
	"credit_card": true,
}

// ValidMidtransMethods berisi metode pembayaran yang valid untuk Midtrans.
var ValidMidtransMethods = map[string]bool{
	"va_bca": true, "va_bni": true, "va_bri": true,
	"va_mandiri": true, "va_permata": true, "qris": true,
	"ewallet_gopay": true, "ewallet_shopeepay": true,
	"credit_card": true,
}

// ValidateEnabledMethods memvalidasi bahwa semua metode dalam daftar valid untuk provider.
// Mengembalikan error jika ada metode yang tidak valid.
func ValidateEnabledMethods(provider GatewayProvider, methods []string) error {
	validMap := ValidXenditMethods
	if provider == GatewayMidtrans {
		validMap = ValidMidtransMethods
	}
	for _, m := range methods {
		if !validMap[m] {
			return fmt.Errorf("%w: %s tidak valid untuk %s", ErrInvalidEnabledMethods, m, provider)
		}
	}
	return nil
}
