// invoice_prepaid.go berisi business logic untuk pembuatan invoice prepaid.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// CreatePrepaid membuat invoice prepaid untuk beberapa bulan sekaligus.
// Flow: validasi customer → generate nomor invoice → buat line items per bulan →
// tambah diskon jika ada → buat invoice → tulis audit log.
func (uc *InvoiceUsecase) CreatePrepaid(ctx context.Context, tenantID string, req domain.CreatePrepaidInvoiceRequest, actor domain.ActorInfo) (*domain.Invoice, error) {
	// Validasi customer ada dan aktif
	customer, err := uc.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, domain.ErrCustomerNotFound
	}
	if customer.Status != domain.CustomerStatusAktif {
		return nil, fmt.Errorf("pelanggan tidak aktif (status: %s)", customer.Status)
	}

	// Ambil paket customer untuk harga bulanan
	pkg, err := uc.packageRepo.GetByID(ctx, customer.PackageID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil paket: %w", err)
	}
	var monthlyPrice int64
	if pkg.MonthlyPrice != nil {
		monthlyPrice = *pkg.MonthlyPrice
	}

	// Generate nomor invoice
	settings, _ := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	prefix := "INV"
	if settings != nil && settings.InvoicePrefix != "" {
		prefix = settings.InvoicePrefix
	}

	seq, err := uc.sequenceRepo.NextSequence(ctx, tenantID, req.StartPeriodYear, req.StartPeriodMonth)
	if err != nil {
		return nil, fmt.Errorf("gagal generate nomor invoice: %w", err)
	}
	invoiceNumber := domain.FormatInvoiceNumber(prefix, req.StartPeriodYear, req.StartPeriodMonth, seq)

	// Buat line items untuk setiap bulan
	items := make([]*domain.InvoiceItem, 0, req.Months+1)
	var subtotal int64
	currentMonth := req.StartPeriodMonth
	currentYear := req.StartPeriodYear

	for i := 0; i < req.Months; i++ {
		amount := monthlyPrice
		subtotal += amount
		desc := fmt.Sprintf("Tagihan %s - %s %d", pkg.Name, monthName(currentMonth), currentYear)
		items = append(items, &domain.InvoiceItem{
			TenantID:    tenantID,
			ItemType:    domain.ItemTypeMonthly,
			Description: desc,
			Quantity:    1,
			UnitPrice:   amount,
			Amount:      amount,
			SortOrder:   i + 1,
		})
		// Maju ke bulan berikutnya
		currentMonth++
		if currentMonth > 12 {
			currentMonth = 1
			currentYear++
		}
	}

	// Tambah diskon jika discount_months > 0
	var discountAmount int64
	if req.DiscountMonths > 0 {
		discountAmount = monthlyPrice * int64(req.DiscountMonths)
		items = append(items, &domain.InvoiceItem{
			TenantID:    tenantID,
			ItemType:    domain.ItemTypeDiscount,
			Description: fmt.Sprintf("Diskon %d bulan gratis", req.DiscountMonths),
			Quantity:    1,
			UnitPrice:   discountAmount,
			Amount:      discountAmount,
			SortOrder:   len(items) + 1,
		})
	}

	totalAmount := subtotal - discountAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	// Hitung due date dari periode awal
	dueDate := time.Date(req.StartPeriodYear, time.Month(req.StartPeriodMonth), customer.DueDate, 0, 0, 0, 0, time.UTC)

	prepaidMonths := req.Months
	invoice := &domain.Invoice{
		TenantID:       tenantID,
		CustomerID:     req.CustomerID,
		InvoiceNumber:  invoiceNumber,
		PeriodMonth:    req.StartPeriodMonth,
		PeriodYear:     req.StartPeriodYear,
		DueDate:        dueDate,
		Subtotal:       subtotal,
		DiscountAmount: discountAmount,
		TotalAmount:    totalAmount,
		Status:         domain.InvoiceStatusBelumBayar,
		IsPrepaid:      true,
		PrepaidMonths:  &prepaidMonths,
		Version:        1,
	}

	created, err := uc.invoiceRepo.Create(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat invoice prepaid: %w", err)
	}

	// Set invoice_id pada semua items dan bulk create
	for _, item := range items {
		item.InvoiceID = created.ID
	}
	if _, err := uc.itemRepo.BulkCreate(ctx, items); err != nil {
		return nil, fmt.Errorf("gagal membuat invoice items: %w", err)
	}

	// Tulis audit log
	uc.writeInvoiceAuditLog(ctx, tenantID, created.ID, "invoice.created_prepaid", actor, map[string]interface{}{
		"months":          req.Months,
		"discount_months": req.DiscountMonths,
	})

	return created, nil
}

// monthName mengembalikan nama bulan dalam bahasa Indonesia.
func monthName(month int) string {
	names := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	if month >= 1 && month <= 12 {
		return names[month]
	}
	return fmt.Sprintf("Bulan-%d", month)
}
