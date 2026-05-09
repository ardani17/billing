// package_usecase.go berisi business logic untuk manajemen paket (CRUD).
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// packageUsecase mengimplementasikan business logic untuk manajemen paket.
type packageUsecase struct {
	packageRepo  domain.PackageRepository
	auditLogRepo domain.AuditLogRepository
	queueClient  *asynq.Client
	logger       zerolog.Logger
}

// NewPackageUsecase membuat instance baru PackageUsecase.
func NewPackageUsecase(
	packageRepo domain.PackageRepository,
	auditLogRepo domain.AuditLogRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) domain.PackageUsecase {
	return &packageUsecase{
		packageRepo:  packageRepo,
		auditLogRepo: auditLogRepo,
		queueClient:  queueClient,
		logger:       logger,
	}
}

// Buat membuat paket baru dengan validasi type-conditional.
// Alur: validasi -> cek duplikat nama -> auto-buat profile -> buat -> audit log.
func (uc *packageUsecase) Create(ctx context.Context, tenantID string, req domain.CreatePackageRequest, actor domain.ActorInfo) (*domain.Package, error) {
	pkgType := domain.PackageType(req.Type)

	// Validasi type-conditional: field wajib per tipe
	if pkgType.IsMonthlyBilling() {
		if req.MonthlyPrice == nil || req.BandwidthType == "" {
			return nil, fmt.Errorf("monthly_price dan bandwidth_type wajib untuk paket bulanan")
		}
	} else {
		if req.SellPrice == nil || req.ResellerPrice == nil || req.DurationValue == nil || req.DurationUnit == "" {
			return nil, fmt.Errorf("sell_price, reseller_price, duration_value, duration_unit wajib untuk paket voucher")
		}
		// Validasi margin reseller
		if err := domain.ValidateResellerMargin(*req.SellPrice, *req.ResellerPrice); err != nil {
			return nil, err
		}
	}

	// Validasi burst field (all-atau-nothing)
	if err := domain.ValidateBurstFields(req.BurstDownloadMbps, req.BurstUploadMbps, req.BurstThresholdMbps, req.BurstTimeSeconds); err != nil {
		return nil, err
	}

	// Validasi quota conditional
	qt := domain.QuotaType(req.QuotaType)
	if qt != domain.QuotaUnlimited {
		if req.QuotaMB == nil {
			return nil, fmt.Errorf("quota_mb wajib jika quota_type bukan unlimited")
		}
		if qt == domain.QuotaMonthlyQuota || qt == domain.QuotaFUP {
			if req.QuotaAction == "" {
				return nil, fmt.Errorf("quota_action wajib jika quota_type = monthly_quota atau fup")
			}
		}
		if req.QuotaAction == string(domain.QuotaActionThrottle) && req.ThrottleMbps == nil {
			return nil, fmt.Errorf("throttle_mbps wajib jika quota_action = throttle")
		}
	}

	// Cek duplikat nama
	exists, err := uc.packageRepo.NameExists(ctx, tenantID, req.Name, "")
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal cek nama duplikat: %w", err)
	}
	if exists {
		return nil, domain.ErrPackageNameDuplicate
	}

	// Auto-buat profile name jika tidak disediakan
	mikrotikProfile := req.MikrotikProfileName
	hotspotProfile := req.HotspotProfileName
	if pkgType == domain.PackageTypePPPoE && mikrotikProfile == "" {
		mikrotikProfile = domain.GenerateProfileName(req.Name)
	}
	if pkgType == domain.PackageTypeVoucher && hotspotProfile == "" {
		hotspotProfile = domain.GenerateProfileName(req.Name)
	}

	// Bawaan shared_users = 1 untuk voucher
	sharedUsers := 1
	if req.SharedUsers != nil {
		sharedUsers = *req.SharedUsers
	}

	// Bawaan installation_fee = 0
	var installationFee int64
	if req.InstallationFee != nil {
		installationFee = *req.InstallationFee
	}

	// Bangun entity paket
	pkg := &domain.Package{
		TenantID:            tenantID,
		Type:                pkgType,
		Name:                req.Name,
		Description:         req.Description,
		IsActive:            true,
		DownloadMbps:        req.DownloadMbps,
		UploadMbps:          req.UploadMbps,
		BurstDownloadMbps:   req.BurstDownloadMbps,
		BurstUploadMbps:     req.BurstUploadMbps,
		BurstThresholdMbps:  req.BurstThresholdMbps,
		BurstTimeSeconds:    req.BurstTimeSeconds,
		QuotaType:           qt,
		QuotaMB:             req.QuotaMB,
		QuotaAction:         req.QuotaAction,
		ThrottleMbps:        req.ThrottleMbps,
		SharedUsers:         sharedUsers,
		MikrotikProfileName: mikrotikProfile,
		AddressPool:         req.AddressPool,
		ParentQueue:         req.ParentQueue,
		HotspotProfileName:  hotspotProfile,
		InstallationFee:     installationFee,
	}

	// Set field sesuai tipe, null-kan field yang tidak relevan
	if pkgType.IsMonthlyBilling() {
		pkg.MonthlyPrice = req.MonthlyPrice
		pkg.BandwidthType = req.BandwidthType
		pkg.SellPrice = nil
		pkg.ResellerPrice = nil
		pkg.DurationValue = nil
		pkg.DurationUnit = ""
	} else {
		pkg.SellPrice = req.SellPrice
		pkg.ResellerPrice = req.ResellerPrice
		pkg.DurationValue = req.DurationValue
		pkg.DurationUnit = req.DurationUnit
		pkg.MonthlyPrice = nil
		pkg.BandwidthType = ""
	}

	// Simpan ke database
	created, err := uc.packageRepo.Create(ctx, pkg)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat paket: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, created.ID, "package.created", actor, nil)

	return created, nil
}

