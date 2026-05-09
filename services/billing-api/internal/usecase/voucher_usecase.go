// voucher_usecase.go berisi business logic untuk manajemen voucher (admin).
// Mengimplementasikan Buat, List, GetByID pada VoucherUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// VoucherUsecase mengimplementasikan business logic untuk manajemen voucher.
type VoucherUsecase struct {
	voucherRepo     domain.VoucherRepository
	voucherAuditLog domain.VoucherAuditLogRepository
	packageRepo     domain.PackageRepository
	queueClient     *asynq.Client
	logger          zerolog.Logger
}

// NewVoucherUsecase membuat instance baru VoucherUsecase.
func NewVoucherUsecase(
	voucherRepo domain.VoucherRepository,
	voucherAuditLog domain.VoucherAuditLogRepository,
	packageRepo domain.PackageRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *VoucherUsecase {
	return &VoucherUsecase{
		voucherRepo:     voucherRepo,
		voucherAuditLog: voucherAuditLog,
		packageRepo:     packageRepo,
		queueClient:     queueClient,
		logger:          logger,
	}
}

// Buat menghasilkan batch voucher baru untuk paket tertentu.
// Alur: validasi paket ada dan type=voucher -> jika quantity ≤ 500: buat kode
// via GenerateVoucherCodes -> BulkCreate voucher dengan status tersedia ->
// tulis voucher audit logs (voucher.generated) -> terbitkan event voucher.batch_generated ->
// kembalikan hasil dengan voucher.
// Jika quantity > 500: antrekan job voucher.async_generate -> kembalikan hasil dengan job_id.
func (uc *VoucherUsecase) Generate(ctx context.Context, tenantID string, req domain.GenerateVoucherRequest, actor domain.ActorInfo) (*domain.GenerateVoucherResult, error) {
	// Validasi paket ada dan bertipe voucher
	pkg, err := uc.packageRepo.GetByID(ctx, req.PackageID)
	if err != nil {
		return nil, err
	}
	if pkg.Type != domain.PackageTypeVoucher {
		return nil, domain.ErrInvalidPackageType
	}

	// Jika quantity > 500, antrekan job async
	if req.Quantity > 500 {
		return uc.enqueueAsyncGenerate(tenantID, req, actor)
	}

	// Buat kode voucher secara sinkron
	return uc.generateSync(ctx, tenantID, req, actor)
}

// generateSync menghasilkan voucher secara sinkron (quantity ≤ 500).
// Alur: buat kode -> buat voucher di database -> tulis audit log -> terbitkan event.
func (uc *VoucherUsecase) generateSync(ctx context.Context, tenantID string, req domain.GenerateVoucherRequest, actor domain.ActorInfo) (*domain.GenerateVoucherResult, error) {
	// Buat kode voucher unik menggunakan crypto/rand
	codeFormat := domain.CodeFormat(req.CodeFormat)
	existingCodes := make(map[string]struct{})
	codes, failed := domain.GenerateVoucherCodes(codeFormat, req.CodeLength, req.Prefix, req.Quantity, existingCodes, 3)

	// Bangun slice voucher untuk BulkCreate
	vouchers := make([]*domain.Voucher, 0, len(codes))
	for _, code := range codes {
		vouchers = append(vouchers, &domain.Voucher{
			TenantID:  tenantID,
			Code:      code,
			PackageID: req.PackageID,
			Status:    domain.VoucherStatusTersedia,
		})
	}

	// Simpan voucher ke database
	created, err := uc.voucherRepo.BulkCreate(ctx, vouchers)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal bulk create voucher: %w", err)
	}

	// Tulis voucher audit log untuk setiap voucher yang berhasil dibuat
	auditLogs := make([]*domain.VoucherAuditLog, 0, len(created))
	for _, v := range created {
		auditLogs = append(auditLogs, &domain.VoucherAuditLog{
			TenantID:  tenantID,
			VoucherID: v.ID,
			Action:    "voucher.generated",
			ActorID:   actor.ActorID,
			ActorName: actor.ActorName,
			Metadata: map[string]interface{}{
				"package_id":  req.PackageID,
				"code_format": req.CodeFormat,
				"code_length": req.CodeLength,
				"prefix":      req.Prefix,
			},
		})
	}

	if len(auditLogs) > 0 {
		if err := uc.voucherAuditLog.BulkCreate(ctx, auditLogs); err != nil {
			uc.logger.Error().Err(err).
				Int("count", len(auditLogs)).
				Msg("gagal menulis voucher audit logs saat generate")
		}
	}

	// Terbitkan event voucher.batch_generated
	uc.publishEvent(tenantID, "voucher.batch_generated", domain.VoucherBatchGeneratedPayload{
		TenantID:    tenantID,
		PackageID:   req.PackageID,
		Quantity:    len(created),
		GeneratedBy: actor.ActorID,
	})

	return &domain.GenerateVoucherResult{
		TotalRequested: req.Quantity,
		TotalGenerated: len(created),
		TotalFailed:    failed,
		Vouchers:       created,
	}, nil
}

