// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan VLANManager untuk CRUD VLAN dan resolusi VLAN
// berdasarkan strategy tenant saat provisioning.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: vlanManager harus mengimplementasikan domain.VLANManager.
var _ domain.VLANManager = (*vlanManager)(nil)

// vlanManager mengimplementasikan domain.VLANManager.
// Mengelola CRUD VLAN per OLT dan resolusi VLAN berdasarkan strategy saat provisioning.
type vlanManager struct {
	vlanRepo domain.VLANRepository
	oltRepo  domain.OLTRepository
}

// NewVLANManager membuat instance VLANManager baru dengan dependensi yang diperlukan.
func NewVLANManager(
	vlanRepo domain.VLANRepository,
	oltRepo domain.OLTRepository,
) domain.VLANManager {
	return &vlanManager{
		vlanRepo: vlanRepo,
		oltRepo:  oltRepo,
	}
}

// Buat membuat VLAN baru untuk OLT tertentu.
// Validasi: OLT harus ada, VLAN ID belum digunakan pada OLT yang sama.
func (vm *vlanManager) Create(ctx context.Context, tenantID string, req domain.CreateVLANRequest) (*domain.VLANResponse, error) {
	// Ambil OLT untuk mendapatkan tenant_id dan validasi keberadaan
	olt, err := vm.oltRepo.GetByID(ctx, req.OLTID)
	if err != nil {
		return nil, domain.ErrOLTNotFound
	}

	// Cek apakah VLAN ID sudah ada pada OLT ini
	exists, err := vm.vlanRepo.VLANIDExists(ctx, req.OLTID, req.VLANID, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrVLANIDExists
	}

	now := time.Now()
	vlan := &domain.VLAN{
		ID:          uuid.New().String(),
		TenantID:    olt.TenantID,
		OLTID:       req.OLTID,
		VLANID:      req.VLANID,
		Name:        req.Name,
		VLANType:    domain.VLANType(req.VLANType),
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := vm.vlanRepo.Create(ctx, vlan)
	if err != nil {
		return nil, err
	}

	return vm.toResponse(ctx, created), nil
}

// GetByID mengambil detail VLAN berdasarkan ID.
func (vm *vlanManager) GetByID(ctx context.Context, id string) (*domain.VLANResponse, error) {
	vlan, err := vm.vlanRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return vm.toResponse(ctx, vlan), nil
}

// Perbarui memperbarui data VLAN yang sudah ada.
func (vm *vlanManager) Update(ctx context.Context, id string, req domain.UpdateVLANRequest) (*domain.VLANResponse, error) {
	vlan, err := vm.vlanRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		vlan.Name = req.Name
	}
	if req.VLANType != "" {
		vlan.VLANType = domain.VLANType(req.VLANType)
	}
	if req.Description != "" {
		vlan.Description = req.Description
	}
	vlan.UpdatedAt = time.Now()

	updated, err := vm.vlanRepo.Update(ctx, vlan)
	if err != nil {
		return nil, err
	}

	return vm.toResponse(ctx, updated), nil
}

// Hapus melakukan hapus lunak VLAN setelah memastikan tidak ada ONT aktif yang menggunakan.
func (vm *vlanManager) Delete(ctx context.Context, id string) error {
	// Pastikan VLAN ada
	_, err := vm.vlanRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Cek apakah ada ONT aktif yang menggunakan VLAN ini
	count, err := vm.vlanRepo.CountActiveONTs(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrVLANInUse
	}

	return vm.vlanRepo.SoftDelete(ctx, id)
}

// List mengambil daftar VLAN per OLT dengan paginasi.
func (vm *vlanManager) List(ctx context.Context, oltID string, params domain.VLANListParams) (*domain.VLANListResult, error) {
	return vm.vlanRepo.List(ctx, oltID, params)
}

// ResolveVLAN menentukan VLAN yang akan digunakan saat provisioning
// berdasarkan strategy tenant: single, per_paket, per_odp, per_pelanggan.
func (vm *vlanManager) ResolveVLAN(
	ctx context.Context,
	oltID string,
	strategy domain.VLANStrategy,
	resolveCtx domain.VLANResolveContext,
) (*domain.VLAN, error) {
	switch strategy {
	case domain.VLANStrategySingle:
		return vm.resolveSingle(ctx, oltID)
	case domain.VLANStrategyPerPaket:
		return vm.resolvePerPaket(ctx, oltID, resolveCtx.PackageID)
	case domain.VLANStrategyPerODP:
		return vm.resolvePerODP(ctx, oltID, resolveCtx.ODPID)
	case domain.VLANStrategyPerPelanggan:
		return vm.resolvePerPelanggan(ctx, oltID, resolveCtx.CustomerID)
	default:
		return nil, domain.ErrInvalidVLANStrategy
	}
}

