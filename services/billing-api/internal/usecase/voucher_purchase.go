// voucher_purchase.go berisi business logic untuk pembelian voucher oleh reseller.
// Mengimplementasikan Buy pada VoucherPurchaseUsecase.
// Semua operasi balance bersifat atomik menggunakan database transaction + row lock.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
)

// DefaultVoucherExpiryDays adalah default masa berlaku voucher setelah pembelian (dalam hari).
// Dapat dikonfigurasi per tenant, namun saat ini menggunakan nilai default 90 hari.
const DefaultVoucherExpiryDays = 90

// VoucherPurchaseUsecase mengimplementasikan business logic untuk pembelian voucher oleh reseller.
type VoucherPurchaseUsecase struct {
	resellerRepo    domain.ResellerRepository
	voucherRepo     domain.VoucherRepository
	voucherAuditLog domain.VoucherAuditLogRepository
	packageRepo     domain.PackageRepository
	txRepo          domain.ResellerTransactionRepository
	pool            *pgxpool.Pool
	queries         *repository.Queries
	queueClient     *asynq.Client
	logger          zerolog.Logger
}

// NewVoucherPurchaseUsecase membuat instance baru VoucherPurchaseUsecase.
func NewVoucherPurchaseUsecase(
	resellerRepo domain.ResellerRepository,
	voucherRepo domain.VoucherRepository,
	voucherAuditLog domain.VoucherAuditLogRepository,
	packageRepo domain.PackageRepository,
	txRepo domain.ResellerTransactionRepository,
	pool *pgxpool.Pool,
	queries *repository.Queries,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *VoucherPurchaseUsecase {
	return &VoucherPurchaseUsecase{
		resellerRepo:    resellerRepo,
		voucherRepo:     voucherRepo,
		voucherAuditLog: voucherAuditLog,
		packageRepo:     packageRepo,
		txRepo:          txRepo,
		pool:            pool,
		queries:         queries,
		queueClient:     queueClient,
		logger:          logger,
	}
}

// Buy melakukan pembelian voucher oleh reseller secara atomik.
// Flow: BEGIN TX → GetForUpdate(resellerID) (row lock) → verifikasi status aktif →
// cek batas pembelian harian → ambil paket (verifikasi type=voucher, is_active=true) →
// hitung totalCost = quantity × reseller_price → verifikasi saldo cukup →
// ambil voucher tersedia berdasarkan paket → untuk setiap voucher: AssignToReseller
// (set sell_price_snapshot, reseller_price_snapshot, purchased_at, expires_at) →
// tulis voucher audit log (voucher.sold, actor=reseller) →
// UpdateBalance(balance - totalCost) → buat reseller_transaction (type=purchase) →
// COMMIT → publish event voucher.purchased → kembalikan BuyVoucherResult.
func (uc *VoucherPurchaseUsecase) Buy(ctx context.Context, resellerID string, req domain.BuyVoucherRequest) (*domain.BuyVoucherResult, error) {
	// Mulai database transaction
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memulai transaksi pembelian voucher: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Buat queries transaksional
	txQueries := uc.queries.WithTx(tx)
	txResellerRepo := repository.NewResellerRepo(txQueries, nil)
	txVoucherRepo := repository.NewVoucherRepo(txQueries, nil)
	txVoucherAuditRepo := repository.NewVoucherAuditRepo(txQueries)
	txTxRepo := repository.NewResellerTxRepo(txQueries)

	// Ambil reseller dengan row lock (SELECT ... FOR UPDATE) untuk mencegah race condition
	reseller, err := txResellerRepo.GetForUpdate(ctx, resellerID)
	if err != nil {
		return nil, err
	}

	// Verifikasi status reseller harus aktif
	if reseller.Status != domain.ResellerStatusAktif {
		return nil, domain.ErrResellerAccountDisabled
	}

	// Cek batas pembelian harian (0 = unlimited)
	if reseller.DailyPurchaseLimit > 0 {
		todayCount, err := txResellerRepo.CountTodayPurchases(ctx, resellerID)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal menghitung pembelian hari ini: %w", err)
		}
		if todayCount+req.Quantity > reseller.DailyPurchaseLimit {
			return nil, domain.ErrDailyLimitExceeded
		}
	}

	// Ambil paket dan verifikasi type=voucher dan is_active=true
	pkg, err := uc.packageRepo.GetByID(ctx, req.PackageID)
	if err != nil {
		return nil, err
	}
	if pkg.Type != domain.PackageTypeVoucher {
		return nil, domain.ErrInvalidPackageType
	}
	if !pkg.IsActive {
		return nil, domain.ErrPackageNotActive
	}

	// Pastikan paket memiliki reseller_price yang valid
	if pkg.ResellerPrice == nil || *pkg.ResellerPrice <= 0 {
		return nil, fmt.Errorf("usecase: paket tidak memiliki reseller_price yang valid")
	}
	if pkg.SellPrice == nil || *pkg.SellPrice <= 0 {
		return nil, fmt.Errorf("usecase: paket tidak memiliki sell_price yang valid")
	}

	// Hitung total biaya pembelian
	resellerPrice := *pkg.ResellerPrice
	sellPrice := *pkg.SellPrice
	totalCost := int64(req.Quantity) * resellerPrice

	// Verifikasi saldo reseller mencukupi
	if reseller.Balance < totalCost {
		return nil, domain.ErrInsufficientBalance
	}

	// Ambil voucher tersedia berdasarkan paket
	availableVouchers, err := txVoucherRepo.GetAvailableByPackage(ctx, req.PackageID, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil voucher tersedia: %w", err)
	}
	if len(availableVouchers) < req.Quantity {
		return nil, fmt.Errorf("usecase: voucher tersedia tidak mencukupi (diminta: %d, tersedia: %d)", req.Quantity, len(availableVouchers))
	}

	// Hitung tanggal kedaluwarsa voucher (default 90 hari dari sekarang)
	now := time.Now()
	expiresAt := now.AddDate(0, 0, DefaultVoucherExpiryDays)

	// Assign setiap voucher ke reseller dan tulis audit log
	assignedVouchers := make([]*domain.Voucher, 0, req.Quantity)
	for _, v := range availableVouchers[:req.Quantity] {
		// Assign voucher ke reseller dengan snapshot harga
		assigned, err := txVoucherRepo.AssignToReseller(ctx, v.ID, resellerID, sellPrice, resellerPrice, expiresAt)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal assign voucher %s ke reseller: %w", v.ID, err)
		}
		assignedVouchers = append(assignedVouchers, assigned)

		// Tulis voucher audit log (voucher.sold, actor=reseller)
		if err := txVoucherAuditRepo.Create(ctx, &domain.VoucherAuditLog{
			TenantID:  reseller.TenantID,
			VoucherID: v.ID,
			Action:    "voucher.sold",
			ActorID:   resellerID,
			ActorName: reseller.Name,
			Metadata: map[string]interface{}{
				"package_id":              req.PackageID,
				"sell_price_snapshot":     sellPrice,
				"reseller_price_snapshot": resellerPrice,
			},
		}); err != nil {
			return nil, fmt.Errorf("usecase: gagal menulis voucher audit log untuk voucher %s: %w", v.ID, err)
		}
	}

	// Hitung saldo baru dan update balance reseller
	balanceBefore := reseller.Balance
	balanceAfter := balanceBefore - totalCost

	if err := txResellerRepo.UpdateBalance(ctx, resellerID, balanceAfter); err != nil {
		return nil, fmt.Errorf("usecase: gagal update saldo reseller setelah pembelian: %w", err)
	}

	// Buat catatan transaksi reseller (type=purchase)
	_, err = txTxRepo.Create(ctx, &domain.ResellerTransaction{
		TenantID:      reseller.TenantID,
		ResellerID:    resellerID,
		Type:          domain.TransactionPurchase,
		Amount:        totalCost,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Notes:         fmt.Sprintf("Pembelian %d voucher paket %s", req.Quantity, pkg.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat transaksi pembelian: %w", err)
	}

	// Commit transaksi database
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("usecase: gagal commit transaksi pembelian voucher: %w", err)
	}

	// Publish event voucher.purchased (di luar transaksi, tidak boleh gagalkan operasi utama)
	uc.publishEvent(reseller.TenantID, "voucher.purchased", domain.VoucherPurchasedPayload{
		TenantID:   reseller.TenantID,
		ResellerID: resellerID,
		PackageID:  req.PackageID,
		Quantity:   req.Quantity,
		TotalCost:  totalCost,
	})

	return &domain.BuyVoucherResult{
		Vouchers:     assignedVouchers,
		TotalCost:    totalCost,
		BalanceAfter: balanceAfter,
	}, nil
}

// --- Helper methods ---

// publishEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *VoucherPurchaseUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
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
