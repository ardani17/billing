// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan HealthChecker untuk monitoring periodik router MikroTik.
package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// failureThreshold adalah jumlah kegagalan berturut-turut sebelum router dianggap offline.
const failureThreshold = 3

// routerWorker menyimpan state goroutine health check untuk satu router.
type routerWorker struct {
	cancel context.CancelFunc
	router *domain.Router
}

// healthChecker mengimplementasikan domain.HealthChecker.
// Menjalankan satu goroutine ticker per router untuk monitoring periodik.
type healthChecker struct {
	repo           domain.RouterRepository
	poolMgr        domain.PoolManager
	metrics        domain.MetricsStore
	events         domain.EventPublisher
	crypto         domain.CredentialEncryptor
	adapterFactory AdapterFactory

	mu      sync.Mutex
	workers map[string]*routerWorker // key: router ID
	stopped bool
}

// NewHealthChecker membuat instance HealthChecker baru dengan semua dependensi.
func NewHealthChecker(
	repo domain.RouterRepository,
	poolMgr domain.PoolManager,
	metrics domain.MetricsStore,
	events domain.EventPublisher,
	crypto domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) domain.HealthChecker {
	return &healthChecker{
		repo:           repo,
		poolMgr:        poolMgr,
		metrics:        metrics,
		events:         events,
		crypto:         crypto,
		adapterFactory: adapterFactory,
		workers:        make(map[string]*routerWorker),
	}
}

// Start memulai health check goroutine untuk semua router aktif.
// Mengambil daftar router dari database dan menjalankan worker per router.
func (hc *healthChecker) Start(ctx context.Context) error {
	routers, err := hc.repo.GetActiveRouters(ctx)
	if err != nil {
		log.Error().Err(err).Msg("gagal mengambil daftar router aktif untuk health check")
		return err
	}

	log.Info().Int("count", len(routers)).Msg("memulai health checker untuk router aktif")

	for _, r := range routers {
		hc.AddRouter(r)
	}

	return nil
}

// Stop menghentikan semua goroutine health check.
func (hc *healthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.stopped = true
	for id, w := range hc.workers {
		w.cancel()
		delete(hc.workers, id)
	}

	log.Info().Msg("health checker dihentikan, semua goroutine dibatalkan")
}

// AddRouter menambahkan router baru ke health check schedule.
// Jika router sudah ada, worker lama dihentikan dan diganti yang baru.
func (hc *healthChecker) AddRouter(router *domain.Router) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.stopped {
		return
	}

	// Hentikan worker lama jika ada (misalnya saat update)
	if existing, ok := hc.workers[router.ID]; ok {
		existing.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := &routerWorker{
		cancel: cancel,
		router: router,
	}
	hc.workers[router.ID] = w

	go hc.runWorker(ctx, router)

	log.Info().
		Str("router_id", router.ID).
		Str("router_name", router.Name).
		Int("interval_sec", router.HealthCheckIntervalSec).
		Msg("health check worker dimulai untuk router")
}

// RemoveRouter menghapus router dari health check schedule dan menghentikan goroutine-nya.
func (hc *healthChecker) RemoveRouter(routerID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if w, ok := hc.workers[routerID]; ok {
		w.cancel()
		delete(hc.workers, routerID)
		log.Info().Str("router_id", routerID).Msg("health check worker dihentikan untuk router")
	}
}

// UpdateInterval mengubah interval health check untuk router tertentu.
// Menghentikan worker lama dan membuat worker baru dengan interval baru.
func (hc *healthChecker) UpdateInterval(routerID string, intervalSec int) {
	hc.mu.Lock()
	w, ok := hc.workers[routerID]
	if !ok || hc.stopped {
		hc.mu.Unlock()
		return
	}

	// Salin data router dan update interval
	routerCopy := *w.router
	routerCopy.HealthCheckIntervalSec = intervalSec

	// Hentikan worker lama
	w.cancel()
	delete(hc.workers, routerID)
	hc.mu.Unlock()

	// Tambahkan kembali dengan interval baru
	hc.AddRouter(&routerCopy)

	log.Info().
		Str("router_id", routerID).
		Int("interval_sec", intervalSec).
		Msg("interval health check diperbarui")
}

// runWorker menjalankan loop health check untuk satu router dengan ticker.
func (hc *healthChecker) runWorker(ctx context.Context, router *domain.Router) {
	interval := time.Duration(router.HealthCheckIntervalSec) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Jalankan health check pertama segera
	hc.checkRouter(ctx, router.ID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkRouter(ctx, router.ID)
		}
	}
}

