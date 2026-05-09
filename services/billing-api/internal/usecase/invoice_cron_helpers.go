// invoice_cron_helpers.go berisi helper methods untuk InvoiceCronUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ProcessOverdueUpdate memperbarui status invoice yang sudah melewati jatuh tempo.
// Mencari semua invoice dengan status belum_bayar yang due_date < tanggal saat ini ->
// transisi ke terlambat -> tulis audit log -> terbitkan event.
func (uc *InvoiceCronUsecase) ProcessOverdueUpdate(ctx context.Context) error {
	now := time.Now()

	// Cari semua invoice yang sudah melewati jatuh tempo
	overdueInvoices, err := uc.invoiceRepo.FindOverdue(ctx, now)
	if err != nil {
		return fmt.Errorf("gagal mencari invoice overdue: %w", err)
	}

	for _, invoice := range overdueInvoices {
		// Transisi status ke terlambat dengan optimistic locking
		updated, err := uc.invoiceRepo.UpdateStatus(ctx, invoice.ID, domain.InvoiceStatusTerlambat, invoice.Version)
		if err != nil {
			uc.logger.Error().Err(err).
				Str("invoice_id", invoice.ID).
				Str("tenant_id", invoice.TenantID).
				Msg("gagal update status invoice ke terlambat")
			continue // Lanjutkan ke invoice berikutnya
		}

		// Tulis audit log
		uc.writeCronAuditLog(ctx, invoice.TenantID, invoice.ID, "invoice.overdue")

		// Hitung hari keterlambatan untuk event payload
		daysOverdue := int(math.Ceil(now.Sub(updated.DueDate).Hours() / 24))
		if daysOverdue < 1 {
			daysOverdue = 1
		}

		// Terbitkan event invoice.terlambat
		uc.publishCronEvent(invoice.TenantID, "invoice.overdue", domain.InvoiceOverduePayload{
			InvoiceID:     invoice.ID,
			TenantID:      invoice.TenantID,
			CustomerID:    invoice.CustomerID,
			InvoiceNumber: invoice.InvoiceNumber,
			TotalAmount:   invoice.TotalAmount,
			DaysOverdue:   daysOverdue,
		})
	}

	return nil
}

// buildInvoiceItems membangun daftar item invoice untuk auto-buat.
// Termasuk: item bulanan, item instalasi (jika invoice pertama), dan item berulangs.
func (uc *InvoiceCronUsecase) buildInvoiceItems(
	ctx context.Context,
	settings *domain.BillingSettings,
	customer *domain.Customer,
	pkg *domain.Package,
	monthlyPrice int64,
	periodMonth, periodYear int,
) ([]*domain.InvoiceItem, int64) {
	var items []*domain.InvoiceItem
	var subtotal int64
	sortOrder := 1

	// Item bulanan (snapshot harga paket saat ini)
	if monthlyPrice > 0 {
		items = append(items, &domain.InvoiceItem{
			TenantID:    settings.TenantID,
			ItemType:    domain.ItemTypeMonthly,
			Description: fmt.Sprintf("Tagihan bulanan - %s", pkg.Name),
			Quantity:    1,
			UnitPrice:   monthlyPrice,
			Amount:      monthlyPrice,
			SortOrder:   sortOrder,
		})
		subtotal += monthlyPrice
		sortOrder++
	}

	// Item instalasi: hanya pada invoice pertama pelanggan
	if pkg.InstallationFee > 0 {
		isFirst, err := uc.isFirstInvoice(ctx, settings.TenantID, customer.ID)
		if err != nil {
			uc.logger.Error().Err(err).
				Str("customer_id", customer.ID).
				Msg("gagal cek invoice pertama, skip installation fee")
		} else if isFirst {
			items = append(items, &domain.InvoiceItem{
				TenantID:    settings.TenantID,
				ItemType:    domain.ItemTypeInstallation,
				Description: "Biaya instalasi",
				Quantity:    1,
				UnitPrice:   pkg.InstallationFee,
				Amount:      pkg.InstallationFee,
				SortOrder:   sortOrder,
			})
			subtotal += pkg.InstallationFee
			sortOrder++
		}
	}

	// Recurring items aktif untuk pelanggan
	periodDate := time.Date(periodYear, time.Month(periodMonth), 1, 0, 0, 0, 0, time.UTC)
	recurringItems, err := uc.recurringItemRepo.ListActiveByCustomer(ctx, customer.ID, periodDate)
	if err != nil {
		uc.logger.Error().Err(err).
			Str("customer_id", customer.ID).
			Msg("gagal mengambil recurring items")
	} else {
		for _, ri := range recurringItems {
			items = append(items, &domain.InvoiceItem{
				TenantID:    settings.TenantID,
				ItemType:    domain.ItemTypeRecurring,
				Description: ri.Description,
				Quantity:    1,
				UnitPrice:   ri.Amount,
				Amount:      ri.Amount,
				SortOrder:   sortOrder,
			})
			subtotal += ri.Amount
			sortOrder++
		}
	}

	return items, subtotal
}

