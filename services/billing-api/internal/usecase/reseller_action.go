// reseller_action.go berisi business logic untuk aksi reseller (suspend, activate,
// deactivate, reset password, deposit, withdraw).
// Mengimplementasikan ResellerActionUsecase pada struct ResellerActionUsecase.
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
)

// ResellerActionUsecase mengimplementasikan business logic untuk aksi reseller
// (suspend, activate, deactivate, reset password, deposit, withdraw).
type ResellerActionUsecase struct {
	resellerRepo    domain.ResellerRepository
	voucherRepo     domain.VoucherRepository
	voucherAuditLog domain.VoucherAuditLogRepository
	txRepo          domain.ResellerTransactionRepository
	auditLogRepo    domain.AuditLogRepository
	sessionRepo     domain.SessionRepository
	pool            *pgxpool.Pool
	queries         *repository.Queries
	queueClient     *asynq.Client
	logger          zerolog.Logger
}

// NewResellerActionUsecase membuat instance baru ResellerActionUsecase.
func NewResellerActionUsecase(
	resellerRepo domain.ResellerRepository,
	voucherRepo domain.VoucherRepository,
	voucherAuditLog domain.VoucherAuditLogRepository,
	txRepo domain.ResellerTransactionRepository,
	auditLogRepo domain.AuditLogRepository,
	sessionRepo domain.SessionRepository,
	pool *pgxpool.Pool,
	queries *repository.Queries,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *ResellerActionUsecase {
	return &ResellerActionUsecase{
		resellerRepo:    resellerRepo,
		voucherRepo:     voucherRepo,
		voucherAuditLog: voucherAuditLog,
		txRepo:          txRepo,
		auditLogRepo:    auditLogRepo,
		sessionRepo:     sessionRepo,
		pool:            pool,
		queries:         queries,
		queueClient:     queueClient,
		logger:          logger,
	}
}

// Suspend mengubah status reseller dari aktif ke suspended.
// Flow: fetch reseller → validasi transisi → update status → tulis audit log → publish event.
func (uc *ResellerActionUsecase) Suspend(ctx context.Context, id string, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Ambil reseller yang ada
	reseller, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi transisi status menggunakan state machine
	oldStatus := reseller.Status
	_, err = domain.ResellerTransition(reseller.Status, domain.ResellerStatusSuspended)
	if err != nil {
		return nil, err
	}

	// Update status di database
	updated, err := uc.resellerRepo.UpdateStatus(ctx, id, domain.ResellerStatusSuspended)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal suspend reseller: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, reseller.TenantID, id, "reseller.status_changed", actor, map[string]interface{}{
		"old_status": string(oldStatus),
		"new_status": string(domain.ResellerStatusSuspended),
	})

	// Publish event reseller.status_changed
	uc.publishEvent(reseller.TenantID, "reseller.status_changed", domain.ResellerStatusChangedPayload{
		ResellerID: id,
		TenantID:   reseller.TenantID,
		OldStatus:  string(oldStatus),
		NewStatus:  string(domain.ResellerStatusSuspended),
	})

	return updated, nil
}

// Activate mengubah status reseller dari suspended ke aktif.
// Flow: fetch reseller → validasi transisi → update status → tulis audit log → publish event.
func (uc *ResellerActionUsecase) Activate(ctx context.Context, id string, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Ambil reseller yang ada
	reseller, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi transisi status menggunakan state machine
	oldStatus := reseller.Status
	_, err = domain.ResellerTransition(reseller.Status, domain.ResellerStatusAktif)
	if err != nil {
		return nil, err
	}

	// Update status di database
	updated, err := uc.resellerRepo.UpdateStatus(ctx, id, domain.ResellerStatusAktif)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal activate reseller: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, reseller.TenantID, id, "reseller.status_changed", actor, map[string]interface{}{
		"old_status": string(oldStatus),
		"new_status": string(domain.ResellerStatusAktif),
	})

	// Publish event reseller.status_changed
	uc.publishEvent(reseller.TenantID, "reseller.status_changed", domain.ResellerStatusChangedPayload{
		ResellerID: id,
		TenantID:   reseller.TenantID,
		OldStatus:  string(oldStatus),
		NewStatus:  string(domain.ResellerStatusAktif),
	})

	return updated, nil
}

