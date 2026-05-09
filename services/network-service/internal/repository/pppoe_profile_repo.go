package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// PPPoEProfileRepo mengimplementasikan domain.PPPoEProfileRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.PPPoEProfile.
type PPPoEProfileRepo struct {
	queries *Queries
}

// NewPPPoEProfileRepo membuat instance baru PPPoEProfileRepo.
func NewPPPoEProfileRepo(queries *Queries) *PPPoEProfileRepo {
	return &PPPoEProfileRepo{queries: queries}
}

// --- Mapping sqlc PppoeProfile -> domain.PPPoEProfile ---

// mapPPPoEProfileRow memetakan PppoeProfile (sqlc model) ke domain.PPPoEProfile.
func mapPPPoEProfileRow(row PppoeProfile) *domain.PPPoEProfile {
	return &domain.PPPoEProfile{
		ID:                     uuidToString(row.ID),
		TenantID:               uuidToString(row.TenantID),
		PackageID:              uuidToString(row.PackageID),
		ProfileName:            row.ProfileName,
		DownloadLimit:          row.DownloadLimit,
		UploadLimit:            row.UploadLimit,
		BurstDownload:          textToString(row.BurstDownload),
		BurstUpload:            textToString(row.BurstUpload),
		BurstThresholdDownload: textToString(row.BurstThresholdDownload),
		BurstThresholdUpload:   textToString(row.BurstThresholdUpload),
		BurstTime:              textToString(row.BurstTime),
		AddressPool:            textToString(row.AddressPool),
		LocalAddress:           row.LocalAddress,
		OnlyOne:                row.OnlyOne,
		CreatedAt:              timestamptzToTime(row.CreatedAt),
		UpdatedAt:              timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.PPPoEProfileRepository ---

// Buat membuat record PPPoE profile baru.
func (r *PPPoEProfileRepo) Create(ctx context.Context, profile *domain.PPPoEProfile) (*domain.PPPoEProfile, error) {
	row, err := r.queries.CreatePPPoEProfile(ctx, CreatePPPoEProfileParams{
		TenantID:               stringToUUID(profile.TenantID),
		PackageID:              stringToUUID(profile.PackageID),
		ProfileName:            profile.ProfileName,
		DownloadLimit:          profile.DownloadLimit,
		UploadLimit:            profile.UploadLimit,
		BurstDownload:          stringToText(profile.BurstDownload),
		BurstUpload:            stringToText(profile.BurstUpload),
		BurstThresholdDownload: stringToText(profile.BurstThresholdDownload),
		BurstThresholdUpload:   stringToText(profile.BurstThresholdUpload),
		BurstTime:              stringToText(profile.BurstTime),
		AddressPool:            stringToText(profile.AddressPool),
		LocalAddress:           profile.LocalAddress,
		OnlyOne:                profile.OnlyOne,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrProfileNameExists
		}
		return nil, fmt.Errorf("repository: gagal membuat pppoe profile: %w", err)
	}
	return mapPPPoEProfileRow(row), nil
}

// GetByID mengambil PPPoE profile berdasarkan ID.
func (r *PPPoEProfileRepo) GetByID(ctx context.Context, id string) (*domain.PPPoEProfile, error) {
	row, err := r.queries.GetPPPoEProfileByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEProfileNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil pppoe profile by ID: %w", err)
	}
	return mapPPPoEProfileRow(row), nil
}

// GetByPackageID mengambil PPPoE profile berdasarkan package_id.
func (r *PPPoEProfileRepo) GetByPackageID(ctx context.Context, packageID string) (*domain.PPPoEProfile, error) {
	row, err := r.queries.GetPPPoEProfileByPackageID(ctx, stringToUUID(packageID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEProfileNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil pppoe profile by package ID: %w", err)
	}
	return mapPPPoEProfileRow(row), nil
}

// GetByProfileName mengambil PPPoE profile berdasarkan tenant_id dan profile_name.
func (r *PPPoEProfileRepo) GetByProfileName(ctx context.Context, tenantID, profileName string) (*domain.PPPoEProfile, error) {
	row, err := r.queries.GetPPPoEProfileByProfileName(ctx, GetPPPoEProfileByProfileNameParams{
		TenantID:    stringToUUID(tenantID),
		ProfileName: profileName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEProfileNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil pppoe profile by profile name: %w", err)
	}
	return mapPPPoEProfileRow(row), nil
}

// Perbarui memperbarui record PPPoE profile.
func (r *PPPoEProfileRepo) Update(ctx context.Context, profile *domain.PPPoEProfile) (*domain.PPPoEProfile, error) {
	row, err := r.queries.UpdatePPPoEProfile(ctx, UpdatePPPoEProfileParams{
		ID:                     stringToUUID(profile.ID),
		ProfileName:            profile.ProfileName,
		DownloadLimit:          profile.DownloadLimit,
		UploadLimit:            profile.UploadLimit,
		BurstDownload:          stringToText(profile.BurstDownload),
		BurstUpload:            stringToText(profile.BurstUpload),
		BurstThresholdDownload: stringToText(profile.BurstThresholdDownload),
		BurstThresholdUpload:   stringToText(profile.BurstThresholdUpload),
		BurstTime:              stringToText(profile.BurstTime),
		AddressPool:            stringToText(profile.AddressPool),
		LocalAddress:           profile.LocalAddress,
		OnlyOne:                profile.OnlyOne,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEProfileNotFound
		}
		if isUniqueViolation(err) {
			return nil, domain.ErrProfileNameExists
		}
		return nil, fmt.Errorf("repository: gagal memperbarui pppoe profile: %w", err)
	}
	return mapPPPoEProfileRow(row), nil
}

// ListByTenant mengambil semua profile untuk satu tenant.
func (r *PPPoEProfileRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.PPPoEProfile, error) {
	rows, err := r.queries.ListPPPoEProfilesByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar pppoe profiles: %w", err)
	}

	profiles := make([]*domain.PPPoEProfile, 0, len(rows))
	for _, row := range rows {
		profiles = append(profiles, mapPPPoEProfileRow(row))
	}
	return profiles, nil
}
