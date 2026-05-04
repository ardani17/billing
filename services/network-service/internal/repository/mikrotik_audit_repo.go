package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

type MikroTikAuditRepo struct {
	queries *Queries
}

func NewMikroTikAuditRepo(queries *Queries) *MikroTikAuditRepo {
	return &MikroTikAuditRepo{queries: queries}
}

func (r *MikroTikAuditRepo) Create(ctx context.Context, item domain.MikroTikCommandAuditLog) error {
	if err := r.queries.CreateMikroTikCommandAuditLog(ctx, CreateMikroTikCommandAuditLogParams{
		TenantID:     stringToUUID(item.TenantID),
		RouterID:     stringToUUID(item.RouterID),
		UserID:       optionalUUID(item.UserID),
		Action:       item.Action,
		Command:      item.Command,
		TargetType:   stringToText(item.TargetType),
		TargetID:     stringToText(item.TargetID),
		Status:       item.Status,
		ErrorMessage: stringToText(item.ErrorMessage),
		RemoteAddr:   stringToText(item.RemoteAddr),
	}); err != nil {
		return fmt.Errorf("repository: gagal menulis audit command mikrotik: %w", err)
	}
	return nil
}

func (r *MikroTikAuditRepo) List(ctx context.Context, params domain.MikroTikCommandAuditListParams) (*domain.MikroTikCommandAuditListResult, error) {
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
FROM mikrotik_command_audit_logs
WHERE router_id = $1
  AND ($2 = '' OR status = $2)
`
	var total int64
	if err := r.queries.db.QueryRow(ctx, countSQL, stringToUUID(params.RouterID), params.Status).Scan(&total); err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung audit command mikrotik: %w", err)
	}

	const listSQL = `
SELECT id, tenant_id, router_id, user_id, action, command, target_type, target_id, status, error_message, remote_addr, created_at
FROM mikrotik_command_audit_logs
WHERE router_id = $1
  AND ($2 = '' OR status = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
`
	rows, err := r.queries.db.Query(ctx, listSQL, stringToUUID(params.RouterID), params.Status, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil audit command mikrotik: %w", err)
	}
	defer rows.Close()

	items := make([]domain.MikroTikCommandAuditLog, 0, pageSize)
	for rows.Next() {
		var item domain.MikroTikCommandAuditLog
		var id, tenantID, routerID, userID pgtype.UUID
		var targetType, targetID, errorMessage, remoteAddr pgtype.Text
		var createdAt pgtype.Timestamptz
		if err := rows.Scan(
			&id, &tenantID, &routerID, &userID, &item.Action, &item.Command,
			&targetType, &targetID, &item.Status, &errorMessage, &remoteAddr, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal membaca audit command mikrotik: %w", err)
		}
		item.ID = uuidToString(id)
		item.TenantID = uuidToString(tenantID)
		item.RouterID = uuidToString(routerID)
		item.UserID = uuidToString(userID)
		item.TargetType = textToString(targetType)
		item.TargetID = textToString(targetID)
		item.ErrorMessage = textToString(errorMessage)
		item.RemoteAddr = textToString(remoteAddr)
		item.CreatedAt = timestamptzToTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi audit command mikrotik: %w", err)
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	return &domain.MikroTikCommandAuditListResult{
		Data:       items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(totalPages),
	}, nil
}
