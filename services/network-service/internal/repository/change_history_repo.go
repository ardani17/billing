package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ChangeHistoryRepo mengimplementasikan domain.ChangeHistoryRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.MapChangeHistory.
// Tabel ini bersifat append-only: hanya Buat dan ListByNode, tidak ada Perbarui atau Hapus.
type ChangeHistoryRepo struct {
	db DBTX
}

// NewChangeHistoryRepo membuat instance baru ChangeHistoryRepo.
func NewChangeHistoryRepo(db DBTX) *ChangeHistoryRepo {
	return &ChangeHistoryRepo{db: db}
}

// scanChangeHistory memindai satu baris hasil kueri ke domain.MapChangeHistory.
func scanChangeHistory(row pgx.Row) (*domain.MapChangeHistory, error) {
	var h domain.MapChangeHistory
	err := row.Scan(
		&h.ID, &h.TenantID, &h.MapNodeID, &h.Action,
		&h.OldValue, &h.NewValue, &h.PerformedBy, &h.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// Buat menyimpan entri riwayat perubahan baru.
func (r *ChangeHistoryRepo) Create(ctx context.Context, entry *domain.MapChangeHistory) (*domain.MapChangeHistory, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO map_change_history (tenant_id, map_node_id, action, old_value, new_value, performed_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, map_node_id, action, old_value, new_value, performed_by, created_at`,
		entry.TenantID, entry.MapNodeID, entry.Action,
		entry.OldValue, entry.NewValue, entry.PerformedBy,
	)
	result, err := scanChangeHistory(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat change history: %w", err)
	}
	return result, nil
}

// ListByNode mengambil daftar riwayat perubahan untuk satu node dengan paginasi.
func (r *ChangeHistoryRepo) ListByNode(ctx context.Context, nodeID string, limit, offset int) ([]*domain.MapChangeHistory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, map_node_id, action, old_value, new_value, performed_by, created_at
		 FROM map_change_history WHERE map_node_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		nodeID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil change history: %w", err)
	}
	defer rows.Close()

	var results []*domain.MapChangeHistory
	for rows.Next() {
		var h domain.MapChangeHistory
		err := rows.Scan(
			&h.ID, &h.TenantID, &h.MapNodeID, &h.Action,
			&h.OldValue, &h.NewValue, &h.PerformedBy, &h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan change history: %w", err)
		}
		results = append(results, &h)
	}
	return results, rows.Err()
}

// Compile-time cek: ChangeHistoryRepo mengimplementasikan domain.ChangeHistoryRepository.
var _ domain.ChangeHistoryRepository = (*ChangeHistoryRepo)(nil)
