// gateway_link_test.go berisi unit test untuk GatewayUsecase - manajemen tautan pembayaran.
// Adapter melakukan HTTP call ke gateway asli, sehingga GeneratePaymentLink
// akan mendapat ErrGatewayUnavailable. Ini diharapkan dan diverifikasi.
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// setupLinkTest menyiapkan usecase dengan config aktif, customer, dan invoice.
func setupLinkTest() *gwTestSetup {
	s := setupGatewayUsecase()
	enc, _ := gateway.EncryptAESGCM("xnd_production_test_key_12345", testMasterKey)
	secEnc, _ := gateway.EncryptAESGCM("whsec_test_secret_12345", testMasterKey)
	s.configRepo.configs["cfg-1"] = &domain.GatewayConfig{
		ID: "cfg-1", TenantID: "tenant-1",
		GatewayProvider: domain.GatewayXendit, IsActive: true,
		APIKeyEncrypted: enc, WebhookSecretEncrypted: secEnc,
		EnabledMethods: []string{"va_bca"}, PaymentLinkExpiryDays: 7,
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1",
		Name: "Budi Santoso", Email: "budi@example.com",
	}
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", TenantID: "tenant-1", CustomerID: "cust-1",
		InvoiceNumber: "INV-2024-001", PeriodMonth: 1, PeriodYear: 2024,
		TotalAmount: 300000, PaidAmount: 0,
		Status:  domain.InvoiceStatusBelumBayar,
		DueDate: time.Now().Add(7 * 24 * time.Hour),
	}
	return s
}

// TestGeneratePaymentLink_SingleInvoice - adapter gagal, verifikasi ErrGatewayUnavailable.
func TestGeneratePaymentLink_SingleInvoice(t *testing.T) {
	s := setupLinkTest()
	_, err := s.uc.GeneratePaymentLink(context.Background(), domain.GeneratePaymentLinkRequest{
		TenantID: "tenant-1", CustomerID: "cust-1", InvoiceIDs: []string{"inv-1"},
	})
	if !errors.Is(err, domain.ErrGatewayUnavailable) {
		t.Fatalf("expected ErrGatewayUnavailable, got %v", err)
	}
}

// TestGeneratePaymentLink_MultiInvoice - multi invoice diproses sebelum adapter.
func TestGeneratePaymentLink_MultiInvoice(t *testing.T) {
	s := setupLinkTest()
	s.invoiceRepo.invoices["inv-2"] = &domain.Invoice{
		ID: "inv-2", TenantID: "tenant-1", CustomerID: "cust-1",
		InvoiceNumber: "INV-2024-002", PeriodMonth: 2, PeriodYear: 2024,
		TotalAmount: 200000, PaidAmount: 50000,
		Status:  domain.InvoiceStatusBayarSebagian,
		DueDate: time.Now().Add(7 * 24 * time.Hour),
	}
	_, err := s.uc.GeneratePaymentLink(context.Background(), domain.GeneratePaymentLinkRequest{
		TenantID: "tenant-1", CustomerID: "cust-1",
		InvoiceIDs: []string{"inv-1", "inv-2"},
	})
	if !errors.Is(err, domain.ErrGatewayUnavailable) {
		t.Fatalf("expected ErrGatewayUnavailable, got %v", err)
	}
}

// TestGeneratePaymentLink_SkipsLunasBatal - invoice lunas/batal tidak diproses.
func TestGeneratePaymentLink_SkipsLunasBatal(t *testing.T) {
	s := setupLinkTest()
	s.invoiceRepo.invoices["inv-1"].Status = domain.InvoiceStatusLunas
	_, err := s.uc.GeneratePaymentLink(context.Background(), domain.GeneratePaymentLinkRequest{
		TenantID: "tenant-1", CustomerID: "cust-1", InvoiceIDs: []string{"inv-1"},
	})
	if err == nil {
		t.Fatal("expected error untuk invoice lunas, got nil")
	}
	if errors.Is(err, domain.ErrGatewayUnavailable) {
		t.Fatal("seharusnya error sebelum adapter, bukan ErrGatewayUnavailable")
	}
}