// isFirstInvoice memeriksa apakah ini invoice pertama untuk pelanggan.
// Menggunakan ExistsForPeriod dengan bulan/tahun 0 sebagai penanda cek umum,
// atau cek via list dengan page_size=1.
func (uc *InvoiceCronUsecase) isFirstInvoice(ctx context.Context, tenantID, customerID string) (bool, error) {
	// Cek apakah ada invoice sebelumnya untuk pelanggan ini
	result, err := uc.invoiceRepo.List(ctx, domain.InvoiceListParams{
		TenantID:   tenantID,
		CustomerID: customerID,
		Page:       1,
		PageSize:   1,
	})
	if err != nil {
		return false, err
	}
	// Jika tidak ada invoice sebelumnya, ini adalah invoice pertama
	return result.Pagination.Total == 0, nil
}

// calculatePeriod menghitung bulan dan tahun periode invoice berdasarkan due_date pelanggan.
func (uc *InvoiceCronUsecase) calculatePeriod(dueDay int, now time.Time) (int, int) {
	// Periode invoice adalah bulan saat ini jika due_date >= hari ini,
	// atau bulan depan jika due_date < hari ini
	currentDay := now.Day()
	month := int(now.Month())
	year := now.Year()

	if dueDay < currentDay {
		// Due date sudah lewat bulan ini, buat untuk bulan depan
		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return month, year
}

// writeCronAuditLog menulis audit log untuk operasi cron (aktor = System).
func (uc *InvoiceCronUsecase) writeCronAuditLog(ctx context.Context, tenantID, invoiceID, action string) {
	log := &domain.InvoiceAuditLog{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Action:    action,
		ActorID:   "system",
		ActorName: "System",
	}

	if err := uc.auditRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).
			Str("invoice_id", invoiceID).
			Str("action", action).
			Msg("gagal menulis cron audit log")
	}
}

// publishCronEvent mempublikasikan event ke Redis queue dari job cron.
func (uc *InvoiceCronUsecase) publishCronEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal cron event payload")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish cron event")
	}
}

// adjustCreditBalance mengubah credit_balance pelanggan secara atomik menggunakan SQL langsung.
// delta positif = tambah kredit, delta negatif = kurangi kredit.
// Menggunakan UPDATE ... SET credit_balance = credit_balance + $1 untuk atomicity.
func (uc *InvoiceCronUsecase) adjustCreditBalance(ctx context.Context, customerID string, delta int64) error {
	if uc.pool == nil || delta == 0 {
		return nil
	}
	_, err := uc.pool.Exec(ctx,
		`UPDATE customers SET credit_balance = credit_balance + $1, updated_at = NOW() WHERE id = $2`,
		delta, customerID,
	)
	if err != nil {
		return fmt.Errorf("gagal update credit_balance: %w", err)
	}
	return nil
}
