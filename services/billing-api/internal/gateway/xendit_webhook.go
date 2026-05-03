package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// xenditWebhookPayload merepresentasikan payload notifikasi webhook dari Xendit.
type xenditWebhookPayload struct {
	ID             string  `json:"id"`
	ExternalID     string  `json:"external_id"`
	Status         string  `json:"status"`
	Amount         float64 `json:"amount"`
	PaidAmount     float64 `json:"paid_amount"`
	PaymentMethod  string  `json:"payment_method"`
	PaymentChannel string  `json:"payment_channel"`
}

// xenditEventMapping memetakan status Xendit ke event type internal.
var xenditEventMapping = map[string]string{
	"PAID":    "payment.paid",
	"EXPIRED": "payment.expired",
	"FAILED":  "payment.failed",
}

// VerifyWebhookSignature memverifikasi webhook Xendit dengan membandingkan
// header x-callback-token dengan webhook secret yang tersimpan.
// Xendit menggunakan perbandingan string sederhana untuk verifikasi.
func (a *XenditAdapter) VerifyWebhookSignature(_ context.Context, headers map[string]string, _ []byte, secret string) (bool, error) {
	callbackToken := headers["x-callback-token"]
	if callbackToken == "" {
		return false, nil
	}
	return callbackToken == secret, nil
}

// ParseWebhookPayload mem-parse body webhook Xendit menjadi WebhookEvent.
// Memetakan status Xendit (PAID, EXPIRED, FAILED) ke event type internal
// (payment.paid, payment.expired, payment.failed).
// Mengekstrak metode pembayaran dari field payment_method dan payment_channel.
func (a *XenditAdapter) ParseWebhookPayload(body []byte) (*domain.WebhookEvent, error) {
	var payload xenditWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("gagal parse webhook xendit: %w", err)
	}

	eventType, ok := xenditEventMapping[payload.Status]
	if !ok {
		return nil, fmt.Errorf("status xendit tidak dikenal: %s", payload.Status)
	}

	// Tentukan jumlah pembayaran — gunakan paid_amount jika tersedia
	amount := int64(math.Round(payload.Amount))
	if payload.PaidAmount > 0 {
		amount = int64(math.Round(payload.PaidAmount))
	}

	// Tentukan metode pembayaran dari payment_method + payment_channel
	paidMethod := mapXenditPaymentMethod(payload.PaymentMethod, payload.PaymentChannel)

	return &domain.WebhookEvent{
		EventType:       eventType,
		ExternalID:      payload.ExternalID,
		TransactionID:   payload.ID,
		Amount:          amount,
		PaidMethod:      paidMethod,
		GatewayProvider: domain.GatewayXendit,
		RawPayload:      body,
	}, nil
}

// mapXenditPaymentMethod mengkonversi payment_method dan payment_channel Xendit
// ke format metode pembayaran internal (e.g., "va_bca", "qris", "ewallet_gopay").
func mapXenditPaymentMethod(method, channel string) string {
	switch method {
	case "BANK_TRANSFER", "VIRTUAL_ACCOUNT":
		return mapXenditBankChannel(channel)
	case "EWALLET":
		return mapXenditEwalletChannel(channel)
	case "QR_CODE":
		return "qris"
	case "CREDIT_CARD":
		return "credit_card"
	default:
		// Fallback: gunakan kombinasi method_channel dalam lowercase
		if channel != "" {
			return method + "_" + channel
		}
		return method
	}
}

// mapXenditBankChannel memetakan channel bank Xendit ke format internal.
func mapXenditBankChannel(channel string) string {
	channelMap := map[string]string{
		"BCA":     "va_bca",
		"BNI":     "va_bni",
		"BRI":     "va_bri",
		"MANDIRI": "va_mandiri",
		"PERMATA": "va_permata",
	}
	if mapped, ok := channelMap[channel]; ok {
		return mapped
	}
	return "va_" + channel
}

// mapXenditEwalletChannel memetakan channel e-wallet Xendit ke format internal.
func mapXenditEwalletChannel(channel string) string {
	channelMap := map[string]string{
		"OVO":       "ewallet_ovo",
		"GOPAY":     "ewallet_gopay",
		"DANA":      "ewallet_dana",
		"SHOPEEPAY": "ewallet_shopeepay",
	}
	if mapped, ok := channelMap[channel]; ok {
		return mapped
	}
	return "ewallet_" + channel
}