// resolveSingle mengembalikan VLAN bawaan untuk OLT (strategy "single").
func (vm *vlanManager) resolveSingle(ctx context.Context, oltID string) (*domain.VLAN, error) {
	vlan, err := vm.vlanRepo.GetDefaultVLAN(ctx, oltID)
	if err != nil {
		return nil, domain.ErrVLANResolutionFailed
	}
	return vlan, nil
}

// resolvePerPaket mengembalikan VLAN berdasarkan package_id (strategy "per_paket").
// Menggunakan GetByOLTAndVLANID dengan konvensi: package_id di-map ke VLAN tertentu.
// Untuk saat ini, lookup VLAN by name yang mengandung package reference.
func (vm *vlanManager) resolvePerPaket(ctx context.Context, oltID, packageID string) (*domain.VLAN, error) {
	if packageID == "" {
		return nil, domain.ErrVLANResolutionFailed
	}
	// List semua VLAN untuk OLT, cari yang memiliki description mengandung package_id
	result, err := vm.vlanRepo.List(ctx, oltID, domain.VLANListParams{Page: 1, PageSize: 100})
	if err != nil {
		return nil, domain.ErrVLANResolutionFailed
	}
	for _, v := range result.Data {
		if v.Description == packageID {
			// Ambil VLAN entity lengkap dari repo
			return vm.vlanRepo.GetByID(ctx, v.ID)
		}
	}
	// Cadangan ke bawaan VLAN jika tidak ada mapping spesifik
	return vm.resolveSingle(ctx, oltID)
}

// resolvePerODP mengembalikan VLAN berdasarkan ODP ID (strategy "per_odp").
func (vm *vlanManager) resolvePerODP(ctx context.Context, oltID, odpID string) (*domain.VLAN, error) {
	if odpID == "" {
		return nil, domain.ErrVLANResolutionFailed
	}
	// List semua VLAN untuk OLT, cari yang memiliki description mengandung odp_id
	result, err := vm.vlanRepo.List(ctx, oltID, domain.VLANListParams{Page: 1, PageSize: 100})
	if err != nil {
		return nil, domain.ErrVLANResolutionFailed
	}
	for _, v := range result.Data {
		if v.Description == odpID {
			return vm.vlanRepo.GetByID(ctx, v.ID)
		}
	}
	// Cadangan ke bawaan VLAN jika tidak ada mapping spesifik
	return vm.resolveSingle(ctx, oltID)
}

// resolvePerPelanggan mengembalikan VLAN unik per pelanggan (strategy "per_pelanggan").
func (vm *vlanManager) resolvePerPelanggan(ctx context.Context, oltID, customerID string) (*domain.VLAN, error) {
	if customerID == "" {
		return nil, domain.ErrVLANResolutionFailed
	}
	// List semua VLAN untuk OLT, cari yang memiliki description = customer_id
	result, err := vm.vlanRepo.List(ctx, oltID, domain.VLANListParams{Page: 1, PageSize: 100})
	if err != nil {
		return nil, domain.ErrVLANResolutionFailed
	}
	for _, v := range result.Data {
		if v.Description == customerID {
			return vm.vlanRepo.GetByID(ctx, v.ID)
		}
	}
	// Untuk per_pelanggan, tidak ada cadangan - harus ada VLAN unik
	return nil, domain.ErrVLANResolutionFailed
}

// toResponse mengkonversi entity VLAN ke VLANResponse dengan jumlah ONT aktif.
func (vm *vlanManager) toResponse(ctx context.Context, vlan *domain.VLAN) *domain.VLANResponse {
	activeONTs, _ := vm.vlanRepo.CountActiveONTs(ctx, vlan.ID)
	return &domain.VLANResponse{
		ID:          vlan.ID,
		OLTID:       vlan.OLTID,
		VLANID:      vlan.VLANID,
		Name:        vlan.Name,
		VLANType:    string(vlan.VLANType),
		Description: vlan.Description,
		ActiveONTs:  activeONTs,
		CreatedAt:   vlan.CreatedAt,
		UpdatedAt:   vlan.UpdatedAt,
	}
}
