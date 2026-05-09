// gateway_usecase_testcfg_test.go berisi unit test untuk TestConfig pada GatewayUsecase.
package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// =============================================================================
// Tes: TestConfig - sukses (config ditemukan, key bisa didekripsi)
// =============================================================================

// Karena adapter melakukan HTTP call ke gateway, kita verifikasi bahwa
func TestTestConfig_Success(t *testing.T) {
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

	result, err := s.uc.TestConfig(ctx, created.ID)
	if err != nil {
		t.Fatalf("TestConfig gagal: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

// =============================================================================
// Tes: TestConfig - config tidak ditemukan
// =============================================================================

// TestTestConfig_NotFound menguji TestConfig dengan config yang tidak ada.
func TestTestConfig_NotFound(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	_, err := s.uc.TestConfig(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrGatewayConfigNotFound) {
		t.Fatalf("expected ErrGatewayConfigNotFound, got %v", err)
	}
}

// =============================================================================
// Tes: TestConfig - dekripsi gagal (key corrupt)
// =============================================================================

// TestTestConfig_DecryptionFailed menguji TestConfig saat API key corrupt.
func TestTestConfig_DecryptionFailed(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	// Simpan config langsung ke repo dengan encrypted key yang corrupt
	s.configRepo.configs["cfg-corrupt"] = &domain.GatewayConfig{
		ID:                     "cfg-corrupt",
		TenantID:               "tenant-1",
		GatewayProvider:        domain.GatewayXendit,
		IsActive:               true,
		APIKeyEncrypted:        "data-corrupt-bukan-base64-valid!!!",
		WebhookSecretEncrypted: "secret-corrupt",
		EnabledMethods:         []string{"va_bca"},
		PaymentLinkExpiryDays:  7,
	}

	result, err := s.uc.TestConfig(ctx, "cfg-corrupt")
	if err != nil {
		t.Fatalf("TestConfig seharusnya tidak return error, got %v", err)
	}

	// Result harus menunjukkan kegagalan dekripsi
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Success {
		t.Fatal("expected success=false untuk key corrupt")
	}
	if result.ErrorCode != "decryption_failed" {
		t.Fatalf("expected error_code decryption_failed, got %s", result.ErrorCode)
	}
}

// =============================================================================
// Tes: CreateConfig - tenant berbeda bisa punya provider yang sama
// =============================================================================

// TestCreateConfig_DifferentTenantSameProvider menguji bahwa tenant berbeda
// bisa memiliki provider yang sama.
func TestCreateConfig_DifferentTenantSameProvider(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_test_key_12345",
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}

	// Tenant 1
	_, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig tenant-1 gagal: %v", err)
	}

	// Tenant 2 dengan provider yang sama - harus berhasil
	req2 := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          "xnd_production_other_key_67890",
		WebhookSecret:   "whsec_other_secret_67890",
		EnabledMethods:  []string{"qris"},
	}
	_, err = s.uc.CreateConfig(ctx, "tenant-2", req2)
	if err != nil {
		t.Fatalf("CreateConfig tenant-2 gagal: %v", err)
	}

	if len(s.configRepo.configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(s.configRepo.configs))
	}
}

// =============================================================================
// =============================================================================

func TestCreateConfig_MidtransInvalidMethod(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "midtrans",
		APIKey:          "SB-Mid-server-test_key_12345",
		WebhookSecret:   "whsec_midtrans_secret_12345",
		EnabledMethods:  []string{"va_bca", "ewallet_ovo"}, // ewallet_ovo tidak valid untuk Midtrans
	}

	_, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidEnabledMethods) {
		t.Fatalf("expected ErrInvalidEnabledMethods, got %v", err)
	}
}

// =============================================================================
// =============================================================================

func TestUpdateConfig_InvalidMethods(t *testing.T) {
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

	updateReq := domain.UpdateGatewayConfigRequest{
		EnabledMethods: []string{"metode_tidak_valid"},
	}
	_, err = s.uc.UpdateConfig(ctx, created.ID, updateReq)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidEnabledMethods) {
		t.Fatalf("expected ErrInvalidEnabledMethods, got %v", err)
	}
}

// =============================================================================
// =============================================================================

// TestEncryptionRoundTrip_InUsecase memverifikasi bahwa key yang disimpan
// melalui CreateConfig bisa didekripsi kembali.
func TestEncryptionRoundTrip_InUsecase(t *testing.T) {
	s := setupGatewayUsecase()
	ctx := context.Background()

	originalKey := "xnd_production_very_long_api_key_for_testing"
	req := domain.CreateGatewayConfigRequest{
		GatewayProvider: "xendit",
		APIKey:          originalKey,
		WebhookSecret:   "whsec_test_secret_12345",
		EnabledMethods:  []string{"va_bca"},
	}

	created, err := s.uc.CreateConfig(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateConfig gagal: %v", err)
	}

	// Dekripsi dari repo
	stored := s.configRepo.configs[created.ID]
	decrypted, err := gateway.DecryptAESGCM(stored.APIKeyEncrypted, testMasterKey)
	if err != nil {
		t.Fatalf("gagal dekripsi: %v", err)
	}
	if decrypted != originalKey {
		t.Fatalf("round-trip gagal: expected %s, got %s", originalKey, decrypted)
	}
}
