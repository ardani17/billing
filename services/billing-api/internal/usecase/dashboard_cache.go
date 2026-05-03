// dashboard_cache.go berisi DashboardCache yang mengelola cache Redis
// untuk data dashboard widget. TTL 5 menit untuk response cepat.
// Key format: report:dashboard:{tenant_id}
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// dashboardCacheTTL adalah TTL cache Redis untuk dashboard data (5 menit).
const dashboardCacheTTL = 5 * time.Minute

// dashboardCacheKeyPrefix adalah prefix key Redis untuk dashboard cache.
const dashboardCacheKeyPrefix = "report:dashboard:"

// DashboardCache mengelola cache data dashboard widget.
type DashboardCache struct {
	aggregationRepo domain.ReportAggregationRepository
	networkClient   domain.NetworkServiceClient
	kpiTargetRepo   domain.KPITargetRepository
	redisClient     *redis.Client
	logger          zerolog.Logger
}

// NewDashboardCache membuat instance baru DashboardCache.
func NewDashboardCache(
	aggregationRepo domain.ReportAggregationRepository,
	networkClient domain.NetworkServiceClient,
	kpiTargetRepo domain.KPITargetRepository,
	redisClient *redis.Client,
	logger zerolog.Logger,
) *DashboardCache {
	return &DashboardCache{
		aggregationRepo: aggregationRepo,
		networkClient:   networkClient,
		kpiTargetRepo:   kpiTargetRepo,
		redisClient:     redisClient,
		logger:          logger.With().Str("component", "dashboard_cache").Logger(),
	}
}

// dashboardCacheKey menghasilkan Redis key untuk dashboard cache tenant.
func dashboardCacheKey(tenantID string) string {
	return fmt.Sprintf("%s%s", dashboardCacheKeyPrefix, tenantID)
}

// GetDashboardData mengambil data dashboard widget.
// Flow: cek Redis cache → jika hit → return cached → jika miss →
// query aggregation + network data → assemble → store cache → return.
func (dc *DashboardCache) GetDashboardData(ctx context.Context, tenantID string) (*domain.DashboardData, error) {
	key := dashboardCacheKey(tenantID)

	// Cek cache Redis
	cached, err := dc.redisClient.Get(ctx, key).Bytes()
	if err == nil {
		var data domain.DashboardData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, nil
		}
		dc.logger.Warn().Err(err).Str("key", key).Msg("gagal parse cached dashboard data")
	}

	// Cache miss — query data dari aggregation repo
	data, err := dc.aggregationRepo.GetDashboardData(ctx, tenantID)
	if err != nil {
		dc.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil dashboard data")
		return nil, err
	}

	// Ambil data network (router online/offline) dari network client
	dc.enrichWithNetworkData(ctx, tenantID, data)

	// Tambahkan KPI targets jika tersedia
	dc.enrichWithKPITargets(ctx, tenantID, data)

	// Simpan ke Redis cache
	if cacheData, err := json.Marshal(data); err == nil {
		if err := dc.redisClient.Set(ctx, key, cacheData, dashboardCacheTTL).Err(); err != nil {
			dc.logger.Warn().Err(err).Str("key", key).Msg("gagal menyimpan dashboard cache")
		}
	}

	return data, nil
}

// enrichWithNetworkData menambahkan data network ke dashboard.
func (dc *DashboardCache) enrichWithNetworkData(ctx context.Context, tenantID string, data *domain.DashboardData) {
	capacity, err := dc.networkClient.GetCapacityReport(ctx, tenantID)
	if err != nil || capacity == nil {
		if data.ModuleInactive == nil {
			data.ModuleInactive = make(map[string]bool)
		}
		data.ModuleInactive["mikrotik"] = true
		return
	}

	// Hitung router online/offline dari capacity data
	for _, r := range capacity.RouterCapacity {
		if r.CurrentCustomers > 0 {
			data.RoutersOnline++
		} else {
			data.RoutersOffline++
		}
	}
}

// enrichWithKPITargets menambahkan KPI targets ke dashboard data.
func (dc *DashboardCache) enrichWithKPITargets(ctx context.Context, tenantID string, data *domain.DashboardData) {
	kpi, err := dc.kpiTargetRepo.GetByTenant(ctx, tenantID)
	if err != nil || kpi == nil {
		return
	}

	if kpi.MonthlyRevenueTarget != nil {
		data.RevenueTarget = kpi.MonthlyRevenueTarget
		if *kpi.MonthlyRevenueTarget > 0 {
			progress := float64(data.MonthlyRevenue) / float64(*kpi.MonthlyRevenueTarget) * 100
			data.RevenueProgress = &progress
		}
	}
	if kpi.CollectionRateTarget != nil {
		data.CollectionTarget = kpi.CollectionRateTarget
	}
	if kpi.MaxChurnRate != nil {
		data.ChurnTarget = kpi.MaxChurnRate
	}
}

// InvalidateCache menghapus cache dashboard untuk tenant tertentu.
// Dipanggil saat data berubah (pembayaran baru, pelanggan baru, dll).
func (dc *DashboardCache) InvalidateCache(ctx context.Context, tenantID string) error {
	key := dashboardCacheKey(tenantID)
	if err := dc.redisClient.Del(ctx, key).Err(); err != nil {
		dc.logger.Warn().Err(err).Str("key", key).Msg("gagal menghapus dashboard cache")
		return err
	}
	return nil
}
