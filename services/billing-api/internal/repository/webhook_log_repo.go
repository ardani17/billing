package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// WebhookLogRepo mengimplementasikan domain.WebhookLogRepository
// dengan membungkus sqlc-generated Queries.
type WebhookLogRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi webhook_logs.
	queries *Queries
}

// NewWebhookLogRepo membuat instance baru WebhookLogRepo.
func NewWebhookLogRepo(queries *Queries) *WebhookLogRepo {
	return &WebhookLogRepo{queries: queries}
}

// --- Helper: mapping sqlc WebhookLog → domain.WebhookLog ---

// mapWebhookLogRow memetakan WebhookLog (sqlc model) ke domain.WebhookLog.
// Konversi: pgtype.UUID → string, pgtype.Timestamptz → time.Time,
// pgtype.Bool → *bool, pgtype.Text → string, netip.Addr → string,
// []byte (JSONB) → json.RawMessage.
func mapWebhookLogRow(row WebhookLog) *domain.WebhookLog {
	// Konversi tenant_id (nullable UUID) ke *string
	var tenantID *string
	if row.TenantID.Valid {
		s := uuidToString(row.TenantID)
		tenantID = &s
	}

	// Konversi signature_valid (nullable bool) ke *bool
	var sigValid *bool
	if row.SignatureValid.Valid {
		sigValid = &row.SignatureValid.Bool
	}

	return &domain.WebhookLog{
		ID:               uuidToString(row.ID),
		TenantID:         tenantID,
		GatewayProvider:  domain.GatewayProvider(row.GatewayProvider),
		EventType:        row.EventType,
		ExternalID:       row.ExternalID,
		RequestBody:      json.RawMessage(row.RequestBody),
		SourceIP:         row.SourceIp.String(),
		SignatureValid:   sigValid,
		ProcessingStatus: domain.WebhookProcessingStatus(row.ProcessingStatus),
		ErrorMessage:     textToString(row.ErrorMessage),
		CreatedAt:        timestamptzToTime(row.CreatedAt),
	}
}

// --- Helper: konversi domain → sqlc params ---

// stringToPgUUIDNullable mengkonversi *string ke pgtype.UUID (nullable).
func stringToPgUUIDNullable(s *string) pgtype.UUID {
	if s == nil || *s == "" {
		return pgtype.UUID{}
	}
	return stringToUUID(*s)
}

// boolToPgBool mengkonversi *bool ke pgtype.Bool (nullable).
func boolToPgBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

// --- Implementasi domain.WebhookLogRepository ---

// Create membuat log webhook baru dan mengembalikan log yang dibuat.
func (r *WebhookLogRepo) Create(ctx context.Context, log *domain.WebhookLog) (*domain.WebhookLog, error) {
	row, err := r.queries.CreateWebhookLog(ctx, CreateWebhookLogParams{
		TenantID:         stringToPgUUIDNullable(log.TenantID),
		GatewayProvider:  string(log.GatewayProvider),
		EventType:        log.EventType,
		ExternalID:       log.ExternalID,
		RequestBody:      []byte(log.RequestBody),
		SourceIp:         netip.MustParseAddr(log.SourceIP),
		SignatureValid:   boolToPgBool(log.SignatureValid),
		ProcessingStatus: string(log.ProcessingStatus),
		ErrorMessage:     pgtype.Text{String: log.ErrorMessage, Valid: log.ErrorMessage != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat webhook log: %w", err)
	}
	return mapWebhookLogRow(row), nil
}

// GetByID mengambil webhook log berdasarkan ID.
func (r *WebhookLogRepo) GetByID(ctx context.Context, id string) (*domain.WebhookLog, error) {
	row, err := r.queries.GetWebhookLogByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("repository: webhook log tidak ditemukan: %s", id)
		}
		return nil, fmt.Errorf("repository: gagal mengambil webhook log by ID: %w", err)
	}
	return mapWebhookLogRow(row), nil
}

// UpdateStatus memperbarui status pemrosesan dan pesan error webhook log.
func (r *WebhookLogRepo) UpdateStatus(ctx context.Context, id string, status domain.WebhookProcessingStatus, errMsg string) error {
	err := r.queries.UpdateWebhookLogStatus(ctx, UpdateWebhookLogStatusParams{
		ID:               stringToUUID(id),
		ProcessingStatus: string(status),
		ErrorMessage:     pgtype.Text{String: errMsg, Valid: errMsg != ""},
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui status webhook log: %w", err)
	}
	return nil
}

// UpdateSignatureValid memperbarui flag signature_valid pada webhook log.
func (r *WebhookLogRepo) UpdateSignatureValid(ctx context.Context, id string, valid bool) error {
	err := r.queries.UpdateWebhookLogSignatureValid(ctx, UpdateWebhookLogSignatureValidParams{
		ID:             stringToUUID(id),
		SignatureValid: pgtype.Bool{Bool: valid, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui signature_valid webhook log: %w", err)
	}
	return nil
}

// IsAlreadyProcessed mengecek apakah webhook dengan external_id dan event_type sudah diproses.
// Mengembalikan true jika sudah ada webhook dengan status 'processed'.
func (r *WebhookLogRepo) IsAlreadyProcessed(ctx context.Context, externalID, eventType string) (bool, error) {
	exists, err := r.queries.IsWebhookAlreadyProcessed(ctx, IsWebhookAlreadyProcessedParams{
		ExternalID: externalID,
		EventType:  eventType,
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek duplikasi webhook: %w", err)
	}
	return exists, nil
}

// ListByPaymentLink mengambil semua webhook logs berdasarkan external_id payment link.
// Diurutkan berdasarkan created_at DESC (terbaru di atas).
func (r *WebhookLogRepo) ListByPaymentLink(ctx context.Context, externalID string) ([]*domain.WebhookLog, error) {
	rows, err := r.queries.ListWebhookLogsByExternalID(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil webhook logs by external ID: %w", err)
	}
	logs := make([]*domain.WebhookLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, mapWebhookLogRow(row))
	}
	return logs, nil
}

// DeleteOlderThan menghapus webhook logs yang lebih tua dari waktu yang ditentukan.
// Tidak menghapus logs dengan processing_status=failed atau signature_valid=false.
// Mengembalikan jumlah baris yang dihapus (int64).
func (r *WebhookLogRepo) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error) {
	count, err := r.queries.DeleteWebhookLogsOlderThan(ctx, timeToTimestamptz(olderThan))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghapus webhook logs lama: %w", err)
	}
	return count, nil
}
