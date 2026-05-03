package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PendingSyncRepo mengimplementasikan domain.PendingSyncRepository.
type PendingSyncRepo struct{ queries *Queries }

func NewPendingSyncRepo(q *Queries) *PendingSyncRepo { return &PendingSyncRepo{queries: q} }

func timePtrToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func mapPendingSync(row PendingSync) *domain.PendingSync {
	ps := &domain.PendingSync{
		ID: uuidToString(row.ID), TenantID: uuidToString(row.TenantID),
		CustomerID: uuidToString(row.CustomerID), OperationType: domain.SyncOperationType(row.OperationType),
		Status: domain.SyncStatus(row.Status), RetryCount: int(row.RetryCount), MaxRetries: int(row.MaxRetries),
		LastRetryAt: timestamptzToTimePtr(row.LastRetryAt), NextRetryAt: timestamptzToTimePtr(row.NextRetryAt),
		ErrorMessage: textToString(row.ErrorMessage),
		CreatedAt: timestamptzToTime(row.CreatedAt), UpdatedAt: timestamptzToTime(row.UpdatedAt),
	}
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &ps.Metadata)
	}
	return ps
}

func mapPendingSyncSlice(rows []PendingSync) []*domain.PendingSync {
	out := make([]*domain.PendingSync, 0, len(rows))
	for _, r := range rows {
		out = append(out, mapPendingSync(r))
	}
	return out
}

func (r *PendingSyncRepo) Create(ctx context.Context, s *domain.PendingSync) (*domain.PendingSync, error) {
	var mb []byte
	if s.Metadata != nil {
		b, err := json.Marshal(s.Metadata)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal marshal metadata: %w", err)
		}
		mb = b
	}
	row, err := r.queries.CreatePendingSync(ctx, CreatePendingSyncParams{
		TenantID: stringToUUID(s.TenantID), CustomerID: stringToUUID(s.CustomerID),
		OperationType: string(s.OperationType), Status: string(s.Status),
		RetryCount: int32(s.RetryCount), MaxRetries: int32(s.MaxRetries),
		NextRetryAt: timePtrToTimestamptz(s.NextRetryAt), Metadata: mb,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat pending sync: %w", err)
	}
	return mapPendingSync(row), nil
}

func (r *PendingSyncRepo) GetByID(ctx context.Context, id string) (*domain.PendingSync, error) {
	row, err := r.queries.GetPendingSyncByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNoPendingSync
		}
		return nil, fmt.Errorf("repository: gagal mengambil pending sync: %w", err)
	}
	return mapPendingSync(row), nil
}

func (r *PendingSyncRepo) UpdateStatus(ctx context.Context, id string, status domain.SyncStatus) error {
	return r.queries.UpdatePendingSyncStatus(ctx, UpdatePendingSyncStatusParams{ID: stringToUUID(id), Status: string(status)})
}

func (r *PendingSyncRepo) UpdateRetry(ctx context.Context, id string, retryCount int, nextRetryAt time.Time, errMsg string) error {
	return r.queries.UpdatePendingSyncRetry(ctx, UpdatePendingSyncRetryParams{
		ID: stringToUUID(id), RetryCount: int32(retryCount), NextRetryAt: timeToTimestamptz(nextRetryAt),
		LastRetryAt: timeToTimestamptz(time.Now()), ErrorMessage: stringToText(errMsg),
	})
}

func (r *PendingSyncRepo) MarkCompleted(ctx context.Context, id string) error {
	return r.queries.MarkPendingSyncCompleted(ctx, stringToUUID(id))
}

func (r *PendingSyncRepo) MarkFailed(ctx context.Context, id string, errMsg string) error {
	return r.queries.MarkPendingSyncFailed(ctx, MarkPendingSyncFailedParams{ID: stringToUUID(id), ErrorMessage: stringToText(errMsg)})
}

