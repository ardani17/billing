// isolir_summary.go berisi business logic untuk dashboard summary dan daftar pending sync.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetDashboardSummary mengambil ringkasan statistik isolir untuk dashboard tenant.
// Menghitung jumlah pelanggan isolir, suspend, pending sync, dan revenue at risk.
func (uc *IsolirUsecase) GetDashboardSummary(ctx context.Context, tenantID string) (*domain.IsolirSummary, error) {
	// Hitung pelanggan per status (RLS-scoped via context)
	statusCounts, err := uc.customerRepo.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung pelanggan per status: %w", err)
	}

	// Hitung pending sync dengan status pending atau failed
	pendingSyncCount, err := uc.pendingSyncRepo.CountByTenantAndStatuses(ctx, tenantID,
		[]domain.SyncStatus{domain.SyncStatusPending, domain.SyncStatusFailed})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung pending sync: %w", err)
	}

	// Hitung revenue at risk: total outstanding untuk pelanggan isolir dan suspend
	revenueAtRisk, err := uc.sumRevenueAtRisk(ctx, tenantID)
	if err != nil {
		uc.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal menghitung revenue at risk")
		// Tidak fatal, lanjutkan dengan 0
	}

	return &domain.IsolirSummary{
		TotalIsolir:      statusCounts[domain.CustomerStatusIsolir],
		TotalSuspend:     statusCounts[domain.CustomerStatusSuspend],
		TotalPendingSync: pendingSyncCount,
		RevenueAtRisk:    revenueAtRisk,
	}, nil
}

// sumRevenueAtRisk menghitung total tagihan outstanding untuk pelanggan isolir dan suspend.
// Mengambil daftar pelanggan dengan status isolir/suspend, lalu menjumlahkan outstanding masing-masing.
func (uc *IsolirUsecase) sumRevenueAtRisk(ctx context.Context, tenantID string) (int64, error) {
	var total int64
	for _, status := range []string{"isolir", "suspend"} {
		result, err := uc.customerRepo.List(ctx, domain.CustomerListParams{
			TenantID: tenantID,
			Status:   status,
			Page:     1,
			PageSize: 50,
		})
		if err != nil {
			return 0, fmt.Errorf("gagal mengambil pelanggan %s: %w", status, err)
		}
		for _, cust := range result.Data {
			amount, err := uc.invoiceRepo.SumOutstandingAmount(ctx, cust.ID)
			if err != nil {
				uc.logger.Error().Err(err).Str("customer_id", cust.ID).Msg("gagal sum outstanding")
				continue
			}
			total += amount
		}
	}
	return total, nil
}

// GetPendingSyncs mengambil daftar pending sync dengan paginasi dan filter status.
// Digunakan oleh endpoint GET /v1/isolir/pending-syncs.
func (uc *IsolirUsecase) GetPendingSyncs(ctx context.Context, tenantID string,
	status *domain.SyncStatus, page, pageSize int) (*domain.PendingSyncListResult, error) {
	// Validasi dan bawaan paginasi
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 50 {
		pageSize = 50
	}

	result, err := uc.pendingSyncRepo.FindByTenantAndStatus(ctx, tenantID, status, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar pending sync: %w", err)
	}
	return result, nil
}

// CountOutstandingInvoices menghitung jumlah invoice outstanding untuk customer.
// Digunakan oleh handler untuk menyertakan detail pada error reactivate.
func (uc *IsolirUsecase) CountOutstandingInvoices(ctx context.Context, customerID string) (int, error) {
	return uc.invoiceRepo.CountOutstandingInvoices(ctx, customerID)
}

// SumOutstandingAmount menghitung total tagihan outstanding untuk customer.
// Digunakan oleh handler untuk menyertakan detail pada error reactivate.
func (uc *IsolirUsecase) SumOutstandingAmount(ctx context.Context, customerID string) (int64, error) {
	return uc.invoiceRepo.SumOutstandingAmount(ctx, customerID)
}
