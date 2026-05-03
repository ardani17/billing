package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- Implementasi method sync-related untuk PPPoEUserRepo ---

// GetByRouterID mengambil semua PPPoE user aktif untuk satu router.
func (r *PPPoEUserRepo) GetByRouterID(ctx context.Context, routerID string) ([]*domain.PPPoEUser, error) {
	rows, err := r.queries.GetPPPoEUsersByRouterID(ctx, stringToUUID(routerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil pppoe users by router ID: %w", err)
	}

	users := make([]*domain.PPPoEUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, mapPPPoEUserRow(row))
	}
	return users, nil
}

// GetSyncStatusSummary mengambil ringkasan sync status per router.
// Memetakan count dari database ke domain.SyncStatusSummary.
func (r *PPPoEUserRepo) GetSyncStatusSummary(ctx context.Context, routerID string) (*domain.SyncStatusSummary, error) {
	row, err := r.queries.GetSyncStatusSummary(ctx, stringToUUID(routerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil sync status summary: %w", err)
	}

	// Konversi last_sync_at dari interface{} ke *time.Time
	var lastSyncAt *time.Time
	if row.LastSyncAt != nil {
		if t, ok := row.LastSyncAt.(time.Time); ok {
			lastSyncAt = &t
		}
	}

	return &domain.SyncStatusSummary{
		SyncedCount:    int(row.SyncedCount),
		MissingCount:   int(row.PendingCreateCount),
		OutOfSyncCount: int(row.OutOfSyncCount),
		LastSyncAt:     lastSyncAt,
	}, nil
}

// UpdateSyncStatus memperbarui sync_status dan last_sync_at untuk satu user.
func (r *PPPoEUserRepo) UpdateSyncStatus(ctx context.Context, id string, status domain.SyncStatus, syncAt *time.Time) error {
	err := r.queries.UpdateSyncStatus(ctx, UpdateSyncStatusParams{
		ID:         stringToUUID(id),
		SyncStatus: string(status),
		LastSyncAt: timePtrToTimestamptz(syncAt),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui sync status: %w", err)
	}
	return nil
}

// BulkUpdateSyncStatus memperbarui sync_status untuk banyak user sekaligus.
func (r *PPPoEUserRepo) BulkUpdateSyncStatus(ctx context.Context, ids []string, status domain.SyncStatus, syncAt *time.Time) error {
	// Konversi []string ke []pgtype.UUID
	uuids := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = stringToUUID(id)
	}

	err := r.queries.BulkUpdateSyncStatus(ctx, BulkUpdateSyncStatusParams{
		Column1:    uuids,
		SyncStatus: string(status),
		LastSyncAt: timePtrToTimestamptz(syncAt),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal bulk update sync status: %w", err)
	}
	return nil
}
