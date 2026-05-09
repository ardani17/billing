// voucher_action.go berisi business logic untuk aksi voucher (admin).
// Mengimplementasikan BulkVoid, BulkAssign, ExportCSV pada VoucherActionUsecase.
package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// VoucherActionUsecase mengimplementasikan business logic untuk aksi voucher (admin).
type VoucherActionUsecase struct {
	voucherRepo     domain.VoucherRepository
	voucherAuditLog domain.VoucherAuditLogRepository
	resellerRepo    domain.ResellerRepository
	logger          zerolog.Logger
}

// NewVoucherActionUsecase membuat instance baru VoucherActionUsecase.
func NewVoucherActionUsecase(
	voucherRepo domain.VoucherRepository,
	voucherAuditLog domain.VoucherAuditLogRepository,
	resellerRepo domain.ResellerRepository,
	logger zerolog.Logger,
) *VoucherActionUsecase {
	return &VoucherActionUsecase{
		voucherRepo:     voucherRepo,
		voucherAuditLog: voucherAuditLog,
		resellerRepo:    resellerRepo,
		logger:          logger,
	}
}

// BulkVoid mem-void beberapa voucher sekaligus.
// Alur: ambil voucher by IDs -> untuk setiap voucher, cek status == tersedia ->
// transisi ke void -> tulis voucher audit log (voucher.voided) ->
// kembalikan BulkActionResult dengan jumlah sukses/gagal dan detail kegagalan.
func (uc *VoucherActionUsecase) BulkVoid(ctx context.Context, ids []string, actor domain.ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	// Ambil semua voucher berdasarkan IDs
	vouchers, err := uc.voucherRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil voucher by IDs: %w", err)
	}

	// Buat map untuk lookup cepat
	voucherMap := make(map[string]*domain.Voucher, len(vouchers))
	for _, v := range vouchers {
		voucherMap[v.ID] = v
	}

	// Proses setiap voucher ID
	for _, id := range ids {
		v, exists := voucherMap[id]
		if !exists {
			// Voucher tidak ditemukan
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     domain.ErrVoucherNotFound.Error(),
			})
			continue
		}

		// Cek status harus tersedia
		if v.Status != domain.VoucherStatusTersedia {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     fmt.Sprintf("voucher status %s, hanya voucher dengan status tersedia yang bisa di-void", v.Status),
			})
			continue
		}

		// Transisi status ke void
		_, transErr := domain.VoucherTransition(v.Status, domain.VoucherStatusVoid)
		if transErr != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     transErr.Error(),
			})
			continue
		}

		// Perbarui status di database
		_, updateErr := uc.voucherRepo.UpdateStatus(ctx, id, domain.VoucherStatusVoid)
		if updateErr != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     updateErr.Error(),
			})
			continue
		}

		// Tulis voucher audit log
		if err := uc.voucherAuditLog.Create(ctx, &domain.VoucherAuditLog{
			TenantID:  v.TenantID,
			VoucherID: id,
			Action:    "voucher.voided",
			ActorID:   actor.ActorID,
			ActorName: actor.ActorName,
			Metadata: map[string]interface{}{
				"reason": "bulk_void",
			},
		}); err != nil {
			uc.logger.Error().Err(err).Str("voucher_id", id).Msg("gagal menulis voucher audit log saat bulk void")
		}

		result.SuccessCount++
	}

	return result, nil
}

