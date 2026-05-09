// schedule_manager.go berisi ScheduleManager yang mengimplementasikan
// domain.ScheduleUsecase untuk CRUD jadwal laporan otomatis.
package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ScheduleManager mengimplementasikan business logic untuk jadwal laporan.
type ScheduleManager struct {
	scheduleRepo domain.ReportScheduleRepository
	jobRepo      domain.ReportJobRepository
	logger       zerolog.Logger
}

// NewScheduleManager membuat instance baru ScheduleManager.
func NewScheduleManager(
	scheduleRepo domain.ReportScheduleRepository,
	jobRepo domain.ReportJobRepository,
	logger zerolog.Logger,
) *ScheduleManager {
	return &ScheduleManager{
		scheduleRepo: scheduleRepo,
		jobRepo:      jobRepo,
		logger:       logger.With().Str("component", "schedule_manager").Logger(),
	}
}

// Buat membuat jadwal laporan baru.
func (sm *ScheduleManager) Create(ctx context.Context, tenantID string, req domain.CreateScheduleRequest, actor domain.ActorInfo) (*domain.ReportSchedule, error) {
	// Konversi recipients dari permintaan ke domain
	recipients := make([]domain.Recipient, len(req.Recipients))
	for i, r := range req.Recipients {
		recipients[i] = domain.Recipient{
			Type:    r.Type,
			Address: r.Address,
		}
	}

	// Bangun filter dari permintaan (opsional)
	var filters domain.ReportFilter
	if req.Filters != nil {
		filters = *req.Filters
	}

	schedule := &domain.ReportSchedule{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		ReportType:   req.ReportType,
		ScheduleType: domain.ScheduleType(req.ScheduleType),
		Format:       req.Format,
		Recipients:   recipients,
		Filters:      filters,
		IsActive:     true,
		CreatedByID:  actor.ActorID,
	}

	created, err := sm.scheduleRepo.Create(ctx, schedule)
	if err != nil {
		sm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal membuat jadwal laporan")
		return nil, err
	}
	return created, nil
}

// Perbarui memperbarui konfigurasi jadwal laporan.
func (sm *ScheduleManager) Update(ctx context.Context, id string, req domain.UpdateScheduleRequest) (*domain.ReportSchedule, error) {
	existing, err := sm.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrReportScheduleNotFound
	}

	// Terapkan perubahan dari permintaan
	if req.ReportType != "" {
		existing.ReportType = req.ReportType
	}
	if req.ScheduleType != "" {
		existing.ScheduleType = domain.ScheduleType(req.ScheduleType)
	}
	if req.Format != "" {
		existing.Format = req.Format
	}
	if len(req.Recipients) > 0 {
		recipients := make([]domain.Recipient, len(req.Recipients))
		for i, r := range req.Recipients {
			recipients[i] = domain.Recipient{
				Type:    r.Type,
				Address: r.Address,
			}
		}
		existing.Recipients = recipients
	}
	if req.Filters != nil {
		existing.Filters = *req.Filters
	}

	updated, err := sm.scheduleRepo.Update(ctx, existing)
	if err != nil {
		sm.logger.Error().Err(err).Str("id", id).Msg("gagal memperbarui jadwal laporan")
		return nil, err
	}
	return updated, nil
}

// Hapus menonaktifkan jadwal laporan (hapus lunak via is_active = false).
func (sm *ScheduleManager) Delete(ctx context.Context, id string) error {
	return sm.scheduleRepo.Deactivate(ctx, id)
}

// List mengambil semua jadwal laporan aktif untuk tenant.
func (sm *ScheduleManager) List(ctx context.Context, tenantID string) ([]*domain.ReportSchedule, error) {
	return sm.scheduleRepo.ListByTenant(ctx, tenantID)
}
