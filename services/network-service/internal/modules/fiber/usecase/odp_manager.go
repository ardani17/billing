// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan ODP Manager: CRUD ODP/splitter dengan auto-capacity.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: odpManager harus mengimplementasikan domain.ODPManager.
var _ domain.ODPManager = (*odpManager)(nil)

// odpManager mengimplementasikan domain.ODPManager.
// Mengelola business logic CRUD ODP/splitter dengan capacity tracking.
type odpManager struct {
	odpRepo domain.ODPRepository
	oltRepo domain.OLTRepository
}

// NewODPManager membuat instance ODPManager baru dengan dependensi repositori.
func NewODPManager(odpRepo domain.ODPRepository, oltRepo domain.OLTRepository) domain.ODPManager {
	return &odpManager{odpRepo: odpRepo, oltRepo: oltRepo}
}

// Buat membuat ODP baru. Validasi nama unik, validasi splitter_type,
// auto-atur capacity berdasarkan splitter_type, simpan ke DB, kembalikan ODPResponse.
func (m *odpManager) Create(ctx context.Context, tenantID string, req domain.CreateODPRequest) (*domain.ODPResponse, error) {
	// Validasi nama unik di tenant
	exists, err := m.odpRepo.NameExists(ctx, tenantID, req.Name, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrODPNameExists
	}

	// Validasi splitter_type via SplitterCapacity
	capacity := domain.SplitterCapacity(req.SplitterType)
	if capacity == 0 {
		return nil, domain.ErrInvalidSplitterType
	}

	odp := &domain.ODP{
		TenantID:     tenantID,
		OLTID:        req.OLTID,
		PONPortIndex: req.PONPortIndex,
		Name:         req.Name,
		SplitterType: req.SplitterType,
		Capacity:     capacity,
		UsedPorts:    0,
		Address:      req.Address,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		Notes:        req.Notes,
	}

	created, err := m.odpRepo.Create(ctx, odp)
	if err != nil {
		return nil, err
	}

	return odpToResponse(created), nil
}

// GetByID mengambil detail ODP. Menambahkan warning jika ODP penuh.
func (m *odpManager) GetByID(ctx context.Context, id string) (*domain.ODPDetailResponse, error) {
	odp, err := m.odpRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &domain.ODPDetailResponse{ODPResponse: *odpToResponse(odp)}

	// Warning jika ODP penuh (used_ports >= capacity)
	if odp.UsedPorts >= odp.Capacity {
		resp.Warning = "ODP sudah penuh, semua port terpakai"
	}

	return resp, nil
}

// Perbarui memperbarui data ODP. Validasi nama unik jika berubah.
func (m *odpManager) Update(ctx context.Context, id string, req domain.UpdateODPRequest) (*domain.ODPResponse, error) {
	odp, err := m.odpRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi nama unik jika nama berubah
	if req.Name != "" && req.Name != odp.Name {
		exists, nameErr := m.odpRepo.NameExists(ctx, odp.TenantID, req.Name, odp.ID)
		if nameErr != nil {
			return nil, nameErr
		}
		if exists {
			return nil, domain.ErrODPNameExists
		}
		odp.Name = req.Name
	}

	// Perbarui field opsional
	if req.Address != "" {
		odp.Address = req.Address
	}
	if req.Latitude != nil {
		odp.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		odp.Longitude = req.Longitude
	}
	if req.Notes != "" {
		odp.Notes = req.Notes
	}
	odp.UpdatedAt = time.Now()

	updated, err := m.odpRepo.Update(ctx, odp)
	if err != nil {
		return nil, err
	}

	return odpToResponse(updated), nil
}

// Hapus melakukan hapus lunak ODP.
func (m *odpManager) Delete(ctx context.Context, id string) error {
	if err := m.odpRepo.SoftDelete(ctx, id); err != nil {
		log.Error().Err(err).Str("odp_id", id).Msg("gagal soft-delete ODP")
		return err
	}
	return nil
}

// List mengambil daftar ODP dengan paginasi dan filter.
func (m *odpManager) List(ctx context.Context, params domain.ODPListParams) (*domain.ODPListResult, error) {
	return m.odpRepo.List(ctx, params)
}

// odpToResponse mengkonversi entity ODP ke ODPResponse.
func odpToResponse(odp *domain.ODP) *domain.ODPResponse {
	return &domain.ODPResponse{
		ID:           odp.ID,
		OLTID:        odp.OLTID,
		PONPortIndex: odp.PONPortIndex,
		Name:         odp.Name,
		SplitterType: odp.SplitterType,
		Capacity:     odp.Capacity,
		UsedPorts:    odp.UsedPorts,
		Address:      odp.Address,
		Latitude:     odp.Latitude,
		Longitude:    odp.Longitude,
		Notes:        odp.Notes,
		CreatedAt:    odp.CreatedAt,
		UpdatedAt:    odp.UpdatedAt,
	}
}
