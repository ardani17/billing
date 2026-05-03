// payment_multi.go berisi business logic untuk pembayaran multi-invoice dan pay-all.
package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// RecordMultiPayment mencatat pembayaran multi-invoice dengan alokasi FIFO.
func (uc *PaymentUsecase) RecordMultiPayment(ctx context.Context, req domain.MultiPaymentRequest, actor domain.ActorInfo) (*domain.MultiPaymentResponse, error) {
	tenantID := tenant.FromContext(ctx)

	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		return nil, fmt.Errorf("format payment_date tidak valid: %w", err)
	}

	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Ambil invoices dengan FOR UPDATE
	var invoices []*domain.Invoice
	if len(req.InvoiceIDs) > 0 {
		invoices, err = uc.invoiceRepo.GetByIDsForUpdate(ctx, req.InvoiceIDs)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil invoices: %w", err)
		}
		// Validasi semua invoice milik customer dan berstatus terbuka
		for _, inv := range invoices {
			if inv.CustomerID != req.CustomerID {
				return nil, domain.ErrInvalidInvoiceSelection
			}
			if inv.Status == domain.InvoiceStatusLunas || inv.Status == domain.InvoiceStatusBatal {
				return nil, domain.ErrInvalidInvoiceSelection
			}
		}
	} else {
		invoices, err = uc.invoiceRepo.FindOpenByCustomerForUpdate(ctx, req.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil invoice terbuka: %w", err)
		}
	}

	if len(invoices) == 0 {
		return nil, domain.ErrNoOpenInvoices
	}

	// Cek denda untuk invoice terlambat
	settings, _ := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	for _, inv := range invoices {
		if inv.Status != domain.InvoiceStatusTerlambat || settings == nil || !settings.PenaltyEnabled || inv.PenaltyAmount != 0 {
			continue
		}
		daysOverdue := int(math.Ceil(paymentDate.Sub(inv.DueDate).Hours() / 24))
		if daysOverdue < 1 {
			daysOverdue = 1
		}
		lateFee := domain.CalculateLateFee(settings, inv.Subtotal, daysOverdue)
		if lateFee > 0 {
			penaltyItem := &domain.InvoiceItem{
				TenantID: tenantID, InvoiceID: inv.ID, ItemType: domain.ItemTypePenalty,
				Description: "Denda keterlambatan", Quantity: 1, UnitPrice: lateFee, Amount: lateFee, SortOrder: 999,
			}
			if _, err := uc.itemRepo.BulkCreate(ctx, []*domain.InvoiceItem{penaltyItem}); err != nil {
				return nil, fmt.Errorf("gagal menambahkan item denda: %w", err)
			}
			inv.PenaltyAmount = lateFee
			inv.TotalAmount += lateFee
			if _, err := uc.invoiceRepo.Update(ctx, inv); err != nil {
				return nil, fmt.Errorf("gagal memperbarui total invoice: %w", err)
			}
		}
	}

	// Siapkan input FIFO dari invoices (sudah terurut due_date ASC dari query)
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
	result := domain.AllocatePaymentFIFO(fifoInputs, req.Amount)

	// Generate receipt_group_id dan nomor kwitansi
	receiptGroupID := uuid.New().String()
	now := time.Now()
	seq, err := uc.receiptSeqRepo.NextSequence(ctx, tenantID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, fmt.Errorf("gagal generate nomor kwitansi: %w", err)
	}
	receiptNumber := domain.FormatReceiptNumber(now.Year(), int(now.Month()), seq)

	// Proses setiap alokasi
	for _, alloc := range result.Allocations {
		payment := &domain.InvoicePayment{
			TenantID: tenantID, InvoiceID: alloc.InvoiceID, Amount: alloc.AllocatedAmt,
			PaymentMethod: req.PaymentMethod, PaymentDate: paymentDate,
			ReferenceNumber: req.ReferenceNumber, Notes: req.Notes,
			RecordedByID: actor.ActorID, RecordedByName: actor.ActorName,
			ReceiptNumber: receiptNumber, ReceiptGroupID: receiptGroupID,
		}
		if _, err := uc.paymentRepo.Create(ctx, payment); err != nil {
			return nil, fmt.Errorf("gagal mencatat pembayaran: %w", err)
		}

		// Cari invoice asli untuk optimistic locking version
		var version int
		for _, inv := range invoices {
			if inv.ID == alloc.InvoiceID {
				version = inv.Version
				break
			}
		}

		// Update paid_amount
		updated, err := uc.invoiceRepo.UpdatePaidAmount(ctx, alloc.InvoiceID, alloc.NewPaidAmount, version)
		if err != nil {
			return nil, fmt.Errorf("%w: gagal update paid_amount invoice", domain.ErrConcurrentModification)
		}

		// Update status jika berubah
		for _, inv := range invoices {
			if inv.ID == alloc.InvoiceID && alloc.NewStatus != inv.Status {
				if _, err := uc.invoiceRepo.UpdateStatus(ctx, alloc.InvoiceID, alloc.NewStatus, updated.Version); err != nil {
					return nil, fmt.Errorf("gagal update status invoice: %w", err)
				}
				break
			}
		}

		// Tulis audit log per invoice
		metadata := map[string]interface{}{
			"amount":          alloc.AllocatedAmt,
			"payment_method":  req.PaymentMethod,
			"new_paid_amount": alloc.NewPaidAmount,
			"new_status":      string(alloc.NewStatus),
			"receipt_number":  receiptNumber,
		}
		uc.writePaymentAuditLog(ctx, tenantID, alloc.InvoiceID, "invoice.payment_recorded", actor, metadata)
	}

	// Handle kelebihan bayar → tambah ke credit_balance
	if result.ExcessToCredit > 0 {
		if err := uc.adjustPaymentCreditBalance(ctx, req.CustomerID, result.ExcessToCredit); err != nil {
			return nil, fmt.Errorf("gagal menambah kredit pelanggan: %w", err)
		}
	}

	// COMMIT transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	// Publish event (non-blocking, setelah commit)
	uc.publishPaymentEvent(tenantID, "payment.recorded", domain.PaymentRecordedPayload{
		TenantID:      tenantID,
		CustomerID:    req.CustomerID,
		ReceiptNumber: receiptNumber,
		TotalAmount:   req.Amount,
		PaymentMethod: req.PaymentMethod,
		InvoiceCount:  len(result.Allocations),
	})

	// Publish event sinkronisasi payment link untuk setiap invoice yang menerima alokasi.
	// Memicu SyncPaymentLinkAmount di gateway worker agar payment link yang aktif
	// di-expire dan di-regenerate dengan jumlah terbaru.
	for _, alloc := range result.Allocations {
		uc.publishSyncPaymentLinkEvent(tenantID, alloc.InvoiceID, req.CustomerID)
	}

	return &domain.MultiPaymentResponse{
		Allocations:    result.Allocations,
		TotalAllocated: result.TotalAllocated,
		ExcessToCredit: result.ExcessToCredit,
		ReceiptNumber:  receiptNumber,
		ReceiptID:      receiptGroupID,
	}, nil
}

// PayAll membayar semua invoice terbuka untuk customer.
// Menghitung total tunggakan dan mendelegasikan ke RecordMultiPayment.
func (uc *PaymentUsecase) PayAll(ctx context.Context, req domain.PayAllRequest, actor domain.ActorInfo) (*domain.MultiPaymentResponse, error) {
	// Ambil invoice terbuka untuk menghitung total tunggakan
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice terbuka: %w", err)
	}
	if len(invoices) == 0 {
		return nil, domain.ErrNoOpenInvoices
	}

	// Hitung total tunggakan
	var totalArrears int64
	for _, inv := range invoices {
		totalArrears += inv.TotalAmount - inv.PaidAmount
	}

	// Delegasikan ke RecordMultiPayment dengan amount = total tunggakan
	multiReq := domain.MultiPaymentRequest{
		CustomerID:      req.CustomerID,
		Amount:          totalArrears,
		PaymentMethod:   req.PaymentMethod,
		PaymentDate:     req.PaymentDate,
		ReferenceNumber: req.ReferenceNumber,
		Notes:           req.Notes,
	}
	return uc.RecordMultiPayment(ctx, multiReq, actor)
}
