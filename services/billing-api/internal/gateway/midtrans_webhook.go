package gateway

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// midtransNotificationPayload merepresentasikan payload notifikasi webhook dari Midtrans.
type midtransNotificationPayload struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	TransactionStatus string `json:"transaction_status"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	SignatureKey      string `json:"signature_key"`
	FraudStatus       string `json:"fraud_status"`
}

// midtransEventMapping memetakan transaction_status Midtrans ke event type internal.
var midtransEventMapping = map[string]string{
	"capture":    "payment.paid",
	"settlement": "payment.paid",
	"expire":     "payment.expired",
	"deny":       "payment.failed",
	"cancel":     "payment.failed",
}

// VerifyWebhookSignature memverifikasi webhook Midtrans dengan menghitung
// SHA-512 hash dari order_id + status_code + gross_amount + server_key
// dan membandingkan dengan signature_key di body notifikasi.
func (a *MidtransAdapter) VerifyWebhookSignature(_ context.Context, _ map[string]string, body []byte, _ string) (bool, error) {
	var payload midtransNotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("gagal parse body webhook midtrans: %w", err)
	}

	if payload.SignatureKey == "" {
		return false, nil
	}

	// Hitung SHA-512: order_id + status_code + gross_amount + server_key
	raw := payload.OrderID + payload.StatusCode + payload.GrossAmount + a.serverKey
	hash := sha512.Sum512([]byte(raw))
	computed := hex.EncodeToString(hash[:])

	return strings.EqualFold(computed, payload.SignatureKey), nil
}

// ParseWebhookPayload mem-parse body webhook Midtrans menjadi WebhookEvent.
// Memetakan transaction_status Midtrans (capture, settlement, expire, deny, cancel)
// ke event type internal (payment.paid, payment.expired, payment.failed).
// Mengekstrak payment_type sebagai paid_method.
func (a *MidtransAdapter) ParseWebhookPayload(body []byte) (*domain.WebhookEvent, error) {
	var payload midtransNotificationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("gagal parse webhook midtrans: %w", err)
	}

	eventType, ok := midtransEventMapping[payload.TransactionStatus]
	if !ok {
		return nil, fmt.Errorf("transaction_status midtrans tidak dikenal: %s", payload.TransactionStatus)
	}

	// Untuk status "capture", pastikan fraud_status = "accept"
	if payload.TransactionStatus == "capture" && payload.FraudStatus != "accept" {
		eventType = "payment.failed"
	}

	// Parse gross_amount dari string ke int64
	amount := parseMidtransAmount(payload.GrossAmount)

	// Konversi payment_type Midtrans ke format metode internal
	paidMethod := mapMidtransPaymentType(payload.PaymentType)

	return &domain.WebhookEvent{
		EventType:       eventType,
		ExternalID:      payload.OrderID,
		TransactionID:   payload.TransactionID,
		Amount:          amount,
		PaidMethod:      paidMethod,
		GatewayProvider: domain.GatewayMidtrans,
		RawPayload:      body,
	}, nil
}

// parseMidtransAmount mengkonversi gross_amount string dari Midtrans ke int64 (Rupiah).
// Midtrans mengirim gross_amount sebagai string, contoh: "100000.00".
func parseMidtransAmount(amountStr string) int64 {
	var amount float64
	_, _ = fmt.Sscanf(amountStr, "%f", &amount)
	return int64(math.Round(amount))
}

// mapMidtransPaymentType mengkonversi payment_type Midtrans ke format metode internal.
// Contoh: "bank_transfer" → "va_*", "gopay" → "ewallet_gopay", "qris" → "qris".
func mapMidtransPaymentType(paymentType string) string {
	switch paymentType {
	case "bank_transfer":
		// Default VA, channel spesifik ditentukan dari field lain jika tersedia
		return "va_transfer"
	case "echannel":
		return "va_mandiri"
	case "gopay":
		return "ewallet_gopay"
	case "shopeepay":
		return "ewallet_shopeepay"
	case "qris":
		return "qris"
	case "credit_card":
		return "credit_card"
	case "cstore":
		return "convenience_store"
	default:
		return paymentType
	}
}
