package repository

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ODPRepo mengimplementasikan domain.ODPRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.ODP.
type ODPRepo struct {
	queries *Queries
}

// NewODPRepo membuat instance baru ODPRepo.
func NewODPRepo(queries *Queries) *ODPRepo {
	return &ODPRepo{queries: queries}
}

// --- Fungsi bantu functions untuk konversi pgtype.Numeric ↔ float64 ---

// numericToFloat64Ptr mengkonversi pgtype.Numeric ke *float64.
// Mengembalikan nil jika Numeric tidak valid.
func numericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, _ := n.Float64Value()
	v := f.Float64
	return &v
}

// float64PtrToNumeric mengkonversi *float64 ke pgtype.Numeric.
// Mengembalikan Numeric tidak valid jika pointer nil.
func float64PtrToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	var n pgtype.Numeric
	n.Valid = true
	bf := new(big.Float).SetFloat64(*f)
	_ = n.Scan(bf.Text('f', 7))
	return n
}

// --- Mapping sqlc Odp -> domain.ODP ---

// mapODPRow memetakan Odp (sqlc model) ke domain.ODP.
func mapODPRow(row Odp) *domain.ODP {
	return &domain.ODP{
		ID:           uuidToString(row.ID),
		TenantID:     uuidToString(row.TenantID),
		OLTID:        uuidToString(row.OltID),
		PONPortIndex: int(row.PonPortIndex),
		Name:         row.Name,
		SplitterType: row.SplitterType,
		Capacity:     int(row.Capacity),
		UsedPorts:    int(row.UsedPorts),
		Address:      textToString(row.Address),
		Latitude:     numericToFloat64Ptr(row.Latitude),
		Longitude:    numericToFloat64Ptr(row.Longitude),
		Notes:        textToString(row.Notes),
		DeletedAt:    timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
		UpdatedAt:    timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi CRUD domain.ODPRepository ---

// Buat membuat ODP baru dan mengembalikan ODP yang dibuat.
func (r *ODPRepo) Create(ctx context.Context, odp *domain.ODP) (*domain.ODP, error) {
	row, err := r.queries.CreateODP(ctx, CreateODPParams{
		TenantID:     stringToUUID(odp.TenantID),
		OltID:        stringToUUID(odp.OLTID),
		PonPortIndex: int32(odp.PONPortIndex),
		Name:         odp.Name,
		SplitterType: odp.SplitterType,
		Capacity:     int32(odp.Capacity),
		UsedPorts:    int32(odp.UsedPorts),
		Address:      stringToText(odp.Address),
		Latitude:     float64PtrToNumeric(odp.Latitude),
		Longitude:    float64PtrToNumeric(odp.Longitude),
		Notes:        stringToText(odp.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat ODP: %w", err)
	}
	return mapODPRow(row), nil
}

// GetByID mengambil ODP berdasarkan ID (tenant-scoped via RLS).
func (r *ODPRepo) GetByID(ctx context.Context, id string) (*domain.ODP, error) {
	row, err := r.queries.GetODPByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrODPNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil ODP by ID: %w", err)
	}
	return mapODPRow(row), nil
}

// Perbarui memperbarui data ODP dan mengembalikan ODP yang diperbarui.
func (r *ODPRepo) Update(ctx context.Context, odp *domain.ODP) (*domain.ODP, error) {
	row, err := r.queries.UpdateODP(ctx, UpdateODPParams{
		ID:        stringToUUID(odp.ID),
		Name:      odp.Name,
		Address:   stringToText(odp.Address),
		Latitude:  float64PtrToNumeric(odp.Latitude),
		Longitude: float64PtrToNumeric(odp.Longitude),
		Notes:     stringToText(odp.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrODPNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui ODP: %w", err)
	}
	return mapODPRow(row), nil
}

// SoftDelete melakukan hapus lunak ODP (atur deleted_at).
func (r *ODPRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteODP(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete ODP: %w", err)
	}
	return nil
}

// Compile-time cek: ODPRepo mengimplementasikan domain.ODPRepository.
var _ domain.ODPRepository = (*ODPRepo)(nil)