// Deactivate mengubah status reseller ke nonaktif (terminal state).
// Flow: fetch reseller → verifikasi confirmation_name → validasi transisi → update status →
// void semua voucher tersedia milik reseller → tulis voucher audit logs →
// tulis audit log → publish event → invalidasi semua session reseller.
func (uc *ResellerActionUsecase) Deactivate(ctx context.Context, id string, confirmName string, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Ambil reseller yang ada
	reseller, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verifikasi confirmation_name cocok dengan nama reseller (case-sensitive)
	if confirmName != reseller.Name {
		return nil, domain.ErrConfirmationMismatch
	}

	// Validasi transisi status menggunakan state machine
	oldStatus := reseller.Status
	_, err = domain.ResellerTransition(reseller.Status, domain.ResellerStatusNonaktif)
	if err != nil {
		return nil, err
	}

	// Update status di database
	updated, err := uc.resellerRepo.UpdateStatus(ctx, id, domain.ResellerStatusNonaktif)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal deactivate reseller: %w", err)
	}

	// Void semua voucher tersedia milik reseller
	uc.voidResellerVouchers(ctx, reseller, actor)

	// Tulis audit log
	uc.writeAuditLog(ctx, reseller.TenantID, id, "reseller.status_changed", actor, map[string]interface{}{
		"old_status": string(oldStatus),
		"new_status": string(domain.ResellerStatusNonaktif),
	})

	// Publish event reseller.status_changed
	uc.publishEvent(reseller.TenantID, "reseller.status_changed", domain.ResellerStatusChangedPayload{
		ResellerID: id,
		TenantID:   reseller.TenantID,
		OldStatus:  string(oldStatus),
		NewStatus:  string(domain.ResellerStatusNonaktif),
	})

	// Invalidasi semua session reseller
	if err := uc.sessionRepo.DeleteByUserID(ctx, id); err != nil {
		uc.logger.Error().Err(err).Str("reseller_id", id).Msg("gagal menghapus session reseller saat deactivate")
	}

	return updated, nil
}

// voidResellerVouchers mem-void semua voucher dengan status tersedia milik reseller.
// Menulis voucher audit log untuk setiap voucher yang di-void.
func (uc *ResellerActionUsecase) voidResellerVouchers(ctx context.Context, reseller *domain.Reseller, actor domain.ActorInfo) {
	// Ambil semua voucher tersedia milik reseller
	vouchers, err := uc.voucherRepo.ListByReseller(ctx, domain.ResellerVoucherListParams{
		ResellerID: reseller.ID,
		TenantID:   reseller.TenantID,
		Status:     string(domain.VoucherStatusTersedia),
		PageSize:   50, // batch size
		Page:       1,
	})
	if err != nil {
		uc.logger.Error().Err(err).Str("reseller_id", reseller.ID).Msg("gagal mengambil voucher tersedia untuk void")
		return
	}

	// Void setiap voucher dan tulis audit log
	for _, v := range vouchers.Data {
		_, err := uc.voucherRepo.UpdateStatus(ctx, v.ID, domain.VoucherStatusVoid)
		if err != nil {
			uc.logger.Error().Err(err).Str("voucher_id", v.ID).Msg("gagal void voucher saat deactivate reseller")
			continue
		}

		// Tulis voucher audit log
		if err := uc.voucherAuditLog.Create(ctx, &domain.VoucherAuditLog{
			TenantID:  reseller.TenantID,
			VoucherID: v.ID,
			Action:    "voucher.voided",
			ActorID:   actor.ActorID,
			ActorName: actor.ActorName,
			Metadata: map[string]interface{}{
				"reason": "reseller_deactivated",
			},
		}); err != nil {
			uc.logger.Error().Err(err).Str("voucher_id", v.ID).Msg("gagal menulis voucher audit log saat void")
		}
	}

	// Proses halaman berikutnya jika ada lebih banyak voucher
	totalPages := vouchers.Pagination.TotalPages
	for page := 2; page <= totalPages; page++ {
		pageVouchers, err := uc.voucherRepo.ListByReseller(ctx, domain.ResellerVoucherListParams{
			ResellerID: reseller.ID,
			TenantID:   reseller.TenantID,
			Status:     string(domain.VoucherStatusTersedia),
			PageSize:   50,
			Page:       page,
		})
		if err != nil {
			uc.logger.Error().Err(err).Str("reseller_id", reseller.ID).Int("page", page).Msg("gagal mengambil voucher tersedia halaman berikutnya")
			break
		}

		for _, v := range pageVouchers.Data {
			_, err := uc.voucherRepo.UpdateStatus(ctx, v.ID, domain.VoucherStatusVoid)
			if err != nil {
				uc.logger.Error().Err(err).Str("voucher_id", v.ID).Msg("gagal void voucher saat deactivate reseller")
				continue
			}

			if err := uc.voucherAuditLog.Create(ctx, &domain.VoucherAuditLog{
				TenantID:  reseller.TenantID,
				VoucherID: v.ID,
				Action:    "voucher.voided",
				ActorID:   actor.ActorID,
				ActorName: actor.ActorName,
				Metadata: map[string]interface{}{
					"reason": "reseller_deactivated",
				},
			}); err != nil {
				uc.logger.Error().Err(err).Str("voucher_id", v.ID).Msg("gagal menulis voucher audit log saat void")
			}
		}
	}
}