// BulkAssign meng-assign beberapa voucher ke reseller (admin assignment, tanpa potong saldo, tanpa snapshot).
// Alur: validasi reseller ada -> ambil voucher by IDs -> untuk setiap voucher, cek status == tersedia ->
// assign ke reseller -> tulis voucher audit log (voucher.assigned) ->
// kembalikan BulkActionResult dengan jumlah sukses/gagal dan detail kegagalan.
func (uc *VoucherActionUsecase) BulkAssign(ctx context.Context, ids []string, resellerID string, actor domain.ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	// Validasi reseller ada
	reseller, err := uc.resellerRepo.GetByID(ctx, resellerID)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil reseller: %w", err)
	}

	// Ambil semua voucher berdasarkan IDs
	vouchers, err := uc.voucherRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil voucher by IDs: %w", err)
	}

	// Buat map untuk lookup cepat
	voucherMap := make(map[string]*domain.Voucher, len(vouchers))
	for _, v := range vouchers {
		voucherMap[v.ID] = v
	}

	// Proses setiap voucher ID
	for _, id := range ids {
		v, exists := voucherMap[id]
		if !exists {
			// Voucher tidak ditemukan
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     domain.ErrVoucherNotFound.Error(),
			})
			continue
		}

		// Cek status harus tersedia
		if v.Status != domain.VoucherStatusTersedia {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     fmt.Sprintf("voucher status %s, hanya voucher dengan status tersedia yang bisa di-assign", v.Status),
			})
			continue
		}

		// Assign voucher ke reseller via repositori (admin assignment, tanpa snapshot)
		bulkResults, bulkErr := uc.voucherRepo.BulkAssign(ctx, []string{id}, resellerID)
		if bulkErr != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     bulkErr.Error(),
			})
			continue
		}

		// Cek hasil assign per item
		if len(bulkResults) > 0 && !bulkResults[0].Success {
			result.FailureCount++
			reason := "gagal assign voucher"
			if bulkResults[0].Error != nil {
				reason = bulkResults[0].Error.Error()
			}
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     reason,
			})
			continue
		}

		// Tulis voucher audit log
		if err := uc.voucherAuditLog.Create(ctx, &domain.VoucherAuditLog{
			TenantID:  v.TenantID,
			VoucherID: id,
			Action:    "voucher.assigned",
			ActorID:   actor.ActorID,
			ActorName: actor.ActorName,
			Metadata: map[string]interface{}{
				"reseller_id":   resellerID,
				"reseller_name": reseller.Name,
				"reason":        "bulk_assign",
			},
		}); err != nil {
			uc.logger.Error().Err(err).Str("voucher_id", id).Msg("gagal menulis voucher audit log saat bulk assign")
		}

		result.SuccessCount++
	}

	return result, nil
}

// ExportCSV mengekspor daftar voucher ke format CSV berdasarkan filter yang diberikan.
// Alur: ambil voucher dengan filter -> format sebagai CSV bytes -> kembalikan.
func (uc *VoucherActionUsecase) ExportCSV(ctx context.Context, params domain.VoucherListParams) ([]byte, error) {
	// Set page_size besar untuk mengambil semua data (export tidak menggunakan paginasi)
	params.Page = 1
	params.PageSize = 50

	var allVouchers []*domain.Voucher

	// Ambil semua voucher secara bertahap (batch)
	for {
		result, err := uc.voucherRepo.List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal mengambil voucher untuk export CSV: %w", err)
		}

		allVouchers = append(allVouchers, result.Data...)

		// Cek apakah masih ada halaman berikutnya
		if params.Page >= result.Pagination.TotalPages {
			break
		}
		params.Page++
	}

	// Format sebagai CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Tulis header CSV
	header := []string{
		"ID", "Kode", "Paket", "Reseller", "Status",
		"Harga Jual", "Harga Reseller",
		"Tanggal Beli", "Tanggal Aktivasi", "Tanggal Kedaluwarsa",
		"Tanggal Void", "Tanggal Dibuat",
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("usecase: gagal menulis header CSV: %w", err)
	}

	// Tulis data voucher
	for _, v := range allVouchers {
		row := []string{
			v.ID,
			v.Code,
			v.PackageName,
			v.ResellerName,
			string(v.Status),
			formatInt64Ptr(v.SellPriceSnapshot),
			formatInt64Ptr(v.ResellerPriceSnapshot),
			formatTimePtr(v.PurchasedAt),
			formatTimePtr(v.ActivatedAt),
			formatTimePtr(v.ExpiresAt),
			formatTimePtr(v.VoidedAt),
			v.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("usecase: gagal menulis baris CSV: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("usecase: gagal flush CSV writer: %w", err)
	}

	return buf.Bytes(), nil
}

// --- Fungsi bantu functions ---

// formatInt64Ptr memformat pointer int64 ke string, mengembalikan string kosong jika nil.
func formatInt64Ptr(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}

// formatTimePtr memformat pointer time.Time ke string, mengembalikan string kosong jika nil.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
