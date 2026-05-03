// gateway_usecase.go berisi struct GatewayUsecase, constructor, dan method CRUD konfigurasi gateway.
// Method payment link ada di file terpisah (gateway_link.go).
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// GatewayUsecase mengimplementasikan business logic untuk konfigurasi gateway
// dan manajemen payment link.
type GatewayUsecase struct {
	configRepo   domain.GatewayConfigRepository
	linkRepo     domain.PaymentLinkRepository
	invoiceRepo  domain.InvoiceRepository
	customerRepo domain.CustomerRepository
	settingsRepo domain.BillingSettingsRepository
	pool         *pgxpool.Pool
	queueClient  *asynq.Client
	masterKey    []byte // AES-256 master key untuk enkripsi API keys
	logger       zerolog.Logger
}

// NewGatewayUsecase membuat instance baru GatewayUsecase.
func NewGatewayUsecase(
	configRepo domain.GatewayConfigRepository,
	linkRepo domain.PaymentLinkRepository,
	invoiceRepo domain.InvoiceRepository,
	customerRepo domain.CustomerRepository,
	settingsRepo domain.BillingSettingsRepository,
	pool *pgxpool.Pool,
	queueClient *asynq.Client,
	masterKey []byte,
	logger zerolog.Logger,
) *GatewayUsecase {
	return &GatewayUsecase{
		configRepo:   configRepo,
		linkRepo:     linkRepo,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
		settingsRepo: settingsRepo,
		pool:         pool,
		queueClient:  queueClient,
		masterKey:    masterKey,
		logger:       logger,
	}
}

// CreateConfig membuat konfigurasi gateway baru untuk tenant.
func (uc *GatewayUsecase) CreateConfig(ctx context.Context, tenantID string, req domain.CreateGatewayConfigRequest) (*domain.GatewayConfig, error) {
	provider := domain.GatewayProvider(req.GatewayProvider)

	// Validasi enabled_methods sesuai provider
	if err := domain.ValidateEnabledMethods(provider, req.EnabledMethods); err != nil {
		return nil, err
	}

	// Cek duplikat provider aktif
	exists, err := uc.configRepo.ExistsByProvider(ctx, tenantID, provider)
	if err != nil {
		return nil, fmt.Errorf("gagal cek duplikat provider: %w", err)
	}
	if exists {
		return nil, domain.ErrGatewayConfigDuplicate
	}

	// Enkripsi API key dan webhook secret
	apiKeyEnc, err := gateway.EncryptAESGCM(req.APIKey, uc.masterKey)
	if err != nil {
		return nil, fmt.Errorf("%w: api_key", domain.ErrEncryptionFailed)
	}
	webhookSecretEnc, err := gateway.EncryptAESGCM(req.WebhookSecret, uc.masterKey)
	if err != nil {
		return nil, fmt.Errorf("%w: webhook_secret", domain.ErrEncryptionFailed)
	}

	expiryDays := 7
	if req.PaymentLinkExpiryDays != nil {
		expiryDays = *req.PaymentLinkExpiryDays
	}

	config := &domain.GatewayConfig{
		TenantID:               tenantID,
		GatewayProvider:        provider,
		IsActive:               true,
		APIKeyEncrypted:        apiKeyEnc,
		WebhookSecretEncrypted: webhookSecretEnc,
		EnabledMethods:         req.EnabledMethods,
		PaymentLinkExpiryDays:  expiryDays,
	}

	created, err := uc.configRepo.Create(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat konfigurasi gateway: %w", err)
	}
	created.APIKeyMasked = gateway.MaskAPIKey(req.APIKey)

	uc.logger.Info().
		Str("tenant_id", tenantID).
		Str("provider", string(provider)).
		Msg("konfigurasi gateway berhasil dibuat")

	return created, nil
}

