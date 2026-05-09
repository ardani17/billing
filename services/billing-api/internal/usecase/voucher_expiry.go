// voucher_expiry.go berisi business logic untuk proses expiry voucher (job cron).
// Mengimplementasikan ProcessExpiredVouchers pada VoucherExpiryUsecase.
// Memproses voucher terjual yang sudah melewati expires_at secara batch,
// mengembalikan saldo reseller_price_snapshot ke reseller, dan mencatat transaksi refund.
package usecase

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
)

// batchSizeExpiry adalah jumlah voucher yang diproses per batch saat expiry.
const batchSizeExpiry = 100

// VoucherExpiryUsecase mengimplementasikan business logic untuk proses expiry voucher.
type VoucherExpiryUsecase struct {
	voucherRepo     domain.VoucherRepository
	voucherAuditLog domain.VoucherAuditLogRepository
	resellerRepo    domain.ResellerRepository
	txRepo          domain.ResellerTransactionRepository
	pool            *pgxpool.Pool
	queries         *repository.Queries
	logger          zerolog.Logger
}

// NewVoucherExpiryUsecase membuat instance baru VoucherExpiryUsecase.
func NewVoucherExpiryUsecase(
	voucherRepo domain.VoucherRepository,
	voucherAuditLog domain.VoucherAuditLogRepository,
	resellerRepo domain.ResellerRepository,
	txRepo domain.ResellerTransactionRepository,
	pool *pgxpool.Pool,
	queries *repository.Queries,
	logger zerolog.Logger,
) *VoucherExpiryUsecase {
	return &VoucherExpiryUsecase{
		voucherRepo:     voucherRepo,
		voucherAuditLog: voucherAuditLog,
		resellerRepo:    resellerRepo,
		txRepo:          txRepo,
		pool:            pool,
		queries:         queries,
		logger:          logger,
	}
}

// ProcessExpiredVouchers memproses semua voucher terjual yang sudah melewati expires_at.
// Alur: loop dalam batch (batchSize=100) -> GetExpiredVouchers(batchSize) ->
// untuk setiap voucher expired: BEGIN TX -> GetForUpdate(resellerID) (row lock) ->
// transisi voucher ke expired -> refund reseller_price_snapshot ke saldo reseller ->
// buat reseller_transaction (type=refund, reference_id=voucher_id) ->
// tulis voucher audit log (voucher.expired, actor=System) -> COMMIT ->
// lanjutkan sampai tidak ada lagi voucher expired.
func (uc *VoucherExpiryUsecase) ProcessExpiredVouchers(ctx context.Context) error {
	totalProcessed := 0

	for {
		// Ambil batch voucher yang sudah expired
		expiredVouchers, err := uc.voucherRepo.GetExpiredVouchers(ctx, batchSizeExpiry)
		if err != nil {
			return fmt.Errorf("usecase: gagal mengambil voucher expired: %w", err)
		}

		// Jika tidak ada lagi voucher expired, selesai
		if len(expiredVouchers) == 0 {
			break
		}

		// Proses setiap voucher expired satu per satu dalam transaksi terpisah
		for _, voucher := range expiredVouchers {
			if err := uc.processOneExpiredVoucher(ctx, voucher); err != nil {
				// Log error tapi lanjutkan ke voucher berikutnya agar tidak menghentikan seluruh batch
				uc.logger.Error().Err(err).
					Str("voucher_id", voucher.ID).
					Str("reseller_id", voucher.ResellerID).
					Msg("gagal memproses voucher expired")
				continue
			}
			totalProcessed++
		}
	}

	if totalProcessed > 0 {
		uc.logger.Info().Int("total_processed", totalProcessed).Msg("selesai memproses voucher expired")
	}

	return nil
}

