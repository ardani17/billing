// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan ServiceProfileManager untuk CRUD service profile
// dan resolusi profile berdasarkan package_id + olt_id saat provisioning.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: serviceProfileManager harus mengimplementasikan domain.ServiceProfileManager.
var _ domain.ServiceProfileManager = (*serviceProfileManager)(nil)

// serviceProfileManager mengimplementasikan domain.ServiceProfileManager.
// Mengelola CRUD service profile per OLT dan resolusi profile berdasarkan
// package_id saat provisioning.
type serviceProfileManager struct {
	profileRepo domain.ServiceProfileRepository
	oltRepo     domain.OLTRepository
}

// NewServiceProfileManager membuat instance ServiceProfileManager baru.
func NewServiceProfileManager(
	profileRepo domain.ServiceProfileRepository,
	oltRepo domain.OLTRepository,
) domain.ServiceProfileManager {
	return &serviceProfileManager{
		profileRepo: profileRepo,
		oltRepo:     oltRepo,
	}
}

// Buat membuat service profile baru untuk OLT tertentu.
// Validasi: OLT harus ada, kombinasi profile belum ada pada OLT yang sama.
func (spm *serviceProfileManager) Create(
	ctx context.Context,
	tenantID string,
	req domain.CreateServiceProfileRequest,
) (*domain.ServiceProfileResponse, error) {
	// Ambil OLT untuk validasi keberadaan dan mendapatkan tenant_id
	olt, err := spm.oltRepo.GetByID(ctx, req.OLTID)
	if err != nil {
		return nil, domain.ErrOLTNotFound
	}

	// Cek apakah kombinasi profile sudah ada pada OLT ini
	exists, err := spm.profileRepo.ProfileExists(ctx, req.OLTID, req.LineProfileID, req.ServiceProfileID, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrServiceProfileExists
	}

	now := time.Now()
	var packageID *string
	if req.PackageID != "" {
		packageID = &req.PackageID
	}

	profile := &domain.ServiceProfile{
		ID:               uuid.New().String(),
		TenantID:         olt.TenantID,
		OLTID:            req.OLTID,
		Name:             req.Name,
		LineProfileID:    req.LineProfileID,
		ServiceProfileID: req.ServiceProfileID,
		PackageID:        packageID,
		Description:      req.Description,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	created, err := spm.profileRepo.Create(ctx, profile)
	if err != nil {
		return nil, err
	}

	return spm.toResponse(ctx, created), nil
}

// GetByID mengambil detail service profile berdasarkan ID.
func (spm *serviceProfileManager) GetByID(ctx context.Context, id string) (*domain.ServiceProfileResponse, error) {
	profile, err := spm.profileRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return spm.toResponse(ctx, profile), nil
}

// Perbarui memperbarui data service profile yang sudah ada.
func (spm *serviceProfileManager) Update(
	ctx context.Context,
	id string,
	req domain.UpdateServiceProfileRequest,
) (*domain.ServiceProfileResponse, error) {
	profile, err := spm.profileRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		profile.Name = req.Name
	}
	if req.LineProfileID != nil {
		profile.LineProfileID = *req.LineProfileID
	}
	if req.ServiceProfileID != nil {
		profile.ServiceProfileID = *req.ServiceProfileID
	}
	if req.PackageID != "" {
		profile.PackageID = &req.PackageID
	}
	if req.Description != "" {
		profile.Description = req.Description
	}
	profile.UpdatedAt = time.Now()

	updated, err := spm.profileRepo.Update(ctx, profile)
	if err != nil {
		return nil, err
	}

	return spm.toResponse(ctx, updated), nil
}

// Hapus melakukan hapus lunak service profile setelah memastikan tidak ada ONT aktif.
func (spm *serviceProfileManager) Delete(ctx context.Context, id string) error {
	// Pastikan profile ada
	_, err := spm.profileRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Cek apakah ada ONT aktif yang menggunakan profile ini
	count, err := spm.profileRepo.CountActiveONTs(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrServiceProfileInUse
	}

	return spm.profileRepo.SoftDelete(ctx, id)
}

// List mengambil daftar service profile per OLT dengan paginasi.
func (spm *serviceProfileManager) List(
	ctx context.Context,
	oltID string,
	params domain.ServiceProfileListParams,
) (*domain.ServiceProfileListResult, error) {
	return spm.profileRepo.List(ctx, oltID, params)
}

// ResolveProfile menentukan service profile berdasarkan package_id dan olt_id.
// Digunakan saat provisioning untuk mendapatkan line_profile_id dan service_profile_id.
func (spm *serviceProfileManager) ResolveProfile(
	ctx context.Context,
	oltID string,
	packageID string,
) (*domain.ServiceProfile, error) {
	profile, err := spm.profileRepo.GetByPackageAndOLT(ctx, oltID, packageID)
	if err != nil {
		return nil, domain.ErrNoProfileMapping
	}
	return profile, nil
}

// toResponse mengkonversi entity ServiceProfile ke ServiceProfileResponse.
func (spm *serviceProfileManager) toResponse(
	ctx context.Context,
	profile *domain.ServiceProfile,
) *domain.ServiceProfileResponse {
	activeONTs, _ := spm.profileRepo.CountActiveONTs(ctx, profile.ID)
	return &domain.ServiceProfileResponse{
		ID:               profile.ID,
		OLTID:            profile.OLTID,
		Name:             profile.Name,
		LineProfileID:    profile.LineProfileID,
		ServiceProfileID: profile.ServiceProfileID,
		PackageID:        profile.PackageID,
		Description:      profile.Description,
		ActiveONTs:       activeONTs,
		CreatedAt:        profile.CreatedAt,
		UpdatedAt:        profile.UpdatedAt,
	}
}
