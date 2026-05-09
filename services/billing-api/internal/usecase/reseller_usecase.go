// reseller_usecase.go berisi business logic untuk manajemen reseller (CRUD).
// Mengimplementasikan Buat, GetByID, Perbarui, List pada ResellerUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ResellerUsecase mengimplementasikan business logic untuk manajemen reseller.
type ResellerUsecase struct {
	resellerRepo domain.ResellerRepository
	auditLogRepo domain.AuditLogRepository
	queueClient  *asynq.Client
	logger       zerolog.Logger
}

// NewResellerUsecase membuat instance baru ResellerUsecase.
func NewResellerUsecase(
	resellerRepo domain.ResellerRepository,
	auditLogRepo domain.AuditLogRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *ResellerUsecase {
	return &ResellerUsecase{
		resellerRepo: resellerRepo,
		auditLogRepo: auditLogRepo,
		queueClient:  queueClient,
		logger:       logger,
	}
}

// Buat membuat reseller baru.
// Alur: validasi phone uniqueness -> hash password dengan bcrypt -> buat reseller
// dengan status=aktif, balance=0 (atau sesuai permintaan) -> tulis audit log ->
// terbitkan event reseller.created.
func (uc *ResellerUsecase) Create(ctx context.Context, tenantID string, req domain.CreateResellerRequest, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Cek duplikat nomor telepon dalam tenant
	exists, err := uc.resellerRepo.PhoneExists(ctx, tenantID, req.Phone, "")
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal cek phone duplicate: %w", err)
	}
	if exists {
		return nil, domain.ErrResellerPhoneDuplicate
	}

	// Hash password menggunakan bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal hash password: %w", err)
	}

	// Tentukan balance awal (bawaan 0)
	var balance int64
	if req.Balance != nil {
		balance = *req.Balance
	}

	// Tentukan daily purchase limit (bawaan 0 = unlimited)
	var dailyPurchaseLimit int
	if req.DailyPurchaseLimit != nil {
		dailyPurchaseLimit = *req.DailyPurchaseLimit
	}

	// Bangun entity reseller
	reseller := &domain.Reseller{
		TenantID:           tenantID,
		Name:               req.Name,
		Phone:              req.Phone,
		Email:              req.Email,
		Address:            req.Address,
		PasswordHash:       string(hash),
		Balance:            balance,
		DailyPurchaseLimit: dailyPurchaseLimit,
		Status:             domain.ResellerStatusAktif,
	}

	// Simpan ke database
	created, err := uc.resellerRepo.Create(ctx, reseller)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat reseller: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, created.ID, "reseller.created", actor, nil)

	// Terbitkan event reseller.created
	uc.publishEvent(tenantID, "reseller.created", domain.ResellerCreatedPayload{
		ResellerID: created.ID,
		TenantID:   tenantID,
		Name:       created.Name,
	})

	return created, nil
}

// GetByID mengambil detail reseller berdasarkan ID.
// Jika includeAudit true, audit logs juga disertakan.
func (uc *ResellerUsecase) GetByID(ctx context.Context, id string, includeAudit bool) (*domain.ResellerDetail, error) {
	reseller, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &domain.ResellerDetail{
		Reseller: reseller,
	}

	// Opsional: ambil audit logs
	if includeAudit {
		logs, err := uc.auditLogRepo.ListByEntity(ctx, "reseller", id)
		if err != nil {
			uc.logger.Error().Err(err).Str("reseller_id", id).Msg("gagal mengambil audit logs")
			// Jangan gagalkan permintaan, skip audit logs saja
		} else {
			detail.AuditLogs = logs
		}
	}

	return detail, nil
}

// Perbarui memperbarui data reseller.
// Alur: ambil existing -> validasi phone uniqueness (exclude self) -> perbarui ->
// hitung field yang berubah -> tulis audit log dengan nilai lama/baru.
func (uc *ResellerUsecase) Update(ctx context.Context, id string, req domain.UpdateResellerRequest, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Ambil reseller yang ada
	existing, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cek duplikat phone jika phone berubah
	if req.Phone != "" && req.Phone != existing.Phone {
		exists, err := uc.resellerRepo.PhoneExists(ctx, existing.TenantID, req.Phone, id)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal cek phone duplicate: %w", err)
		}
		if exists {
			return nil, domain.ErrResellerPhoneDuplicate
		}
	}

	// Terapkan perubahan ke reseller (hanya field non-zero)
	updated := applyResellerUpdates(existing, req)

	// Simpan ke database
	result, err := uc.resellerRepo.Update(ctx, updated)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memperbarui reseller: %w", err)
	}

	// Hitung field yang berubah untuk audit log
	changes := computeResellerChanges(existing, result)

	// Tulis audit log dengan nilai lama/baru
	if len(changes) > 0 {
		uc.writeAuditLog(ctx, existing.TenantID, id, "reseller.updated", actor, changes)
	}

	return result, nil
}

// List mengambil daftar reseller dengan paginasi, filter, dan pengurutan.
// Menerapkan bawaan: page=1, page_size=25.
func (uc *ResellerUsecase) List(ctx context.Context, params domain.ResellerListParams) (*domain.ResellerListResult, error) {
	// Terapkan bawaan
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	return uc.resellerRepo.List(ctx, params)
}

// --- Fungsi bantu methods ---

// writeAuditLog menulis audit log. Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *ResellerUsecase) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor domain.ActorInfo, changes map[string]interface{}) {
	log := &domain.AuditLog{
		TenantID:   tenantID,
		EntityType: "reseller",
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ActorID,
		ActorName:  actor.ActorName,
		Changes:    changes,
	}
	if err := uc.auditLogRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).
			Str("entity_id", entityID).
			Str("action", action).
			Msg("gagal menulis audit log")
	}
}

// publishEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *ResellerUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
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

// applyResellerUpdates menerapkan perubahan dari UpdateResellerRequest ke Reseller.
// Hanya field yang non-zero/non-empty yang diperbarui.
func applyResellerUpdates(existing *domain.Reseller, req domain.UpdateResellerRequest) *domain.Reseller {
	updated := *existing // copy

	if req.Name != "" {
		updated.Name = req.Name
	}
	if req.Phone != "" {
		updated.Phone = req.Phone
	}
	if req.Email != "" {
		updated.Email = req.Email
	}
	if req.Address != "" {
		updated.Address = req.Address
	}
	if req.DailyPurchaseLimit != nil {
		updated.DailyPurchaseLimit = *req.DailyPurchaseLimit
	}

	return &updated
}

// computeResellerChanges menghitung field yang berubah antara old dan new reseller.
// Mengembalikan map dengan format {"field": {"old": oldVal, "new": newVal}}.
func computeResellerChanges(old, updated *domain.Reseller) map[string]interface{} {
	changes := make(map[string]interface{})

	if old.Name != updated.Name {
		changes["name"] = map[string]interface{}{"old": old.Name, "new": updated.Name}
	}
	if old.Phone != updated.Phone {
		changes["phone"] = map[string]interface{}{"old": old.Phone, "new": updated.Phone}
	}
	if old.Email != updated.Email {
		changes["email"] = map[string]interface{}{"old": old.Email, "new": updated.Email}
	}
	if old.Address != updated.Address {
		changes["address"] = map[string]interface{}{"old": old.Address, "new": updated.Address}
	}
	if old.DailyPurchaseLimit != updated.DailyPurchaseLimit {
		changes["daily_purchase_limit"] = map[string]interface{}{"old": old.DailyPurchaseLimit, "new": updated.DailyPurchaseLimit}
	}

	return changes
}
