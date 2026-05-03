// gateway_usecase_config_test.go berisi unit test untuk UpdateConfig, DeactivateConfig,
// ListConfigs pada GatewayUsecase.
package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// TestUpdateConfig_PartialUpdate menguji update parsial config.
func TestUpdateConfig_PartialUpdate(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	// Buat config awal
	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca", "qris"},
	}
	created, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	// Update hanya enabled_methods
	updateReq := domain.UpdateGatewayConfigRequest{
		EnabledMethods: []string{"va_bca", "qris", "ewallet_ovo"},
	}
	updated, err := s.uc.UpdateConfig(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateConfig gagal: %v", err)
	}

	// Verifikasi enabled_methods diperbarui
	if len(updated.EnabledMethods) != 3 {
		t.Fatalf("expected 3 methods, got %d", len(updated.EnabledMethods))
	}

	// Verifikasi API key tidak berubah (masih bisa didekripsi ke nilai awal)
	decrypted, err := gateway.DecryptAESGCM(updated.APIKeyEncrypted, testMasterKey)
	if err != nil {
		t.Fatalf("gagal dekripsi API key: %v", err)
	}
	if decrypted != req.APIKey {
		t.Fatalf("API key berubah: expected %s, got %s", req.APIKey, decrypted)
	}
}

// TestUpdateConfig_NewAPIKey menguji update API key baru.
func TestUpdateConfig_NewAPIKey(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	// Buat config awal
	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}
	created, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	// Update API key
	newKey := "xnd_production_new_key_67890"
	updateReq := domain.UpdateGatewayConfigRequest{
		APIKey: newKey,
	}
	updated, err := s.uc.UpdateConfig(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateConfig gagal: %v", err)
	}

	// Verifikasi API key baru terenkripsi
	decrypted, err := gateway.DecryptAESGCM(updated.APIKeyEncrypted, testMasterKey)
	if err != nil {
		t.Fatalf("gagal dekripsi API key baru: %v", err)
	}
	if decrypted != newKey {
		t.Fatalf("expected new key %s, got %s", newKey, decrypted)
	}
}

// TestUpdateConfig_ExpiryDays menguji update payment_link_expiry_days.
func TestUpdateConfig_ExpiryDays(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}
	created, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	newExpiry := 21
	updateReq := domain.UpdateGatewayConfigRequest{
		PaymentLinkExpiryDays: &newExpiry,
	}
	updated, err := s.uc.UpdateConfig(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateConfig gagal: %v", err)
	}

	if updated.PaymentLinkExpiryDays != 21 {
		t.Fatalf("expected expiry 21, got %d", updated.PaymentLinkExpiryDays)
	}
}

// TestDeactivateConfig_Success menguji deaktivasi config berhasil.
func TestDeactivateConfig_Success(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	// Buat config
	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}
	created, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	// Deaktivasi
	err = s.uc.DeactivateConfig(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeactivateConfig gagal: %v", err)
	}

	// Verifikasi config sudah tidak aktif
	stored := s.configRepo.configs[created.ID]
	if stored.IsActive {
		t.Fatal("expected is_active false setelah deaktivasi")
	}
}

// TestDeactivateConfig_NotFound menguji deaktivasi config yang tidak ada.
func TestDeactivateConfig_NotFound(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	err := s.uc.DeactivateConfig(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrGatewayConfigNotFound) {
		t.Fatalf("expected ErrGatewayConfigNotFound, got %v", err)
	}
}

// TestListConfigs_MaskedKeys menguji bahwa ListConfigs mengembalikan API key yang di-mask.
func TestListConfigs_MaskedKeys(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	// Buat 2 config untuk tenant yang sama
	req1 := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_ABCD",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}
	_, err := s.uc.CreateConfig(ctx, "tenant-1", req1)
	if err != nil {
		t.Fatalf("CreateConfig xendit gagal: %v", err)
	}

	req2 := domain.CreateGatewayConfigRequest{
		GatewayProvider: "midtrans",
		APIKey:          "SB-Mid-server-test_key_WXYZ",
		WebhookSecret:   "whsec_midtrans_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}
	_, err = s.uc.CreateConfig(ctx, "tenant-1", req2)
	if err != nil {
		t.Fatalf("CreateConfig midtrans gagal: %v", err)
	}

	// List configs
	configs, err := s.uc.ListConfigs(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ListConfigs gagal: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}

	// Verifikasi semua config memiliki API key yang di-mask
	for _, cfg := range configs {
		if cfg.APIKeyMasked == "" {
			t.Fatal("expected non-empty masked key")
		}
		if !strings.Contains(cfg.APIKeyMasked, "****") {
			t.Fatalf("expected masked key with asterisks, got %s", cfg.APIKeyMasked)
		}
	}
}

// TestListConfigs_EmptyTenant menguji ListConfigs untuk tenant tanpa config.
func TestListConfigs_EmptyTenant(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	configs, err := s.uc.ListConfigs(ctx, "tenant-empty")
	if err != nil {
		t.Fatalf("ListConfigs gagal: %v", err)
	}

	if len(configs) != 0 {
		t.Fatalf("expected 0 configs, got %d", len(configs))
	}
}
