// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan OLTHealthChecker untuk pemantauan periodik OLT via SNMP Ping.
package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// oltFailureThreshold adalah jumlah kegagalan berturut-turut sebelum OLT dianggap offline.
const oltFailureThreshold = 3

// Compile-time cek: oltHealthChecker harus mengimplementasikan domain.OLTHealthChecker.
var _ domain.OLTHealthChecker = (*oltHealthChecker)(nil)

// oltWorker menyimpan state goroutine health cek untuk satu OLT.
type oltWorker struct {
	cancel context.CancelFunc
	olt    *domain.OLT
}

// oltHealthChecker mengimplementasikan domain.OLTHealthChecker.
// Menjalankan satu goroutine ticker per OLT untuk pemantauan periodik via SNMP Ping.
type oltHealthChecker struct {
	oltRepo   domain.OLTRepository
	factory   domain.OLTAdapterFactory
	encryptor domain.CredentialEncryptor
	eventPub  domain.OLTEventPublisher

	mu      sync.Mutex
	workers map[string]*oltWorker // key: OLT ID
	stopped bool
}

// NewOLTHealthChecker membuat instance OLTHealthChecker baru dengan semua dependensi.
func NewOLTHealthChecker(
	oltRepo domain.OLTRepository,
	factory domain.OLTAdapterFactory,
	encryptor domain.CredentialEncryptor,
	eventPub domain.OLTEventPublisher,
) domain.OLTHealthChecker {
	return &oltHealthChecker{
		oltRepo:   oltRepo,
		factory:   factory,
		encryptor: encryptor,
		eventPub:  eventPub,
		workers:   make(map[string]*oltWorker),
	}
}

// Start memulai health cek goroutine untuk semua OLT aktif.
// Mengambil daftar OLT dari database dan menjalankan worker per OLT.
func (hc *oltHealthChecker) Start(ctx context.Context) error {
	olts, err := hc.oltRepo.GetActiveOLTs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("gagal mengambil daftar OLT aktif untuk health check")
		return err
	}

	log.Info().Int("count", len(olts)).Msg("memulai OLT health checker untuk OLT aktif")

	for _, olt := range olts {
		hc.AddOLT(olt)
	}

	return nil
}

// Stop menghentikan semua goroutine health cek OLT.
func (hc *oltHealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.stopped = true
	for id, w := range hc.workers {
		w.cancel()
		delete(hc.workers, id)
	}

	log.Info().Msg("OLT health checker dihentikan, semua goroutine dibatalkan")
}

// AddOLT menambahkan OLT baru ke health cek jadwal.
// Jika OLT sudah ada, worker lama dihentikan dan diganti yang baru.
func (hc *oltHealthChecker) AddOLT(olt *domain.OLT) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.stopped {
		return
	}

	// Hentikan worker lama jika ada
	if existing, ok := hc.workers[olt.ID]; ok {
		existing.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := &oltWorker{cancel: cancel, olt: olt}
	hc.workers[olt.ID] = w

	go hc.runWorker(ctx, olt)

	log.Info().
		Str("olt_id", olt.ID).
		Str("olt_name", olt.Name).
		Int("interval_sec", olt.HealthCheckIntervalSec).
		Msg("OLT health check worker dimulai")
}

// RemoveOLT menghapus OLT dari health cek jadwal dan menghentikan goroutine-nya.
func (hc *oltHealthChecker) RemoveOLT(oltID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if w, ok := hc.workers[oltID]; ok {
		w.cancel()
		delete(hc.workers, oltID)
		log.Info().Str("olt_id", oltID).Msg("OLT health check worker dihentikan")
	}
}

// UpdateInterval mengubah interval health cek untuk OLT tertentu.
// Menghentikan worker lama dan membuat worker baru dengan interval baru.
func (hc *oltHealthChecker) UpdateInterval(oltID string, intervalSec int) {
	hc.mu.Lock()
	w, ok := hc.workers[oltID]
	if !ok || hc.stopped {
		hc.mu.Unlock()
		return
	}

	// Salin data OLT dan perbarui interval
	oltCopy := *w.olt
	oltCopy.HealthCheckIntervalSec = intervalSec

	// Hentikan worker lama
	w.cancel()
	delete(hc.workers, oltID)
	hc.mu.Unlock()

	// Tambahkan kembali dengan interval baru
	hc.AddOLT(&oltCopy)

	log.Info().
		Str("olt_id", oltID).
		Int("interval_sec", intervalSec).
		Msg("interval OLT health check diperbarui")
}

// runWorker menjalankan loop health cek untuk satu OLT dengan ticker.
func (hc *oltHealthChecker) runWorker(ctx context.Context, olt *domain.OLT) {
	interval := time.Duration(olt.HealthCheckIntervalSec) * time.Second
	if interval <= 0 {
		interval = 300 * time.Second // bawaan 5 menit
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Jalankan health cek pertama segera
	hc.checkOLT(ctx, olt.ID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkOLT(ctx, olt.ID)
		}
	}
}