// TestGetCustomerPaymentLink_ActiveLink - mengembalikan link aktif dengan invoice.
func TestGetCustomerPaymentLink_ActiveLink(t *testing.T) {
	s := setupLinkTest()
	s.linkRepo.links["link-1"] = &domain.PaymentLink{
		ID: "link-1", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, Amount: 300000,
		PaymentURL: "https://checkout.xendit.co/test",
		Status:     domain.PaymentLinkActive,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}
	s.linkRepo.junction["link-1"] = []string{"inv-1"}
	resp, err := s.uc.GetCustomerPaymentLink(context.Background(), "cust-1")
	if err != nil {
		t.Fatalf("GetCustomerPaymentLink gagal: %v", err)
	}
	if resp == nil || resp.PaymentLink.ID != "link-1" {
		t.Fatal("expected link-1 dalam response")
	}
	if resp.TotalArrears != 300000 {
		t.Fatalf("expected total_arrears 300000, got %d", resp.TotalArrears)
	}
	if len(resp.Invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(resp.Invoices))
	}
}

// TestGetCustomerPaymentLink_NoActiveLink - nil saat tidak ada link aktif.
func TestGetCustomerPaymentLink_NoActiveLink(t *testing.T) {
	s := setupLinkTest()
	resp, err := s.uc.GetCustomerPaymentLink(context.Background(), "cust-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Fatal("expected nil response, got non-nil")
	}
}

// TestRegeneratePaymentLink_ExpiresOldAndCreatesNew - expire link lama.
func TestRegeneratePaymentLink_ExpiresOldAndCreatesNew(t *testing.T) {
	s := setupLinkTest()
	s.linkRepo.links["link-old"] = &domain.PaymentLink{
		ID: "link-old", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		Amount: 300000, Status: domain.PaymentLinkActive,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	_, err := s.uc.RegeneratePaymentLink(context.Background(), "cust-1")
	// Buat baru gagal di adapter
	if !errors.Is(err, domain.ErrGatewayUnavailable) {
		t.Fatalf("expected ErrGatewayUnavailable, got %v", err)
	}
	// Link lama harus sudah di-expire
	if s.linkRepo.links["link-old"].Status != domain.PaymentLinkExpired {
		t.Fatal("expected link lama status expired")
	}
}

// TestWalledGardenPaymentInfo_NoActiveLink - buat on-demand gagal, info tetap ada.
func TestWalledGardenPaymentInfo_NoActiveLink(t *testing.T) {
	s := setupLinkTest()
	info, err := s.uc.GetWalledGardenPaymentInfo(context.Background(), "cust-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected info, got nil")
	}
	if info.TotalArrears != 300000 {
		t.Fatalf("expected total_arrears 300000, got %d", info.TotalArrears)
	}
	if info.CustomerName != "Budi Santoso" {
		t.Fatalf("expected Budi Santoso, got %s", info.CustomerName)
	}
	// PaymentURL kosong karena buat gagal
	if info.PaymentURL != "" {
		t.Fatalf("expected empty payment_url, got %s", info.PaymentURL)
	}
}

// TestWalledGardenPaymentInfo_ExpiredLink - link expired di-regenerasi.
func TestWalledGardenPaymentInfo_ExpiredLink(t *testing.T) {
	s := setupLinkTest()
	s.linkRepo.links["link-exp"] = &domain.PaymentLink{
		ID: "link-exp", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		Amount: 300000, Status: domain.PaymentLinkActive,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // sudah expired
	}
	info, err := s.uc.GetWalledGardenPaymentInfo(context.Background(), "cust-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected info, got nil")
	}
	if info.TotalArrears != 300000 {
		t.Fatalf("expected total_arrears 300000, got %d", info.TotalArrears)
	}
}

// TestSyncPaymentLinkAmount_ExpiresAndRegenerates - sync expire link lama.
func TestSyncPaymentLinkAmount_ExpiresAndRegenerates(t *testing.T) {
	s := setupLinkTest()
	s.linkRepo.links["link-sync"] = &domain.PaymentLink{
		ID: "link-sync", TenantID: "tenant-1", CustomerID: "cust-1",
		GatewayProvider: domain.GatewayXendit, GatewayConfigID: "cfg-1",
		Amount: 300000, Status: domain.PaymentLinkActive,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	s.linkRepo.junction["link-sync"] = []string{"inv-1"}
	err := s.uc.SyncPaymentLinkAmount(context.Background(), "inv-1")
	// Buat baru gagal di adapter, tapi link lama harus di-expire
	if err != nil && !errors.Is(err, domain.ErrGatewayUnavailable) {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.linkRepo.links["link-sync"].Status != domain.PaymentLinkExpired {
		t.Fatal("expected link lama status expired setelah sync")
	}
}
