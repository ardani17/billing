// gateway_usecase_test.go berisi unit test untuk GatewayUsecase — config management.
package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// =============================================================================
// Test: CreateConfig — input valid (enkripsi key, simpan config)
// =============================================================================

// TestCreateConfig_ValidInput menguji pembuatan config dengan input valid.
func TestCreateConfig_ValidInput(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca", "qris"},
	}

	config, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	// Verifikasi config dibuat dengan benar
	if config.GatewayProvider != domain.GatewayXendit {
		t.Fatalf("expected provider xendit, got %s", config.GatewayProvider)
	}
	if !config.IsActive {
		t.Fatal("expected is_active true")
	}
	if config.PaymentLinkExpiryDays != 7 {
		t.Fatalf("expected expiry_days 7, got %d", config.PaymentLinkExpiryDays)
	}

	// Verifikasi API key terenkripsi (bukan plaintext)
	if config.APIKeyEncrypted == req.APIKey {
		t.Fatal("API key tidak terenkripsi")
	}
	if config.APIKeyEncrypted == "" {
		t.Fatal("API key encrypted kosong")
	}

	// Verifikasi API key bisa didekripsi kembali
	decrypted, err := gateway.DecryptAESGCM(config.APIKeyEncrypted, testMasterKey)
	if err != nil {
		t.Fatalf("gagal dekripsi API key: %v", err)
	}
	if decrypted != req.APIKey {
		t.Fatalf("expected decrypted key %s, got %s", req.APIKey, decrypted)
	}

	// Verifikasi webhook secret terenkripsi
	decryptedSecret, err := gateway.DecryptAESGCM(config.WebhookSecretEncrypted, testMasterKey)
	if err != nil {
		t.Fatalf("gagal dekripsi webhook secret: %v", err)
	}
	if decryptedSecret != req.WebhookSecret {
		t.Fatalf("expected decrypted secret %s, got %s", req.WebhookSecret, decryptedSecret)
	}

	// Verifikasi API key masked
	if !strings.HasSuffix(config.APIKeyMasked, "2345") {
		t.Fatalf("expected masked key ending with 2345, got %s", config.APIKeyMasked)
	}

	// Verifikasi enabled_methods tersimpan
	if len(config.EnabledMethods) != 2 {
		t.Fatalf("expected 2 enabled methods, got %d", len(config.EnabledMethods))
	}

	// Verifikasi config tersimpan di repo
	if len(s.configRepo.configs) != 1 {
		t.Fatalf("expected 1 config in repo, got %d", len(s.configRepo.configs))
	}
}

// =============================================================================
// Test: CreateConfig — custom expiry days
// =============================================================================

// TestCreateConfig_CustomExpiryDays menguji pembuatan config dengan expiry days kustom.
func TestCreateConfig_CustomExpiryDays(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	expiryDays := 14
	req := domain.CreateGatewayConfigRequest{
		GatewayProvider:       "midtrans",
		APIKey:                "SB-Mid-server-test_key_12345",
		WebhookSecret:         "whsec_midtrans_secret_12345",
		EnabledMethods:        []string{"va_bca", "ewallet_gopay"},
		PaymentLinkExpiryDays: &expiryDays,
	}

	config, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	if config.PaymentLinkExpiryDays != 14 {
		t.Fatalf("expected expiry_days 14, got %d", config.PaymentLinkExpiryDays)
	}
}

// =============================================================================
// Test: CreateConfig — invalid methods (return ErrInvalidEnabledMethods)
// =============================================================================

// TestCreateConfig_InvalidMethods menguji error saat enabled_methods tidak valid.
func TestCreateConfig_InvalidMethods(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca", "metode_tidak_valid"},
	}

	_, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidEnabledMethods) {
		t.Fatalf("expected ErrInvalidEnabledMethods, got %v", err)
	}

	// Verifikasi tidak ada config yang tersimpan
	if len(s.configRepo.configs) != 0 {
		t.Fatalf("expected 0 configs, got %d", len(s.configRepo.configs))
	}
}

// =============================================================================
// Test: CreateConfig — duplicate provider (return ErrGatewayConfigDuplicate)
// =============================================================================

// TestCreateConfig_DuplicateProvider menguji error saat provider sudah ada.
func TestCreateConfig_DuplicateProvider(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	// Buat config pertama
	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}
	_, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig pertama gagal: %v", err)
	}

	// Coba buat config kedua dengan provider yang sama
	req2 := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_another_key_123",
		WebhookSecret:   "whsec_another_secret_12345",
		EnabledMethods:  []string{"qris"},
	}
	_, err = s.uc.CreateConfig(ctx, "tenant-1", req2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrGatewayConfigDuplicate) {
		t.Fatalf("expected ErrGatewayConfigDuplicate, got %v", err)
	}
}
