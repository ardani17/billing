package gateway

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property-based test untuk verifikasi signature webhook Midtrans
// =============================================================================

// TestProperty_MidtransWebhookSignature memverifikasi bahwa untuk sembarang
// kombinasi order_id, status_code, gross_amount, dan server_key:
// - VerifyWebhookSignature mengembalikan true jika signature_key di body
//   sama dengan SHA512(order_id + status_code + gross_amount + server_key).
// - VerifyWebhookSignature mengembalikan false jika signature_key berbeda
//   (signature acak yang salah).
//
// **Validates: Requirements 6.2**
func TestProperty_MidtransWebhookSignature(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate field acak menggunakan karakter alfanumerik untuk menghindari
		// masalah JSON escaping
		orderID := rapid.StringMatching("[a-zA-Z0-9]+").Draw(t, "orderID")
		statusCode := rapid.StringMatching("[a-zA-Z0-9]+").Draw(t, "statusCode")
		grossAmount := rapid.StringMatching("[a-zA-Z0-9]+").Draw(t, "grossAmount")
		serverKey := rapid.StringMatching("[a-zA-Z0-9]+").Draw(t, "serverKey")

		// Hitung signature yang benar: SHA512(order_id + status_code + gross_amount + server_key)
		raw := orderID + statusCode + grossAmount + serverKey
		hash := sha512.Sum512([]byte(raw))
		correctSignature := hex.EncodeToString(hash[:])

		// Buat adapter Midtrans dengan server key
		adapter := NewMidtransAdapter(serverKey)

		// --- Test 1: Signature benar harus mengembalikan true ---
		validBody := buildMidtransNotificationBody(t, orderID, statusCode, grossAmount, correctSignature)
		valid, err := adapter.VerifyWebhookSignature(context.Background(), nil, validBody, "")
		if err != nil {
			t.Fatalf("VerifyWebhookSignature gagal dengan signature benar: %v", err)
		}
		if !valid {
			t.Errorf("VerifyWebhookSignature mengembalikan false untuk signature yang benar")
		}

		// --- Test 2: Signature salah harus mengembalikan false ---
		wrongSignature := rapid.StringMatching("[a-zA-Z0-9]+").Draw(t, "wrongSignature")
		// Pastikan signature salah berbeda dari yang benar
		if wrongSignature == correctSignature {
			wrongSignature = wrongSignature + "x"
		}
		invalidBody := buildMidtransNotificationBody(t, orderID, statusCode, grossAmount, wrongSignature)
		invalid, err := adapter.VerifyWebhookSignature(context.Background(), nil, invalidBody, "")
		if err != nil {
			t.Fatalf("VerifyWebhookSignature gagal dengan signature salah: %v", err)
		}
		if invalid {
			t.Errorf("VerifyWebhookSignature mengembalikan true untuk signature yang salah")
		}
	})
}

// buildMidtransNotificationBody membuat JSON body notifikasi Midtrans untuk testing.
func buildMidtransNotificationBody(t *rapid.T, orderID, statusCode, grossAmount, signatureKey string) []byte {
	payload := map[string]string{
		"order_id":      orderID,
		"status_code":   statusCode,
		"gross_amount":  grossAmount,
		"signature_key": signatureKey,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("gagal marshal notification body: %v", err)
	}
	return body
}
