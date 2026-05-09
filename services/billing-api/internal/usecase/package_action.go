// package_action.go berisi business logic untuk aksi paket: aktifkan, deactivate, duplicate.
// Mengimplementasikan Activate, Deactivate, Duplicate pada packageUsecase.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Activate mengaktifkan paket yang sedang nonaktif.
// Alur: ambil -> cek sudah aktif -> perbarui is_active=true -> audit log.
func (uc *packageUsecase) Activate(ctx context.Context, id string, actor domain.ActorInfo) (*domain.Package, error) {
	pkg, err := uc.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Jika sudah aktif, kembalikan error
	if pkg.IsActive {
		return nil, domain.ErrPackageAlreadyActive
	}

	// Perbarui status aktif
	updated, err := uc.packageRepo.UpdateIsActive(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengaktifkan paket: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, pkg.TenantID, id, "package.activated", actor, map[string]interface{}{
		"is_active": map[string]interface{}{"old": false, "new": true},
	})

	return updated, nil
}

// Deactivate menonaktifkan paket yang sedang aktif.
// Alur: ambil -> cek sudah nonaktif -> perbarui is_active=false -> audit log.
func (uc *packageUsecase) Deactivate(ctx context.Context, id string, actor domain.ActorInfo) (*domain.Package, error) {
	pkg, err := uc.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Jika sudah nonaktif, kembalikan error
	if !pkg.IsActive {
		return nil, domain.ErrPackageAlreadyInactive
	}

	// Perbarui status nonaktif
	updated, err := uc.packageRepo.UpdateIsActive(ctx, id, false)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal menonaktifkan paket: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, pkg.TenantID, id, "package.deactivated", actor, map[string]interface{}{
		"is_active": map[string]interface{}{"old": true, "new": false},
	})

	return updated, nil
}

// Duplicate menduplikasi paket yang sudah ada dengan nama unik.
// Alur: ambil sumber -> list nama by prefix -> buat nama duplikat -> buat -> audit log.
func (uc *packageUsecase) Duplicate(ctx context.Context, id string, actor domain.ActorInfo) (*domain.Package, error) {
	// Ambil paket sumber
	source, err := uc.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ambil daftar nama yang sudah ada dengan prefix yang sama untuk collision cek
	existingNames, err := uc.packageRepo.ListNamesByPrefix(ctx, source.TenantID, source.Name)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil nama paket by prefix: %w", err)
	}

	// Buat nama duplikat yang unik
	newName := domain.GenerateDuplicateName(source.Name, existingNames)

	// Buat paket baru dengan field yang disalin dari sumber
	newPkg := &domain.Package{
		TenantID:            source.TenantID,
		Type:                source.Type,
		Name:                newName,
		Description:         source.Description,
		IsActive:            true,
		DownloadMbps:        source.DownloadMbps,
		UploadMbps:          source.UploadMbps,
		BandwidthType:       source.BandwidthType,
		BurstDownloadMbps:   source.BurstDownloadMbps,
		BurstUploadMbps:     source.BurstUploadMbps,
		BurstThresholdMbps:  source.BurstThresholdMbps,
		BurstTimeSeconds:    source.BurstTimeSeconds,
		QuotaType:           source.QuotaType,
		QuotaMB:             source.QuotaMB,
		QuotaAction:         source.QuotaAction,
		ThrottleMbps:        source.ThrottleMbps,
		MonthlyPrice:        source.MonthlyPrice,
		InstallationFee:     source.InstallationFee,
		SellPrice:           source.SellPrice,
		ResellerPrice:       source.ResellerPrice,
		DurationValue:       source.DurationValue,
		DurationUnit:        source.DurationUnit,
		SharedUsers:         source.SharedUsers,
		MikrotikProfileName: source.MikrotikProfileName,
		AddressPool:         source.AddressPool,
		ParentQueue:         source.ParentQueue,
		HotspotProfileName:  source.HotspotProfileName,
	}

	// Simpan paket duplikat ke database
	created, err := uc.packageRepo.Create(ctx, newPkg)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal menduplikasi paket: %w", err)
	}

	// Tulis audit log dengan metadata source_id
	log := &domain.AuditLog{
		TenantID:   source.TenantID,
		EntityType: "package",
		EntityID:   created.ID,
		Action:     "package.duplicated",
		ActorID:    actor.ActorID,
		ActorName:  actor.ActorName,
		Metadata:   map[string]interface{}{"source_id": source.ID},
	}
	if err := uc.auditLogRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).Str("entity_id", created.ID).Msg("gagal menulis audit log duplikasi")
	}

	return created, nil
}