// checkRouter melakukan satu kali health check untuk router tertentu.
// Skip jika status maintenance, decrypt password, connect, ambil system resource.
func (hc *healthChecker) checkRouter(ctx context.Context, routerID string) {
	// Ambil data router terbaru dari database
	router, err := hc.repo.GetByID(ctx, routerID)
	if err != nil {
		log.Error().Err(err).Str("router_id", routerID).Msg("gagal ambil data router untuk health check")
		return
	}

	// Skip jika status maintenance
	if router.Status == domain.StatusMaintenance {
		log.Debug().Str("router_id", routerID).Msg("skip health check: router dalam maintenance")
		return
	}

	// Dekripsi password untuk koneksi
	password, err := hc.crypto.Decrypt(router.PasswordEncrypted)
	if err != nil {
		log.Error().Err(err).Str("router_id", routerID).Msg("gagal dekripsi password untuk health check")
		hc.handleFailure(ctx, router)
		return
	}

	// Buat adapter sementara dan coba connect + ambil system resource
	adapter := hc.adapterFactory()
	cfg := domain.ConnectionConfig{
		Host:           router.Host,
		Port:           router.Port,
		Username:       router.Username,
		Password:       password,
		UseSSL:         router.UseSSL,
		ConnectTimeout: 10 * time.Second,
		CommandTimeout: 10 * time.Second,
	}

	if connErr := adapter.Connect(ctx, cfg); connErr != nil {
		_ = adapter.Close()
		log.Warn().Err(connErr).Str("router_id", routerID).Msg("health check gagal: koneksi gagal")
		hc.handleFailure(ctx, router)
		return
	}
	defer func() { _ = adapter.Close() }()

	// Ping untuk memastikan koneksi aktif
	if pingErr := adapter.Ping(ctx); pingErr != nil {
		log.Warn().Err(pingErr).Str("router_id", routerID).Msg("health check gagal: ping gagal")
		hc.handleFailure(ctx, router)
		return
	}

	// Ambil system resource untuk metrik
	sysRes, err := adapter.GetSystemResource(ctx)
	if err != nil {
		log.Warn().Err(err).Str("router_id", routerID).Msg("health check gagal: GetSystemResource gagal")
		hc.handleFailure(ctx, router)
		return
	}

	// Health check berhasil
	hc.handleSuccess(ctx, router, sysRes)
}

// handleSuccess memproses hasil health check yang berhasil.
// Reset failure_count, update last_checked_at, store metrics, deteksi reboot.
func (hc *healthChecker) handleSuccess(ctx context.Context, router *domain.Router, sysRes *domain.SystemResource) {
	now := time.Now()
	uptime := sysRes.Uptime

	// Siapkan update health check
	update := domain.HealthCheckUpdate{
		LastCheckedAt: &now,
		LastOnlineAt:  &now,
		LastUptimeSec: &uptime,
		FailureCount:  0,
	}
	onlineStatus := domain.StatusOnline
	update.Status = &onlineStatus

	// Jika sebelumnya offline, transisi ke online dan publish event
	wasOffline := router.Status == domain.StatusOffline
	if wasOffline {
		// Hitung downtime duration
		var downtimeDuration time.Duration
		if router.LastOnlineAt != nil {
			downtimeDuration = now.Sub(*router.LastOnlineAt)
		}

		_ = hc.events.PublishRouterOnline(ctx, router, downtimeDuration)
		log.Info().
			Str("router_id", router.ID).
			Str("router_name", router.Name).
			Dur("downtime", downtimeDuration).
			Msg("router kembali online")
	}

	// Deteksi reboot: uptime saat ini lebih kecil dari uptime sebelumnya
	if router.LastUptimeSec != nil && *router.LastUptimeSec > 0 && uptime < *router.LastUptimeSec {
		_ = hc.events.PublishUnexpectedReboot(ctx, router, *router.LastUptimeSec, uptime)
		log.Warn().
			Str("router_id", router.ID).
			Str("router_name", router.Name).
			Int64("prev_uptime", *router.LastUptimeSec).
			Int64("curr_uptime", uptime).
			Msg("reboot tak terduga terdeteksi")
	}

	// Update health check di database
	if err := hc.repo.UpdateHealthCheck(ctx, router.ID, update); err != nil {
		log.Error().Err(err).Str("router_id", router.ID).Msg("gagal update health check setelah sukses")
	}

	// Simpan metrik ke metrics store
	totalRAM := sysRes.TotalRAM
	ramUsage := 0
	if totalRAM > 0 {
		ramUsage = int(((totalRAM - sysRes.FreeRAM) * 100) / totalRAM)
	}

	routerMetrics := domain.RouterMetrics{
		CPULoad:         sysRes.CPULoad,
		RAMUsagePercent: ramUsage,
		UptimeSeconds:   sysRes.Uptime,
	}

	if err := hc.metrics.Store(ctx, router.ID, routerMetrics); err != nil {
		log.Error().Err(err).Str("router_id", router.ID).Msg("gagal simpan metrik router")
	}

	log.Debug().
		Str("router_id", router.ID).
		Int("cpu_load", sysRes.CPULoad).
		Int("ram_usage", ramUsage).
		Int64("uptime", sysRes.Uptime).
		Msg("health check berhasil")
}

// handleFailure memproses hasil health check yang gagal.
// Increment failure_count, jika >= 3 set offline dan publish event.
func (hc *healthChecker) handleFailure(ctx context.Context, router *domain.Router) {
	now := time.Now()
	newFailureCount := router.FailureCount + 1

	update := domain.HealthCheckUpdate{
		LastCheckedAt: &now,
		FailureCount:  newFailureCount,
	}

	// Jika sudah mencapai threshold, set status offline
	if newFailureCount >= failureThreshold && router.Status != domain.StatusOffline {
		offlineStatus := domain.StatusOffline
		update.Status = &offlineStatus

		_ = hc.events.PublishRouterOffline(ctx, router)
		log.Warn().
			Str("router_id", router.ID).
			Str("router_name", router.Name).
			Int("failure_count", newFailureCount).
			Msg("router dianggap offline setelah kegagalan berturut-turut")
	}

	// Update health check di database
	if err := hc.repo.UpdateHealthCheck(ctx, router.ID, update); err != nil {
		log.Error().Err(err).Str("router_id", router.ID).Msg("gagal update health check setelah gagal")
	}

	log.Debug().
		Str("router_id", router.ID).
		Int("failure_count", newFailureCount).
		Msg("health check gagal, failure count bertambah")
}
