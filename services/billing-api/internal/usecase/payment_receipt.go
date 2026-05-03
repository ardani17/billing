// payment_receipt.go berisi business logic untuk pengambilan data kwitansi pembayaran.
package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetReceipt mengambil data kwitansi pembayaran untuk cetak thermal.
// Jika payment memiliki receipt_group_id, ambil semua payment dalam grup
// untuk kwitansi multi-invoice.
func (uc *PaymentUsecase) GetReceipt(ctx context.Context, paymentID string) (*domain.ReceiptData, error) {
	// Ambil payment utama
	payment, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, domain.ErrPaymentNotFound
	}

	// Ambil invoice untuk data customer
	invoice, err := uc.invoiceRepo.GetByID(ctx, payment.InvoiceID)
	if err != nil {
		return nil, err
	}

	// Ambil customer untuk nama dan ID
	customer, err := uc.customerRepo.GetByID(ctx, invoice.CustomerID)
	if err != nil {
		return nil, err
	}

	// Ambil nama tenant dari billing settings
	tenantName := ""
	settings, _ := uc.settingsRepo.GetByTenantID(ctx, invoice.TenantID)
	if settings != nil {
		tenantName = settings.InvoicePrefix // Gunakan prefix sebagai fallback
	}

	// Kumpulkan daftar invoice dalam kwitansi
	var invoices []domain.ReceiptInvoice
	var totalAmount int64

	if payment.ReceiptGroupID != "" {
		// Multi-invoice: ambil semua payment dengan receipt_group_id yang sama
		groupPayments, err := uc.paymentRepo.ListByInvoice(ctx, payment.InvoiceID)
		if err == nil {
			// Cari semua payment dengan receipt_group_id yang sama
			// Karena ListByInvoice hanya per invoice, kita perlu pendekatan berbeda
			// Gunakan payment yang sudah ada dan tambahkan invoice-nya
			seen := make(map[string]bool)
			// Tambahkan payment utama dulu
			invoices = append(invoices, domain.ReceiptInvoice{
				InvoiceNumber: invoice.InvoiceNumber,
				Amount:        payment.Amount,
			})
			totalAmount += payment.Amount
			seen[payment.InvoiceID] = true

			// Cek payment lain dalam grup yang sama dari invoice yang sama
			for _, gp := range groupPayments {
				if gp.ID != payment.ID && gp.ReceiptGroupID == payment.ReceiptGroupID && !seen[gp.InvoiceID] {
					inv, err := uc.invoiceRepo.GetByID(ctx, gp.InvoiceID)
					if err == nil {
						invoices = append(invoices, domain.ReceiptInvoice{
							InvoiceNumber: inv.InvoiceNumber,
							Amount:        gp.Amount,
						})
						totalAmount += gp.Amount
						seen[gp.InvoiceID] = true
					}
				}
			}
		}
	}

	// Fallback: jika tidak ada receipt_group_id atau gagal ambil grup
	if len(invoices) == 0 {
		invoices = []domain.ReceiptInvoice{
			{
				InvoiceNumber: invoice.InvoiceNumber,
				Amount:        payment.Amount,
			},
		}
		totalAmount = payment.Amount
	}

	return &domain.ReceiptData{
		ReceiptNumber:  payment.ReceiptNumber,
		TenantName:     tenantName,
		PaymentDate:    payment.PaymentDate,
		CustomerName:   customer.Name,
		CustomerIDSeq:  customer.CustomerIDSeq,
		Invoices:       invoices,
		TotalAmount:    totalAmount,
		PaymentMethod:  payment.PaymentMethod,
		RecordedByName: payment.RecordedByName,
		Voided:         payment.Voided,
		VoidReason:     payment.VoidReason,
	}, nil
}
