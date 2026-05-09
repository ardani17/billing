package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// VoucherAuditRepo mengimplementasikan domain.VoucherAuditLogRepository dengan membungkus
// sqlc-generated Queries untuk operasi voucher audit log.
type VoucherAuditRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi voucher audit log.
	queries *Queries
}

// NewVoucherAuditRepo membuat instance baru VoucherAuditRepo.
func NewVoucherAuditRepo(queries *Queries) *VoucherAuditRepo {
	return &VoucherAuditRepo{
		queries: queries,
	}
}

// --- Helper function untuk mapping sqlc VoucherAuditLog -> domain.VoucherAuditLog ---

// mapVoucherAuditLogRow memetakan VoucherAuditLog (sqlc model) ke domain.VoucherAuditLog.
func mapVoucherAuditLogRow(row VoucherAuditLog) *domain.VoucherAuditLog {
	var metadata map[string]interface{}
	if row.Metadata != nil {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}

	return &domain.VoucherAuditLog{
		ID:        uuidToString(row.ID),
		TenantID:  uuidToString(row.TenantID),
		VoucherID: uuidToString(row.VoucherID),
		Action:    row.Action,
		ActorID:   row.ActorID,
		ActorName: row.ActorName,
		Metadata:  metadata,
		CreatedAt: timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.VoucherAuditLogRepository ---

// Buat membuat satu entri audit log voucher.
func (r *VoucherAuditRepo) Create(ctx context.Context, log *domain.VoucherAuditLog) error {
	var metadataJSON []byte
	if log.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(log.Metadata)
		if err != nil {
			return fmt.Errorf("repository: gagal marshal metadata: %w", err)
		}
	}

	_, err := r.queries.CreateVoucherAuditLog(ctx, CreateVoucherAuditLogParams{
		TenantID:  stringToUUID(log.TenantID),
		VoucherID: stringToUUID(log.VoucherID),
		Action:    log.Action,
		ActorID:   log.ActorID,
		ActorName: log.ActorName,
		Metadata:  metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal membuat voucher audit log: %w", err)
	}
	return nil
}

// BulkCreate membuat beberapa entri audit log voucher sekaligus.
// Mengiterasi setiap entri dan memanggil CreateVoucherAuditLog untuk masing-masing.
func (r *VoucherAuditRepo) BulkCreate(ctx context.Context, logs []*domain.VoucherAuditLog) error {
	for _, log := range logs {
		if err := r.Create(ctx, log); err != nil {
			return fmt.Errorf("repository: gagal bulk create voucher audit log: %w", err)
		}
	}
	return nil
}

// ListByVoucher mengambil semua audit log untuk voucher tertentu, diurutkan berdasarkan waktu pembuatan (ASC).
func (r *VoucherAuditRepo) ListByVoucher(ctx context.Context, voucherID string) ([]*domain.VoucherAuditLog, error) {
	rows, err := r.queries.ListVoucherAuditLogsByVoucher(ctx, stringToUUID(voucherID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil voucher audit logs: %w", err)
	}

	logs := make([]*domain.VoucherAuditLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, mapVoucherAuditLogRow(row))
	}
	return logs, nil
}