// GetByID mengambil detail paket berdasarkan ID.
// Jika includeAudit true, audit logs juga disertakan.
func (uc *packageUsecase) GetByID(ctx context.Context, id string, includeAudit bool) (*domain.PackageDetail, error) {
	pkg, err := uc.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &domain.PackageDetail{Package: pkg}

	if includeAudit {
		logs, err := uc.auditLogRepo.ListByEntity(ctx, "package", id)
		if err != nil {
			uc.logger.Error().Err(err).Str("package_id", id).Msg("gagal mengambil audit logs")
		} else {
			detail.AuditLogs = logs
		}
	}

	return detail, nil
}

// Perbarui memperbarui data paket.
// Alur: ambil -> gabungkan -> validasi -> cek duplikat -> perbarui -> audit -> event.
func (uc *packageUsecase) Update(ctx context.Context, id string, req domain.UpdatePackageRequest, actor domain.ActorInfo) (*domain.Package, error) {
	existing, err := uc.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Merge permintaan ke existing (hanya field non-zero/non-nil)
	merged := applyPackageUpdates(existing, req)

	// Validasi type-conditional pada hasil gabungkan
	if err := uc.validateMergedPackage(merged); err != nil {
		return nil, err
	}

	// Cek duplikat nama (exclude ID saat ini)
	if merged.Name != existing.Name {
		exists, err := uc.packageRepo.NameExists(ctx, existing.TenantID, merged.Name, id)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal cek nama duplikat: %w", err)
		}
		if exists {
			return nil, domain.ErrPackageNameDuplicate
		}
	}

	// Perbarui ke database
	updated, err := uc.packageRepo.Update(ctx, merged)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memperbarui paket: %w", err)
	}

	// Hitung field yang berubah untuk audit log
	changes := computePackageChanges(existing, updated)
	if len(changes) > 0 {
		uc.writeAuditLog(ctx, existing.TenantID, id, "package.updated", actor, changes)
	}

	// Terbitkan event jika harga berubah
	uc.publishPriceChangeEvent(existing, updated)

	return updated, nil
}

// Hapus menghapus paket secara permanen (hapus permanen).
// Alur: ambil -> verifikasi nama -> cek jumlah pelanggan -> hapus -> audit log.
func (uc *packageUsecase) Delete(ctx context.Context, id string, confirmName string, actor domain.ActorInfo) error {
	pkg, err := uc.packageRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verifikasi nama konfirmasi (case-sensitive)
	if confirmName != pkg.Name {
		return domain.ErrConfirmationMismatch
	}

	// Cek jumlah pelanggan yang menggunakan paket
	count, err := uc.packageRepo.CustomerCount(ctx, id)
	if err != nil {
		return fmt.Errorf("usecase: gagal menghitung pelanggan: %w", err)
	}
	if count > 0 {
		return domain.ErrPackageHasCustomers
	}

	// Hard hapus
	if err := uc.packageRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrPackageHasCustomers) || errors.Is(err, domain.ErrPackageHasVouchers) {
			return err
		}
		return fmt.Errorf("usecase: gagal menghapus paket: %w", err)
	}

	uc.writeAuditLog(ctx, pkg.TenantID, id, "package.deleted", actor, nil)
	return nil
}

