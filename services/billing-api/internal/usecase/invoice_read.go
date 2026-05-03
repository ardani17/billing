// invoice_read.go berisi business logic untuk query invoice (GetByID, Edit, List, Summary, GeneratePDF).
package usecase

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetByID mengambil detail invoice lengkap termasuk items, payments, dan audit logs.
func (uc *InvoiceUsecase) GetByID(ctx context.Context, id string, includeAudit bool) (*domain.InvoiceDetail, error) {
	// Ambil invoice
	invoice, err := uc.invoiceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ambil items
	items, err := uc.itemRepo.ListByInvoice(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice items: %w", err)
	}

	// Ambil payments (non-voided)
	payments, err := uc.paymentRepo.ListByInvoice(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil payments: %w", err)
	}

	detail := &domain.InvoiceDetail{
		Invoice:  invoice,
		Items:    items,
		Payments: payments,
	}

	// Ambil audit logs jika diminta
	if includeAudit {
		auditLogs, err := uc.auditRepo.ListByInvoice(ctx, id)
		if err != nil {
			uc.logger.Error().Err(err).Str("invoice_id", id).Msg("gagal mengambil audit logs")
		} else {
			detail.AuditLogs = auditLogs
		}
	}

	return detail, nil
}

// Edit memperbarui invoice yang masih berstatus belum_bayar.
// Flow: ambil invoice → verifikasi status → hapus items lama → hitung ulang →
// update invoice → buat items baru → tulis audit log.
func (uc *InvoiceUsecase) Edit(ctx context.Context, id string, req domain.EditInvoiceRequest, actor domain.ActorInfo) (*domain.Invoice, error) {
	// Ambil invoice yang ada
	invoice, err := uc.invoiceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verifikasi status harus belum_bayar
	if invoice.Status != domain.InvoiceStatusBelumBayar {
		return nil, domain.ErrInvoiceNotEditable
	}

	// Update due date jika diberikan
	if req.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("format due_date tidak valid: %w", err)
		}
		invoice.DueDate = dueDate
	}

	// Update notes jika diberikan
	if req.Notes != "" {
		invoice.Notes = req.Notes
	}

	// Jika items baru diberikan, hapus items lama dan hitung ulang
	if len(req.Items) > 0 {
		if err := uc.itemRepo.DeleteByInvoice(ctx, id); err != nil {
			return nil, fmt.Errorf("gagal menghapus items lama: %w", err)
		}

		var subtotal int64
		items := make([]*domain.InvoiceItem, 0, len(req.Items))
		for i, item := range req.Items {
			amount := int64(item.Quantity) * item.UnitPrice
			subtotal += amount
			items = append(items, &domain.InvoiceItem{
				TenantID:    invoice.TenantID,
				InvoiceID:   id,
				ItemType:    domain.ItemTypeCustom,
				Description: item.Description,
				Quantity:    item.Quantity,
				UnitPrice:   item.UnitPrice,
				Amount:      amount,
				SortOrder:   i + 1,
			})
		}

		// Hitung ulang pajak jika sebelumnya ada
		var taxAmount int64
		if invoice.TaxAmount > 0 {
			settings, _ := uc.settingsRepo.GetByTenantID(ctx, invoice.TenantID)
			if settings != nil && settings.TaxEnabled {
				taxAmount = subtotal * int64(settings.TaxRate) / 100
				items = append(items, &domain.InvoiceItem{
					TenantID:    invoice.TenantID,
					InvoiceID:   id,
					ItemType:    domain.ItemTypeTax,
					Description: fmt.Sprintf("PPN %v%%", settings.TaxRate),
					Quantity:    1,
					UnitPrice:   taxAmount,
					Amount:      taxAmount,
					SortOrder:   len(items) + 1,
				})
			}
		}

		// Hitung ulang kredit jika sebelumnya ada
		totalBeforeCredit := subtotal + taxAmount
		var creditApplied int64
		if invoice.CreditApplied > 0 && totalBeforeCredit > 0 {
			creditApplied = invoice.CreditApplied
			if creditApplied > totalBeforeCredit {
				creditApplied = totalBeforeCredit
			}
			items = append(items, &domain.InvoiceItem{
				TenantID:    invoice.TenantID,
				InvoiceID:   id,
				ItemType:    domain.ItemTypeCreditApplied,
				Description: "Kredit diterapkan",
				Quantity:    1,
				UnitPrice:   creditApplied,
				Amount:      creditApplied,
				SortOrder:   len(items) + 1,
			})
		}

		invoice.Subtotal = subtotal
		invoice.TaxAmount = taxAmount
		invoice.CreditApplied = creditApplied
		invoice.TotalAmount = totalBeforeCredit - creditApplied

		if _, err := uc.itemRepo.BulkCreate(ctx, items); err != nil {
			return nil, fmt.Errorf("gagal membuat items baru: %w", err)
		}
	}

	// Increment version untuk optimistic locking
	invoice.Version++

	updated, err := uc.invoiceRepo.Update(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("gagal update invoice: %w", err)
	}

	// Tulis audit log
	uc.writeInvoiceAuditLog(ctx, invoice.TenantID, id, "invoice.edited", actor, nil)

	return updated, nil
}