// enqueueAsyncGenerate mengirim job buat voucher ke queue untuk diproses secara async.
// Digunakan saat quantity > 500 agar HTTP respons tetap cepat.
func (uc *VoucherUsecase) enqueueAsyncGenerate(tenantID string, req domain.GenerateVoucherRequest, actor domain.ActorInfo) (*domain.GenerateVoucherResult, error) {
	// Bangun payload job async
	payload := map[string]interface{}{
		"package_id":  req.PackageID,
		"quantity":    req.Quantity,
		"code_format": req.CodeFormat,
		"code_length": req.CodeLength,
		"prefix":      req.Prefix,
		"actor_id":    actor.ActorID,
		"actor_name":  actor.ActorName,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal marshal async generate payload: %w", err)
	}

	// Enqueue job ke queue
	envelope := queue.TaskEnvelope{
		EventType: "voucher.async_generate",
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		return nil, fmt.Errorf("usecase: gagal enqueue async generate job: %w", err)
	}

	// Gunakan correlation ID sebagai job_id
	jobID := envelope.CorrelationID
	if jobID == "" {
		jobID = "voucher-generate-" + tenantID
	}

	return &domain.GenerateVoucherResult{
		TotalRequested: req.Quantity,
		TotalGenerated: 0,
		TotalFailed:    0,
		JobID:          jobID,
	}, nil
}

// List mengambil daftar voucher dengan paginasi, filter, dan pengurutan.
// Menerapkan bawaan: page=1, page_size=25.
func (uc *VoucherUsecase) List(ctx context.Context, params domain.VoucherListParams) (*domain.VoucherListResult, error) {
	// Terapkan bawaan paginasi
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	return uc.voucherRepo.List(ctx, params)
}

// GetByID mengambil detail voucher berdasarkan ID, termasuk audit logs.
// Alur: ambil voucher -> ambil voucher audit logs -> kembalikan VoucherDetail.
func (uc *VoucherUsecase) GetByID(ctx context.Context, id string) (*domain.VoucherDetail, error) {
	// Ambil voucher berdasarkan ID
	voucher, err := uc.voucherRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ambil audit logs untuk voucher ini
	auditLogs, err := uc.voucherAuditLog.ListByVoucher(ctx, id)
	if err != nil {
		uc.logger.Error().Err(err).Str("voucher_id", id).Msg("gagal mengambil voucher audit logs")
		// Jangan gagalkan permintaan, skip audit logs saja
	}

	return &domain.VoucherDetail{
		Voucher:   voucher,
		AuditLogs: auditLogs,
	}, nil
}

