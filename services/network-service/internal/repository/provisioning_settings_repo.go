package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ProvisioningSettingsRepo mengimplementasikan domain.ProvisioningSettingsRepository
// dengan membungkus sqlc-generated Queries dan memetakan tipe database ke
// domain.ProvisioningSettings. Satu record per tenant, menggunakan upsert.
type ProvisioningSettingsRepo struct {
	queries *Queries
}

// NewProvisioningSettingsRepo membuat instance baru ProvisioningSettingsRepo.
func NewProvisioningSettingsRepo(queries *Queries) *ProvisioningSettingsRepo {
	return &ProvisioningSettingsRepo{queries: queries}
}

// --- Mapping sqlc ProvisioningSetting -> domain.ProvisioningSettings ---

// mapSettingsRow memetakan ProvisioningSetting (sqlc model) ke domain.ProvisioningSettings.
func mapSettingsRow(row ProvisioningSetting) *domain.ProvisioningSettings {
	return &domain.ProvisioningSettings{
		ID:                       uuidToString(row.ID),
		TenantID:                 uuidToString(row.TenantID),
		AutoProvisioningEnabled:  row.AutoProvisioningEnabled,
		AutoPortMigrationEnabled: row.AutoPortMigrationEnabled,
		VLANStrategy:             domain.VLANStrategy(row.VlanStrategy),
		CreatedAt:                timestamptzToTime(row.CreatedAt),
		UpdatedAt:                timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.ProvisioningSettingsRepository ---

// GetByTenantID mengambil settings berdasarkan tenant_id.
// Mengembalikan nil dan ErrONTNotFound jika tidak ditemukan - caller
// harus menggunakan DefaultProvisioningSettings sebagai cadangan.
func (r *ProvisioningSettingsRepo) GetByTenantID(ctx context.Context, tenantID string) (*domain.ProvisioningSettings, error) {
	row, err := r.queries.GetProvisioningSettingsByTenantID(ctx, stringToUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: gagal mengambil provisioning settings: %w", err)
	}
	return mapSettingsRow(row), nil
}

// Upsert membuat atau memperbarui settings untuk tenant.
func (r *ProvisioningSettingsRepo) Upsert(ctx context.Context, settings *domain.ProvisioningSettings) (*domain.ProvisioningSettings, error) {
	row, err := r.queries.UpsertProvisioningSettings(ctx, UpsertProvisioningSettingsParams{
		TenantID:                 stringToUUID(settings.TenantID),
		AutoProvisioningEnabled:  settings.AutoProvisioningEnabled,
		AutoPortMigrationEnabled: settings.AutoPortMigrationEnabled,
		VlanStrategy:             string(settings.VLANStrategy),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal upsert provisioning settings: %w", err)
	}
	return mapSettingsRow(row), nil
}

// Compile-time cek: ProvisioningSettingsRepo mengimplementasikan domain.ProvisioningSettingsRepository.
var _ domain.ProvisioningSettingsRepository = (*ProvisioningSettingsRepo)(nil)
