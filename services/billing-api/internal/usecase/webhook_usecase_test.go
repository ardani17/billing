// webhook_usecase_test.go berisi unit test untuk WebhookUsecase — early-exit path.
// Fokus pada: external_id not found, invalid signature, duplicate.
// pool=nil sehingga payment.paid flow (butuh transaksi DB) tidak ditest di sini.
package usecase

import (
	"context"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Test: ProcessWebhook — external_id tidak ditemukan
// =============================================================================

// TestProcessWebhook_ExternalIDNotFound menguji bahwa webhook dengan external_id
// yang tidak ada di payment_links ditandai failed dengan "payment_link_not_found".
func TestProcessWebhook_ExternalIDNotFound(t *testing.T) {
	s := setupWebhookUsecase()

	// Buat webhook log dengan external_id yang tidak ada di linkRepo
	logID := seedWebhookLog(s, "ext-tidak-ada", map[string]interface{}{
		"id": "ext-tidak-ada", "status": "PAID",
	})

	err := s.uc.ProcessWebhook(context.Background(), logID)
	if err != nil {
		t.Fatalf("expected nil error (early exit), got %v", err)
	}

	// Verifikasi webhook log ditandai failed
	wlog := s.webhookRepo.logs[logID]
	if wlog.ProcessingStatus != domain.WebhookFailed {
		t.Fatalf("expected status failed, got %s", wlog.ProcessingStatus)
	}
	if wlog.ErrorMessage != "payment_link_not_found" {
		t.Fatalf("expected error_message 'payment_link_not_found', got '%s'", wlog.ErrorMessage)
	}

	// Verifikasi tidak ada pembayaran yang tercatat
	if len(s.paymentRepo.payments) != 0 {
		t.Fatalf("expected 0 payments, got %d", len(s.paymentRepo.payments))
	}
}

// =============================================================================
// Test: ProcessWebhook — signature tidak valid
// =============================================================================

// TestProcessWebhook_InvalidSignature menguji bahwa webhook dengan signature
// tidak valid ditandai failed dan signature_valid=false.
func TestProcessWebhook_InvalidSignature(t *testing.T) {
	s := setupWebhookUsecase()
	seedWebhookConfig(s)

	// Buat payment link yang cocok dengan external_id
	s.linkRepo.links["link-1"] = &domain.PaymentLink{
		ID: "link-1", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		ExternalID: "ext-123", Amount: 300000,
		Status: domain.PaymentLinkActive,
	}

	// Buat webhook log TANPA _headers (x-callback-token kosong → signature invalid)
	logID := seedWebhookLog(s, "ext-123", map[string]interface{}{
		"id": "ext-123", "status": "PAID", "amount": 300000,
	})

	err := s.uc.ProcessWebhook(context.Background(), logID)
	if err != nil {
		t.Fatalf("expected nil error (early exit), got %v", err)
	}

	// Verifikasi webhook log ditandai failed
	wlog := s.webhookRepo.logs[logID]
	if wlog.ProcessingStatus != domain.WebhookFailed {
		t.Fatalf("expected status failed, got %s", wlog.ProcessingStatus)
	}
	if wlog.ErrorMessage != "signature_invalid" {
		t.Fatalf("expected error_message 'signature_invalid', got '%s'", wlog.ErrorMessage)
	}

	// Verifikasi signature_valid = false
	if wlog.SignatureValid == nil || *wlog.SignatureValid != false {
		t.Fatal("expected signature_valid = false")
	}

	// Verifikasi tidak ada pembayaran yang tercatat
	if len(s.paymentRepo.payments) != 0 {
		t.Fatalf("expected 0 payments, got %d", len(s.paymentRepo.payments))
	}
}

// =============================================================================
// Test: ProcessWebhook — duplikat (sudah diproses sebelumnya)
// =============================================================================

// TestProcessWebhook_Duplicate menguji bahwa webhook duplikat ditandai duplicate
// tanpa pemrosesan ulang.
func TestProcessWebhook_Duplicate(t *testing.T) {
	s := setupWebhookUsecase()
	seedWebhookConfig(s)

	// Buat payment link
	s.linkRepo.links["link-1"] = &domain.PaymentLink{
		ID: "link-1", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		ExternalID: "ext-123", Amount: 300000,
		Status: domain.PaymentLinkActive,
	}

	// Buat webhook log dengan _headers yang valid (signature cocok)
	body := map[string]interface{}{
		"id": "ext-123", "status": "PAID", "amount": 300000,
		"_headers": map[string]interface{}{
			"x-callback-token": "whsec_callback_token_12345",
		},
	}
	logID := seedWebhookLog(s, "ext-123", body)

	// Tandai bahwa webhook ini sudah pernah diproses
	s.webhookRepo.alreadyProcessed["ext-123|payment.paid"] = true

	err := s.uc.ProcessWebhook(context.Background(), logID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verifikasi webhook log ditandai duplicate
	wlog := s.webhookRepo.logs[logID]
	if wlog.ProcessingStatus != domain.WebhookDuplicate {
		t.Fatalf("expected status duplicate, got %s", wlog.ProcessingStatus)
	}

	// Verifikasi tidak ada pembayaran yang tercatat
	if len(s.paymentRepo.payments) != 0 {
		t.Fatalf("expected 0 payments, got %d", len(s.paymentRepo.payments))
	}
}
