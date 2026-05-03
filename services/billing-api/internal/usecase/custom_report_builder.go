// custom_report_builder.go berisi CustomReportBuilder yang mengimplementasikan
// domain.CustomReportTemplateUsecase untuk laporan custom dengan metrik dan
// dimensi yang dipilih pengguna. Maksimal 3 metrik per laporan.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// CustomReportBuilder mengimplementasikan business logic untuk laporan custom.
type CustomReportBuilder struct {
	aggregationRepo domain.ReportAggregationRepository
	templateRepo    domain.CustomReportTemplateRepository
	logger          zerolog.Logger
}

// NewCustomReportBuilder membuat instance baru CustomReportBuilder.
func NewCustomReportBuilder(
	aggregationRepo domain.ReportAggregationRepository,
	templateRepo domain.CustomReportTemplateRepository,
	logger zerolog.Logger,
) *CustomReportBuilder {
	return &CustomReportBuilder{
		aggregationRepo: aggregationRepo,
		templateRepo:    templateRepo,
		logger:          logger.With().Str("component", "custom_report_builder").Logger(),
	}
}

// PreviewCustomReport menjalankan laporan custom tanpa menyimpan template.
// Validasi: maksimal 3 metrik. Query aggregation dengan dynamic grouping.
func (crb *CustomReportBuilder) PreviewCustomReport(ctx context.Context, tenantID string, metrics []string, groupBy, subGroupBy string, periodStart, periodEnd time.Time, displayType string) (interface{}, error) {
	// Validasi jumlah metrik
	if len(metrics) > 3 {
		return nil, domain.ErrMaxMetricsExceeded
	}

	// Query data dari aggregation repo dengan dynamic grouping
	data, err := crb.aggregationRepo.GetCustomReportData(ctx, tenantID, metrics, groupBy, subGroupBy, periodStart, periodEnd)
	if err != nil {
		crb.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil custom report data")
		return nil, err
	}

	return data, nil
}

// CreateTemplate menyimpan konfigurasi laporan custom sebagai template.
func (crb *CustomReportBuilder) CreateTemplate(ctx context.Context, tenantID string, req domain.CreateTemplateRequest, actor domain.ActorInfo) (*domain.CustomReportTemplate, error) {
	// Validasi jumlah metrik
	if len(req.Metrics) > 3 {
		return nil, domain.ErrMaxMetricsExceeded
	}

	template := &domain.CustomReportTemplate{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		Name:               req.Name,
		Metrics:            req.Metrics,
		GroupBy:            req.GroupBy,
		SubGroupBy:         req.SubGroupBy,
		DisplayType:        req.DisplayType,
		DefaultPeriodRange: req.DefaultPeriodRange,
		CreatedByID:        actor.ActorID,
	}

	created, err := crb.templateRepo.Create(ctx, template)
	if err != nil {
		crb.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal membuat template laporan")
		return nil, err
	}
	return created, nil
}

// DeleteTemplate menghapus template laporan custom.
func (crb *CustomReportBuilder) DeleteTemplate(ctx context.Context, id string) error {
	return crb.templateRepo.Delete(ctx, id)
}

// ListTemplates mengambil semua template laporan custom untuk tenant.
func (crb *CustomReportBuilder) ListTemplates(ctx context.Context, tenantID string) ([]*domain.CustomReportTemplate, error) {
	return crb.templateRepo.ListByTenant(ctx, tenantID)
}
