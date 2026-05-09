package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportJobRepo mengimplementasikan domain.ReportJobRepository
// dengan membungkus sqlc-generated Queries dan pgxpool.Pool.
type ReportJobRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi report_jobs.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewReportJobRepo membuat instance baru ReportJobRepo.
func NewReportJobRepo(queries *Queries, pool *pgxpool.Pool) *ReportJobRepo {
	return &ReportJobRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper mapping sqlc ↔ domain ---

// mapReportJobRow memetakan sqlc ReportJob ke domain.ReportJob.
func mapReportJobRow(row ReportJob) *domain.ReportJob {
	var filters domain.ReportFilter
	_ = json.Unmarshal(row.Filters, &filters)

	return &domain.ReportJob{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		ReportType:  row.ReportType,
		Format:      row.Format,
		Filters:     filters,
		Status:      domain.ReportJobStatus(row.Status),
		DownloadURL: textToString(row.DownloadUrl),
		Error:       textToString(row.Error),
		RequestedBy: uuidToString(row.RequestedBy),
		CreatedAt:   timestamptzToTime(row.CreatedAt),
		UpdatedAt:   timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.ReportJobRepository ---

// Buat membuat job export baru dan mengembalikan job yang dibuat.
func (r *ReportJobRepo) Create(ctx context.Context, job *domain.ReportJob) (*domain.ReportJob, error) {
	filtersJSON, err := json.Marshal(job.Filters)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal filters: %w", err)
	}

	row, err := r.queries.CreateReportJob(ctx, CreateReportJobParams{
		TenantID:    stringToUUID(job.TenantID),
		ReportType:  job.ReportType,
		Format:      job.Format,
		Filters:     filtersJSON,
		RequestedBy: stringToUUID(job.RequestedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat report job: %w", err)
	}
	return mapReportJobRow(row), nil
}

// GetByID mengambil job export berdasarkan ID.
func (r *ReportJobRepo) GetByID(ctx context.Context, id string) (*domain.ReportJob, error) {
	row, err := r.queries.GetReportJobByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrReportJobNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil report job by ID: %w", err)
	}
	return mapReportJobRow(row), nil
}

// UpdateStatus memperbarui status job beserta download URL dan pesan error.
func (r *ReportJobRepo) UpdateStatus(ctx context.Context, id string, status domain.ReportJobStatus, downloadURL, errMsg string) error {
	err := r.queries.UpdateReportJobStatus(ctx, UpdateReportJobStatusParams{
		ID:          stringToUUID(id),
		Status:      string(status),
		DownloadUrl: stringToText(downloadURL),
		Error:       stringToText(errMsg),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui status report job: %w", err)
	}
	return nil
}

// CleanupOld menghapus job yang lebih lama dari waktu yang ditentukan.
func (r *ReportJobRepo) CleanupOld(ctx context.Context, olderThan time.Time) error {
	err := r.queries.CleanupOldReportJobs(ctx, pgtype.Timestamptz{
		Time:  olderThan,
		Valid: true,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal cleanup old report jobs: %w", err)
	}
	return nil
}
