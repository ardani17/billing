// isolir_unisolir.go berisi business logic untuk un-isolir, reactivate, dan re-isolir pelanggan.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ProcessUnIsolir memproses buka isolir pelanggan setelah pembayaran diterima.
// Mengecek auto_open_isolir aktif, status isolir, dan semua invoice lunas.
func (uc *IsolirUsecase) ProcessUnIsolir(ctx context.Context, tenantID, customerID, trigger string) error {
	// Ambil billing settings untuk cek auto_open_isolir
	settings, err := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("gagal mengambil billing settings: %w", err)
	}
	if !settings.AutoOpenIsolir {
		return nil // fitur auto buka isolir tidak aktif
	}

	// Ambil data pelanggan
	cust, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer: %w", err)
	}

	// Idempotency: skip jika bukan status isolir
	if cust.Status != domain.CustomerStatusIsolir {
		return nil
	}

	// Cek apakah masih ada invoice outstanding
	hasOutstanding, err := uc.invoiceRepo.HasOutstandingInvoices(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal cek outstanding invoices: %w", err)
	}
	if hasOutstanding {
		return nil // masih ada tagihan, tidak bisa buka isolir
	}

	// Transisi status isolir -> aktif
	newStatus, err := domain.Transition(cust.Status, domain.CustomerStatusAktif)
	if err != nil {
		return err
	}
	if _, err := uc.customerRepo.UpdateStatus(ctx, cust.ID, newStatus); err != nil {
		return fmt.Errorf("gagal update status ke aktif: %w", err)
	}

	// Terbitkan event customer.un_isolir dan notification.un_isolir
	p := domain.CustomerUnIsolirPayload{
		CustomerID:       cust.ID,
		TenantID:         cust.TenantID,
		CustomerName:     cust.Name,
		RouterID:         cust.RouterID,
		PPPoEUsername:    cust.PPPoEUsername,
		ConnectionMethod: string(cust.ConnectionMethod),
		Trigger:          trigger,
	}
	if uc.mikrotikEnabled(ctx, cust.TenantID) {
		uc.createPendingSync(ctx, cust.TenantID, cust.ID, domain.SyncOpUnIsolir)
		uc.publishEvent(cust.TenantID, domain.TaskCustomerUnIsolir, p)
	}
	uc.publishEvent(cust.TenantID, domain.TaskNotifUnIsolir, p)

	// Tulis audit log
	uc.writeAuditLog(ctx, cust.TenantID, "", "customer.un_isolir",
		map[string]interface{}{"customer_id": cust.ID, "trigger": trigger})
	return nil
}

// ProcessReactivate memproses reaktivasi pelanggan suspend oleh admin.
// Memerlukan semua invoice lunas sebelum bisa diaktifkan kembali.
func (uc *IsolirUsecase) ProcessReactivate(ctx context.Context, customerID, actorID, actorName string) error {
	// Ambil data pelanggan
	cust, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer: %w", err)
	}

	// Validasi status harus suspend
	if cust.Status != domain.CustomerStatusSuspend {
		return fmt.Errorf("%w: status saat ini %s, harus suspend", domain.ErrInvalidStatusTransition, cust.Status)
	}

	// Cek apakah masih ada invoice outstanding
	hasOutstanding, err := uc.invoiceRepo.HasOutstandingInvoices(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal cek outstanding invoices: %w", err)
	}
	if hasOutstanding {
		return domain.ErrOutstandingInvoicesExist
	}

	// Transisi status suspend -> aktif
	newStatus, err := domain.Transition(cust.Status, domain.CustomerStatusAktif)
	if err != nil {
		return err
	}
	if _, err := uc.customerRepo.UpdateStatus(ctx, cust.ID, newStatus); err != nil {
		return fmt.Errorf("gagal update status ke aktif: %w", err)
	}

	// Terbitkan event customer.un_isolir dan notification.reactivated
	p := domain.CustomerUnIsolirPayload{
		CustomerID:       cust.ID,
		TenantID:         cust.TenantID,
		CustomerName:     cust.Name,
		RouterID:         cust.RouterID,
		PPPoEUsername:    cust.PPPoEUsername,
		ConnectionMethod: string(cust.ConnectionMethod),
		Trigger:          "admin_manual",
	}
	if uc.mikrotikEnabled(ctx, cust.TenantID) {
		uc.createPendingSync(ctx, cust.TenantID, cust.ID, domain.SyncOpUnIsolir)
		uc.publishEvent(cust.TenantID, domain.TaskCustomerUnIsolir, p)
	}
	uc.publishEvent(cust.TenantID, domain.TaskNotifReactivated, p)

	// Tulis audit log dengan aktor admin
	uc.writeAuditLog(ctx, cust.TenantID, "", "customer.reactivated",
		map[string]interface{}{"customer_id": cust.ID, "actor_id": actorID, "actor_name": actorName})
	return nil
}

// ProcessReIsolir memproses re-isolir pelanggan setelah pembayaran di-void.
// Mengecek status aktif dan apakah ada invoice outstanding melewati grace period.
func (uc *IsolirUsecase) ProcessReIsolir(ctx context.Context, tenantID, customerID string) error {
	// Ambil data pelanggan
	cust, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer: %w", err)
	}

	// Skip jika bukan status aktif
	if cust.Status != domain.CustomerStatusAktif {
		return nil
	}

	// Ambil billing settings untuk grace period
	settings, err := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("gagal mengambil billing settings: %w", err)
	}

	// Cek apakah ada invoice terlambat melewati grace period
	now := domain.CurrentDateInTimezone(settings.Timezone)
	invoices, err := uc.invoiceRepo.FindOverdueForIsolir(ctx, tenantID, settings.GracePeriodDays, now.Time)
	if err != nil {
		return fmt.Errorf("gagal mencari invoice overdue: %w", err)
	}

	// Cari invoice milik customer ini
	var targetInv *domain.Invoice
	for _, inv := range invoices {
		if inv.CustomerID == customerID {
			targetInv = inv
			break
		}
	}
	if targetInv == nil {
		return nil // tidak ada invoice terlambat, tidak perlu re-isolir
	}

	// Transisi status aktif -> isolir
	newStatus, err := domain.Transition(cust.Status, domain.CustomerStatusIsolir)
	if err != nil {
		return err
	}
	if _, err := uc.customerRepo.UpdateStatus(ctx, cust.ID, newStatus); err != nil {
		return fmt.Errorf("gagal update status ke isolir: %w", err)
	}

	// Terbitkan event customer.isolir dan notification.isolir
	overdue := domain.DaysOverdue(targetInv.DueDate, now.Time)
	p := domain.CustomerIsolirPayload{
		CustomerID:       cust.ID,
		TenantID:         cust.TenantID,
		CustomerName:     cust.Name,
		RouterID:         cust.RouterID,
		PPPoEUsername:    cust.PPPoEUsername,
		ConnectionMethod: string(cust.ConnectionMethod),
		Reason:           "re_isolir: pembayaran di-void, invoice kembali overdue",
		OverdueDays:      overdue,
	}
	if uc.mikrotikEnabled(ctx, cust.TenantID) {
		uc.createPendingSync(ctx, cust.TenantID, cust.ID, domain.SyncOpIsolir)
		uc.publishEvent(cust.TenantID, domain.TaskCustomerIsolir, p)
	}
	uc.publishEvent(cust.TenantID, domain.TaskNotifIsolir, p)

	// Tulis audit log
	uc.writeAuditLog(ctx, cust.TenantID, targetInv.ID, "customer.re_isolir",
		map[string]interface{}{"customer_id": cust.ID, "void_triggered": true})
	return nil
}
