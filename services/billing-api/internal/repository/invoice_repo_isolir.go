package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Method isolir-specific pada InvoiceRepo ---
// File ini berisi method tambahan pada InvoiceRepo untuk kebutuhan modul isolir.
// Menggunakan fungsi sqlc-generated dari invoice_isolir.sql.go.

// FindOverdueForIsolir mengambil invoice terlambat yang sudah melewati grace period.
// Hanya mengembalikan invoice milik pelanggan dengan status aktif (eligible untuk isolir).
func (r *InvoiceRepo) FindOverdueForIsolir(ctx context.Context, tenantID string, gracePeriodDays int, currentDate time.Time) ([]*domain.Invoice, error) {
	rows, err := r.queries.FindOverdueForIsolir(ctx, FindOverdueForIsolirParams{
		TenantID:        stringToUUID(tenantID),
		CurrentDate:     timeToDate(currentDate),
		GracePeriodDays: int32(gracePeriodDays),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice overdue untuk isolir: %w", err)
	}
	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}

// FindOverdueForSuspend mengambil invoice terlambat yang sudah melewati suspend_days.
// Hanya mengembalikan invoice milik pelanggan dengan status isolir (eligible untuk suspend).
func (r *InvoiceRepo) FindOverdueForSuspend(ctx context.Context, tenantID string, suspendDays int, currentDate time.Time) ([]*domain.Invoice, error) {
	rows, err := r.queries.FindOverdueForSuspend(ctx, FindOverdueForSuspendParams{
		TenantID:    stringToUUID(tenantID),
		CurrentDate: timeToDate(currentDate),
		SuspendDays: int32(suspendDays),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice overdue untuk suspend: %w", err)
	}
	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}

// HasOutstandingInvoices mengecek apakah customer masih punya invoice yang belum lunas.
// Invoice outstanding = status bukan lunas dan bukan batal.
func (r *InvoiceRepo) HasOutstandingInvoices(ctx context.Context, customerID string) (bool, error) {
	exists, err := r.queries.HasOutstandingInvoices(ctx, stringToUUID(customerID))
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek outstanding invoices: %w", err)
	}
	return exists, nil
}

// SumOutstandingAmount menghitung total tagihan outstanding untuk customer tertentu.
// Mengembalikan 0 jika tidak ada invoice outstanding.
func (r *InvoiceRepo) SumOutstandingAmount(ctx context.Context, customerID string) (int64, error) {
	total, err := r.queries.SumOutstandingAmount(ctx, stringToUUID(customerID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung total outstanding: %w", err)
	}
	return total, nil
}

// CountOutstandingInvoices menghitung jumlah invoice outstanding untuk customer tertentu.
// Invoice outstanding = status bukan lunas dan bukan batal.
func (r *InvoiceRepo) CountOutstandingInvoices(ctx context.Context, customerID string) (int, error) {
	count, err := r.queries.CountOutstandingInvoices(ctx, stringToUUID(customerID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung jumlah outstanding invoices: %w", err)
	}
	return int(count), nil
}