// ResetPassword menghasilkan password baru acak untuk reseller.
// Flow: fetch reseller → generate password acak 8 karakter alfanumerik →
// hash dengan bcrypt → update password_hash → invalidasi semua session →
// tulis audit log → kembalikan password plaintext.
func (uc *ResellerActionUsecase) ResetPassword(ctx context.Context, id string, actor domain.ActorInfo) (string, error) {
	// Ambil reseller yang ada (untuk validasi keberadaan dan mendapatkan tenant_id)
	reseller, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Generate password acak 8 karakter alfanumerik menggunakan crypto/rand
	plaintext, err := generateRandomPassword(8)
	if err != nil {
		return "", fmt.Errorf("usecase: gagal generate password acak: %w", err)
	}

	// Hash password dengan bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("usecase: gagal hash password: %w", err)
	}

	// Update password_hash di database
	if err := uc.resellerRepo.UpdatePasswordHash(ctx, id, string(hash)); err != nil {
		return "", fmt.Errorf("usecase: gagal update password hash: %w", err)
	}

	// Invalidasi semua session reseller
	if err := uc.sessionRepo.DeleteByUserID(ctx, id); err != nil {
		uc.logger.Error().Err(err).Str("reseller_id", id).Msg("gagal menghapus session reseller saat reset password")
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, reseller.TenantID, id, "reseller.password_reset", actor, nil)

	return plaintext, nil
}

