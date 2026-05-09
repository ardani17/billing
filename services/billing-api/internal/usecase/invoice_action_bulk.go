// invoice_action_bulk.go berisi business logic untuk bulk aksi invoice
// (BulkReminder, BulkCancel, BulkPDF, ExportCSV).
package usecase

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// BulkReminder mengirim pengingat pembayaran untuk beberapa invoice sekaligus.
// Hanya invoice dengan status belum_bayar atau terlambat yang eligible.
func (uc *InvoiceActionUsecase) BulkReminder(ctx context.Context, req domain.BulkInvoiceIDsRequest, actor domain.ActorInfo) (*domain.InvoiceBulkActionResult, error) {
	// Ambil semua invoice berdasarkan IDs
	invoices, err := uc.invoiceRepo.GetByIDs(ctx, req.InvoiceIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice: %w", err)
	}

	// Buat map untuk lookup cepat
	invoiceMap := make(map[string]*domain.Invoice, len(invoices))
	for _, inv := range invoices {
		invoiceMap[inv.ID] = inv
	}

	result := &domain.InvoiceBulkActionResult{
		Total: len(req.InvoiceIDs),
	}

	for _, id := range req.InvoiceIDs {
		inv, ok := invoiceMap[id]
		if !ok {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.InvoiceBulkFailure{
				InvoiceID: id,
				Reason:    "invoice tidak ditemukan",
			})
			continue
		}

		// Hanya kirim reminder untuk status belum_bayar atau terlambat
		if inv.Status != domain.InvoiceStatusBelumBayar && inv.Status != domain.InvoiceStatusTerlambat {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.InvoiceBulkFailure{
				InvoiceID: id,
				Reason:    fmt.Sprintf("status %s tidak eligible untuk reminder", inv.Status),
			})
			continue
		}

		// Terbitkan event reminder
		uc.publishEvent(inv.TenantID, "invoice.reminder", domain.InvoiceReminderPayload{
			InvoiceID:     inv.ID,
			TenantID:      inv.TenantID,
			CustomerID:    inv.CustomerID,
			InvoiceNumber: inv.InvoiceNumber,
			TotalAmount:   inv.TotalAmount,
			DueDate:       inv.DueDate.Format("2006-01-02"),
		})

		// Tulis audit log
		uc.writeAuditLog(ctx, inv.TenantID, inv.ID, "invoice.reminder_sent", actor, nil)

		result.SuccessCount++
	}

	return result, nil
}

// BulkCancel membatalkan beberapa invoice sekaligus.
// Menggunakan logika Cancel yang sama untuk setiap invoice.
func (uc *InvoiceActionUsecase) BulkCancel(ctx context.Context, req domain.BulkCancelRequest, actor domain.ActorInfo) (*domain.InvoiceBulkActionResult, error) {
	result := &domain.InvoiceBulkActionResult{
		Total: len(req.InvoiceIDs),
	}

	for _, id := range req.InvoiceIDs {
		cancelReq := domain.CancelInvoiceRequest{
			Reason: req.Reason,
		}

		// Ambil invoice untuk mendapatkan invoice_number sebagai confirmation
		inv, err := uc.invoiceRepo.GetByID(ctx, id)
		if err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.InvoiceBulkFailure{
				InvoiceID: id,
				Reason:    "invoice tidak ditemukan",
			})
			continue
		}

		// Untuk bulk cancel, gunakan invoice_number sebagai confirmation_number
		cancelReq.ConfirmationNumber = inv.InvoiceNumber

		// Jalankan logika cancel
		if _, err := uc.Cancel(ctx, id, cancelReq, actor); err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.InvoiceBulkFailure{
				InvoiceID: id,
				Reason:    err.Error(),
			})
			continue
		}

		result.SuccessCount++
	}

	return result, nil
}

// BulkPDF menghasilkan PDF untuk beberapa invoice sekaligus.
// Placeholder - mengembalikan bytes kosong untuk saat ini.
func (uc *InvoiceActionUsecase) BulkPDF(ctx context.Context, req domain.BulkInvoiceIDsRequest) ([]byte, error) {
	// Ambil semua invoice berdasarkan IDs untuk validasi
	_, err := uc.invoiceRepo.GetByIDs(ctx, req.InvoiceIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice: %w", err)
	}

	// Placeholder: kembalikan bytes kosong
	// Implementasi lengkap akan menggunakan maroto untuk buat PDF per invoice
	// dan mengemas ke dalam ZIP file.
	return []byte{}, nil
}

// ExportCSV mengekspor daftar invoice ke format CSV.
// Kolom: invoice_number, customer_name, customer_id_seq, period, due_date,
// subtotal, tax, denda, total, paid, status.
func (uc *InvoiceActionUsecase) ExportCSV(ctx context.Context, params domain.InvoiceListParams) ([]byte, error) {
	// Ambil semua invoice sesuai filter (tanpa paginasi)
	params.Page = 1
	params.PageSize = 50 // ambil per batch
	var allInvoices []*domain.Invoice

	for {
		result, err := uc.invoiceRepo.List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil invoice: %w", err)
		}
		allInvoices = append(allInvoices, result.Data...)

		// Cek apakah masih ada halaman berikutnya
		if params.Page >= result.Pagination.TotalPages {
			break
		}
		params.Page++
	}

	// Format CSV
	var buf bytes.Buffer

	// Header CSV
	buf.WriteString("invoice_number,customer_name,customer_id_seq,period,due_date,subtotal,tax,penalty,total,paid,status\n")

	// Baris data
	for _, inv := range allInvoices {
		period := fmt.Sprintf("%d-%02d", inv.PeriodYear, inv.PeriodMonth)
		dueDate := inv.DueDate.Format("2006-01-02")

		line := fmt.Sprintf("%s,%s,%s,%s,%s,%d,%d,%d,%d,%d,%s\n",
			escapeCsvField(inv.InvoiceNumber),
			escapeCsvField(inv.CustomerName),
			escapeCsvField(inv.CustomerIDSeq),
			period,
			dueDate,
			inv.Subtotal,
			inv.TaxAmount,
			inv.PenaltyAmount,
			inv.TotalAmount,
			inv.PaidAmount,
			string(inv.Status),
		)
		buf.WriteString(line)
	}

	return buf.Bytes(), nil
}

// escapeCsvField meng-escape field CSV yang mengandung koma, kutip, atau newline.
func escapeCsvField(field string) string {
	needsQuote := false
	for _, c := range field {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return field
	}

	// Escape double quotes dengan menggandakannya
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, c := range field {
		if c == '"' {
			buf.WriteByte('"')
		}
		buf.WriteRune(c)
	}
	buf.WriteByte('"')
	return buf.String()
}
