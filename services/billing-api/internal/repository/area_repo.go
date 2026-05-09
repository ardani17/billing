package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
)

// AreaRepo mengimplementasikan domain.AreaRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.Area.
type AreaRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi area.
	queries *Queries
}

// NewAreaRepo membuat instance baru AreaRepo.
func NewAreaRepo(queries *Queries) *AreaRepo {
	return &AreaRepo{
		queries: queries,
	}
}

// --- Helper functions untuk mapping sqlc row -> domain.Area ---

// mapAreaRow memetakan Area (sqlc model) ke domain.Area.
func mapAreaRow(row Area) *domain.Area {
	return &domain.Area{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		Name:        row.Name,
		Description: textToString(row.Description),
		ODPID:       textToString(row.OdpID),
		CenterLat:   numericToFloat64Ptr(row.CenterLat),
		CenterLng:   numericToFloat64Ptr(row.CenterLng),
		CreatedAt:   timestamptzToTime(row.CreatedAt),
		UpdatedAt:   timestamptzToTime(row.UpdatedAt),
	}
}

// mapListAreasRow memetakan ListAreasRow (sqlc model) ke domain.Area.
func mapListAreasRow(row ListAreasRow) *domain.Area {
	return &domain.Area{
		ID:            uuidToString(row.ID),
		TenantID:      uuidToString(row.TenantID),
		Name:          row.Name,
		Description:   textToString(row.Description),
		ODPID:         textToString(row.OdpID),
		CenterLat:     numericToFloat64Ptr(row.CenterLat),
		CenterLng:     numericToFloat64Ptr(row.CenterLng),
		CustomerCount: int(row.CustomerCount),
		CreatedAt:     timestamptzToTime(row.CreatedAt),
		UpdatedAt:     timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.AreaRepository ---

// Buat membuat area baru dan mengembalikan area yang dibuat.
func (r *AreaRepo) Create(ctx context.Context, area *domain.Area) (*domain.Area, error) {
	row, err := r.queries.CreateArea(ctx, CreateAreaParams{
		TenantID:    stringToUUID(area.TenantID),
		Name:        area.Name,
		Description: stringToText(area.Description),
		OdpID:       stringToText(area.ODPID),
		CenterLat:   float64PtrToNumeric(area.CenterLat),
		CenterLng:   float64PtrToNumeric(area.CenterLng),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat area: %w", err)
	}
	return mapAreaRow(row), nil
}

// GetByID mengambil area berdasarkan ID.
func (r *AreaRepo) GetByID(ctx context.Context, id string) (*domain.Area, error) {
	row, err := r.queries.GetAreaByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAreaNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil area by ID: %w", err)
	}
	return mapAreaRow(row), nil
}

// Perbarui memperbarui data area dan mengembalikan area yang diperbarui.
func (r *AreaRepo) Update(ctx context.Context, area *domain.Area) (*domain.Area, error) {
	row, err := r.queries.UpdateArea(ctx, UpdateAreaParams{
		ID:          stringToUUID(area.ID),
		Name:        area.Name,
		Description: stringToText(area.Description),
		OdpID:       stringToText(area.ODPID),
		CenterLat:   float64PtrToNumeric(area.CenterLat),
		CenterLng:   float64PtrToNumeric(area.CenterLng),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAreaNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui area: %w", err)
	}
	return mapAreaRow(row), nil
}

// Hapus menghapus area berdasarkan ID.
func (r *AreaRepo) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteArea(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus area: %w", err)
	}
	return nil
}

// List mengambil semua area untuk tenant tertentu beserta jumlah customer per area.
func (r *AreaRepo) List(ctx context.Context, tenantID string) ([]*domain.Area, error) {
	rows, err := r.queries.ListAreas(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar area: %w", err)
	}

	areas := make([]*domain.Area, 0, len(rows))
	for _, row := range rows {
		areas = append(areas, mapListAreasRow(row))
	}
	return areas, nil
}

// NameExists mengecek apakah nama area sudah terdaftar di tenant yang sama.
// excludeID digunakan untuk mengecualikan area tertentu (saat perbarui).
func (r *AreaRepo) NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}
	exists, err := r.queries.AreaNameExists(ctx, AreaNameExistsParams{
		TenantID: stringToUUID(tenantID),
		Name:     name,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek area name exists: %w", err)
	}
	return exists, nil
}

// CustomerCount mengembalikan jumlah customer yang terkait dengan area tertentu.
func (r *AreaRepo) CustomerCount(ctx context.Context, id string) (int, error) {
	count, err := r.queries.AreaCustomerCount(ctx, stringToUUID(id))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung customer di area: %w", err)
	}
	return int(count), nil
}
