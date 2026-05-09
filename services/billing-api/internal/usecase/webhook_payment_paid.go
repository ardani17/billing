// webhook_payment_paid.go berisi method processPaymentPaid pada WebhookUsecase.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// processPaymentPaid memproses event payment.paid dari webhook.
func (uc *WebhookUsecase) processPaymentPaid(ctx context.Context, event *domain.WebhookEvent, link *domain.PaymentLink) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Ambil invoice IDs yang terkait dengan link pembayaran
	invoiceIDs, err := uc.linkRepo.GetInvoiceIDsByLinkID(ctx, link.ID)
	if err != nil {
		return fmt.Errorf("gagal mengambil invoice IDs: %w", err)
	}
	if len(invoiceIDs) == 0 {
		return fmt.Errorf("tidak ada invoice terkait payment link %s", link.ID)
	}

	// Ambil invoices dengan SELECT FOR UPDATE untuk mencegah race condition
	invoices, err := uc.invoiceRepo.GetByIDsForUpdate(ctx, invoiceIDs)
	if err != nil {
		return fmt.Errorf("gagal mengambil invoices FOR UPDATE: %w", err)
	}

	// Cek apakah semua invoice sudah lunas (double payment)
	allLunas := true
	for _, inv := range invoices {
		if inv.Status != domain.InvoiceStatusLunas {
			allLunas = false
			break
		}
	}

	if allLunas {
		return uc.handleDoublePayment(ctx, event, link, invoices)
	}

	// Siapkan input FIFO dari invoices (sudah terurut due_date ASC dari kueri)
	fifoInputs := make([]domain.FIFOInput, 0, len(invoices))
	for _, inv := range invoices {
		fifoInputs = append(fifoInputs, domain.FIFOInput{
			InvoiceID:     inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			TotalAmount:   inv.TotalAmount,
			PaidAmount:    inv.PaidAmount,
			Status:        inv.Status,
		})
	}

	// Alokasi pembayaran FIFO
	result := domain.AllocatePaymentFIFO(fifoInputs, event.Amount)

	// Buat receipt_group_id dan nomor kwitansi
	receiptGroupID := uuid.New().String()
	now := time.Now()
	seq, err := uc.receiptSeqRepo.NextSequence(ctx, link.TenantID, now.Year(), int(now.Month()))
	if err != nil {
		return fmt.Errorf("gagal generate nomor kwitansi: %w", err)
	}
	receiptNumber := domain.FormatReceiptNumber(now.Year(), int(now.Month()), seq)

	// Proses setiap alokasi
	for _, alloc := range result.Allocations {
		if err := uc.processAllocation(ctx, alloc, link, event, receiptNumber, receiptGroupID, invoices); err != nil {
			return err
		}
	}

	// Tangani kelebihan bayar -> tambah ke credit_balance customer
	if result.ExcessToCredit > 0 {
		if err := uc.adjustCreditBalance(ctx, link.CustomerID, result.ExcessToCredit); err != nil {
			return fmt.Errorf("gagal menambah kredit pelanggan: %w", err)
		}
	}

	// Perbarui link pembayaran status menjadi paid
	if err := uc.linkRepo.UpdateStatusPaid(ctx, link.ID, event.PaidMethod, now); err != nil {
		return fmt.Errorf("gagal update status payment link: %w", err)
	}

	// COMMIT transaksi
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("gagal commit transaksi: %w", err)
	}

	// Terbitkan event setelah commit (non-blocking)
	uc.publishWebhookEvent(link.TenantID, "payment.online.received", map[string]interface{}{
		"tenant_id":      link.TenantID,
		"customer_id":    link.CustomerID,
		"receipt_number": receiptNumber,
		"total_amount":   event.Amount,
		"payment_method": event.PaidMethod,
		"invoice_count":  len(result.Allocations),
	})
	uc.publishWebhookEvent(link.TenantID, "payment.online.confirmation", map[string]interface{}{
		"tenant_id":      link.TenantID,
		"customer_id":    link.CustomerID,
		"receipt_number": receiptNumber,
		"total_amount":   event.Amount,
		"payment_method": event.PaidMethod,
	})

	uc.logger.Info().
		Str("payment_link_id", link.ID).
		Int64("amount", event.Amount).
		Int("invoice_count", len(result.Allocations)).
		Msg("pembayaran online berhasil diproses")

	return nil
}