// List mengambil daftar invoice dengan filter, search, sorting, dan paginasi.
// Menerapkan default: page=1, page_size=25.
func (uc *InvoiceUsecase) List(ctx context.Context, params domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}
	return uc.invoiceRepo.List(ctx, params)
}

// Summary mengambil ringkasan invoice per status untuk dashboard.
func (uc *InvoiceUsecase) Summary(ctx context.Context, tenantID string, periodMonth, periodYear *int) (*domain.InvoiceSummary, error) {
	return uc.invoiceRepo.GetSummary(ctx, tenantID, periodMonth, periodYear)
}

// GeneratePDF menghasilkan PDF untuk invoice menggunakan gofpdf.
// Termasuk: nomor invoice, tanggal jatuh tempo, status, data pelanggan,
// semua line items, subtotal, pajak, denda, diskon, kredit, total, dan riwayat pembayaran.
func (uc *InvoiceUsecase) GeneratePDF(ctx context.Context, id string) ([]byte, string, error) {
	// Ambil detail invoice lengkap (items + payments)
	detail, err := uc.GetByID(ctx, id, false)
	if err != nil {
		return nil, "", err
	}
	inv := detail.Invoice

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// --- Header: Invoice Number dan Status ---
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, fmt.Sprintf("INVOICE %s", inv.InvoiceNumber))
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Status: %s", inv.Status))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Tanggal Jatuh Tempo: %s", inv.DueDate.Format("02 Jan 2006")))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Periode: %02d/%d", inv.PeriodMonth, inv.PeriodYear))
	pdf.Ln(10)

	// --- Data Pelanggan ---
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(0, 6, "Data Pelanggan")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	if inv.CustomerName != "" {
		pdf.Cell(0, 5, fmt.Sprintf("Nama: %s", inv.CustomerName))
		pdf.Ln(5)
	}
	if inv.CustomerIDSeq != "" {
		pdf.Cell(0, 5, fmt.Sprintf("ID Pelanggan: %s", inv.CustomerIDSeq))
		pdf.Ln(5)
	}
	if inv.CustomerPhone != "" {
		pdf.Cell(0, 5, fmt.Sprintf("Telepon: %s", inv.CustomerPhone))
		pdf.Ln(5)
	}
	if inv.CustomerAddress != "" {
		pdf.Cell(0, 5, fmt.Sprintf("Alamat: %s", inv.CustomerAddress))
		pdf.Ln(5)
	}
	pdf.Ln(5)

	// --- Tabel Line Items ---
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(10, 7, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 7, "Deskripsi", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 7, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 7, "Harga Satuan", "1", 0, "R", true, 0, "")
	pdf.CellFormat(40, 7, "Jumlah", "1", 0, "R", true, 0, "")
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	for i, item := range detail.Items {
		pdf.CellFormat(10, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(70, 6, truncateStr(item.Description, 40), "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%d", item.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 6, formatRupiah(item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 6, formatRupiah(item.Amount), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}
	pdf.Ln(3)

	// --- Ringkasan Nominal ---
	pdf.SetFont("Arial", "", 10)
	summaryX := 110.0
	summaryW := 40.0
	valW := 40.0

	pdf.SetX(summaryX)
	pdf.CellFormat(summaryW, 6, "Subtotal", "", 0, "L", false, 0, "")
	pdf.CellFormat(valW, 6, formatRupiah(inv.Subtotal), "", 0, "R", false, 0, "")
	pdf.Ln(6)

	if inv.TaxAmount > 0 {
		pdf.SetX(summaryX)
		pdf.CellFormat(summaryW, 6, "Pajak (PPN)", "", 0, "L", false, 0, "")
		pdf.CellFormat(valW, 6, formatRupiah(inv.TaxAmount), "", 0, "R", false, 0, "")
		pdf.Ln(6)
	}
	if inv.PenaltyAmount > 0 {
		pdf.SetX(summaryX)
		pdf.CellFormat(summaryW, 6, "Denda", "", 0, "L", false, 0, "")
		pdf.CellFormat(valW, 6, formatRupiah(inv.PenaltyAmount), "", 0, "R", false, 0, "")
		pdf.Ln(6)
	}
	if inv.DiscountAmount > 0 {
		pdf.SetX(summaryX)
		pdf.CellFormat(summaryW, 6, "Diskon", "", 0, "L", false, 0, "")
		pdf.CellFormat(valW, 6, fmt.Sprintf("-%s", formatRupiah(inv.DiscountAmount)), "", 0, "R", false, 0, "")
		pdf.Ln(6)
	}
	if inv.CreditApplied > 0 {
		pdf.SetX(summaryX)
		pdf.CellFormat(summaryW, 6, "Kredit Diterapkan", "", 0, "L", false, 0, "")
		pdf.CellFormat(valW, 6, fmt.Sprintf("-%s", formatRupiah(inv.CreditApplied)), "", 0, "R", false, 0, "")
		pdf.Ln(6)
	}

	pdf.SetFont("Arial", "B", 11)
	pdf.SetX(summaryX)
	pdf.CellFormat(summaryW, 7, "TOTAL", "T", 0, "L", false, 0, "")
	pdf.CellFormat(valW, 7, formatRupiah(inv.TotalAmount), "T", 0, "R", false, 0, "")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	pdf.SetX(summaryX)
	pdf.CellFormat(summaryW, 6, "Terbayar", "", 0, "L", false, 0, "")
	pdf.CellFormat(valW, 6, formatRupiah(inv.PaidAmount), "", 0, "R", false, 0, "")
	pdf.Ln(10)

	// --- Riwayat Pembayaran ---
	if len(detail.Payments) > 0 {
		pdf.SetFont("Arial", "B", 11)
		pdf.Cell(0, 6, "Riwayat Pembayaran")
		pdf.Ln(6)

		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		pdf.CellFormat(35, 6, "Tanggal", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 6, "Metode", "1", 0, "C", true, 0, "")
		pdf.CellFormat(40, 6, "Jumlah", "1", 0, "R", true, 0, "")
		pdf.CellFormat(40, 6, "Dicatat Oleh", "1", 0, "L", true, 0, "")
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 9)
		for _, p := range detail.Payments {
			pdf.CellFormat(35, 6, p.PaymentDate.Format("02 Jan 2006"), "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 6, p.PaymentMethod, "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, 6, formatRupiah(p.Amount), "1", 0, "R", false, 0, "")
			pdf.CellFormat(40, 6, truncateStr(p.RecordedByName, 20), "1", 0, "L", false, 0, "")
			pdf.Ln(-1)
		}
	}

	// Generate PDF bytes
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", fmt.Errorf("gagal generate PDF: %w", err)
	}

	filename := fmt.Sprintf("%s.pdf", inv.InvoiceNumber)
	return buf.Bytes(), filename, nil
}

// formatRupiah memformat angka ke format Rupiah sederhana (tanpa simbol).
func formatRupiah(amount int64) string {
	// Format dengan pemisah ribuan
	str := fmt.Sprintf("%d", amount)
	if len(str) <= 3 {
		return "Rp " + str
	}
	var result []byte
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return "Rp " + string(result)
}

// truncateStr memotong string ke panjang maksimum.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
