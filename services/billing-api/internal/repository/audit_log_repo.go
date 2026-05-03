package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// AuditLogRepo mengimplementasikan domain.AuditLogRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.AuditLog.
type AuditLogRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi audit log.
	queries *Queries
}

// NewAuditLogRepo membuat instance baru AuditLogRepo.
func NewAuditLogRepo(queries *Queries) *AuditLogRepo {
	return &AuditLogRepo{
		queries: queries,
	}
}

// --- Helper functions untuk mapping sqlc row → domain.AuditLog ---

// mapAuditLogRow memetakan AuditLog (sqlc model) ke domain.AuditLog.
func mapAuditLogRow(row AuditLog) *domain.AuditLog {
	var changes map[string]interface{}
	if row.Changes != nil {
		_ = json.Unmarshal(row.Changes, &changes)
	}

	var metadata map[string]interface{}
	if row.Metadata != nil {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}

	return &domain.AuditLog{
		ID:         uuidToString(row.ID),
		TenantID:   uuidToString(row.TenantID),
		EntityType: row.EntityType,
		EntityID:   uuidToString(row.EntityID),
		Action:     row.Action,
		ActorID:    uuidToString(row.ActorID),
		ActorName:  row.ActorName,
		Changes:    changes,
		Metadata:   metadata,
		CreatedAt:  timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.AuditLogRepository ---

// Create membuat audit log entry baru.
func (r *AuditLogRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	var changesJSON []byte
	if log.Changes != nil {
		var err error
		changesJSON, err = json.Marshal(log.Changes)
		if err != nil {
			return fmt.Errorf("repository: gagal marshal changes: %w", err)
		}
	}

	var metadataJSON []byte
	if log.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(log.Metadata)
		if err != nil {
			return fmt.Errorf("repository: gagal marshal metadata: %w", err)
		}
	}

	err := r.queries.CreateAuditLog(ctx, CreateAuditLogParams{
		TenantID:   stringToUUID(log.TenantID),
		EntityType: log.EntityType,
		EntityID:   stringToUUID(log.EntityID),
		Action:     log.Action,
		ActorID:    stringToUUID(log.ActorID),
		ActorName:  log.ActorName,
		Changes:    changesJSON,
		Metadata:   metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal membuat audit log: %w", err)
	}
	return nil
}

// ListByEntity mengambil semua audit log untuk entity tertentu.
// Diurutkan berdasarkan created_at DESC.
func (r *AuditLogRepo) ListByEntity(ctx context.Context, entityType, entityID string) ([]*domain.AuditLog, error) {
	rows, err := r.queries.ListAuditLogsByEntity(ctx, ListAuditLogsByEntityParams{
		EntityType: entityType,
		EntityID:   stringToUUID(entityID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil audit logs: %w", err)
	}

	logs := make([]*domain.AuditLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, mapAuditLogRow(row))
	}
	return logs, nil
}
