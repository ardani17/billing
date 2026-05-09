// area_usecase.go berisi business logic untuk manajemen area.
package usecase

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// AreaUsecase mengimplementasikan business logic untuk manajemen area.
type AreaUsecase struct {
	areaRepo     domain.AreaRepository
	auditLogRepo domain.AuditLogRepository
	logger       zerolog.Logger
}

// NewAreaUsecase membuat instance baru AreaUsecase.
func NewAreaUsecase(
	areaRepo domain.AreaRepository,
	auditLogRepo domain.AuditLogRepository,
	logger zerolog.Logger,
) *AreaUsecase {
	return &AreaUsecase{
		areaRepo:     areaRepo,
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

// Buat membuat area baru.
// Alur: validasi -> cek name duplicate -> buat area.
func (uc *AreaUsecase) Create(ctx context.Context, tenantID string, req domain.CreateAreaRequest, actor ActorInfo) (*domain.Area, error) {
	// Periksa name duplicate
	exists, err := uc.areaRepo.NameExists(ctx, tenantID, req.Name, "")
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal cek nama area duplicate: %w", err)
	}
	if exists {
		return nil, domain.ErrAreaNameDuplicate
	}

	// Bangun area entity
	area := &domain.Area{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		ODPID:       req.ODPID,
		CenterLat:   req.CenterLat,
		CenterLng:   req.CenterLng,
	}

	// Buat area in database
	created, err := uc.areaRepo.Create(ctx, area)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat area: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, created.ID, "area.created", actor, nil)

	return created, nil
}

// GetByID mengambil detail area berdasarkan ID.
func (uc *AreaUsecase) GetByID(ctx context.Context, id string) (*domain.Area, error) {
	area, err := uc.areaRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return area, nil
}

// Perbarui memperbarui data area.
// Alur: validasi -> cek name duplicate (jika name changed) -> perbarui area.
func (uc *AreaUsecase) Update(ctx context.Context, id string, req domain.UpdateAreaRequest, actor ActorInfo) (*domain.Area, error) {
	// Ambil existing area
	existing, err := uc.areaRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Periksa name duplicate jika name is being changed
	if req.Name != "" && req.Name != existing.Name {
		exists, err := uc.areaRepo.NameExists(ctx, existing.TenantID, req.Name, id)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal cek nama area duplicate: %w", err)
		}
		if exists {
			return nil, domain.ErrAreaNameDuplicate
		}
	}

	// Terapkan updates
	updated := applyAreaUpdates(existing, req)

	// Simpan to database
	result, err := uc.areaRepo.Update(ctx, updated)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memperbarui area: %w", err)
	}

	// Compute changes untuk audit log
	changes := computeAreaChanges(existing, result)
	if len(changes) > 0 {
		uc.writeAuditLog(ctx, existing.TenantID, id, "area.updated", actor, changes)
	}

	return result, nil
}

// Hapus menghapus area.
func (uc *AreaUsecase) Delete(ctx context.Context, id string, actor ActorInfo) error {
	// Ambil existing area to get tenant_id
	area, err := uc.areaRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Periksa jumlah pelanggan
	count, err := uc.areaRepo.CustomerCount(ctx, id)
	if err != nil {
		return fmt.Errorf("usecase: gagal cek jumlah pelanggan area: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: %d pelanggan", domain.ErrAreaHasCustomers, count)
	}

	// Hapus area
	if err := uc.areaRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("usecase: gagal menghapus area: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, area.TenantID, id, "area.deleted", actor, nil)

	return nil
}

// List mengambil daftar area untuk tenant.
func (uc *AreaUsecase) List(ctx context.Context, tenantID string) ([]*domain.Area, error) {
	return uc.areaRepo.List(ctx, tenantID)
}

// --- Fungsi bantu functions ---

// writeAuditLog menulis audit log entry untuk area operations.
func (uc *AreaUsecase) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor ActorInfo, changes map[string]interface{}) {
	log := &domain.AuditLog{
		TenantID:   tenantID,
		EntityType: "area",
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ID,
		ActorName:  actor.Name,
		Changes:    changes,
	}

	if err := uc.auditLogRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).
			Str("entity_id", entityID).
			Str("action", action).
			Msg("gagal menulis audit log area")
	}
}

// applyAreaUpdates menerapkan perubahan dari UpdateAreaRequest ke Area.
func applyAreaUpdates(existing *domain.Area, req domain.UpdateAreaRequest) *domain.Area {
	updated := *existing

	if req.Name != "" {
		updated.Name = req.Name
	}
	if req.Description != "" {
		updated.Description = req.Description
	}
	if req.ODPID != "" {
		updated.ODPID = req.ODPID
	}
	if req.CenterLat != nil {
		updated.CenterLat = req.CenterLat
	}
	if req.CenterLng != nil {
		updated.CenterLng = req.CenterLng
	}

	return &updated
}

// computeAreaChanges menghitung field yang berubah antara old dan new area.
func computeAreaChanges(old, new *domain.Area) map[string]interface{} {
	changes := make(map[string]interface{})

	if old.Name != new.Name {
		changes["name"] = map[string]interface{}{"old": old.Name, "new": new.Name}
	}
	if old.Description != new.Description {
		changes["description"] = map[string]interface{}{"old": old.Description, "new": new.Description}
	}
	if old.ODPID != new.ODPID {
		changes["odp_id"] = map[string]interface{}{"old": old.ODPID, "new": new.ODPID}
	}

	// Compare pointers untuk lat/lng
	oldLat := float64PtrValue(old.CenterLat)
	newLat := float64PtrValue(new.CenterLat)
	if oldLat != newLat {
		changes["center_lat"] = map[string]interface{}{"old": old.CenterLat, "new": new.CenterLat}
	}

	oldLng := float64PtrValue(old.CenterLng)
	newLng := float64PtrValue(new.CenterLng)
	if oldLng != newLng {
		changes["center_lng"] = map[string]interface{}{"old": old.CenterLng, "new": new.CenterLng}
	}

	return changes
}

func float64PtrValue(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
