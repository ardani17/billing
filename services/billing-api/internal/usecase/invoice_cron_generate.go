// invoice_cron_generate.go berisi logika buat invoice per pelanggan untuk job cron.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// generateInvoiceForCustomer membuat invoice untuk satu pelanggan.
// Alur: cek idempotency -> cek prepaid -> buat nomor -> ambil paket (snapshot harga) ->
// tambah item instalasi jika invoice pertama -> ambil item berulangs -> hitung subtotal ->
// hitung pajak -> terapkan kredit -> buat invoice -> buat items -> tulis audit log -> terbitkan event.
func (uc *InvoiceCronUsecase) generateInvoiceForCustomer(
	ctx context.Context,
	settings *domain.BillingSettings,
	customer *domain.Customer,
	now time.Time,
) error {
	// Tentukan periode invoice (bulan dan tahun jatuh tempo)
	periodMonth, periodYear := uc.calculatePeriod(customer.DueDate, now)

	// Cek idempotency: apakah invoice untuk periode ini sudah ada
	exists, err := uc.invoiceRepo.ExistsForPeriod(ctx, customer.ID, periodMonth, periodYear)
	if err != nil {
		return fmt.Errorf("gagal cek invoice existing: %w", err)
	}
	if exists {
		return nil // sudah ada, skip (idempotent)
	}

	// Cek apakah invoice prepaid sudah mencakup periode ini
	prepaidExists, err := uc.invoiceRepo.ExistsForPeriodPrepaid(ctx, customer.ID, periodMonth, periodYear)
	if err != nil {
		return fmt.Errorf("gagal cek prepaid: %w", err)
	}
	if prepaidExists {
		return nil // periode sudah di-cover prepaid, skip
	}

	// Buat nomor invoice via sequence atomik
	prefix := "INV"
	if settings.InvoicePrefix != "" {
		prefix = settings.InvoicePrefix
	}
	seq, err := uc.sequenceRepo.NextSequence(ctx, settings.TenantID, periodYear, periodMonth)
	if err != nil {
		return fmt.Errorf("gagal generate nomor invoice: %w", err)
	}
	invoiceNumber := domain.FormatInvoiceNumber(prefix, periodYear, periodMonth, seq)

	// Ambil paket untuk snapshot harga saat ini
	pkg, err := uc.packageRepo.GetByID(ctx, customer.PackageID)
	if err != nil {
		return fmt.Errorf("gagal mengambil paket: %w", err)
	}

	var monthlyPrice int64
	if pkg.MonthlyPrice != nil {
		monthlyPrice = *pkg.MonthlyPrice
	}

	// Bangun daftar item invoice
	items, subtotal := uc.buildInvoiceItems(ctx, settings, customer, pkg, monthlyPrice, periodMonth, periodYear)

	// Hitung pajak jika diaktifkan
	var taxAmount int64
	if settings.TaxEnabled && settings.TaxRate > 0 {
		taxAmount = subtotal * int64(settings.TaxRate) / 100
		items = append(items, &domain.InvoiceItem{
			TenantID:    settings.TenantID,
			ItemType:    domain.ItemTypeTax,
			Description: fmt.Sprintf("PPN %v%%", settings.TaxRate),
			Quantity:    1,
			UnitPrice:   taxAmount,
			Amount:      taxAmount,
			SortOrder:   len(items) + 1,
		})
	}

	totalAmount := subtotal + taxAmount

	// Terapkan kredit jika pelanggan punya saldo kredit
	var creditApplied int64
	if customer.CreditBalance > 0 && totalAmount > 0 {
		creditApplied = customer.CreditBalance
		if creditApplied > totalAmount {
			creditApplied = totalAmount
		}
		// Kurangi credit_balance pelanggan secara atomik
		if err := uc.adjustCreditBalance(ctx, customer.ID, -creditApplied); err != nil {
			return fmt.Errorf("gagal update credit balance: %w", err)
		}
		items = append(items, &domain.InvoiceItem{
			TenantID:    settings.TenantID,
			ItemType:    domain.ItemTypeCreditApplied,
			Description: "Kredit diterapkan",
			Quantity:    1,
			UnitPrice:   creditApplied,
			Amount:      creditApplied,
			SortOrder:   len(items) + 1,
		})
		totalAmount -= creditApplied
	}

	// Hitung tanggal jatuh tempo
	dueDate := time.Date(periodYear, time.Month(periodMonth), customer.DueDate, 0, 0, 0, 0, time.UTC)

	// Buat invoice
	invoice := &domain.Invoice{
		TenantID:      settings.TenantID,
		CustomerID:    customer.ID,
		InvoiceNumber: invoiceNumber,
		PeriodMonth:   periodMonth,
		PeriodYear:    periodYear,
		DueDate:       dueDate,
		Subtotal:      subtotal,
		TaxAmount:     taxAmount,
		CreditApplied: creditApplied,
		TotalAmount:   totalAmount,
		Status:        domain.InvoiceStatusBelumBayar,
		Version:       1,
	}

	created, err := uc.invoiceRepo.Create(ctx, invoice)
	if err != nil {
		return fmt.Errorf("gagal membuat invoice: %w", err)
	}

	// Set invoice_id pada semua items dan bulk buat
	for _, item := range items {
		item.InvoiceID = created.ID
	}
	if _, err := uc.itemRepo.BulkCreate(ctx, items); err != nil {
		return fmt.Errorf("gagal membuat invoice items: %w", err)
	}

	// Tulis audit log (aktor = System untuk job cron)
	uc.writeCronAuditLog(ctx, settings.TenantID, created.ID, "invoice.generated")

	// Terbitkan event invoice.created
	uc.publishCronEvent(settings.TenantID, "invoice.created", domain.InvoiceCreatedPayload{
		InvoiceID:     created.ID,
		TenantID:      settings.TenantID,
		CustomerID:    customer.ID,
		InvoiceNumber: invoiceNumber,
		TotalAmount:   totalAmount,
		DueDate:       dueDate.Format("2006-01-02"),
	})

	return nil
}
