package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuditLogRepo mengimplementasikan domain.AuditLogRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.ProvisioningAuditLog.
// Tabel ini append-only: hanya Buat dan List, tidak ada Perbarui atau Hapus.
type AuditLogRepo struct {
	queries *Queries
}

// NewAuditLogRepo membuat instance baru AuditLogRepo.
func NewAuditLogRepo(queries *Queries) *AuditLogRepo {
	return &AuditLogRepo{queries: queries}
}

// --- Mapping sqlc ProvisioningAuditLog -> domain.ProvisioningAuditLog ---

// mapAuditLogRow memetakan ProvisioningAuditLog (sqlc model) ke domain.ProvisioningAuditLog.
func mapAuditLogRow(row ProvisioningAuditLog) *domain.ProvisioningAuditLog {
	return &domain.ProvisioningAuditLog{
		ID:               uuidToString(row.ID),
		TenantID:         uuidToString(row.TenantID),
		OLTID:            uuidToString(row.OltID),
		ONTID:            uuidToStringPtr(row.OntID),
		Action:           domain.AuditAction(row.Action),
		CommandsSent:     jsonToStringSlice(row.CommandsSent),
		CommandResponses: jsonToStringSlice(row.CommandResponses),
		Status:           row.Status,
		ErrorMessage:     textToString(row.ErrorMessage),
		PerformedBy:      row.PerformedBy,
		Brand:            textToString(row.Brand),
		Model:            textToString(row.Model),
		Transport:        textToString(row.Transport),
		Operation:        textToString(row.Operation),
		CorrelationID:    uuidToString(row.CorrelationID),
		CreatedAt:        timestamptzToTime(row.CreatedAt),
	}
}

// --- Fungsi bantu konversi JSON ↔ []string ---

// stringSliceToJSON mengkonversi []string ke []byte (JSON) untuk JSONB column.
func stringSliceToJSON(s []string) []byte {
	if s == nil {
		s = []string{}
	}
	data, _ := json.Marshal(s)
	return data
}

// jsonToStringSlice mengkonversi []byte (JSON) ke []string dari JSONB column.
func jsonToStringSlice(data []byte) []string {
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return []string{}
	}
	return result
}

// --- Implementasi domain.AuditLogRepository ---

// Buat menyimpan record audit log baru.
func (r *AuditLogRepo) Create(ctx context.Context, log *domain.ProvisioningAuditLog) (*domain.ProvisioningAuditLog, error) {
	row, err := r.queries.CreateAuditLog(ctx, CreateAuditLogParams{
		TenantID:         stringToUUID(log.TenantID),
		OltID:            stringToUUID(log.OLTID),
		OntID:            stringPtrToUUID(log.ONTID),
		Action:           string(log.Action),
		CommandsSent:     stringSliceToJSON(log.CommandsSent),
		CommandResponses: stringSliceToJSON(log.CommandResponses),
		Status:           log.Status,
		ErrorMessage:     stringToText(log.ErrorMessage),
		PerformedBy:      log.PerformedBy,
		Brand:            stringToText(log.Brand),
		Model:            stringToText(log.Model),
		Transport:        stringToText(log.Transport),
		Operation:        stringToText(log.Operation),
		CorrelationID:    stringToUUID(log.CorrelationID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat audit log: %w", err)
	}
	return mapAuditLogRow(row), nil
}

// List mengambil daftar audit log dengan paginasi dan filter.
func (r *AuditLogRepo) List(ctx context.Context, params domain.AuditLogListParams) (*domain.AuditLogListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Ambil total count untuk paginasi
	total, err := r.queries.CountAuditLogs(ctx, CountAuditLogsParams{
		OltID:    stringToNullableUUID(params.OLTID),
		OntID:    stringToNullableUUID(params.ONTID),
		Action:   stringToText(params.Action),
		DateFrom: timePtrToTimestamptz(params.DateFrom),
		DateTo:   timePtrToTimestamptz(params.DateTo),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total audit log: %w", err)
	}

	// Ambil data audit log
	rows, err := r.queries.ListAuditLogs(ctx, ListAuditLogsParams{
		Limit:    int32(params.PageSize),
		Offset:   int32(offset),
		OltID:    stringToNullableUUID(params.OLTID),
		OntID:    stringToNullableUUID(params.ONTID),
		Action:   stringToText(params.Action),
		DateFrom: timePtrToTimestamptz(params.DateFrom),
		DateTo:   timePtrToTimestamptz(params.DateTo),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar audit log: %w", err)
	}

	// Konversi ke domain.ProvisioningAuditLog
	logs := make([]*domain.ProvisioningAuditLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, mapAuditLogRow(row))
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.AuditLogListResult{
		Data:       logs,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// --- Fungsi bantu untuk nullable timestamptz filter ---

// timePtrToNullableTimestamptz mengkonversi *time.Time ke pgtype.Timestamptz.
// nil -> Timestamptz tidak valid (NULL) agar filter diabaikan.
// Catatan: menggunakan timePtrToTimestamptz yang sudah ada di router_repo.go.
var _ pgtype.Timestamptz // memastikan import pgtype digunakan

// Compile-time cek: AuditLogRepo mengimplementasikan domain.AuditLogRepository.
var _ domain.AuditLogRepository = (*AuditLogRepo)(nil)
