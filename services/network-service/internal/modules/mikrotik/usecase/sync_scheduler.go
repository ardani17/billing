// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi SyncScheduler untuk periodic sync PPPoE user
// antara database dan router MikroTik.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// SyncScheduler menjalankan periodic sync PPPoE user untuk semua router online
// dengan service_type "pppoe". Interval dikonfigurasi via config (bawaan 15 menit).
type SyncScheduler struct {
	manager    PPPoEManager
	routerRepo domain.RouterRepository
	interval   time.Duration
	logger     zerolog.Logger
	stopCh     chan struct{}
}

// NewSyncScheduler membuat instance SyncScheduler baru.
// intervalMinutes menentukan interval sync dalam menit (bawaan 15).
func NewSyncScheduler(
	manager PPPoEManager,
	routerRepo domain.RouterRepository,
	intervalMinutes int,
	logger zerolog.Logger,
) *SyncScheduler {
	if intervalMinutes <= 0 {
		intervalMinutes = 15
	}
	return &SyncScheduler{
		manager:    manager,
		routerRepo: routerRepo,
		interval:   time.Duration(intervalMinutes) * time.Minute,
		logger:     logger.With().Str("component", "sync_scheduler").Logger(),
		stopCh:     make(chan struct{}),
	}
}

// Start memulai periodic sync dalam goroutine terpisah.
// Menggunakan time.Ticker dengan interval yang dikonfigurasi.
// Pada setiap tick: ambil router aktif, filter online+pppoe, sync per router.
func (s *SyncScheduler) Start(ctx context.Context) {
	go s.run(ctx)
	s.logger.Info().
		Dur("interval", s.interval).
		Msg("sync scheduler dimulai")
}

// Stop menghentikan periodic sync dengan menutup stopCh.
func (s *SyncScheduler) Stop() {
	close(s.stopCh)
	s.logger.Info().Msg("sync scheduler dihentikan")
}

// run menjalankan loop periodic sync. Dipanggil dalam goroutine oleh Start.
func (s *SyncScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runSync(ctx)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// runSync menjalankan satu siklus sync untuk semua router online dengan service_type pppoe.
func (s *SyncScheduler) runSync(ctx context.Context) {
	s.logger.Info().Msg("memulai periodic sync untuk semua router PPPoE")

	routers, err := s.routerRepo.GetActiveRouters(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("gagal ambil daftar router aktif")
		return
	}

	// Filter router yang online dan memiliki service_type "pppoe"
	pppoeRouters := filterPPPoERouters(routers)
	if len(pppoeRouters) == 0 {
		s.logger.Info().Msg("tidak ada router PPPoE online, skip sync")
		return
	}

	s.logger.Info().Int("router_count", len(pppoeRouters)).
		Msg("memulai sync untuk router PPPoE online")

	// Sync per router, lanjutkan meskipun ada yang gagal
	for _, router := range pppoeRouters {
		result, syncErr := s.manager.SyncRouter(ctx, router.ID)
		if syncErr != nil {
			s.logger.Error().Err(syncErr).
				Str("router_id", router.ID).
				Str("router_name", router.Name).
				Msg("gagal sync router")
			continue
		}

		s.logger.Info().
			Str("router_id", router.ID).
			Str("router_name", router.Name).
			Int("synced", result.SyncedCount).
			Int("orphan", result.OrphanCount).
			Int("missing", result.MissingCount).
			Int("out_of_sync", result.OutOfSyncCount).
			Int("error", result.ErrorCount).
			Msg("sync router selesai")
	}

	s.logger.Info().Msg("periodic sync selesai")
}

// filterPPPoERouters memfilter router yang online dan memiliki service_type "pppoe".
// Digunakan oleh SyncScheduler untuk menentukan router mana yang perlu di-sync.
func filterPPPoERouters(routers []*domain.Router) []*domain.Router {
	var result []*domain.Router
	for _, r := range routers {
		if r.Status != domain.StatusOnline {
			continue
		}
		if !hasServiceType(r.ServiceTypes, "pppoe") {
			continue
		}
		result = append(result, r)
	}
	return result
}
