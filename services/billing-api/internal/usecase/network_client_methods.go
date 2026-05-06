package usecase

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetUptimeReport mengambil laporan uptime router dari network-service.
// Graceful degradation: jika gagal → cache stale → module_inactive.
func (nc *NetworkClient) GetUptimeReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, routerID string) (*domain.UptimeReport, error) {
	extra := map[string]string{"router_id": routerID}
	reqURL, params := nc.buildURL("uptime", tenantID, periodStart, periodEnd, extra)

	var report domain.UptimeReport
	stale, lastUpdated, err := nc.fetchAndCache(ctx, "uptime", tenantID, reqURL, params, &report)
	if err != nil {
		return nil, err
	}

	// Jika tidak ada data dari HTTP maupun cache → module inactive
	if !stale && lastUpdated == nil && len(report.Routers) == 0 {
		return &domain.UptimeReport{ModuleInactive: true}, nil
	}

	report.StaleData = stale
	report.LastUpdated = lastUpdated
	return &report, nil
}

// GetTrafficReport mengambil laporan traffic jaringan dari network-service.
// Graceful degradation: jika gagal → cache stale → module_inactive.
func (nc *NetworkClient) GetTrafficReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, routerID string) (*domain.TrafficReport, error) {
	extra := map[string]string{"router_id": routerID}
	reqURL, params := nc.buildURL("traffic", tenantID, periodStart, periodEnd, extra)

	var report domain.TrafficReport
	stale, lastUpdated, err := nc.fetchAndCache(ctx, "traffic", tenantID, reqURL, params, &report)
	if err != nil {
		return nil, err
	}

	if !stale && lastUpdated == nil && report.TotalTrafficBytes == 0 && len(report.ByRouter) == 0 {
		return &domain.TrafficReport{ModuleInactive: true}, nil
	}

	// TrafficReport tidak punya field StaleData/LastUpdated di struct,
	// tapi data tetap valid dari cache jika stale
	_ = stale
	_ = lastUpdated
	return &report, nil
}

// GetSignalQualityReport mengambil laporan kualitas signal OLT dari network-service.
// Graceful degradation: jika gagal → cache stale → module_inactive.
func (nc *NetworkClient) GetSignalQualityReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, oltID string) (*domain.SignalQualityReport, error) {
	extra := map[string]string{"olt_id": oltID}
	reqURL, params := nc.buildURL("signal-quality", tenantID, periodStart, periodEnd, extra)

	var report domain.SignalQualityReport
	stale, lastUpdated, err := nc.fetchAndCache(ctx, "signal-quality", tenantID, reqURL, params, &report)
	if err != nil {
		return nil, err
	}

	if !stale && lastUpdated == nil && report.TotalONTCount == 0 {
		return &domain.SignalQualityReport{ModuleInactive: true}, nil
	}

	// SignalQualityReport tidak punya field StaleData/LastUpdated,
	// tapi data tetap valid dari cache jika stale
	_ = stale
	_ = lastUpdated
	return &report, nil
}

// GetCapacityReport mengambil laporan kapasitas jaringan dari network-service.
// Graceful degradation: jika gagal → cache stale → module_inactive.
func (nc *NetworkClient) GetCapacityReport(ctx context.Context, tenantID string) (*domain.CapacityReport, error) {
	reqURL, params := nc.buildURL("capacity", tenantID, time.Time{}, time.Time{}, nil)

	var report domain.CapacityReport
	stale, lastUpdated, err := nc.fetchAndCache(ctx, "capacity", tenantID, reqURL, params, &report)
	if err != nil {
		return nil, err
	}

	if !stale && lastUpdated == nil && len(report.RouterCapacity) == 0 && len(report.ODPCapacity) == 0 {
		return &domain.CapacityReport{
			ModuleInactive: map[string]bool{"mikrotik": true, "fiber_network": true, "olt": true},
		}, nil
	}

	_ = stale
	_ = lastUpdated
	return &report, nil
}

// GetSyncReport mengambil laporan status sync MikroTik dan OLT dari network-service.
// Graceful degradation: jika gagal → cache stale → module_inactive.
func (nc *NetworkClient) GetSyncReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.SyncReport, error) {
	reqURL, params := nc.buildURL("sync", tenantID, periodStart, periodEnd, nil)

	var report domain.SyncReport
	stale, lastUpdated, err := nc.fetchAndCache(ctx, "sync", tenantID, reqURL, params, &report)
	if err != nil {
		return nil, err
	}

	if !stale && lastUpdated == nil && len(report.MikrotikSync) == 0 && len(report.OLTSync) == 0 {
		return &domain.SyncReport{
			ModuleInactive: map[string]bool{"mikrotik": true, "fiber_network": true, "olt": true},
		}, nil
	}

	_ = stale
	_ = lastUpdated
	return &report, nil
}

// GetNotificationReport mengambil laporan statistik notifikasi dari network-service.
// Graceful degradation: jika gagal → cache stale → module_inactive.
func (nc *NetworkClient) GetNotificationReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*domain.NotificationReport, error) {
	reqURL, params := nc.buildURL("notification", tenantID, periodStart, periodEnd, nil)

	var report domain.NotificationReport
	stale, lastUpdated, err := nc.fetchAndCache(ctx, "notification", tenantID, reqURL, params, &report)
	if err != nil {
		return nil, err
	}

	if !stale && lastUpdated == nil && report.TotalSent == 0 && len(report.PerChannel) == 0 {
		return &domain.NotificationReport{ModuleInactive: true}, nil
	}

	_ = stale
	_ = lastUpdated
	return &report, nil
}
