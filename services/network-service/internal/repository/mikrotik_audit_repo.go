package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
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