// List mengambil daftar paket dengan paginasi, filter, dan pengurutan.
func (uc *packageUsecase) List(ctx context.Context, params domain.PackageListParams) (*domain.PackageListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}
	return uc.packageRepo.List(ctx, params)
}

// --- Fungsi bantu methods (method pada packageUsecase) ---

// writeAuditLog menulis audit log. Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *packageUsecase) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor domain.ActorInfo, changes map[string]interface{}) {
	log := &domain.AuditLog{
		TenantID:   tenantID,
		EntityType: "package",
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ActorID,
		ActorName:  actor.ActorName,
		Changes:    changes,
	}
	if err := uc.auditLogRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).Str("entity_id", entityID).Str("action", action).Msg("gagal menulis audit log")
	}
}

// publishEvent mempublikasikan event ke Redis queue.
func (uc *packageUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal event payload")
		return
	}
	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}
	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish event")
	}
}

// publishPriceChangeEvent mempublikasikan event jika harga berubah.
func (uc *packageUsecase) publishPriceChangeEvent(old, updated *domain.Package) {
	var oldPrice, newPrice int64
	changed := false

	if old.Type.IsMonthlyBilling() && updated.Type.IsMonthlyBilling() && old.MonthlyPrice != nil && updated.MonthlyPrice != nil {
		if *old.MonthlyPrice != *updated.MonthlyPrice {
			oldPrice, newPrice = *old.MonthlyPrice, *updated.MonthlyPrice
			changed = true
		}
	} else if old.Type == domain.PackageTypeVoucher && old.SellPrice != nil && updated.SellPrice != nil {
		if *old.SellPrice != *updated.SellPrice {
			oldPrice, newPrice = *old.SellPrice, *updated.SellPrice
			changed = true
		}
	}

	if !changed {
		return
	}

	uc.publishEvent(updated.TenantID, "package.price_changed", domain.PackagePriceChangedPayload{
		TenantID:    updated.TenantID,
		PackageID:   updated.ID,
		PackageName: updated.Name,
		PackageType: string(updated.Type),
		OldPrice:    oldPrice,
		NewPrice:    newPrice,
	})
}

// validateMergedPackage memvalidasi paket hasil gabungkan sebelum perbarui.
func (uc *packageUsecase) validateMergedPackage(pkg *domain.Package) error {
	// Validasi field wajib per tipe
	if pkg.Type.IsMonthlyBilling() {
		if pkg.MonthlyPrice == nil || pkg.BandwidthType == "" {
			return fmt.Errorf("monthly_price dan bandwidth_type wajib untuk paket bulanan")
		}
	} else {
		if pkg.SellPrice == nil || pkg.ResellerPrice == nil || pkg.DurationValue == nil || pkg.DurationUnit == "" {
			return fmt.Errorf("sell_price, reseller_price, duration_value, duration_unit wajib untuk paket voucher")
		}
		if err := domain.ValidateResellerMargin(*pkg.SellPrice, *pkg.ResellerPrice); err != nil {
			return err
		}
	}

	// Validasi quota_type sesuai tipe paket
	qt := pkg.QuotaType
	if pkg.Type.IsMonthlyBilling() {
		if qt != domain.QuotaUnlimited && qt != domain.QuotaMonthlyQuota && qt != domain.QuotaFUP {
			return fmt.Errorf("quota_type untuk paket bulanan harus unlimited, monthly_quota, atau fup")
		}
	} else {
		if qt != domain.QuotaUnlimited && qt != domain.QuotaQuota {
			return fmt.Errorf("quota_type untuk paket voucher harus unlimited atau quota")
		}
	}

	// Validasi burst field (all-atau-nothing)
	if err := domain.ValidateBurstFields(pkg.BurstDownloadMbps, pkg.BurstUploadMbps, pkg.BurstThresholdMbps, pkg.BurstTimeSeconds); err != nil {
		return err
	}

	// Validasi quota conditional
	if qt != domain.QuotaUnlimited {
		if pkg.QuotaMB == nil {
			return fmt.Errorf("quota_mb wajib jika quota_type bukan unlimited")
		}
		if (qt == domain.QuotaMonthlyQuota || qt == domain.QuotaFUP) && pkg.QuotaAction == "" {
			return fmt.Errorf("quota_action wajib jika quota_type = monthly_quota atau fup")
		}
		if pkg.QuotaAction == string(domain.QuotaActionThrottle) && pkg.ThrottleMbps == nil {
			return fmt.Errorf("throttle_mbps wajib jika quota_action = throttle")
		}
	}
	return nil
}
