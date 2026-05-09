package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ServiceProfileRepo mengimplementasikan domain.ServiceProfileRepository dengan
// membungkus sqlc-generated Queries dan memetakan tipe database ke domain.ServiceProfile.
type ServiceProfileRepo struct {
	queries *Queries
}

// NewServiceProfileRepo membuat instance baru ServiceProfileRepo.
func NewServiceProfileRepo(queries *Queries) *ServiceProfileRepo {
	return &ServiceProfileRepo{queries: queries}
}

// --- Mapping sqlc ServiceProfile -> domain.ServiceProfile ---

// mapServiceProfileRow memetakan ServiceProfile (sqlc model) ke domain.ServiceProfile.
func mapServiceProfileRow(row ServiceProfile) *domain.ServiceProfile {
	return &domain.ServiceProfile{
		ID:               uuidToString(row.ID),
		TenantID:         uuidToString(row.TenantID),
		OLTID:            uuidToString(row.OltID),
		Name:             row.Name,
		LineProfileID:    int(row.LineProfileID),
		ServiceProfileID: int(row.ServiceProfileID),
		PackageID:        uuidToStringPtr(row.PackageID),
		Description:      textToString(row.Description),
		DeletedAt:        timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:        timestamptzToTime(row.CreatedAt),
		UpdatedAt:        timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi CRUD domain.ServiceProfileRepository ---

// Buat membuat service profile baru dan mengembalikan profile yang dibuat.
func (r *ServiceProfileRepo) Create(ctx context.Context, profile *domain.ServiceProfile) (*domain.ServiceProfile, error) {
	row, err := r.queries.CreateServiceProfile(ctx, CreateServiceProfileParams{
		TenantID:         stringToUUID(profile.TenantID),
		OltID:            stringToUUID(profile.OLTID),
		Name:             profile.Name,
		LineProfileID:    int32(profile.LineProfileID),
		ServiceProfileID: int32(profile.ServiceProfileID),
		PackageID:        stringPtrToUUID(profile.PackageID),
		Description:      stringToText(profile.Description),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat service profile: %w", err)
	}
	return mapServiceProfileRow(row), nil
}

// GetByID mengambil service profile berdasarkan ID (tenant-scoped via RLS).
func (r *ServiceProfileRepo) GetByID(ctx context.Context, id string) (*domain.ServiceProfile, error) {
	row, err := r.queries.GetServiceProfileByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrServiceProfileNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil service profile by ID: %w", err)
	}
	return mapServiceProfileRow(row), nil
}

// Perbarui memperbarui data service profile dan mengembalikan profile yang diperbarui.
func (r *ServiceProfileRepo) Update(ctx context.Context, profile *domain.ServiceProfile) (*domain.ServiceProfile, error) {
	row, err := r.queries.UpdateServiceProfile(ctx, UpdateServiceProfileParams{
		ID:               stringToUUID(profile.ID),
		Name:             profile.Name,
		LineProfileID:    int32(profile.LineProfileID),
		ServiceProfileID: int32(profile.ServiceProfileID),
		PackageID:        stringPtrToUUID(profile.PackageID),
		Description:      stringToText(profile.Description),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrServiceProfileNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui service profile: %w", err)
	}
	return mapServiceProfileRow(row), nil
}

// SoftDelete melakukan hapus lunak service profile (atur deleted_at).
func (r *ServiceProfileRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteServiceProfile(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete service profile: %w", err)
	}
	return nil
}

// List mengambil daftar service profile per OLT dengan paginasi.
func (r *ServiceProfileRepo) List(ctx context.Context, oltID string, params domain.ServiceProfileListParams) (*domain.ServiceProfileListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Ambil total count untuk paginasi
	total, err := r.queries.CountServiceProfiles(ctx, stringToUUID(oltID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total service profile: %w", err)
	}

	// Ambil data service profile
	rows, err := r.queries.ListServiceProfiles(ctx, ListServiceProfilesParams{
		OltID:  stringToUUID(oltID),
		Limit:  int32(params.PageSize),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar service profile: %w", err)
	}

	// Konversi ke domain.ServiceProfileResponse untuk list
	responses := make([]*domain.ServiceProfileResponse, 0, len(rows))
	for _, row := range rows {
		sp := mapServiceProfileRow(row)
		responses = append(responses, &domain.ServiceProfileResponse{
			ID:               sp.ID,
			OLTID:            sp.OLTID,
			Name:             sp.Name,
			LineProfileID:    sp.LineProfileID,
			ServiceProfileID: sp.ServiceProfileID,
			PackageID:        sp.PackageID,
			Description:      sp.Description,
			CreatedAt:        sp.CreatedAt,
			UpdatedAt:        sp.UpdatedAt,
		})
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.ServiceProfileListResult{
		Data:       responses,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByPackageAndOLT mengambil service profile berdasarkan package_id dan olt_id.
func (r *ServiceProfileRepo) GetByPackageAndOLT(ctx context.Context, oltID, packageID string) (*domain.ServiceProfile, error) {
	row, err := r.queries.GetServiceProfileByPackageAndOLT(ctx, GetServiceProfileByPackageAndOLTParams{
		OltID:     stringToUUID(oltID),
		PackageID: stringToUUID(packageID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrServiceProfileNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil service profile by package: %w", err)
	}
	return mapServiceProfileRow(row), nil
}

// ProfileExists mengecek apakah kombinasi profile sudah ada pada OLT.
func (r *ServiceProfileRepo) ProfileExists(ctx context.Context, oltID string, lineProfileID, serviceProfileID int, excludeID string) (bool, error) {
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.ServiceProfileExists(ctx, ServiceProfileExistsParams{
		OltID:            stringToUUID(oltID),
		LineProfileID:    int32(lineProfileID),
		ServiceProfileID: int32(serviceProfileID),
		ID:               stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek service profile: %w", err)
	}
	return exists, nil
}

// CountActiveONTs menghitung jumlah ONT aktif yang menggunakan profile ini.
func (r *ServiceProfileRepo) CountActiveONTs(ctx context.Context, profileID string) (int64, error) {
	count, err := r.queries.CountServiceProfileActiveONTs(ctx, stringToUUID(profileID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung ONT aktif service profile: %w", err)
	}
	return count, nil
}

// Compile-time cek: ServiceProfileRepo mengimplementasikan domain.ServiceProfileRepository.
var _ domain.ServiceProfileRepository = (*ServiceProfileRepo)(nil)
