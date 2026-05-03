// report_network.go berisi methods ReportManager untuk laporan jaringan:
// uptime, traffic, signal quality, dan kapasitas.
// Semua data didelegasikan ke NetworkServiceClient dengan graceful degradation.
package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetUptimeReport mengambil laporan uptime router dari network-service.
// Menambahkan SLA target dari KPI targets jika tersedia.
// Graceful degradation: jika network-service down → stale cache → module_inactive.
func (rm *ReportManager) GetUptimeReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.UptimeReport, error) {
	report, err := rm.networkClient.GetUptimeReport(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd, filter.RouterID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil uptime report")
		return nil, err
	}

	// Tambahkan SLA target dari KPI jika tersedia
	rm.attachUptimeKPI(ctx, tenantID, report)

	// Identifikasi router di bawah SLA target
	if report.SLATarget != nil && len(report.Routers) > 0 {
		var belowSLA []domain.RouterUptimeItem
		for _, r := range report.Routers {
			if r.UptimePercentage < *report.SLATarget {
				belowSLA = append(belowSLA, r)
			}
		}
		report.RoutersBelowSLA = belowSLA
	}

	return report, nil
}

// attachUptimeKPI menambahkan SLA uptime target dari KPI ke uptime report.
func (rm *ReportManager) attachUptimeKPI(ctx context.Context, tenantID string, report *domain.UptimeReport) {
	kpi, err := rm.kpiTargetRepo.GetByTenant(ctx, tenantID)
	if err != nil || kpi == nil || kpi.SLAUptimeTarget == nil {
		return
	}
	report.SLATarget = kpi.SLAUptimeTarget
}

// GetTrafficReport mengambil laporan traffic jaringan dari network-service.
// Graceful degradation: jika network-service down → stale cache → module_inactive.
func (rm *ReportManager) GetTrafficReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.TrafficReport, error) {
	report, err := rm.networkClient.GetTrafficReport(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd, filter.RouterID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil traffic report")
		return nil, err
	}
	return report, nil
}

// GetSignalQualityReport mengambil laporan kualitas signal OLT dari network-service.
// Graceful degradation: jika network-service down → stale cache → module_inactive.
func (rm *ReportManager) GetSignalQualityReport(ctx context.Context, tenantID string, filter domain.ReportFilter) (*domain.SignalQualityReport, error) {
	report, err := rm.networkClient.GetSignalQualityReport(ctx, tenantID, filter.PeriodStart, filter.PeriodEnd, filter.RouterID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil signal quality report")
		return nil, err
	}
	return report, nil
}

// GetCapacityReport mengambil laporan kapasitas jaringan dari network-service.
// Graceful degradation: jika network-service down → stale cache → module_inactive.
func (rm *ReportManager) GetCapacityReport(ctx context.Context, tenantID string) (*domain.CapacityReport, error) {
	report, err := rm.networkClient.GetCapacityReport(ctx, tenantID)
	if err != nil {
		rm.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil capacity report")
		return nil, err
	}
	return report, nil
}