// Deposit menambah saldo reseller secara atomik menggunakan database transaction.
// Flow: BEGIN TX → GetForUpdate (row lock) → update balance → create transaksi →
// tulis audit log → COMMIT → kembalikan reseller yang diperbarui.
func (uc *ResellerActionUsecase) Deposit(ctx context.Context, id string, req domain.DepositRequest, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Mulai database transaction
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memulai transaksi deposit: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Buat queries transaksional
	txQueries := uc.queries.WithTx(tx)
	txResellerRepo := repository.NewResellerRepo(txQueries, nil)
	txTxRepo := repository.NewResellerTxRepo(txQueries)

	// Ambil reseller dengan row lock (SELECT ... FOR UPDATE)
	reseller, err := txResellerRepo.GetForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}

	// Hitung saldo baru
	balanceBefore := reseller.Balance
	balanceAfter := balanceBefore + req.Amount

	// Update saldo reseller
	if err := txResellerRepo.UpdateBalance(ctx, id, balanceAfter); err != nil {
		return nil, fmt.Errorf("usecase: gagal update saldo deposit: %w", err)
	}

	// Buat catatan transaksi reseller
	_, err = txTxRepo.Create(ctx, &domain.ResellerTransaction{
		TenantID:      reseller.TenantID,
		ResellerID:    reseller.ID,
		Type:          domain.TransactionDeposit,
		Amount:        req.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Notes:         req.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat transaksi deposit: %w", err)
	}

	// Commit transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("usecase: gagal commit transaksi deposit: %w", err)
	}

	// Tulis audit log (di luar transaksi, tidak boleh gagalkan operasi utama)
	uc.writeAuditLog(ctx, reseller.TenantID, id, "reseller.deposit", actor, map[string]interface{}{
		"amount":        req.Amount,
		"balance_after": balanceAfter,
		"notes":         req.Notes,
	})

	// Ambil reseller terbaru untuk dikembalikan
	updated, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil reseller setelah deposit: %w", err)
	}

	return updated, nil
}

// Withdraw mengurangi saldo reseller secara atomik menggunakan database transaction.
// Flow: BEGIN TX → GetForUpdate (row lock) → verifikasi saldo cukup →
// update balance → create transaksi → tulis audit log → COMMIT →
// kembalikan reseller yang diperbarui.
// Mengembalikan ErrInsufficientBalance jika saldo tidak mencukupi.
func (uc *ResellerActionUsecase) Withdraw(ctx context.Context, id string, req domain.WithdrawRequest, actor domain.ActorInfo) (*domain.Reseller, error) {
	// Mulai database transaction
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memulai transaksi withdraw: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Buat queries transaksional
	txQueries := uc.queries.WithTx(tx)
	txResellerRepo := repository.NewResellerRepo(txQueries, nil)
	txTxRepo := repository.NewResellerTxRepo(txQueries)

	// Ambil reseller dengan row lock (SELECT ... FOR UPDATE)
	reseller, err := txResellerRepo.GetForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verifikasi saldo mencukupi
	if reseller.Balance < req.Amount {
		return nil, domain.ErrInsufficientBalance
	}

	// Hitung saldo baru
	balanceBefore := reseller.Balance
	balanceAfter := balanceBefore - req.Amount

	// Update saldo reseller
	if err := txResellerRepo.UpdateBalance(ctx, id, balanceAfter); err != nil {
		return nil, fmt.Errorf("usecase: gagal update saldo withdraw: %w", err)
	}

	// Buat catatan transaksi reseller
	_, err = txTxRepo.Create(ctx, &domain.ResellerTransaction{
		TenantID:      reseller.TenantID,
		ResellerID:    reseller.ID,
		Type:          domain.TransactionWithdraw,
		Amount:        req.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Notes:         req.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat transaksi withdraw: %w", err)
	}

	// Commit transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("usecase: gagal commit transaksi withdraw: %w", err)
	}

	// Tulis audit log (di luar transaksi, tidak boleh gagalkan operasi utama)
	uc.writeAuditLog(ctx, reseller.TenantID, id, "reseller.withdraw", actor, map[string]interface{}{
		"amount":        req.Amount,
		"balance_after": balanceAfter,
		"notes":         req.Notes,
	})

	// Ambil reseller terbaru untuk dikembalikan
	updated, err := uc.resellerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil reseller setelah withdraw: %w", err)
	}

	return updated, nil
}

// --- Helper methods ---

// generateRandomPassword menghasilkan password acak alfanumerik dengan panjang tertentu
// menggunakan crypto/rand untuk keamanan kriptografis.
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("gagal membaca crypto/rand: %w", err)
	}

	password := make([]byte, length)
	for i := 0; i < length; i++ {
		password[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(password), nil
}

// writeAuditLog menulis audit log. Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *ResellerActionUsecase) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor domain.ActorInfo, changes map[string]interface{}) {
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
func (uc *ResellerActionUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
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