// processOneExpiredVoucher memproses satu voucher expired secara atomik dalam database transaction.
// Alur: BEGIN TX -> GetForUpdate(resellerID) -> transisi voucher ke expired ->
// refund reseller_price_snapshot -> buat transaksi refund -> tulis audit log -> COMMIT.
func (uc *VoucherExpiryUsecase) processOneExpiredVoucher(ctx context.Context, voucher *domain.Voucher) error {
	// Validasi voucher memiliki reseller_price_snapshot untuk refund
	if voucher.ResellerPriceSnapshot == nil {
		return fmt.Errorf("voucher %s tidak memiliki reseller_price_snapshot", voucher.ID)
	}
	refundAmount := *voucher.ResellerPriceSnapshot

	// Validasi voucher memiliki reseller_id
	if voucher.ResellerID == "" {
		return fmt.Errorf("voucher %s tidak memiliki reseller_id", voucher.ID)
	}

	// Mulai database transaction
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi expiry voucher %s: %w", voucher.ID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Buat queries transaksional
	txQueries := uc.queries.WithTx(tx)
	txResellerRepo := repository.NewResellerRepo(txQueries, nil)
	txVoucherRepo := repository.NewVoucherRepo(txQueries, nil)
	txVoucherAuditRepo := repository.NewVoucherAuditRepo(txQueries)
	txTxRepo := repository.NewResellerTxRepo(txQueries)

	// Ambil reseller dengan row lock (SELECT ... FOR UPDATE) untuk mencegah race condition
	reseller, err := txResellerRepo.GetForUpdate(ctx, voucher.ResellerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil reseller %s untuk update: %w", voucher.ResellerID, err)
	}

	// Transisi status voucher ke expired
	_, err = domain.VoucherTransition(voucher.Status, domain.VoucherStatusExpired)
	if err != nil {
		return fmt.Errorf("gagal transisi voucher %s ke expired: %w", voucher.ID, err)
	}

	_, err = txVoucherRepo.UpdateStatus(ctx, voucher.ID, domain.VoucherStatusExpired)
	if err != nil {
		return fmt.Errorf("gagal update status voucher %s ke expired: %w", voucher.ID, err)
	}

	// Hitung saldo baru setelah refund
	balanceBefore := reseller.Balance
	balanceAfter := balanceBefore + refundAmount

	// Perbarui saldo reseller (refund reseller_price_snapshot)
	if err := txResellerRepo.UpdateBalance(ctx, reseller.ID, balanceAfter); err != nil {
		return fmt.Errorf("gagal update saldo reseller %s saat refund: %w", reseller.ID, err)
	}

	// Buat catatan transaksi reseller (type=refund, reference_id=voucher_id)
	_, err = txTxRepo.Create(ctx, &domain.ResellerTransaction{
		TenantID:      reseller.TenantID,
		ResellerID:    reseller.ID,
		Type:          domain.TransactionRefund,
		Amount:        refundAmount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		ReferenceID:   voucher.ID,
		Notes:         fmt.Sprintf("Refund voucher expired %s", voucher.Code),
	})
	if err != nil {
		return fmt.Errorf("gagal membuat transaksi refund untuk voucher %s: %w", voucher.ID, err)
	}

	// Tulis voucher audit log (voucher.expired, actor=System)
	if err := txVoucherAuditRepo.Create(ctx, &domain.VoucherAuditLog{
		TenantID:  voucher.TenantID,
		VoucherID: voucher.ID,
		Action:    "voucher.expired",
		ActorID:   "system",
		ActorName: "System",
		Metadata: map[string]interface{}{
			"refund_amount":  refundAmount,
			"reseller_id":    reseller.ID,
			"balance_before": balanceBefore,
			"balance_after":  balanceAfter,
		},
	}); err != nil {
		return fmt.Errorf("gagal menulis voucher audit log untuk voucher %s: %w", voucher.ID, err)
	}

	// Commit transaksi
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("gagal commit transaksi expiry voucher %s: %w", voucher.ID, err)
	}

	return nil
}