// processAllocation memproses satu alokasi pembayaran: insert payment, perbarui invoice, audit log.
func (uc *WebhookUsecase) processAllocation(
	ctx context.Context,
	alloc domain.PaymentAllocation,
	link *domain.PaymentLink,
	event *domain.WebhookEvent,
	receiptNumber, receiptGroupID string,
	invoices []*domain.Invoice,
) error {
	// Insert invoice_payment
	payment := &domain.InvoicePayment{
		TenantID:       link.TenantID,
		InvoiceID:      alloc.InvoiceID,
		Amount:         alloc.AllocatedAmt,
		PaymentMethod:  event.PaidMethod,
		PaymentDate:    time.Now(),
		RecordedByID:   "system",
		RecordedByName: "Payment Gateway",
		ReceiptNumber:  receiptNumber,
		ReceiptGroupID: receiptGroupID,
	}
	if _, err := uc.paymentRepo.Create(ctx, payment); err != nil {
		return fmt.Errorf("gagal mencatat pembayaran: %w", err)
	}

	// Cari invoice asli untuk optimistic locking version
	var version int
	for _, inv := range invoices {
		if inv.ID == alloc.InvoiceID {
			version = inv.Version
			break
		}
	}

	// Perbarui paid_amount dengan optimistic locking
	updated, err := uc.invoiceRepo.UpdatePaidAmount(ctx, alloc.InvoiceID, alloc.NewPaidAmount, version)
	if err != nil {
		return fmt.Errorf("%w: gagal update paid_amount invoice", domain.ErrConcurrentModification)
	}

	// Perbarui status jika berubah
	for _, inv := range invoices {
		if inv.ID == alloc.InvoiceID && alloc.NewStatus != inv.Status {
			if _, err := uc.invoiceRepo.UpdateStatus(ctx, alloc.InvoiceID, alloc.NewStatus, updated.Version); err != nil {
				return fmt.Errorf("gagal update status invoice: %w", err)
			}
			break
		}
	}

	// Tulis audit log per invoice
	uc.writeWebhookAuditLog(ctx, link.TenantID, alloc.InvoiceID, "invoice.payment_online", map[string]interface{}{
		"amount":           alloc.AllocatedAmt,
		"payment_method":   event.PaidMethod,
		"new_paid_amount":  alloc.NewPaidAmount,
		"new_status":       string(alloc.NewStatus),
		"receipt_number":   receiptNumber,
		"gateway_provider": string(link.GatewayProvider),
		"transaction_id":   event.TransactionID,
	})
	return nil
}

// handleDoublePayment menangani kasus double payment (semua invoice sudah lunas).
func (uc *WebhookUsecase) handleDoublePayment(
	ctx context.Context,
	event *domain.WebhookEvent,
	link *domain.PaymentLink,
	invoices []*domain.Invoice,
) error {
	// Tambah seluruh nominal ke credit_balance customer
	if err := uc.adjustCreditBalance(ctx, link.CustomerID, event.Amount); err != nil {
		return fmt.Errorf("gagal menambah kredit double payment: %w", err)
	}

	// Perbarui link pembayaran status menjadi paid
	now := time.Now()
	if err := uc.linkRepo.UpdateStatusPaid(ctx, link.ID, event.PaidMethod, now); err != nil {
		return fmt.Errorf("gagal update status payment link: %w", err)
	}

	// Tulis audit log untuk setiap invoice
	for _, inv := range invoices {
		uc.writeWebhookAuditLog(ctx, link.TenantID, inv.ID, "invoice.double_payment_detected", map[string]interface{}{
			"amount":           event.Amount,
			"payment_method":   event.PaidMethod,
			"gateway_provider": string(link.GatewayProvider),
			"reason":           "invoice sudah lunas, amount ditambahkan ke credit_balance",
		})
	}

	// Terbitkan event double payment
	uc.publishWebhookEvent(link.TenantID, "payment.double_payment", map[string]interface{}{
		"tenant_id":        link.TenantID,
		"customer_id":      link.CustomerID,
		"payment_link_id":  link.ID,
		"amount":           event.Amount,
		"gateway_provider": string(link.GatewayProvider),
	})

	uc.logger.Warn().Str("payment_link_id", link.ID).Int64("amount", event.Amount).
		Msg("double payment terdeteksi, amount ditambahkan ke credit_balance")

	return nil
}