func (r *PendingSyncRepo) FindPendingForRetry(ctx context.Context, batchSize int) ([]*domain.PendingSync, error) {
	rows, err := r.queries.FindPendingSyncsForRetry(ctx, int32(batchSize))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil pending syncs for retry: %w", err)
	}
	return mapPendingSyncSlice(rows), nil
}

func (r *PendingSyncRepo) FindByCustomer(ctx context.Context, customerID string) ([]*domain.PendingSync, error) {
	rows, err := r.queries.FindPendingSyncsByCustomer(ctx, stringToUUID(customerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil pending syncs by customer: %w", err)
	}
	return mapPendingSyncSlice(rows), nil
}

// FindByTenantAndStatus mengambil pending_syncs (paginated), termasuk CustomerName/CustomerIDSeq dari JOIN.
func (r *PendingSyncRepo) FindByTenantAndStatus(ctx context.Context, tenantID string, status *domain.SyncStatus, page, pageSize int) (*domain.PendingSyncListResult, error) {
	statuses := []string{"pending", "in_progress", "completed", "failed"}
	if status != nil {
		statuses = []string{string(*status)}
	}
	total, err := r.queries.CountPendingSyncsByTenantAndStatuses(ctx, CountPendingSyncsByTenantAndStatusesParams{
		TenantID: stringToUUID(tenantID), Statuses: statuses,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung pending syncs: %w", err)
	}
	var sf pgtype.Text
	if status != nil {
		sf = pgtype.Text{String: string(*status), Valid: true}
	}
	rows, err := r.queries.FindPendingSyncsByTenantAndStatus(ctx, FindPendingSyncsByTenantAndStatusParams{
		TenantID: stringToUUID(tenantID), Status: sf, Offset: int32((page - 1) * pageSize), PageSize: int32(pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil pending syncs by tenant: %w", err)
	}
	items := make([]*domain.PendingSync, 0, len(rows))
	for _, row := range rows {
		ps := &domain.PendingSync{
			ID: uuidToString(row.ID), TenantID: uuidToString(row.TenantID),
			CustomerID: uuidToString(row.CustomerID), OperationType: domain.SyncOperationType(row.OperationType),
			Status: domain.SyncStatus(row.Status), RetryCount: int(row.RetryCount), MaxRetries: int(row.MaxRetries),
			LastRetryAt: timestamptzToTimePtr(row.LastRetryAt), NextRetryAt: timestamptzToTimePtr(row.NextRetryAt),
			ErrorMessage: textToString(row.ErrorMessage), CreatedAt: timestamptzToTime(row.CreatedAt),
			UpdatedAt: timestamptzToTime(row.UpdatedAt), CustomerName: textToString(row.CustomerName),
			CustomerIDSeq: textToString(row.CustomerIDSeq),
		}
		if len(row.Metadata) > 0 {
			_ = json.Unmarshal(row.Metadata, &ps.Metadata)
		}
		items = append(items, ps)
	}
	tp := int((total + int64(pageSize) - 1) / int64(pageSize))
	if tp < 1 { tp = 1 }
	return &domain.PendingSyncListResult{Items: items, Total: total, Page: page, PageSize: pageSize, TotalPages: tp}, nil
}

func (r *PendingSyncRepo) ResetRetryForCustomer(ctx context.Context, customerID string) error {
	return r.queries.ResetRetryForCustomer(ctx, stringToUUID(customerID))
}

func (r *PendingSyncRepo) ResetRetryAll(ctx context.Context, tenantID string) (int, error) {
	count, err := r.queries.ResetRetryAll(ctx, stringToUUID(tenantID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal reset retry all: %w", err)
	}
	return int(count), nil
}

func (r *PendingSyncRepo) CountByTenantAndStatuses(ctx context.Context, tenantID string, statuses []domain.SyncStatus) (int64, error) {
	strs := make([]string, len(statuses))
	for i, s := range statuses {
		strs[i] = string(s)
	}
	return r.queries.CountPendingSyncsByTenantAndStatuses(ctx, CountPendingSyncsByTenantAndStatusesParams{
		TenantID: stringToUUID(tenantID), Statuses: strs,
	})
}
