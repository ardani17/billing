package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// InvoiceAuditLogRepo mengimplementasikan domain.InvoiceAuditLogRepository dengan membungkus
// sqlc-generated Queries untuk operasi invoice audit logs.
type InvoiceAuditLogRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi invoice audit logs.
	queries *Queries
}

// NewInvoiceAuditLogRepo membuat instance baru InvoiceAuditLogRepo.
func NewInvoiceAuditLogRepo(queries *Queries) *InvoiceAuditLogRepo {
	return &InvoiceAuditLogRepo{
		queries: queries,
	}
}

// --- Helper function untuk mapping sqlc InvoiceAuditLog → domain.InvoiceAuditLog ---

// mapInvoiceAuditLogRow memetakan InvoiceAuditLog (sqlc model) ke domain.InvoiceAuditLog.
func mapInvoiceAuditLogRow(row InvoiceAuditLog) *domain.InvoiceAuditLog {
	log := &domain.InvoiceAuditLog{
		ID:        uuidToString(row.ID),
		TenantID:  uuidToString(row.TenantID),
		InvoiceID: uuidToString(row.InvoiceID),
		Action:    row.Action,
		ActorID:   row.ActorID,
		ActorName: row.ActorName,
		CreatedAt: timestamptzToTime(row.CreatedAt),
	}

	// Konversi metadata JSON jika ada
	if len(row.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(row.Metadata, &meta); err == nil {
			log.Metadata = meta
		}
	}

	return log
}

// --- Implementasi domain.InvoiceAuditLogRepository ---

// Create membuat satu entri audit log invoice.
func (r *InvoiceAuditLogRepo) Create(ctx context.Context, log *domain.InvoiceAuditLog) error {
	// Serialisasi metadata ke JSON
	var metadata []byte
	if log.Metadata != nil {
		var err error
		metadata, err = json.Marshal(log.Metadata)
		if err != nil {
			return fmt.Errorf("repository: gagal serialisasi metadata audit log: %w", err)
		}
	}

	_, err := r.queries.CreateInvoiceAuditLog(ctx, CreateInvoiceAuditLogParams{
		TenantID:  stringToUUID(log.TenantID),
		InvoiceID: stringToUUID(log.InvoiceID),
		Action:    log.Action,
		ActorID:   log.ActorID,
		ActorName: log.ActorName,
		Metadata:  metadata,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal membuat invoice audit log: %w", err)
	}
	return nil
}

// ListByInvoice mengambil semua audit log untuk invoice tertentu (urut berdasarkan created_at).
func (r *InvoiceAuditLogRepo) ListByInvoice(ctx context.Context, invoiceID string) ([]*domain.InvoiceAuditLog, error) {
	rows, err := r.queries.ListAuditLogsByInvoice(ctx, stringToUUID(invoiceID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice audit logs: %w", err)
	}

	result := make([]*domain.InvoiceAuditLog, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapInvoiceAuditLogRow(row))
	}
	return result, nil
}
