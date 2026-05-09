// report_operational.go berisi methods ReportManager untuk laporan operasional:
// aktivitas admin, notifikasi, dan status sync.
// Dipisah dari report_manager.go agar tidak melebihi batas 200 baris per file.
package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetActivityReport mengambil laporan aktivitas admin/user.
// Data diambil dari audit_logs melalui aggregation repositori.
func (rm *ReportManager) GetActivityReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.ActivityReport, error) {
	report, err := rm.aggregationRepo.GetAdminActivity(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil activity report")
		return nil, err
	}
	return report, nil
}

// GetNotificationReport mengambil laporan statistik notifikasi.
// Data didelegasikan ke NetworkServiceClient (notification-service).
// Graceful degradation: jika service down -> module_inactive.
func (rm *ReportManager) GetNotificationReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.NotificationReport, error) {
	report, err := rm.networkClient.GetNotificationReport(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil notification report")
		return nil, err
	}
	return report, nil
}

// GetSyncReport mengambil laporan status sync MikroTik dan OLT.
// Data didelegasikan ke NetworkServiceClient (network-service).
// Graceful degradation: jika service down -> module_inactive.
func (rm *ReportManager) GetSyncReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.SyncReport, error) {
	report, err := rm.networkClient.GetSyncReport(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil sync report")
		return nil, err
	}
	return report, nil
}
