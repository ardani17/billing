package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// AlarmRepo mengimplementasikan domain.AlarmRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.OLTAlarmRecord.
type AlarmRepo struct {
	queries *Queries
}

// NewAlarmRepo membuat instance baru AlarmRepo.
func NewAlarmRepo(queries *Queries) *AlarmRepo {
	return &AlarmRepo{queries: queries}
}

// --- Mapping sqlc OltAlarm → domain.OLTAlarmRecord ---
// Menggunakan helper int4ToIntPtr dan intPtrToInt4 dari vpn_tunnel_repo.go.

// mapAlarmRow memetakan OltAlarm (sqlc model) ke domain.OLTAlarmRecord.
func mapAlarmRow(row OltAlarm) *domain.OLTAlarmRecord {
	return &domain.OLTAlarmRecord{
		ID:           uuidToString(row.ID),
		TenantID:     uuidToString(row.TenantID),
		OLTID:        uuidToString(row.OltID),
		PONPortIndex: int4ToIntPtr(row.PonPortIndex),
		ONTIndex:     int4ToIntPtr(row.OntIndex),
		AlarmType:    row.AlarmType,
		Severity:     row.Severity,
		Message:      textToString(row.Message),
		Source:       row.Source,
		Status:       row.Status,
		ClearedAt:    timestamptzToTimePtr(row.ClearedAt),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.AlarmRepository ---

// Create menyimpan alarm baru dan mengembalikan alarm yang dibuat.
func (r *AlarmRepo) Create(ctx context.Context, alarm *domain.OLTAlarmRecord) (*domain.OLTAlarmRecord, error) {
	row, err := r.queries.CreateOLTAlarm(ctx, CreateOLTAlarmParams{
		TenantID:     stringToUUID(alarm.TenantID),
		OltID:        stringToUUID(alarm.OLTID),
		PonPortIndex: intPtrToInt4(alarm.PONPortIndex),
		OntIndex:     intPtrToInt4(alarm.ONTIndex),
		AlarmType:    alarm.AlarmType,
		Severity:     alarm.Severity,
		Message:      stringToText(alarm.Message),
		Source:       alarm.Source,
		Status:       alarm.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menyimpan alarm: %w", err)
	}
	return mapAlarmRow(row), nil
}

// List mengambil daftar alarm dengan paginasi dan filter.
func (r *AlarmRepo) List(ctx context.Context, oltID string, params domain.AlarmListParams) (*domain.AlarmListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize
	oltUUID := stringToUUID(oltID)

	// Ambil total count untuk paginasi
	total, err := r.queries.CountOLTAlarms(ctx, CountOLTAlarmsParams{
		OltID:    oltUUID,
		Severity: stringToText(params.Severity),
		Status:   stringToText(params.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total alarm: %w", err)
	}

	// Ambil data alarm
	rows, err := r.queries.ListOLTAlarms(ctx, ListOLTAlarmsParams{
		OltID:    oltUUID,
		Limit:    int32(params.PageSize),
		Offset:   int32(offset),
		Severity: stringToText(params.Severity),
		Status:   stringToText(params.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar alarm: %w", err)
	}

	alarms := make([]*domain.OLTAlarmRecord, 0, len(rows))
	for _, row := range rows {
		alarms = append(alarms, mapAlarmRow(row))
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.AlarmListResult{
		Data:       alarms,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CountActive menghitung jumlah alarm aktif per OLT.
func (r *AlarmRepo) CountActive(ctx context.Context, oltID string) (int64, error) {
	count, err := r.queries.CountActiveAlarms(ctx, stringToUUID(oltID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung alarm aktif: %w", err)
	}
	return count, nil
}

// CountActiveByTenant menghitung total alarm aktif untuk tenant (via RLS).
func (r *AlarmRepo) CountActiveByTenant(ctx context.Context) (int64, error) {
	count, err := r.queries.CountActiveAlarmsByTenant(ctx)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung alarm aktif tenant: %w", err)
	}
	return count, nil
}

// ClearAlarm mengubah status alarm menjadi cleared.
func (r *AlarmRepo) ClearAlarm(ctx context.Context, id string) error {
	err := r.queries.ClearOLTAlarm(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal clear alarm: %w", err)
	}
	return nil
}

// PurgeOlderThan menghapus alarm lebih tua dari waktu tertentu.
// Mengembalikan jumlah baris yang dihapus.
func (r *AlarmRepo) PurgeOlderThan(ctx context.Context, before time.Time) (int64, error) {
	count, err := r.queries.PurgeOLTAlarms(ctx, pgtype.Timestamptz{
		Time:  before,
		Valid: true,
	})
	if err != nil {
		return 0, fmt.Errorf("repository: gagal purge alarm lama: %w", err)
	}
	return count, nil
}

// Compile-time check: AlarmRepo mengimplementasikan domain.AlarmRepository.
var _ domain.AlarmRepository = (*AlarmRepo)(nil)