// Activate mengaktifkan voucher Hotspot dan mengirim event agar network-service membuat user RouterOS.
func (uc *VoucherUsecase) Activate(ctx context.Context, tenantID string, req domain.ActivateVoucherRequest, actor domain.ActorInfo) (*domain.Voucher, error) {
	code := strings.TrimSpace(req.Code)
	voucher, err := uc.voucherRepo.GetByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}
	if voucher.Status != domain.VoucherStatusTerjual {
		return nil, fmt.Errorf("%w: voucher status %s, hanya voucher terjual yang bisa diaktifkan", domain.ErrInvalidVoucherTransition, voucher.Status)
	}

	pkg, err := uc.packageRepo.GetByID(ctx, voucher.PackageID)
	if err != nil {
		return nil, err
	}
	if pkg.Type != domain.PackageTypeVoucher {
		return nil, domain.ErrInvalidPackageType
	}

	expiresAt := time.Now().Add(voucherDuration(pkg))
	activated, err := uc.voucherRepo.Activate(ctx, voucher.ID, expiresAt)
	if err != nil {
		return nil, err
	}
	activated.PackageName = pkg.Name

	if err := uc.voucherAuditLog.Create(ctx, &domain.VoucherAuditLog{
		TenantID:  tenantID,
		VoucherID: activated.ID,
		Action:    "voucher.activated",
		ActorID:   actor.ActorID,
		ActorName: actor.ActorName,
		Metadata: map[string]interface{}{
			"router_id":            strings.TrimSpace(req.RouterID),
			"hotspot_profile_name": pkg.HotspotProfileName,
			"limit_uptime":         routerOSLimitUptime(pkg),
			"mac_address":          strings.TrimSpace(req.MACAddress),
		},
	}); err != nil {
		uc.logger.Error().Err(err).Str("voucher_id", activated.ID).Msg("gagal menulis audit log voucher activation")
	}

	uc.publishEvent(tenantID, "voucher.activated", domain.VoucherActivatedPayload{
		TenantID:           tenantID,
		VoucherID:          activated.ID,
		Code:               activated.Code,
		PackageID:          activated.PackageID,
		PackageName:        pkg.Name,
		RouterID:           strings.TrimSpace(req.RouterID),
		HotspotProfileName: pkg.HotspotProfileName,
		LimitUptime:        routerOSLimitUptime(pkg),
		MACAddress:         strings.TrimSpace(req.MACAddress),
	})

	return activated, nil
}

// ListByReseller mengambil daftar voucher milik reseller tertentu dengan paginasi.
// Menerapkan bawaan: page=1, page_size=25.
func (uc *VoucherUsecase) ListByReseller(ctx context.Context, params domain.ResellerVoucherListParams) (*domain.VoucherListResult, error) {
	// Terapkan bawaan paginasi
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	return uc.voucherRepo.ListByReseller(ctx, params)
}

// CountSoldToday menghitung jumlah voucher yang dibeli reseller hari ini.
func (uc *VoucherUsecase) CountSoldToday(ctx context.Context, resellerID string) (int, error) {
	return uc.voucherRepo.CountSoldToday(ctx, resellerID)
}

func voucherDuration(pkg *domain.Package) time.Duration {
	value := 1
	if pkg.DurationValue != nil && *pkg.DurationValue > 0 {
		value = *pkg.DurationValue
	}
	switch pkg.DurationUnit {
	case string(domain.DurationHours):
		return time.Duration(value) * time.Hour
	case string(domain.DurationWeeks):
		return time.Duration(value) * 7 * 24 * time.Hour
	case string(domain.DurationMonths):
		return time.Duration(value) * 30 * 24 * time.Hour
	default:
		return time.Duration(value) * 24 * time.Hour
	}
}

func routerOSLimitUptime(pkg *domain.Package) string {
	value := 1
	if pkg.DurationValue != nil && *pkg.DurationValue > 0 {
		value = *pkg.DurationValue
	}
	switch pkg.DurationUnit {
	case string(domain.DurationHours):
		return fmt.Sprintf("%dh", value)
	case string(domain.DurationWeeks):
		return fmt.Sprintf("%dw", value)
	case string(domain.DurationMonths):
		return fmt.Sprintf("%dw", value*4)
	default:
		return fmt.Sprintf("%dd", value)
	}
}

// CountAvailableByReseller menghitung jumlah voucher tersedia (status terjual) milik reseller.
func (uc *VoucherUsecase) CountAvailableByReseller(ctx context.Context, resellerID string) (int, error) {
	return uc.voucherRepo.CountByResellerAndStatus(ctx, resellerID, []domain.VoucherStatus{domain.VoucherStatusTerjual})
}

// VerifyOwnership memverifikasi bahwa semua voucher milik reseller tertentu.
// Mengembalikan ErrVoucherForbidden jika ada voucher yang bukan milik reseller.
func (uc *VoucherUsecase) VerifyOwnership(ctx context.Context, voucherIDs []string, resellerID string) error {
	vouchers, err := uc.voucherRepo.GetByIDs(ctx, voucherIDs)
	if err != nil {
		return err
	}

	for _, v := range vouchers {
		if v.ResellerID != resellerID {
			return domain.ErrVoucherForbidden
		}
	}

	return nil
}

// --- Fungsi bantu methods ---

// publishEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *VoucherUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
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
