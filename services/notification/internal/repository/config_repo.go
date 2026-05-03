package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// ConfigRepo — implementasi domain.ConfigRepository menggunakan sqlc Queries
// =============================================================================

// ConfigRepo mengimplementasikan domain.ConfigRepository dengan membungkus
// sqlc-generated Queries untuk operasi tabel notification_configs.
type ConfigRepo struct {
	queries *Queries
}

// NewConfigRepo membuat instance baru ConfigRepo.
func NewConfigRepo(queries *Queries) *ConfigRepo {
	return &ConfigRepo{queries: queries}
}

// --- Helper: konversi pgtype.UUID ↔ string ---

// parseUUID mengkonversi string UUID ke pgtype.UUID.
func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

// uuidToString mengkonversi pgtype.UUID ke string.
// Mengembalikan string kosong jika UUID tidak valid.
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// --- Helper: konversi pgtype.Timestamptz ↔ time.Time ---

// timestamptzToTime mengkonversi pgtype.Timestamptz ke time.Time.
// Mengembalikan zero time jika tidak valid (NULL).
func timestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// --- Helper: mapping sqlc NotificationConfig → domain.NotificationConfig ---

// mapConfigRow memetakan NotificationConfig (sqlc model) ke domain.NotificationConfig.
// Konversi: pgtype.UUID → string, []byte → json.RawMessage/ConfigSettings, int32 → int.
func mapConfigRow(row NotificationConfig) (*domain.NotificationConfig, error) {
	// Unmarshal settings dari JSONB ke domain.ConfigSettings
	var settings domain.ConfigSettings
	if len(row.Settings) > 0 {
		if err := json.Unmarshal(row.Settings, &settings); err != nil {
			return nil, fmt.Errorf("repository: gagal unmarshal settings: %w", err)
		}
	}

	return &domain.NotificationConfig{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		Channel:     domain.Channel(row.Channel),
		Provider:    row.Provider,
		Credentials: row.Credentials,
		IsEnabled:   row.IsEnabled,
		Priority:    int(row.Priority),
		Settings:    settings,
		CreatedAt:   timestamptzToTime(row.CreatedAt),
		UpdatedAt:   timestamptzToTime(row.UpdatedAt),
	}, nil
}

// --- Implementasi domain.ConfigRepository ---

// GetByTenant mengambil semua konfigurasi notifikasi untuk tenant tertentu.
// Hasil diurutkan berdasarkan priority ASC.
func (r *ConfigRepo) GetByTenant(ctx context.Context, tenantID string) ([]*domain.NotificationConfig, error) {
	rows, err := r.queries.GetConfigsByTenant(ctx, parseUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil config by tenant: %w", err)
	}

	configs := make([]*domain.NotificationConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := mapConfigRow(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// GetByTenantAndChannel mengambil konfigurasi notifikasi berdasarkan tenant dan channel.
// Mengembalikan domain.ErrConfigNotFound jika tidak ditemukan.
func (r *ConfigRepo) GetByTenantAndChannel(ctx context.Context, tenantID string, ch domain.Channel) (*domain.NotificationConfig, error) {
	row, err := r.queries.GetConfigByTenantAndChannel(ctx, GetConfigByTenantAndChannelParams{
		TenantID: parseUUID(tenantID),
		Channel:  string(ch),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil config by tenant dan channel: %w", err)
	}
	return mapConfigRow(row)
}

// Upsert membuat atau memperbarui konfigurasi notifikasi per tenant per channel.
// Menggunakan INSERT ON CONFLICT untuk upsert berdasarkan (tenant_id, channel).
func (r *ConfigRepo) Upsert(ctx context.Context, cfg *domain.NotificationConfig) (*domain.NotificationConfig, error) {
	// Marshal settings ke JSONB
	settingsJSON, err := json.Marshal(cfg.Settings)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal settings: %w", err)
	}

	// Pastikan credentials tidak nil
	creds := cfg.Credentials
	if creds == nil {
		creds = json.RawMessage("{}")
	}

	row, err := r.queries.UpsertConfig(ctx, UpsertConfigParams{
		TenantID:    parseUUID(cfg.TenantID),
		Channel:     string(cfg.Channel),
		Provider:    cfg.Provider,
		Credentials: creds,
		IsEnabled:   cfg.IsEnabled,
		Priority:    int32(cfg.Priority),
		Settings:    settingsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal upsert config: %w", err)
	}
	return mapConfigRow(row)
}

// GetSettings mengambil pengaturan umum notifikasi untuk tenant tertentu.
// Mengambil settings dari baris pertama config milik tenant.
// Mengembalikan domain.ErrConfigNotFound jika tenant belum punya config.
func (r *ConfigRepo) GetSettings(ctx context.Context, tenantID string) (*domain.ConfigSettings, error) {
	settingsJSON, err := r.queries.GetSettingsByTenant(ctx, parseUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil settings: %w", err)
	}

	var settings domain.ConfigSettings
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			return nil, fmt.Errorf("repository: gagal unmarshal settings: %w", err)
		}
	}
	return &settings, nil
}

// UpdateSettings memperbarui pengaturan umum notifikasi untuk semua config milik tenant.
func (r *ConfigRepo) UpdateSettings(ctx context.Context, tenantID string, s domain.ConfigSettings) error {
	settingsJSON, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("repository: gagal marshal settings: %w", err)
	}

	err = r.queries.UpdateSettings(ctx, UpdateSettingsParams{
		TenantID: parseUUID(tenantID),
		Settings: settingsJSON,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui settings: %w", err)
	}
	return nil
}
