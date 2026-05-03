package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// olt_repo_query.go berisi method query OLTRepo: List, CountByStatus,
// GetActiveOLTs, GetOnlineOLTs, NameExists, UpdateHealthCheck, UpdateONTCounts.

// List mengambil daftar OLT dengan paginasi dan filter (tenant-scoped via RLS).
func (r *OLTRepo) List(ctx context.Context, params domain.OLTListParams) (*domain.OLTListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Ambil total count untuk paginasi
	total, err := r.queries.CountOLTs(ctx, CountOLTsParams{
		Status: stringToText(params.Status),
		Brand:  stringToText(params.Brand),
		Search: stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total OLT: %w", err)
	}

	// Ambil data OLT
	rows, err := r.queries.ListOLTs(ctx, ListOLTsParams{
		Limit:  int32(params.PageSize),
		Offset: int32(offset),
		Status: stringToText(params.Status),
		Brand:  stringToText(params.Brand),
		Search: stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar OLT: %w", err)
	}

	// Konversi ke domain.OLTResponse untuk list
	responses := make([]*domain.OLTResponse, 0, len(rows))
	for _, row := range rows {
		olt := mapOLTRow(row)
		responses = append(responses, &domain.OLTResponse{
			ID:                     olt.ID,
			Name:                   olt.Name,
			Host:                   olt.Host,
			Brand:                  olt.Brand,
			Model:                  olt.Model,
			FirmwareVersion:        olt.FirmwareVersion,
			PONPortCount:           olt.PONPortCount,
			TotalONTCount:          olt.TotalONTCount,
			Status:                 olt.Status,
			HealthCheckIntervalSec: olt.HealthCheckIntervalSec,
			LastOnlineAt:           olt.LastOnlineAt,
			Notes:                  olt.Notes,
			CreatedAt:              olt.CreatedAt,
			UpdatedAt:              olt.UpdatedAt,
		})
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.OLTListResult{
		Data:       responses,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CountByStatus menghitung jumlah OLT per status untuk tenant.
func (r *OLTRepo) CountByStatus(ctx context.Context) (map[domain.OLTStatus]int64, error) {
	rows, err := r.queries.CountOLTsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung OLT per status: %w", err)
	}

	result := make(map[domain.OLTStatus]int64)
	for _, row := range rows {
		result[domain.OLTStatus(row.Status)] = row.Count
	}
	return result, nil
}

// GetActiveOLTs mengambil semua OLT yang tidak di-delete dan bukan maintenance.
func (r *OLTRepo) GetActiveOLTs(ctx context.Context) ([]*domain.OLT, error) {
	rows, err := r.queries.GetActiveOLTs(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil OLT aktif: %w", err)
	}

	olts := make([]*domain.OLT, 0, len(rows))
	for _, row := range rows {
		olts = append(olts, mapOLTRow(row))
	}
	return olts, nil
}

// GetOnlineOLTs mengambil semua OLT dengan status online.
func (r *OLTRepo) GetOnlineOLTs(ctx context.Context) ([]*domain.OLT, error) {
	rows, err := r.queries.GetOnlineOLTs(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil OLT online: %w", err)
	}

	olts := make([]*domain.OLT, 0, len(rows))
	for _, row := range rows {
		olts = append(olts, mapOLTRow(row))
	}
	return olts, nil
}

// NameExists mengecek apakah nama OLT sudah ada di tenant.
// excludeID digunakan untuk mengecualikan OLT tertentu (saat update).
func (r *OLTRepo) NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.OLTNameExists(ctx, OLTNameExistsParams{
		TenantID: stringToUUID(tenantID),
		Name:     name,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek nama OLT: %w", err)
	}
	return exists, nil
}

// UpdateHealthCheck memperbarui field health check OLT.
func (r *OLTRepo) UpdateHealthCheck(ctx context.Context, id string, params domain.OLTHealthCheckUpdate) error {
	// Tentukan status string, gunakan empty string jika nil
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}

	err := r.queries.UpdateOLTHealthCheck(ctx, UpdateOLTHealthCheckParams{
		ID:            stringToUUID(id),
		LastCheckedAt: timePtrToTimestamptz(params.LastCheckedAt),
		LastOnlineAt:  timePtrToTimestamptz(params.LastOnlineAt),
		FailureCount:  int32(params.FailureCount),
		Status:        status,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui health check OLT: %w", err)
	}
	return nil
}

// UpdateONTCounts memperbarui total_ont_count setelah sync.
func (r *OLTRepo) UpdateONTCounts(ctx context.Context, id string, totalONT int) error {
	err := r.queries.UpdateOLTONTCounts(ctx, UpdateOLTONTCountsParams{
		ID:            stringToUUID(id),
		TotalOntCount: int32(totalONT),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui ONT count: %w", err)
	}
	return nil
}
