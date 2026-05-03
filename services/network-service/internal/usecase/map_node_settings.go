// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi operasi riwayat, trash, dan label settings untuk MapNodeManager.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetHistory mengambil riwayat perubahan node dengan paginasi.
func (m *mapNodeManager) GetHistory(ctx context.Context, nodeID string, limit, offset int) ([]*domain.MapChangeHistoryResponse, error) {
	entries, err := m.changeHistoryRepo.ListByNode(ctx, nodeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat: %w", err)
	}

	responses := make([]*domain.MapChangeHistoryResponse, 0, len(entries))
	for _, h := range entries {
		responses = append(responses, domain.ToMapChangeHistoryResponse(h))
	}

	return responses, nil
}

// ListTrashed mengambil daftar node yang ada di trash (sudah di-soft-delete).
func (m *mapNodeManager) ListTrashed(ctx context.Context, tenantID string) ([]*domain.MapNodeResponse, error) {
	nodes, err := m.mapNodeRepo.ListTrashed(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar trash: %w", err)
	}

	responses := make([]*domain.MapNodeResponse, 0, len(nodes))
	for _, n := range nodes {
		responses = append(responses, domain.ToMapNodeResponse(n))
	}

	return responses, nil
}

// GetLabelSettings mengambil konfigurasi label untuk tenant.
// Mengembalikan default settings jika tenant belum memiliki konfigurasi.
func (m *mapNodeManager) GetLabelSettings(ctx context.Context, tenantID string) (*domain.MapLabelSettingsResponse, error) {
	settings, err := m.labelSettingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil label settings: %w", err)
	}

	// Jika belum ada konfigurasi, kembalikan default
	if settings == nil {
		defaults := domain.NewDefaultLabelSettings(tenantID)
		return domain.ToMapLabelSettingsResponse(&defaults), nil
	}

	return domain.ToMapLabelSettingsResponse(settings), nil
}

// UpdateLabelSettings memperbarui konfigurasi label untuk tenant.
// Merge dengan settings yang sudah ada, atau buat baru jika belum ada.
func (m *mapNodeManager) UpdateLabelSettings(ctx context.Context, tenantID string, req domain.UpdateLabelSettingsRequest) (*domain.MapLabelSettingsResponse, error) {
	// Ambil settings yang sudah ada atau buat default
	existing, err := m.labelSettingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil label settings: %w", err)
	}

	var settings domain.MapLabelSettings
	if existing != nil {
		settings = *existing
	} else {
		settings = domain.NewDefaultLabelSettings(tenantID)
	}

	// Merge field yang dikirim
	if req.OLTLabels != nil {
		settings.OLTLabels = req.OLTLabels
	}
	if req.ODPLabels != nil {
		settings.ODPLabels = req.ODPLabels
	}
	if req.ONTLabels != nil {
		settings.ONTLabels = req.ONTLabels
	}
	if req.MinZoomLevel != nil {
		settings.MinZoomLevel = *req.MinZoomLevel
	}

	// Upsert ke database
	saved, err := m.labelSettingsRepo.Upsert(ctx, &settings)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan label settings: %w", err)
	}

	return domain.ToMapLabelSettingsResponse(saved), nil
}
