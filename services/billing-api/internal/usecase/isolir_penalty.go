// isolir_penalty.go berisi business logic untuk denda keterlambatan dan penghapusan denda.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ProcessLateFee menghitung dan menambahkan denda keterlambatan ke invoice.
// Dipanggil saat invoice transisi ke terlambat oleh terlambat cron.
func (uc *IsolirUsecase) ProcessLateFee(ctx context.Context, tenantID string,
	invoice *domain.Invoice, settings *domain.BillingSettings, daysOverdue int) error {
	// Cek apakah fitur denda aktif
	if !settings.PenaltyEnabled {
		return nil
	}

	// Hitung denda menggunakan fungsi domain murni
	fee := domain.CalculateLateFee(settings, invoice.Subtotal, daysOverdue)
	if fee == 0 {
		return nil
	}

	// Tambahkan item denda ke invoice
	penaltyItem := &domain.InvoiceItem{
		TenantID:    tenantID,
		InvoiceID:   invoice.ID,
		ItemType:    domain.ItemTypePenalty,
		Description: "Denda keterlambatan",
		Quantity:    1,
		UnitPrice:   fee,
		Amount:      fee,
		SortOrder:   999, // denda selalu di akhir
	}
	if _, err := uc.invoiceItemRepo.BulkCreate(ctx, []*domain.InvoiceItem{penaltyItem}); err != nil {
		return fmt.Errorf("gagal menambahkan item denda: %w", err)
	}

	// Perbarui penalty_amount dan total_amount invoice
	invoice.PenaltyAmount = fee
	invoice.TotalAmount += fee
	if _, err := uc.invoiceRepo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("gagal memperbarui total invoice: %w", err)
	}

	// Terbitkan event invoice.penalty_added untuk sinkronisasi link pembayaran
	uc.publishEvent(tenantID, domain.TaskInvoicePenaltyAdded, domain.PenaltyAddedPayload{
		InvoiceID:     invoice.ID,
		TenantID:      tenantID,
		CustomerID:    invoice.CustomerID,
		PenaltyAmount: fee,
		PenaltyType:   string(settings.PenaltyType),
		InvoiceNumber: invoice.InvoiceNumber,
	})

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, invoice.ID, "invoice.penalty_added",
		map[string]interface{}{
			"fee_amount":         fee,
			"calculation_method": string(settings.PenaltyType),
			"days_overdue":       daysOverdue,
		})
	return nil
}

// WaivePenalty menghapus denda dari invoice dan menghitung ulang total.
// Dipanggil oleh admin melalui endpoint POST /v1/invoices/:id/waive-denda.
func (uc *IsolirUsecase) WaivePenalty(ctx context.Context, invoiceID, actorID, actorName string) error {
	// Ambil invoice berdasarkan ID
	invoice, err := uc.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("%w", domain.ErrInvoiceNotFound)
	}

	// Validasi status invoice - tidak bisa edit jika sudah lunas atau batal
	if invoice.Status == domain.InvoiceStatusLunas || invoice.Status == domain.InvoiceStatusBatal {
		return domain.ErrInvoiceNotEditable
	}

	// Ambil semua item invoice
	items, err := uc.invoiceItemRepo.ListByInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("gagal mengambil item invoice: %w", err)
	}

	// Filter item denda dan non-denda
	var nonPenaltyItems []*domain.InvoiceItem
	hasPenalty := false
	for _, item := range items {
		if item.ItemType == domain.ItemTypePenalty {
			hasPenalty = true
		} else {
			nonPenaltyItems = append(nonPenaltyItems, item)
		}
	}
	if !hasPenalty {
		return domain.ErrNoPenaltyToWaive
	}

	// Hapus semua item lama dan buat ulang tanpa item denda
	if err := uc.invoiceItemRepo.DeleteByInvoice(ctx, invoiceID); err != nil {
		return fmt.Errorf("gagal menghapus item invoice: %w", err)
	}
	if len(nonPenaltyItems) > 0 {
		if _, err := uc.invoiceItemRepo.BulkCreate(ctx, nonPenaltyItems); err != nil {
			return fmt.Errorf("gagal membuat ulang item invoice: %w", err)
		}
	}

	// Hitung ulang total: atur penalty_amount ke 0, recalculate total_amount
	invoice.TotalAmount -= invoice.PenaltyAmount
	invoice.PenaltyAmount = 0
	if _, err := uc.invoiceRepo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("gagal memperbarui total invoice: %w", err)
	}

	// Terbitkan event invoice.penalty_added (nominal 0) untuk sinkronisasi link pembayaran
	uc.publishEvent(invoice.TenantID, domain.TaskInvoicePenaltyAdded, domain.PenaltyAddedPayload{
		InvoiceID:     invoice.ID,
		TenantID:      invoice.TenantID,
		CustomerID:    invoice.CustomerID,
		PenaltyAmount: 0,
		PenaltyType:   "waived",
		InvoiceNumber: invoice.InvoiceNumber,
	})

	// Tulis audit log dengan aktor admin
	uc.writeAuditLog(ctx, invoice.TenantID, invoice.ID, "invoice.penalty_waived",
		map[string]interface{}{
			"actor_id":   actorID,
			"actor_name": actorName,
		})
	return nil
}
