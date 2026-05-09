// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi operasi hapus, restore, list, search untuk MapNodeManager.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// DeleteNode melakukan hapus lunak node dengan pencatatan riwayat.
func (m *mapNodeManager) DeleteNode(ctx context.Context, id string, performedBy string) error {
	// Pastikan node ada
	node, err := m.mapNodeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := m.mapNodeRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("gagal menghapus node: %w", err)
	}

	// Catat riwayat penghapusan
	m.recordHistory(ctx, node.TenantID, id, domain.ChangeActionDeleted, nil, nil, performedBy)

	return nil
}

// RestoreNode mengembalikan node dari trash dengan pencatatan riwayat.
func (m *mapNodeManager) RestoreNode(ctx context.Context, id, performedBy string) error {
	if err := m.mapNodeRepo.Restore(ctx, id); err != nil {
		return fmt.Errorf("gagal mengembalikan node: %w", err)
	}

	// Ambil node untuk mendapatkan tenant_id
	node, err := m.mapNodeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Catat riwayat restore
	m.recordHistory(ctx, node.TenantID, id, domain.ChangeActionRestored, nil, nil, performedBy)

	return nil
}

// ListNodes mengambil daftar node berdasarkan bounding box dan filter.
// Mengembalikan node dengan data referensi (OLT/ODP/ONT) yang sudah di-join.
func (m *mapNodeManager) ListNodes(ctx context.Context, params domain.MapNodeListParams) ([]*domain.MapNodeWithRefResponse, error) {
	nodes, err := m.mapNodeRepo.ListByBounds(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar node: %w", err)
	}

	responses := make([]*domain.MapNodeWithRefResponse, 0, len(nodes))
	for _, n := range nodes {
		responses = append(responses, domain.ToMapNodeWithRefResponse(n))
	}

	return responses, nil
}

// Pencarian melakukan pencarian full-text di node dan entitas referensi.
// Mengembalikan maksimal 20 hasil pencarian.
func (m *mapNodeManager) Search(ctx context.Context, tenantID, query string) ([]*domain.MapSearchResult, error) {
	const maxSearchResults = 20

	results, err := m.mapNodeRepo.Search(ctx, tenantID, query, maxSearchResults)
	if err != nil {
		return nil, fmt.Errorf("gagal melakukan pencarian: %w", err)
	}

	return results, nil
}
