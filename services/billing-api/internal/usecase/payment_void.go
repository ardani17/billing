// payment_void.go berisi business logic untuk void pembayaran dengan rollback.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// VoidPayment membatalkan pembayaran dan mengembalikan status invoice.
// Flow: BEGIN tx → ambil payment → validasi → void → rollback invoice →
// handle credit → audit log → COMMIT → publish re-isolir event jika perlu.
func (uc *PaymentUsecase) VoidPayment(ctx context.Context, paymentID string, req domain.VoidPaymentRequest, actor domain.ActorInfo) (*domain.VoidPaymentResponse, error) {
	// Mulai transaksi
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Ambil payment
	payment, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, domain.ErrPaymentNotFound
	}

	// Validasi: belum di-void
	if payment.Voided {
		return nil, domain.ErrPaymentAlreadyVoided
	}

	// Validasi: dalam batas waktu 24 jam
	now := time.Now()
	if now.Sub(payment.CreatedAt) > 24*time.Hour {
		return nil, domain.ErrVoidTimeLimitExceeded
	}

	// Void payment
	if err := uc.paymentRepo.VoidPayment(ctx, paymentID, actor.ActorID, req.Reason); err != nil {
		return nil, fmt.Errorf("gagal void pembayaran: %w", err)
	}

	// Ambil invoice untuk rollback
	invoice, err := uc.invoiceRepo.GetByID(ctx, payment.InvoiceID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice: %w", err)
	}

	// Hitung paid_amount baru setelah void
	newPaidAmount := invoice.PaidAmount - payment.Amount
	if newPaidAmount < 0 {
		newPaidAmount = 0
	}

	// Update paid_amount invoice
	updated, err := uc.invoiceRepo.UpdatePaidAmount(ctx, invoice.ID, newPaidAmount, invoice.Version)
	if err != nil {
		return nil, fmt.Errorf("gagal update paid_amount invoice: %w", err)
	}

	// Tentukan status baru setelah void
	newStatus := domain.DeterminePostVoidStatus(newPaidAmount, invoice.TotalAmount, invoice.DueDate, now)

	// Update status jika berubah
	if newStatus != updated.Status {
		if _, err := uc.invoiceRepo.UpdateStatus(ctx, invoice.ID, newStatus, updated.Version); err != nil {
			return nil, fmt.Errorf("gagal update status invoice: %w", err)
		}
	}

	// Jika payment sebelumnya menghasilkan excess ke credit, kurangi credit
	var creditReduced int64
	if invoice.PaidAmount >= invoice.TotalAmount && payment.Amount > 0 {
		// Hitung berapa excess yang mungkin ditambahkan dari payment ini
		excessFromPayment := invoice.PaidAmount - invoice.TotalAmount
		if excessFromPayment > payment.Amount {
			excessFromPayment = payment.Amount
		}
		if excessFromPayment > 0 {
			creditReduced = excessFromPayment
			// Kurangi credit balance, clamp ke 0 jika perlu
			if err := uc.adjustPaymentCreditBalance(ctx, invoice.CustomerID, -creditReduced); err != nil {
				uc.logger.Warn().Err(err).
					Str("customer_id", invoice.CustomerID).
					Int64("credit_reduced", creditReduced).
					Msg("gagal mengurangi credit balance saat void, mungkin sudah 0")
			}
		}
	}

	// Tulis audit log
	metadata := map[string]interface{}{
		"voided_amount":      payment.Amount,
		"new_paid_amount":    newPaidAmount,
		"new_status":         string(newStatus),
		"reason":             req.Reason,
		"credit_reduced":     creditReduced,
	}
	uc.writePaymentAuditLog(ctx, invoice.TenantID, invoice.ID, "invoice.payment_voided", actor, metadata)

	// COMMIT transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	// Publish re-isolir event jika invoice kembali ke terlambat
	if newStatus == domain.InvoiceStatusTerlambat {
		uc.publishPaymentEvent(invoice.TenantID, "payment.voided.re_isolir", domain.PaymentVoidedReIsolirPayload{
			TenantID:   invoice.TenantID,
			CustomerID: invoice.CustomerID,
			InvoiceID:  invoice.ID,
			Reason:     req.Reason,
		})
	}

	// Publish event sinkronisasi payment link setelah void.
	// Memicu SyncPaymentLinkAmount di gateway worker agar payment link yang aktif
	// di-expire dan di-regenerate dengan jumlah terbaru.
	uc.publishSyncPaymentLinkEvent(invoice.TenantID, invoice.ID, invoice.CustomerID)

	return &domain.VoidPaymentResponse{
		PaymentID:        paymentID,
		InvoiceID:        invoice.ID,
		VoidedAmount:     payment.Amount,
		NewPaidAmount:    newPaidAmount,
		NewInvoiceStatus: newStatus,
		CreditReduced:    creditReduced,
	}, nil
}
