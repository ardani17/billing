// webhook_usecase_event_test.go berisi unit test untuk WebhookUsecase —
// event handler sederhana: payment.expired dan payment.failed.
// Event ini tidak memerlukan transaksi DB (pool), sehingga bisa ditest dengan pool=nil.
package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Test: ProcessWebhook — payment.expired (update link, TIDAK ubah invoice)
// =============================================================================

// TestProcessWebhook_PaymentExpired menguji bahwa event payment.expired
// mengupdate status payment link tanpa mengubah status invoice.
func TestProcessWebhook_PaymentExpired(t *testing.T) {
	s := setupWebhookUsecase()
	seedWebhookConfig(s)

	// Buat payment link dan invoice
	s.linkRepo.links["link-1"] = &domain.PaymentLink{
		ID: "link-1", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		ExternalID: "ext-expired", Amount: 300000,
		Status: domain.PaymentLinkActive,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", TenantID: "tenant-1", CustomerID: "cust-1",
		TotalAmount: 300000, PaidAmount: 0,
		Status: domain.InvoiceStatusBelumBayar,
	}
	s.linkRepo.junction["link-1"] = []string{"inv-1"}

	// Buat webhook log untuk event expired dengan signature valid
	expiredBody := map[string]interface{}{
		"id": "ext-expired", "status": "EXPIRED",
		"_headers": map[string]interface{}{
			"x-callback-token": "whsec_callback_token_12345",
		},
	}
	raw, _ := json.Marshal(expiredBody)
	s.webhookRepo.logs["wlog-exp"] = &domain.WebhookLog{
		ID: "wlog-exp", GatewayProvider: domain.GatewayXendit,
		EventType: "invoice.expired", ExternalID: "ext-expired",
		RequestBody: raw, SourceIP: "1.2.3.4",
		ProcessingStatus: domain.WebhookReceived,
	}

	err := s.uc.ProcessWebhook(context.Background(), "wlog-exp")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verifikasi payment link status berubah ke expired
	link := s.linkRepo.links["link-1"]
	if link.Status != domain.PaymentLinkExpired {
		t.Fatalf("expected link status expired, got %s", link.Status)
	}

	// Verifikasi invoice TIDAK berubah
	inv := s.invoiceRepo.invoices["inv-1"]
	if inv.Status != domain.InvoiceStatusBelumBayar {
		t.Fatalf("expected invoice status belum_bayar (tidak berubah), got %s", inv.Status)
	}

	// Verifikasi webhook log ditandai processed
	wlog := s.webhookRepo.logs["wlog-exp"]
	if wlog.ProcessingStatus != domain.WebhookProcessed {
		t.Fatalf("expected status processed, got %s", wlog.ProcessingStatus)
	}
}

// =============================================================================
// Test: ProcessWebhook — payment.failed (log kegagalan, TIDAK ubah invoice)
// =============================================================================

// TestProcessWebhook_PaymentFailed menguji bahwa event payment.failed
// mencatat kegagalan tanpa mengubah status invoice.
func TestProcessWebhook_PaymentFailed(t *testing.T) {
	s := setupWebhookUsecase()
	seedWebhookConfig(s)

	// Buat payment link dan invoice
	s.linkRepo.links["link-1"] = &domain.PaymentLink{
		ID: "link-1", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		ExternalID: "ext-failed", Amount: 300000,
		Status: domain.PaymentLinkActive,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", TenantID: "tenant-1", CustomerID: "cust-1",
		TotalAmount: 300000, PaidAmount: 0,
		Status: domain.InvoiceStatusBelumBayar,
	}
	s.linkRepo.junction["link-1"] = []string{"inv-1"}

	// Buat webhook log untuk event failed dengan signature valid
	failedBody := map[string]interface{}{
		"id": "ext-failed", "status": "FAILED",
		"payment_method": "va_bca",
		"_headers": map[string]interface{}{
			"x-callback-token": "whsec_callback_token_12345",
		},
	}
	raw, _ := json.Marshal(failedBody)
	s.webhookRepo.logs["wlog-fail"] = &domain.WebhookLog{
		ID: "wlog-fail", GatewayProvider: domain.GatewayXendit,
		EventType: "invoice.payment_failure", ExternalID: "ext-failed",
		RequestBody: raw, SourceIP: "1.2.3.4",
		ProcessingStatus: domain.WebhookReceived,
	}

	err := s.uc.ProcessWebhook(context.Background(), "wlog-fail")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verifikasi invoice TIDAK berubah
	inv := s.invoiceRepo.invoices["inv-1"]
	if inv.Status != domain.InvoiceStatusBelumBayar {
		t.Fatalf("expected invoice status belum_bayar (tidak berubah), got %s", inv.Status)
	}

	// Verifikasi payment link status TIDAK berubah (tetap active)
	link := s.linkRepo.links["link-1"]
	if link.Status != domain.PaymentLinkActive {
		t.Fatalf("expected link status active (tidak berubah), got %s", link.Status)
	}

	// Verifikasi tidak ada pembayaran yang tercatat
	if len(s.paymentRepo.payments) != 0 {
		t.Fatalf("expected 0 payments, got %d", len(s.paymentRepo.payments))
	}

	// Verifikasi webhook log ditandai processed
	wlog := s.webhookRepo.logs["wlog-fail"]
	if wlog.ProcessingStatus != domain.WebhookProcessed {
		t.Fatalf("expected status processed, got %s", wlog.ProcessingStatus)
	}
}
