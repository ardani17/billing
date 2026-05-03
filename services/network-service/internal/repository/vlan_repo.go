package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// VLANRepo mengimplementasikan domain.VLANRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.VLAN.
type VLANRepo struct {
	queries *Queries
}

// NewVLANRepo membuat instance baru VLANRepo.
func NewVLANRepo(queries *Queries) *VLANRepo {
	return &VLANRepo{queries: queries}
}

// --- Mapping sqlc Vlan → domain.VLAN ---

// mapVLANRow memetakan Vlan (sqlc model) ke domain.VLAN.
func mapVLANRow(row Vlan) *domain.VLAN {
	return &domain.VLAN{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		OLTID:       uuidToString(row.OltID),
		VLANID:      int(row.VlanID),
		Name:        row.Name,
		VLANType:    domain.VLANType(row.VlanType),
		Description: textToString(row.Description),
		DeletedAt:   timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:   timestamptzToTime(row.CreatedAt),
		UpdatedAt:   timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi CRUD domain.VLANRepository ---

// Create membuat VLAN baru dan mengembalikan VLAN yang dibuat.
func (r *VLANRepo) Create(ctx context.Context, vlan *domain.VLAN) (*domain.VLAN, error) {
	row, err := r.queries.CreateVLAN(ctx, CreateVLANParams{
		TenantID:    stringToUUID(vlan.TenantID),
		OltID:       stringToUUID(vlan.OLTID),
		VlanID:      int32(vlan.VLANID),
		Name:        vlan.Name,
		VlanType:    string(vlan.VLANType),
		Description: stringToText(vlan.Description),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat VLAN: %w", err)
	}
	return mapVLANRow(row), nil
}

// GetByID mengambil VLAN berdasarkan ID (tenant-scoped via RLS).
func (r *VLANRepo) GetByID(ctx context.Context, id string) (*domain.VLAN, error) {
	row, err := r.queries.GetVLANByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVLANNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil VLAN by ID: %w", err)
	}
	return mapVLANRow(row), nil
}

// Update memperbarui data VLAN dan mengembalikan VLAN yang diperbarui.
func (r *VLANRepo) Update(ctx context.Context, vlan *domain.VLAN) (*domain.VLAN, error) {
	row, err := r.queries.UpdateVLAN(ctx, UpdateVLANParams{
		ID:          stringToUUID(vlan.ID),
		Name:        vlan.Name,
		VlanType:    string(vlan.VLANType),
		Description: stringToText(vlan.Description),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVLANNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui VLAN: %w", err)
	}
	return mapVLANRow(row), nil
}

// SoftDelete melakukan soft-delete VLAN (set deleted_at).
func (r *VLANRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteVLAN(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete VLAN: %w", err)
	}
	return nil
}

// List mengambil daftar VLAN per OLT dengan paginasi.
func (r *VLANRepo) List(ctx context.Context, oltID string, params domain.VLANListParams) (*domain.VLANListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Ambil total count untuk paginasi
	total, err := r.queries.CountVLANs(ctx, stringToUUID(oltID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total VLAN: %w", err)
	}

	// Ambil data VLAN
	rows, err := r.queries.ListVLANs(ctx, ListVLANsParams{
		OltID:  stringToUUID(oltID),
		Limit:  int32(params.PageSize),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar VLAN: %w", err)
	}

	// Konversi ke domain.VLANResponse untuk list
	responses := make([]*domain.VLANResponse, 0, len(rows))
	for _, row := range rows {
		vlan := mapVLANRow(row)
		responses = append(responses, &domain.VLANResponse{
			ID:          vlan.ID,
			OLTID:       vlan.OLTID,
			VLANID:      vlan.VLANID,
			Name:        vlan.Name,
			VLANType:    string(vlan.VLANType),
			Description: vlan.Description,
			CreatedAt:   vlan.CreatedAt,
			UpdatedAt:   vlan.UpdatedAt,
		})
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.VLANListResult{
		Data:       responses,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByOLTAndVLANID mengambil VLAN berdasarkan olt_id dan vlan_id.
func (r *VLANRepo) GetByOLTAndVLANID(ctx context.Context, oltID string, vlanID int) (*domain.VLAN, error) {
	row, err := r.queries.GetVLANByOLTAndVLANID(ctx, GetVLANByOLTAndVLANIDParams{
		OltID:  stringToUUID(oltID),
		VlanID: int32(vlanID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVLANNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil VLAN by OLT dan VLAN ID: %w", err)
	}
	return mapVLANRow(row), nil
}

// GetDefaultVLAN mengambil VLAN default untuk OLT (VLAN pertama tipe data).
func (r *VLANRepo) GetDefaultVLAN(ctx context.Context, oltID string) (*domain.VLAN, error) {
	row, err := r.queries.GetDefaultVLAN(ctx, stringToUUID(oltID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVLANNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil default VLAN: %w", err)
	}
	return mapVLANRow(row), nil
}

// VLANIDExists mengecek apakah vlan_id sudah ada pada OLT yang sama.
func (r *VLANRepo) VLANIDExists(ctx context.Context, oltID string, vlanID int, excludeID string) (bool, error) {
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.VLANIDExists(ctx, VLANIDExistsParams{
		OltID:  stringToUUID(oltID),
		VlanID: int32(vlanID),
		ID:     stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek VLAN ID: %w", err)
	}
	return exists, nil
}

// CountActiveONTs menghitung jumlah ONT aktif yang menggunakan VLAN ini.
func (r *VLANRepo) CountActiveONTs(ctx context.Context, vlanID string) (int64, error) {
	count, err := r.queries.CountVLANActiveONTs(ctx, stringToUUID(vlanID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung ONT aktif VLAN: %w", err)
	}
	return count, nil
}

// Compile-time check: VLANRepo mengimplementasikan domain.VLANRepository.
var _ domain.VLANRepository = (*VLANRepo)(nil)
