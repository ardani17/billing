package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportScheduleRepo mengimplementasikan domain.ReportScheduleRepository
// dengan membungkus sqlc-generated Queries dan pgxpool.Pool.
type ReportScheduleRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi report_schedules.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewReportScheduleRepo membuat instance baru ReportScheduleRepo.
func NewReportScheduleRepo(queries *Queries, pool *pgxpool.Pool) *ReportScheduleRepo {
	return &ReportScheduleRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper mapping sqlc ↔ domain ---

// mapReportScheduleRow memetakan sqlc ReportSchedule ke domain.ReportSchedule.
func mapReportScheduleRow(row ReportSchedule) *domain.ReportSchedule {
	var recipients []domain.Recipient
	_ = json.Unmarshal(row.Recipients, &recipients)

	var filters domain.ReportFilter
	_ = json.Unmarshal(row.Filters, &filters)

	return &domain.ReportSchedule{
		ID:           uuidToString(row.ID),
		TenantID:     uuidToString(row.TenantID),
		ReportType:   row.ReportType,
		ScheduleType: domain.ScheduleType(row.ScheduleType),
		Format:       row.Format,
		Recipients:   recipients,
		Filters:      filters,
		IsActive:     row.IsActive,
		CreatedByID:  uuidToString(row.CreatedByID),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
		UpdatedAt:    timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.ReportScheduleRepository ---

// Create membuat jadwal laporan baru dan mengembalikan jadwal yang dibuat.
func (r *ReportScheduleRepo) Create(ctx context.Context, schedule *domain.ReportSchedule) (*domain.ReportSchedule, error) {
	recipientsJSON, err := json.Marshal(schedule.Recipients)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal recipients: %w", err)
	}
	filtersJSON, err := json.Marshal(schedule.Filters)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal filters: %w", err)
	}

	row, err := r.queries.CreateReportSchedule(ctx, CreateReportScheduleParams{
		TenantID:     stringToUUID(schedule.TenantID),
		ReportType:   schedule.ReportType,
		ScheduleType: string(schedule.ScheduleType),
		Format:       schedule.Format,
		Recipients:   recipientsJSON,
		Filters:      filtersJSON,
		CreatedByID:  stringToUUID(schedule.CreatedByID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat report schedule: %w", err)
	}
	return mapReportScheduleRow(row), nil
}

// GetByID mengambil jadwal laporan berdasarkan ID.
func (r *ReportScheduleRepo) GetByID(ctx context.Context, id string) (*domain.ReportSchedule, error) {
	row, err := r.queries.GetReportScheduleByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrReportScheduleNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil report schedule by ID: %w", err)
	}
	return mapReportScheduleRow(row), nil
}

// Update memperbarui konfigurasi jadwal dan mengembalikan jadwal yang diperbarui.
func (r *ReportScheduleRepo) Update(ctx context.Context, schedule *domain.ReportSchedule) (*domain.ReportSchedule, error) {
	recipientsJSON, err := json.Marshal(schedule.Recipients)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal recipients: %w", err)
	}
	filtersJSON, err := json.Marshal(schedule.Filters)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal filters: %w", err)
	}

	row, err := r.queries.UpdateReportSchedule(ctx, UpdateReportScheduleParams{
		ID:           stringToUUID(schedule.ID),
		ReportType:   schedule.ReportType,
		ScheduleType: string(schedule.ScheduleType),
		Format:       schedule.Format,
		Recipients:   recipientsJSON,
		Filters:      filtersJSON,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrReportScheduleNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui report schedule: %w", err)
	}
	return mapReportScheduleRow(row), nil
}

// Deactivate menonaktifkan jadwal laporan (set is_active = false).
func (r *ReportScheduleRepo) Deactivate(ctx context.Context, id string) error {
	err := r.queries.DeactivateReportSchedule(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menonaktifkan report schedule: %w", err)
	}
	return nil
}

// ListByTenant mengambil semua jadwal laporan aktif untuk tenant.
func (r *ReportScheduleRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.ReportSchedule, error) {
	rows, err := r.queries.ListReportSchedulesByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar report schedules: %w", err)
	}
	schedules := make([]*domain.ReportSchedule, 0, len(rows))
	for _, row := range rows {
		schedules = append(schedules, mapReportScheduleRow(row))
	}
	return schedules, nil
}

// ListDue mengambil jadwal yang perlu dijalankan berdasarkan tipe jadwal.
func (r *ReportScheduleRepo) ListDue(ctx context.Context, scheduleType domain.ScheduleType) ([]*domain.ReportSchedule, error) {
	rows, err := r.queries.ListDueSchedules(ctx, string(scheduleType))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil due schedules: %w", err)
	}
	schedules := make([]*domain.ReportSchedule, 0, len(rows))
	for _, row := range rows {
		schedules = append(schedules, mapReportScheduleRow(row))
	}
	return schedules, nil
}
