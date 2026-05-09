package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// LabelSettingsRepo mengimplementasikan domain.LabelSettingsRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.MapLabelSettings.
type LabelSettingsRepo struct {
	db DBTX
}

// NewLabelSettingsRepo membuat instance baru LabelSettingsRepo.
func NewLabelSettingsRepo(db DBTX) *LabelSettingsRepo {
	return &LabelSettingsRepo{db: db}
}

// scanLabelSettings memindai satu baris hasil kueri ke domain.MapLabelSettings.
func scanLabelSettings(row pgx.Row) (*domain.MapLabelSettings, error) {
	var s domain.MapLabelSettings
	err := row.Scan(
		&s.ID, &s.TenantID, &s.OLTLabels, &s.ODPLabels, &s.ONTLabels,
		&s.MinZoomLevel, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByTenantID mengambil konfigurasi label berdasarkan tenant_id.
// Mengembalikan nil jika tenant belum memiliki konfigurasi.
func (r *LabelSettingsRepo) GetByTenantID(ctx context.Context, tenantID string) (*domain.MapLabelSettings, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, olt_labels, odp_labels, ont_labels, min_zoom_level, created_at, updated_at
		 FROM map_label_settings WHERE tenant_id = $1`, tenantID,
	)
	result, err := scanLabelSettings(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: gagal mengambil label settings: %w", err)
	}
	return result, nil
}

// Upsert membuat atau memperbarui konfigurasi label untuk tenant.
func (r *LabelSettingsRepo) Upsert(ctx context.Context, settings *domain.MapLabelSettings) (*domain.MapLabelSettings, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO map_label_settings (tenant_id, olt_labels, odp_labels, ont_labels, min_zoom_level)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id) DO UPDATE SET
			olt_labels = EXCLUDED.olt_labels,
			odp_labels = EXCLUDED.odp_labels,
			ont_labels = EXCLUDED.ont_labels,
			min_zoom_level = EXCLUDED.min_zoom_level,
			updated_at = NOW()
		 RETURNING id, tenant_id, olt_labels, odp_labels, ont_labels, min_zoom_level, created_at, updated_at`,
		settings.TenantID, settings.OLTLabels, settings.ODPLabels,
		settings.ONTLabels, settings.MinZoomLevel,
	)
	result, err := scanLabelSettings(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal upsert label settings: %w", err)
	}
	return result, nil
}

// Compile-time cek: LabelSettingsRepo mengimplementasikan domain.LabelSettingsRepository.
var _ domain.LabelSettingsRepository = (*LabelSettingsRepo)(nil)
