package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
)

// BillingSettingsRepo mengimplementasikan domain.BillingSettingsRepository dengan membungkus
// sqlc-generated Queries untuk operasi billing settings.
type BillingSettingsRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi billing settings.
	queries *Queries
}

// NewBillingSettingsRepo membuat instance baru BillingSettingsRepo.
func NewBillingSettingsRepo(queries *Queries) *BillingSettingsRepo {
	return &BillingSettingsRepo{
		queries: queries,
	}
}

// --- Helper function untuk mapping sqlc BillingSetting → domain.BillingSettings ---

// mapBillingSettingRow memetakan BillingSetting (sqlc model) ke domain.BillingSettings.
func mapBillingSettingRow(row BillingSetting) *domain.BillingSettings {
	return &domain.BillingSettings{
		ID:                 uuidToString(row.ID),
		TenantID:           uuidToString(row.TenantID),
		GenerateDays:       int(row.GenerateDays),
		GracePeriodDays:    int(row.GracePeriodDays),
		SuspendDays:        int(row.SuspendDays),
		TaxEnabled:         row.TaxEnabled,
		TaxRate:            numericToFloat64(row.TaxRate),
		PenaltyEnabled:     row.PenaltyEnabled,
		PenaltyType:        domain.PenaltyType(row.PenaltyType),
		PenaltyAmount:      row.PenaltyAmount,
		PenaltyPercentage:  numericToFloat64(row.PenaltyPercentage),
		PenaltyDailyAmount: row.PenaltyDailyAmount,
		PenaltyMaxAmount:   row.PenaltyMaxAmount,
		InvoicePrefix:      row.InvoicePrefix,
		NewCustomerBilling: row.NewCustomerBilling,
		Timezone:           row.Timezone,
		AutoIsolir:         row.AutoIsolir,
		AutoOpenIsolir:     row.AutoOpenIsolir,
		CreatedAt:          timestamptzToTime(row.CreatedAt),
		UpdatedAt:          timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.BillingSettingsRepository ---

// GetByTenantID mengambil billing settings berdasarkan tenant ID.
// Mengembalikan ErrBillingSettingsNotFound jika tidak ditemukan.
func (r *BillingSettingsRepo) GetByTenantID(ctx context.Context, tenantID string) (*domain.BillingSettings, error) {
	row, err := r.queries.GetBillingSettingsByTenantID(ctx, stringToUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBillingSettingsNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil billing settings: %w", err)
	}
	return mapBillingSettingRow(row), nil
}

// Upsert membuat atau memperbarui billing settings untuk tenant.
// Menggunakan INSERT ON CONFLICT untuk upsert berdasarkan tenant_id.
func (r *BillingSettingsRepo) Upsert(ctx context.Context, settings *domain.BillingSettings) (*domain.BillingSettings, error) {
	row, err := r.queries.UpsertBillingSettings(ctx, UpsertBillingSettingsParams{
		TenantID:           stringToUUID(settings.TenantID),
		GenerateDays:       int32(settings.GenerateDays),
		GracePeriodDays:    int32(settings.GracePeriodDays),
		SuspendDays:        int32(settings.SuspendDays),
		TaxEnabled:         settings.TaxEnabled,
		TaxRate:            float64ToNumeric(settings.TaxRate),
		PenaltyEnabled:     settings.PenaltyEnabled,
		PenaltyType:        string(settings.PenaltyType),
		PenaltyAmount:      settings.PenaltyAmount,
		PenaltyPercentage:  float64ToNumeric(settings.PenaltyPercentage),
		PenaltyDailyAmount: settings.PenaltyDailyAmount,
		PenaltyMaxAmount:   settings.PenaltyMaxAmount,
		InvoicePrefix:      settings.InvoicePrefix,
		NewCustomerBilling: settings.NewCustomerBilling,
		Timezone:           settings.Timezone,
		AutoIsolir:         settings.AutoIsolir,
		AutoOpenIsolir:     settings.AutoOpenIsolir,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal upsert billing settings: %w", err)
	}
	return mapBillingSettingRow(row), nil
}

// ListAll mengambil semua billing settings (untuk cron job lintas tenant).
func (r *BillingSettingsRepo) ListAll(ctx context.Context) ([]*domain.BillingSettings, error) {
	rows, err := r.queries.ListAllBillingSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil semua billing settings: %w", err)
	}

	result := make([]*domain.BillingSettings, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapBillingSettingRow(row))
	}
	return result, nil
}
