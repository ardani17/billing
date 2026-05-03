package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// PaymentGatewayAdapter — kontrak untuk interaksi dengan payment gateway
// =============================================================================

// PaymentGatewayAdapter mendefinisikan kontrak untuk interaksi dengan payment gateway.
// Diimplementasikan oleh XenditAdapter dan MidtransAdapter.
type PaymentGatewayAdapter interface {
	// CreatePaymentLink membuat payment link baru di gateway.
	// Mengembalikan PaymentLinkResponse berisi URL dan external ID.
	CreatePaymentLink(ctx context.Context, req CreateLinkRequest) (*domain.PaymentLinkResponse, error)

	// VerifyWebhookSignature memverifikasi signature/token webhook.
	// Mengembalikan true jika signature valid.
	VerifyWebhookSignature(ctx context.Context, headers map[string]string, body []byte, secret string) (bool, error)

	// ParseWebhookPayload mem-parse body webhook menjadi WebhookEvent.
	ParseWebhookPayload(body []byte) (*domain.WebhookEvent, error)

	// ExpirePaymentLink meng-expire payment link di gateway.
	ExpirePaymentLink(ctx context.Context, externalID string) error

	// TestConnection menguji koneksi dan kredensial ke gateway.
	TestConnection(ctx context.Context) (*domain.GatewayTestResult, error)
}

// =============================================================================
// CreateLinkRequest — parameter untuk membuat payment link
// =============================================================================

// CreateLinkRequest berisi parameter untuk membuat payment link di gateway.
type CreateLinkRequest struct {
	ExternalID     string        // ID unik dari sistem kita (payment_link.id)
	Amount         int64         // Jumlah dalam Rupiah
	Description    string        // Deskripsi pembayaran
	CustomerName   string        // Nama pelanggan
	CustomerEmail  string        // Email pelanggan (opsional)
	ExpiryDuration time.Duration // Durasi sebelum link expired
	EnabledMethods []string      // Metode pembayaran yang diaktifkan
}

// =============================================================================
// NewAdapter — factory untuk membuat adapter berdasarkan provider
// =============================================================================

// NewAdapter membuat adapter berdasarkan provider.
// apiKey sudah dalam bentuk plaintext (sudah didekripsi).
// Mengembalikan error jika provider tidak didukung.
func NewAdapter(provider domain.GatewayProvider, apiKey string) (PaymentGatewayAdapter, error) {
	switch provider {
	case domain.GatewayXendit:
		return NewXenditAdapter(apiKey), nil
	case domain.GatewayMidtrans:
		return NewMidtransAdapter(apiKey), nil
	default:
		return nil, fmt.Errorf("provider tidak didukung: %s", provider)
	}
}
