package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type RouterBackupRepo struct {
	queries *Queries
}

func NewRouterBackupRepo(queries *Queries) *RouterBackupRepo {
	return &RouterBackupRepo{queries: queries}
}

func (r *RouterBackupRepo) Create(ctx context.Context, input domain.CreateRouterBackupInput) (*domain.RouterBackup, error) {
	const sql = `
INSERT INTO router_backups (
    tenant_id, router_id, file_name, format, size_bytes, checksum, content, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, tenant_id, router_id, file_name, format, size_bytes, checksum, content, created_by, created_at
`
	row := r.queries.db.QueryRow(ctx, sql,
		stringToUUID(input.TenantID),
		stringToUUID(input.RouterID),
		input.FileName,
		input.Format,
		input.SizeBytes,
		stringToText(input.Checksum),
		input.Content,
		optionalUUID(input.CreatedBy),
	)
	item, err := scanRouterBackup(row, true)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menyimpan backup router: %w", err)
	}
	return item, nil
}

func (r *RouterBackupRepo) GetByID(ctx context.Context, id string) (*domain.RouterBackup, error) {
	const sql = `
SELECT id, tenant_id, router_id, file_name, format, size_bytes, checksum, content, created_by, created_at
FROM router_backups
WHERE id = $1
`
	item, err := scanRouterBackup(r.queries.db.QueryRow(ctx, sql, stringToUUID(id)), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrRouterBackupNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil backup router: %w", err)
	}
	return item, nil
}

func (r *RouterBackupRepo) List(ctx context.Context, params domain.RouterBackupListParams) (*domain.RouterBackupListResult, error) {
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

	const countSQL = `SELECT COUNT(*) FROM router_backups WHERE router_id = $1`
	var total int64
	if err := r.queries.db.QueryRow(ctx, countSQL, stringToUUID(params.RouterID)).Scan(&total); err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung backup router: %w", err)
	}

	const listSQL = `
SELECT id, tenant_id, router_id, file_name, format, size_bytes, checksum, ''::text AS content, created_by, created_at
FROM router_backups
WHERE router_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`
	rows, err := r.queries.db.Query(ctx, listSQL, stringToUUID(params.RouterID), pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil list backup router: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RouterBackup, 0, pageSize)
	for rows.Next() {
		item, err := scanRouterBackup(rows, false)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal membaca backup router: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi backup router: %w", err)
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	return &domain.RouterBackupListResult{
		Data:       items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(totalPages),
	}, nil
}

func (r *RouterBackupRepo) Delete(ctx context.Context, id string) error {
	const sql = `DELETE FROM router_backups WHERE id = $1`
	tag, err := r.queries.db.Exec(ctx, sql, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus backup router: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrRouterBackupNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRouterBackup(row scanner, includeContent bool) (*domain.RouterBackup, error) {
	var item domain.RouterBackup
	var id, tenantID, routerID, createdBy pgtype.UUID
	var checksum pgtype.Text
	var createdAt pgtype.Timestamptz
	if err := row.Scan(
		&id, &tenantID, &routerID, &item.FileName, &item.Format, &item.SizeBytes,
		&checksum, &item.Content, &createdBy, &createdAt,
	); err != nil {
		return nil, err
	}
	item.ID = uuidToString(id)
	item.TenantID = uuidToString(tenantID)
	item.RouterID = uuidToString(routerID)
	item.Checksum = textToString(checksum)
	item.CreatedBy = uuidToString(createdBy)
	item.CreatedAt = timestamptzToTime(createdAt)
	if !includeContent {
		item.Content = ""
	}
	return &item, nil
}
