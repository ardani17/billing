package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type MikroTikBulkJobRepo struct {
	queries *Queries
}

func NewMikroTikBulkJobRepo(queries *Queries) *MikroTikBulkJobRepo {
	return &MikroTikBulkJobRepo{queries: queries}
}

func (r *MikroTikBulkJobRepo) Create(ctx context.Context, input domain.CreateMikroTikBulkJobInput) (*domain.MikroTikBulkJob, error) {
	const sql = `
INSERT INTO mikrotik_bulk_jobs (
    tenant_id, action, status, router_ids, total_count, requested_by, started_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, tenant_id, action, status, router_ids, total_count, success_count, failed_count,
    results, error_message, requested_by, started_at, finished_at, created_at, updated_at
`
	item, err := scanMikroTikBulkJob(r.queries.db.QueryRow(ctx, sql,
		stringToUUID(input.TenantID),
		string(input.Action),
		string(input.Status),
		stringSliceToUUIDs(input.RouterIDs),
		input.TotalCount,
		optionalUUID(input.RequestedBy),
		timePtrToTimestamptz(input.StartedAt),
	))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat bulk job mikrotik: %w", err)
	}
	return item, nil
}

func (r *MikroTikBulkJobRepo) GetByID(ctx context.Context, id string) (*domain.MikroTikBulkJob, error) {
	const sql = `
SELECT id, tenant_id, action, status, router_ids, total_count, success_count, failed_count,
    results, error_message, requested_by, started_at, finished_at, created_at, updated_at
FROM mikrotik_bulk_jobs
WHERE id = $1
`
	item, err := scanMikroTikBulkJob(r.queries.db.QueryRow(ctx, sql, stringToUUID(id)))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrMikroTikBulkJobNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil bulk job mikrotik: %w", err)
	}
	return item, nil
}

func (r *MikroTikBulkJobRepo) List(ctx context.Context, params domain.MikroTikBulkJobListParams) (*domain.MikroTikBulkJobListResult, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	const countSQL = `
SELECT COUNT(*)
FROM mikrotik_bulk_jobs
WHERE ($1 = '' OR action = $1)
  AND ($2 = '' OR status = $2)
`
	var total int64
	if err := r.queries.db.QueryRow(ctx, countSQL, params.Action, params.Status).Scan(&total); err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung bulk job mikrotik: %w", err)
	}

	const listSQL = `
SELECT id, tenant_id, action, status, router_ids, total_count, success_count, failed_count,
    results, error_message, requested_by, started_at, finished_at, created_at, updated_at
FROM mikrotik_bulk_jobs
WHERE ($1 = '' OR action = $1)
  AND ($2 = '' OR status = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
`
	rows, err := r.queries.db.Query(ctx, listSQL, params.Action, params.Status, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil list bulk job mikrotik: %w", err)
	}
	defer rows.Close()

	items := make([]domain.MikroTikBulkJob, 0, pageSize)
	for rows.Next() {
		item, err := scanMikroTikBulkJob(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal membaca bulk job mikrotik: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi bulk job mikrotik: %w", err)
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	return &domain.MikroTikBulkJobListResult{
		Data: items, Total: total, Page: page, PageSize: pageSize, TotalPages: int(totalPages),
	}, nil
}

func (r *MikroTikBulkJobRepo) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	const sql = `
UPDATE mikrotik_bulk_jobs
SET status = 'running', started_at = $2, updated_at = now()
WHERE id = $1
`
	tag, err := r.queries.db.Exec(ctx, sql, stringToUUID(id), startedAt)
	if err != nil {
		return fmt.Errorf("repository: gagal menandai bulk job running: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMikroTikBulkJobNotFound
	}
	return nil
}

func (r *MikroTikBulkJobRepo) Complete(ctx context.Context, input domain.UpdateMikroTikBulkJobResultInput) (*domain.MikroTikBulkJob, error) {
	resultsJSON, err := json.Marshal(input.Results)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal hasil bulk job: %w", err)
	}
	const sql = `
UPDATE mikrotik_bulk_jobs
SET status = $2,
    success_count = $3,
    failed_count = $4,
    results = $5,
    error_message = $6,
    finished_at = $7,
    updated_at = now()
WHERE id = $1
RETURNING id, tenant_id, action, status, router_ids, total_count, success_count, failed_count,
    results, error_message, requested_by, started_at, finished_at, created_at, updated_at
`
	item, err := scanMikroTikBulkJob(r.queries.db.QueryRow(ctx, sql,
		stringToUUID(input.ID),
		string(input.Status),
		input.SuccessCount,
		input.FailedCount,
		resultsJSON,
		stringToText(input.ErrorMessage),
		timePtrToTimestamptz(input.FinishedAt),
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrMikroTikBulkJobNotFound
		}
		return nil, fmt.Errorf("repository: gagal menyelesaikan bulk job mikrotik: %w", err)
	}
	return item, nil
}

func scanMikroTikBulkJob(row scanner) (*domain.MikroTikBulkJob, error) {
	var item domain.MikroTikBulkJob
	var id, tenantID, requestedBy pgtype.UUID
	var action, status string
	var routerIDs []pgtype.UUID
	var resultsJSON []byte
	var errorMessage pgtype.Text
	var startedAt, finishedAt, createdAt, updatedAt pgtype.Timestamptz
	if err := row.Scan(
		&id, &tenantID, &action, &status, &routerIDs, &item.TotalCount, &item.SuccessCount, &item.FailedCount,
		&resultsJSON, &errorMessage, &requestedBy, &startedAt, &finishedAt, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	item.ID = uuidToString(id)
	item.TenantID = uuidToString(tenantID)
	item.Action = domain.MikroTikBulkAction(action)
	item.Status = domain.MikroTikBulkJobStatus(status)
	item.RouterIDs = uuidsToStringSlice(routerIDs)
	item.ErrorMessage = textToString(errorMessage)
	item.RequestedBy = uuidToString(requestedBy)
	item.StartedAt = timestamptzToTimePtr(startedAt)
	item.FinishedAt = timestamptzToTimePtr(finishedAt)
	item.CreatedAt = timestamptzToTime(createdAt)
	item.UpdatedAt = timestamptzToTime(updatedAt)
	if len(resultsJSON) > 0 {
		_ = json.Unmarshal(resultsJSON, &item.Results)
	}
	if item.Results == nil {
		item.Results = []domain.MikroTikBulkJobResult{}
	}
	return &item, nil
}

func stringSliceToUUIDs(values []string) []pgtype.UUID {
	items := make([]pgtype.UUID, 0, len(values))
	for _, value := range values {
		items = append(items, stringToUUID(value))
	}
	return items
}

func uuidsToStringSlice(values []pgtype.UUID) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, uuidToString(value))
	}
	return items
}
