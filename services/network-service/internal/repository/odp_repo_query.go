package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// odp_repo_kueri.go berisi method kueri ODPRepo: List, NameExists, GetByOLTAndPort.

// List mengambil daftar ODP dengan paginasi dan filter (tenant-scoped via RLS).
func (r *ODPRepo) List(ctx context.Context, params domain.ODPListParams) (*domain.ODPListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Konversi filter olt_id ke pgtype.UUID (NULL jika kosong)
	oltID := pgtype.UUID{Valid: false}
	if params.OLTID != "" {
		oltID = stringToUUID(params.OLTID)
	}

	// Konversi filter pon_port_index ke pgtype.Int4 (NULL jika nil)
	ponPort := pgtype.Int4{Valid: false}
	if params.PONPortIndex != nil {
		ponPort = pgtype.Int4{Int32: int32(*params.PONPortIndex), Valid: true}
	}

	// Ambil total count untuk paginasi
	total, err := r.queries.CountODPs(ctx, CountODPsParams{
		OltID:        oltID,
		PonPortIndex: ponPort,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total ODP: %w", err)
	}

	// Ambil data ODP
	rows, err := r.queries.ListODPs(ctx, ListODPsParams{
		Limit:        int32(params.PageSize),
		Offset:       int32(offset),
		OltID:        oltID,
		PonPortIndex: ponPort,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar ODP: %w", err)
	}

	// Konversi ke domain.ODPResponse untuk list
	responses := make([]*domain.ODPResponse, 0, len(rows))
	for _, row := range rows {
		odp := mapODPRow(row)
		responses = append(responses, &domain.ODPResponse{
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
		})
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.ODPListResult{
		Data:       responses,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// NameExists mengecek apakah nama ODP sudah ada di tenant.
// excludeID digunakan untuk mengecualikan ODP tertentu (saat perbarui).
func (r *ODPRepo) NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.ODPNameExists(ctx, ODPNameExistsParams{
		TenantID: stringToUUID(tenantID),
		Name:     name,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek nama ODP: %w", err)
	}
	return exists, nil
}

// GetByOLTAndPort mengambil semua ODP untuk satu OLT dan PON port.
func (r *ODPRepo) GetByOLTAndPort(ctx context.Context, oltID string, ponPort int) ([]*domain.ODP, error) {
	rows, err := r.queries.GetODPsByOLTAndPort(ctx, GetODPsByOLTAndPortParams{
		OltID:        stringToUUID(oltID),
		PonPortIndex: int32(ponPort),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil ODP by OLT dan port: %w", err)
	}

	odps := make([]*domain.ODP, 0, len(rows))
	for _, row := range rows {
		odps = append(odps, mapODPRow(row))
	}
	return odps, nil
}
