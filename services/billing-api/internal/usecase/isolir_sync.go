// isolir_sync.go berisi business logic untuk sinkronisasi periodik dan manual sync.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ProcessPeriodicSync memproses pending_syncs yang siap di-retry secara periodik.
// Mengambil batch 50 record, re-terbitkan event, dan perbarui retry info.
func (uc *IsolirUsecase) ProcessPeriodicSync(ctx context.Context) error {
	syncs, err := uc.pendingSyncRepo.FindPendingForRetry(ctx, 50)
	if err != nil {
		return fmt.Errorf("gagal mengambil pending_syncs untuk retry: %w", err)
	}
	for _, ps := range syncs {
		if err := uc.processSingleSync(ctx, ps); err != nil {
			uc.logger.Error().Err(err).Str("pending_sync_id", ps.ID).
				Str("customer_id", ps.CustomerID).Msg("gagal memproses pending_sync")
		}
	}
	return nil
}

// processSingleSync memproses satu record pending_sync: re-terbitkan event dan perbarui retry.
func (uc *IsolirUsecase) processSingleSync(ctx context.Context, ps *domain.PendingSync) error {
	// Ambil data customer untuk membangun payload event
	cust, err := uc.customerRepo.GetByID(ctx, ps.CustomerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer %s: %w", ps.CustomerID, err)
	}

	// Re-terbitkan event sesuai operation_type
	uc.publishSyncEvent(ctx, ps, cust)

	// Increment retry_count
	newRetryCount := ps.RetryCount + 1

	// Cek apakah sudah mencapai max_retries
	if newRetryCount >= ps.MaxRetries {
		if err := uc.pendingSyncRepo.MarkFailed(ctx, ps.ID, "max retries tercapai"); err != nil {
			return fmt.Errorf("gagal mark failed pending_sync %s: %w", ps.ID, err)
		}
		// Terbitkan notifikasi pending_sync_failed
		uc.publishEvent(ps.TenantID, domain.TaskNotifPendingSyncFailed, ps)
		return nil
	}

	// Hitung next_retry_at menggunakan backoff
	nextRetry := domain.CalculateNextRetryAt(newRetryCount, time.Now())
	if err := uc.pendingSyncRepo.UpdateRetry(ctx, ps.ID, newRetryCount, nextRetry, ""); err != nil {
		return fmt.Errorf("gagal update retry pending_sync %s: %w", ps.ID, err)
	}
	return nil
}

// publishSyncEvent mempublikasikan event berdasarkan operation_type pending_sync.
func (uc *IsolirUsecase) publishSyncEvent(ctx context.Context, ps *domain.PendingSync, cust *domain.Customer) {
	if !uc.mikrotikEnabled(ctx, ps.TenantID) {
		return
	}
	switch ps.OperationType {
	case domain.SyncOpIsolir:
		uc.publishEvent(ps.TenantID, domain.TaskCustomerIsolir, domain.CustomerIsolirPayload{
			CustomerID: cust.ID, TenantID: cust.TenantID, CustomerName: cust.Name,
			RouterID: cust.RouterID, PPPoEUsername: cust.PPPoEUsername,
			ConnectionMethod: string(cust.ConnectionMethod),
			Reason:           "periodic_sync: retry sinkronisasi router",
		})
	case domain.SyncOpUnIsolir:
		uc.publishEvent(ps.TenantID, domain.TaskCustomerUnIsolir, domain.CustomerUnIsolirPayload{
			CustomerID: cust.ID, TenantID: cust.TenantID, CustomerName: cust.Name,
			RouterID: cust.RouterID, PPPoEUsername: cust.PPPoEUsername,
			ConnectionMethod: string(cust.ConnectionMethod),
			Trigger:          "periodic_sync",
		})
	case domain.SyncOpSuspend:
		uc.publishEvent(ps.TenantID, domain.TaskCustomerSuspend, domain.CustomerSuspendPayload{
			CustomerID: cust.ID, TenantID: cust.TenantID, CustomerName: cust.Name,
			RouterID: cust.RouterID, PPPoEUsername: cust.PPPoEUsername,
			ConnectionMethod: string(cust.ConnectionMethod),
		})
	}
}

// ManualSync memproses manual sync untuk satu pelanggan oleh admin.
// Mereset retry_count dan re-terbitkan event untuk semua pending_sync pelanggan.
func (uc *IsolirUsecase) ManualSync(ctx context.Context, customerID, actorID string) error {
	// Cari pending_syncs untuk customer
	syncs, err := uc.pendingSyncRepo.FindByCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal mencari pending_syncs: %w", err)
	}
	if len(syncs) == 0 {
		return domain.ErrNoPendingSync
	}

	// Reset retry_count ke 0
	if err := uc.pendingSyncRepo.ResetRetryForCustomer(ctx, customerID); err != nil {
		return fmt.Errorf("gagal reset retry untuk customer %s: %w", customerID, err)
	}

	// Ambil data customer untuk membangun payload
	cust, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer: %w", err)
	}

	// Re-terbitkan event untuk setiap pending_sync
	for _, ps := range syncs {
		uc.publishSyncEvent(ctx, ps, cust)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, cust.TenantID, "", "sync.manual_trigger",
		map[string]interface{}{"customer_id": customerID, "actor_id": actorID, "count": len(syncs)})
	return nil
}

// ManualSyncAll mereset semua pending/failed records di tenant dan mengembalikan jumlah record.
func (uc *IsolirUsecase) ManualSyncAll(ctx context.Context, tenantID, actorID string) (int, error) {
	count, err := uc.pendingSyncRepo.ResetRetryAll(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("gagal reset retry semua pending_syncs: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, "", "sync.manual_trigger_all",
		map[string]interface{}{"actor_id": actorID, "count": count})
	return count, nil
}
