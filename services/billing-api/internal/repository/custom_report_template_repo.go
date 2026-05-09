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

// CustomReportTemplateRepo mengimplementasikan domain.CustomReportTemplateRepository
// dengan membungkus sqlc-generated Queries dan pgxpool.Pool.
type CustomReportTemplateRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi custom_report_templates.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewCustomReportTemplateRepo membuat instance baru CustomReportTemplateRepo.
func NewCustomReportTemplateRepo(queries *Queries, pool *pgxpool.Pool) *CustomReportTemplateRepo {
	return &CustomReportTemplateRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper mapping sqlc ↔ domain ---

// mapCustomReportTemplateRow memetakan sqlc CustomReportTemplate ke domain.CustomReportTemplate.
func mapCustomReportTemplateRow(row CustomReportTemplate) *domain.CustomReportTemplate {
	var metrics []string
	_ = json.Unmarshal(row.Metrics, &metrics)

	return &domain.CustomReportTemplate{
		ID:                 uuidToString(row.ID),
		TenantID:           uuidToString(row.TenantID),
		Name:               row.Name,
		Metrics:            metrics,
		GroupBy:            row.GroupBy,
		SubGroupBy:         textToString(row.SubGroupBy),
		DisplayType:        row.DisplayType,
		DefaultPeriodRange: textToString(row.DefaultPeriodRange),
		CreatedByID:        uuidToString(row.CreatedByID),
		CreatedAt:          timestamptzToTime(row.CreatedAt),
		UpdatedAt:          timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.CustomReportTemplateRepository ---

// Buat membuat template laporan kustom baru dan mengembalikan template yang dibuat.
func (r *CustomReportTemplateRepo) Create(ctx context.Context, template *domain.CustomReportTemplate) (*domain.CustomReportTemplate, error) {
	metricsJSON, err := json.Marshal(template.Metrics)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal metrics: %w", err)
	}

	row, err := r.queries.CreateCustomReportTemplate(ctx, CreateCustomReportTemplateParams{
		TenantID:           stringToUUID(template.TenantID),
		Name:               template.Name,
		Metrics:            metricsJSON,
		GroupBy:            template.GroupBy,
		SubGroupBy:         stringToText(template.SubGroupBy),
		DisplayType:        template.DisplayType,
		DefaultPeriodRange: stringToText(template.DefaultPeriodRange),
		CreatedByID:        stringToUUID(template.CreatedByID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat custom report template: %w", err)
	}
	return mapCustomReportTemplateRow(row), nil
}

// GetByID mengambil template laporan berdasarkan ID.
func (r *CustomReportTemplateRepo) GetByID(ctx context.Context, id string) (*domain.CustomReportTemplate, error) {
	row, err := r.queries.GetCustomReportTemplateByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil custom report template by ID: %w", err)
	}
	return mapCustomReportTemplateRow(row), nil
}

// Hapus menghapus template laporan secara permanen.
func (r *CustomReportTemplateRepo) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteCustomReportTemplate(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus custom report template: %w", err)
	}
	return nil
}

// ListByTenant mengambil semua template laporan untuk tenant.
func (r *CustomReportTemplateRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.CustomReportTemplate, error) {
	rows, err := r.queries.ListCustomReportTemplatesByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar custom report templates: %w", err)
	}
	templates := make([]*domain.CustomReportTemplate, 0, len(rows))
	for _, row := range rows {
		templates = append(templates, mapCustomReportTemplateRow(row))
	}
	return templates, nil
}
