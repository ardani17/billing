package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GatewayConfigRepo mengimplementasikan domain.GatewayConfigRepository
// dengan membungkus sqlc-generated Queries.
type GatewayConfigRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi payment_gateway_configs.
	queries *Queries

	// pool digunakan untuk operasi yang membutuhkan koneksi langsung.
	pool *pgxpool.Pool
}

// NewGatewayConfigRepo membuat instance baru GatewayConfigRepo.
func NewGatewayConfigRepo(queries *Queries, pool *pgxpool.Pool) *GatewayConfigRepo {
	return &GatewayConfigRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper: mapping sqlc PaymentGatewayConfig → domain.GatewayConfig ---

// mapGatewayConfigRow memetakan PaymentGatewayConfig (sqlc model) ke domain.GatewayConfig.
// Konversi: pgtype.UUID → string, pgtype.Timestamptz → time.Time,
// []byte (JSONB) → []string, int32 → int.
func mapGatewayConfigRow(row PaymentGatewayConfig) (*domain.GatewayConfig, error) {
	// Konversi enabled_methods dari JSONB ([]byte) ke []string
	var methods []string
	if len(row.EnabledMethods) > 0 {
		if err := json.Unmarshal(row.EnabledMethods, &methods); err != nil {
			return nil, fmt.Errorf("repository: gagal unmarshal enabled_methods: %w", err)
		}
	}

	return &domain.GatewayConfig{
		ID:                     uuidToString(row.ID),
		TenantID:               uuidToString(row.TenantID),
		GatewayProvider:        domain.GatewayProvider(row.GatewayProvider),
		IsActive:               row.IsActive,
		APIKeyEncrypted:        row.ApiKeyEncrypted,
		WebhookSecretEncrypted: row.WebhookSecretEncrypted,
		EnabledMethods:         methods,
		PaymentLinkExpiryDays:  int(row.PaymentLinkExpiryDays),
		CreatedAt:              timestamptzToTime(row.CreatedAt),
		UpdatedAt:              timestamptzToTime(row.UpdatedAt),
	}, nil
}

// --- Implementasi domain.GatewayConfigRepository ---

// Create membuat konfigurasi gateway baru dan mengembalikan konfigurasi yang dibuat.
func (r *GatewayConfigRepo) Create(ctx context.Context, config *domain.GatewayConfig) (*domain.GatewayConfig, error) {
	// Konversi enabled_methods dari []string ke []byte (JSONB)
	methodsJSON, err := json.Marshal(config.EnabledMethods)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal enabled_methods: %w", err)
	}

	row, err := r.queries.CreateGatewayConfig(ctx, CreateGatewayConfigParams{
		TenantID:               stringToUUID(config.TenantID),
		GatewayProvider:        string(config.GatewayProvider),
		ApiKeyEncrypted:        config.APIKeyEncrypted,
		WebhookSecretEncrypted: config.WebhookSecretEncrypted,
		EnabledMethods:         methodsJSON,
		PaymentLinkExpiryDays:  int32(config.PaymentLinkExpiryDays),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat gateway config: %w", err)
	}
	return mapGatewayConfigRow(row)
}

// GetByID mengambil konfigurasi gateway berdasarkan ID (tenant-scoped via RLS).
func (r *GatewayConfigRepo) GetByID(ctx context.Context, id string) (*domain.GatewayConfig, error) {
	row, err := r.queries.GetGatewayConfigByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGatewayConfigNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil gateway config by ID: %w", err)
	}
	return mapGatewayConfigRow(row)
}

// Update memperbarui konfigurasi gateway dan mengembalikan konfigurasi yang diperbarui.
func (r *GatewayConfigRepo) Update(ctx context.Context, config *domain.GatewayConfig) (*domain.GatewayConfig, error) {
	// Konversi enabled_methods dari []string ke []byte (JSONB)
	methodsJSON, err := json.Marshal(config.EnabledMethods)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal enabled_methods: %w", err)
	}

	row, err := r.queries.UpdateGatewayConfig(ctx, UpdateGatewayConfigParams{
		ID:                     stringToUUID(config.ID),
		ApiKeyEncrypted:        config.APIKeyEncrypted,
		WebhookSecretEncrypted: config.WebhookSecretEncrypted,
		EnabledMethods:         methodsJSON,
		PaymentLinkExpiryDays:  int32(config.PaymentLinkExpiryDays),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGatewayConfigNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui gateway config: %w", err)
	}
	return mapGatewayConfigRow(row)
}

// Deactivate menonaktifkan konfigurasi gateway (soft delete, set is_active=false).
func (r *GatewayConfigRepo) Deactivate(ctx context.Context, id string) error {
	err := r.queries.DeactivateGatewayConfig(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menonaktifkan gateway config: %w", err)
	}
	return nil
}

// ListByTenant mengambil semua konfigurasi gateway untuk tenant tertentu.
func (r *GatewayConfigRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	rows, err := r.queries.ListGatewayConfigsByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar gateway config: %w", err)
	}
	configs := make([]*domain.GatewayConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := mapGatewayConfigRow(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// GetActiveByTenant mengambil konfigurasi gateway aktif untuk tenant tertentu.
func (r *GatewayConfigRepo) GetActiveByTenant(ctx context.Context, tenantID string) ([]*domain.GatewayConfig, error) {
	rows, err := r.queries.GetActiveGatewayConfigsByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil gateway config aktif: %w", err)
	}
	configs := make([]*domain.GatewayConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := mapGatewayConfigRow(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// GetActiveByProvider mengambil konfigurasi gateway aktif berdasarkan provider untuk tenant.
func (r *GatewayConfigRepo) GetActiveByProvider(ctx context.Context, tenantID string, provider domain.GatewayProvider) (*domain.GatewayConfig, error) {
	row, err := r.queries.GetActiveGatewayConfigByProvider(ctx, GetActiveGatewayConfigByProviderParams{
		TenantID:        stringToUUID(tenantID),
		GatewayProvider: string(provider),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGatewayConfigNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil gateway config by provider: %w", err)
	}
	return mapGatewayConfigRow(row)
}

// ExistsByProvider mengecek apakah konfigurasi aktif sudah ada untuk provider di tenant.
func (r *GatewayConfigRepo) ExistsByProvider(ctx context.Context, tenantID string, provider domain.GatewayProvider) (bool, error) {
	exists, err := r.queries.ExistsGatewayConfigByProvider(ctx, ExistsGatewayConfigByProviderParams{
		TenantID:        stringToUUID(tenantID),
		GatewayProvider: string(provider),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek exists gateway config: %w", err)
	}
	return exists, nil
}