// UpdateConfig memperbarui konfigurasi gateway. Hanya field yang diisi yang diperbarui.
func (uc *GatewayUsecase) UpdateConfig(ctx context.Context, id string, req domain.UpdateGatewayConfigRequest) (*domain.GatewayConfig, error) {
	existing, err := uc.configRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrGatewayConfigNotFound
	}

	// Validasi enabled_methods jika diisi
	if len(req.EnabledMethods) > 0 {
		if err := domain.ValidateEnabledMethods(existing.GatewayProvider, req.EnabledMethods); err != nil {
			return nil, err
		}
		existing.EnabledMethods = req.EnabledMethods
	}
	// Enkripsi API key baru jika diisi
	if req.APIKey != "" {
		enc, err := gateway.EncryptAESGCM(req.APIKey, uc.masterKey)
		if err != nil {
			return nil, fmt.Errorf("%w: api_key", domain.ErrEncryptionFailed)
		}
		existing.APIKeyEncrypted = enc
	}
	// Enkripsi webhook secret baru jika diisi
	if req.WebhookSecret != "" {
		enc, err := gateway.EncryptAESGCM(req.WebhookSecret, uc.masterKey)
		if err != nil {
			return nil, fmt.Errorf("%w: webhook_secret", domain.ErrEncryptionFailed)
		}
		existing.WebhookSecretEncrypted = enc
	}
	if req.PaymentLinkExpiryDays != nil {
		existing.PaymentLinkExpiryDays = *req.PaymentLinkExpiryDays
	}

	updated, err := uc.configRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui konfigurasi gateway: %w", err)
	}
	uc.logger.Info().Str("config_id", id).Msg("konfigurasi gateway berhasil diperbarui")
	return updated, nil
}

// DeactivateConfig menonaktifkan konfigurasi gateway (soft delete).
func (uc *GatewayUsecase) DeactivateConfig(ctx context.Context, id string) error {
	if err := uc.configRepo.Deactivate(ctx, id); err != nil {
		return fmt.Errorf("gagal menonaktifkan konfigurasi gateway: %w", err)
	}
	uc.logger.Info().Str("config_id", id).Msg("konfigurasi gateway dinonaktifkan")
	return nil
}

// ListConfigs mengambil semua konfigurasi gateway untuk tenant (API key di-mask).
func (uc *GatewayUsecase) ListConfigs(ctx context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	configs, err := uc.configRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar konfigurasi gateway: %w", err)
	}
	for _, cfg := range configs {
		plainKey, err := gateway.DecryptAESGCM(cfg.APIKeyEncrypted, uc.masterKey)
		if err != nil {
			cfg.APIKeyMasked = "****"
			continue
		}
		cfg.APIKeyMasked = gateway.MaskAPIKey(plainKey)
	}
	return configs, nil
}

// TestConfig menguji koneksi ke payment gateway dengan timeout 10 detik.
func (uc *GatewayUsecase) TestConfig(ctx context.Context, id string) (*domain.GatewayTestResult, error) {
	config, err := uc.configRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrGatewayConfigNotFound
	}

	// Dekripsi API key untuk adapter
	plainKey, err := gateway.DecryptAESGCM(config.APIKeyEncrypted, uc.masterKey)
	if err != nil {
		return &domain.GatewayTestResult{
			Success: false, ErrorCode: "decryption_failed",
			ErrorMessage: "gagal mendekripsi API key",
		}, nil
	}

	adapter, err := gateway.NewAdapter(config.GatewayProvider, plainKey)
	if err != nil {
		return &domain.GatewayTestResult{
			Success: false, ErrorCode: "invalid_provider",
			ErrorMessage: err.Error(),
		}, nil
	}

	// Panggil TestConnection dengan timeout 10 detik
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := adapter.TestConnection(testCtx)
	if err != nil {
		return &domain.GatewayTestResult{
			Success: false, ErrorCode: "gateway_unavailable",
			ErrorMessage: err.Error(),
		}, nil
	}
	return result, nil
}
