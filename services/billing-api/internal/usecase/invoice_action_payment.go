// invoice_action_payment.go berisi business logic untuk pencatatan pembayaran invoice.
package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// RecordPayment mencatat pembayaran terhadap invoice.
// Alur: ambil invoice -> verifikasi status -> hitung denda jika terlambat ->
// buat catatan pembayaran -> perbarui paid_amount -> tentukan status baru ->
// tangani kelebihan bayar -> tulis audit log.
func (uc *InvoiceActionUsecase) RecordPayment(ctx context.Context, invoiceID string, req domain.RecordPaymentRequest, actor domain.ActorInfo) (*domain.Invoice, error) {
	// Ambil invoice
	invoice, err := uc.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	// Invoice batal tidak boleh menerima pembayaran baru.
	if invoice.Status == domain.InvoiceStatusBatal {
		return nil, domain.ErrInvoiceNotCancellable
	}

	// Jika invoice sudah lunas tetapi pembayaran masuk lagi, simpan transaksi
	// sebagai jejak kas dan masukkan seluruh nominal ke kredit pelanggan.
	if invoice.Status == domain.InvoiceStatusLunas {
		paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
		if err != nil {
			return nil, fmt.Errorf("format payment_date tidak valid: %w", err)
		}
		payment := &domain.InvoicePayment{
			TenantID:        invoice.TenantID,
			InvoiceID:       invoiceID,
			Amount:          req.Amount,
			PaymentMethod:   req.PaymentMethod,
			PaymentDate:     paymentDate,
			ReferenceNumber: req.ReferenceNumber,
			Notes:           req.Notes,
			RecordedByID:    actor.ActorID,
			RecordedByName:  actor.ActorName,
		}
		if _, err := uc.paymentRepo.Create(ctx, payment); err != nil {
			return nil, fmt.Errorf("gagal mencatat pembayaran dobel: %w", err)
		}
		if err := uc.adjustCreditBalance(ctx, invoice.CustomerID, req.Amount); err != nil {
			uc.logger.Error().Err(err).
				Str("customer_id", invoice.CustomerID).
				Int64("amount", req.Amount).
				Msg("gagal menambah kredit dari pembayaran dobel")
		}
		uc.writeAuditLog(ctx, invoice.TenantID, invoiceID, "invoice.double_payment_credited", actor, map[string]interface{}{
			"amount":         req.Amount,
			"payment_method": req.PaymentMethod,
			"credit_amount":  req.Amount,
			"reason":         "invoice sudah lunas",
		})
		return invoice, nil
	}

	// Jika invoice terlambat dan denda diaktifkan, hitung denda
	if invoice.Status == domain.InvoiceStatusTerlambat {
		settings, _ := uc.settingsRepo.GetByTenantID(ctx, invoice.TenantID)
		if settings != nil && settings.PenaltyEnabled && invoice.PenaltyAmount == 0 {
			// Hitung hari keterlambatan
			paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
			if err != nil {
				return nil, fmt.Errorf("format payment_date tidak valid: %w", err)
			}
			daysOverdue := int(math.Ceil(paymentDate.Sub(invoice.DueDate).Hours() / 24))
			if daysOverdue < 1 {
				daysOverdue = 1
			}

			lateFee := domain.CalculateLateFee(settings, invoice.Subtotal, daysOverdue)
			if lateFee > 0 {
				// Tambahkan item denda ke invoice
				penaltyItem := &domain.InvoiceItem{
					TenantID:    invoice.TenantID,
					InvoiceID:   invoiceID,
					ItemType:    domain.ItemTypePenalty,
					Description: "Denda keterlambatan",
					Quantity:    1,
					UnitPrice:   lateFee,
					Amount:      lateFee,
					SortOrder:   999, // denda selalu di akhir
				}
				if _, err := uc.itemRepo.BulkCreate(ctx, []*domain.InvoiceItem{penaltyItem}); err != nil {
					return nil, fmt.Errorf("gagal menambahkan item denda: %w", err)
				}

				// Perbarui total invoice dengan denda
				invoice.PenaltyAmount = lateFee
				invoice.TotalAmount += lateFee
				if _, err := uc.invoiceRepo.Update(ctx, invoice); err != nil {
					return nil, fmt.Errorf("gagal memperbarui total invoice: %w", err)
				}
			}
		}
	}

	// Parsing tanggal pembayaran
	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		return nil, fmt.Errorf("format payment_date tidak valid: %w", err)
	}

	// Buat catatan pembayaran
	payment := &domain.InvoicePayment{
		TenantID:        invoice.TenantID,
		InvoiceID:       invoiceID,
		Amount:          req.Amount,
		PaymentMethod:   req.PaymentMethod,
		PaymentDate:     paymentDate,
		ReferenceNumber: req.ReferenceNumber,
		Notes:           req.Notes,
		RecordedByID:    actor.ActorID,
		RecordedByName:  actor.ActorName,
	}
	if _, err := uc.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("gagal mencatat pembayaran: %w", err)
	}

	// Hitung paid_amount baru
	newPaidAmount := invoice.PaidAmount + req.Amount

	// Tentukan status baru berdasarkan jumlah pembayaran
	var newStatus domain.InvoiceStatus
	var excessAmount int64

	if newPaidAmount >= invoice.TotalAmount {
		// Lunas - kelebihan bayar menjadi kredit pelanggan
		newStatus = domain.InvoiceStatusLunas
		excessAmount = newPaidAmount - invoice.TotalAmount
		newPaidAmount = invoice.TotalAmount // cap paid_amount pada total_amount
	} else if newPaidAmount > 0 {
		// Bayar sebagian
		newStatus = domain.InvoiceStatusBayarSebagian
	} else {
		newStatus = invoice.Status
	}

	// Perbarui paid_amount dengan optimistic locking
	updated, err := uc.invoiceRepo.UpdatePaidAmount(ctx, invoiceID, newPaidAmount, invoice.Version)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui jumlah pembayaran: %w", err)
	}

	// Perbarui status jika berubah
	if newStatus != invoice.Status {
		updated, err = uc.invoiceRepo.UpdateStatus(ctx, invoiceID, newStatus, updated.Version)
		if err != nil {
			return nil, fmt.Errorf("gagal memperbarui status invoice: %w", err)
		}
	}

	// Jika ada kelebihan bayar, tambahkan ke credit_balance pelanggan
	if excessAmount > 0 {
		if err := uc.adjustCreditBalance(ctx, invoice.CustomerID, excessAmount); err != nil {
			uc.logger.Error().Err(err).
				Str("customer_id", invoice.CustomerID).
				Int64("excess_amount", excessAmount).
				Msg("gagal menambah kredit pelanggan")
		}
	}

	// Tulis audit log
	metadata := map[string]interface{}{
		"amount":          req.Amount,
		"payment_method":  req.PaymentMethod,
		"new_paid_amount": newPaidAmount,
		"new_status":      string(newStatus),
	}
	if excessAmount > 0 {
		metadata["excess_to_credit"] = excessAmount
	}
	uc.writeAuditLog(ctx, invoice.TenantID, invoiceID, "invoice.payment_recorded", actor, metadata)

	return updated, nil
}
