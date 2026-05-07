// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan SyncEngine untuk sinkronisasi periodik
// antara data OLT fisik dan database. OLT = sumber of truth untuk data fisik.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// syncInterval bawaan 30 menit untuk periodic sync.
const defaultSyncInterval = 30 * time.Minute

// Compile-time cek: syncEngine harus mengimplementasikan domain.SyncEngine.
var _ domain.SyncEngine = (*syncEngine)(nil)

// syncEngine mengimplementasikan domain.SyncEngine.
// Menjalankan sinkronisasi periodik antara OLT dan database setiap 30 menit.
type syncEngine struct {
	oltRepo       domain.OLTRepository
	ontRepo       domain.ONTRepository
	factory       domain.OLTAdapterFactory
	encryptor     domain.CredentialEncryptor
	signalStore   domain.SignalStore
	trafficStore  domain.TrafficStore
	syncInterval  time.Duration
	immediateSync bool
	cancel        context.CancelFunc
}

// NewSyncEngine membuat instance SyncEngine baru dengan semua dependensi.
func NewSyncEngine(
	oltRepo domain.OLTRepository,
	factory domain.OLTAdapterFactory,
	encryptor domain.CredentialEncryptor,
	signalStore domain.SignalStore,
	trafficStore domain.TrafficStore,
) domain.SyncEngine {
	return &syncEngine{
		oltRepo:      oltRepo,
		factory:      factory,
		encryptor:    encryptor,
		signalStore:  signalStore,
		trafficStore: trafficStore,
		syncInterval: defaultSyncInterval,
	}
}

// NewSyncEngineWithInterval membuat SyncEngine dengan interval eksplisit dari config.
// Constructor lama tetap dipertahankan untuk kompatibilitas test dan pemanggil lama.
func NewSyncEngineWithInterval(
	oltRepo domain.OLTRepository,
	factory domain.OLTAdapterFactory,
	encryptor domain.CredentialEncryptor,
	signalStore domain.SignalStore,
	trafficStore domain.TrafficStore,
	interval time.Duration,
) domain.SyncEngine {
	se := NewSyncEngine(oltRepo, factory, encryptor, signalStore, trafficStore).(*syncEngine)
	if interval > 0 {
		se.syncInterval = interval
	}
	return se
}

// SetONTRepository menghubungkan repository ONT untuk reconciliation DB vs OLT.
func (se *syncEngine) SetONTRepository(repo domain.ONTRepository) {
	se.ontRepo = repo
}

// SetImmediateSync mengaktifkan sync pertama segera setelah engine start.
func (se *syncEngine) SetImmediateSync(enabled bool) {
	se.immediateSync = enabled
}

// Start memulai goroutine periodic sync dengan ticker 30 menit.
// Pada setiap tick, ambil semua OLT online dan sync masing-masing.
func (se *syncEngine) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	se.cancel = cancel

	go se.runLoop(ctx)

	log.Info().
		Dur("interval", se.syncInterval).
		Msg("sync engine dimulai")

	return nil
}

// Stop menghentikan goroutine periodic sync.
func (se *syncEngine) Stop() {
	if se.cancel != nil {
		se.cancel()
		log.Info().Msg("sync engine dihentikan")
	}
}

// runLoop menjalankan loop periodic sync dengan ticker.
func (se *syncEngine) runLoop(ctx context.Context) {
	ticker := time.NewTicker(se.syncInterval)
	defer ticker.Stop()

	if se.immediateSync {
		se.syncAllOLTs(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			se.syncAllOLTs(ctx)
		}
	}
}

// syncAllOLTs mengambil semua OLT online dan menjalankan sync untuk masing-masing.
func (se *syncEngine) syncAllOLTs(ctx context.Context) {
	olts, err := se.oltRepo.GetOnlineOLTs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("gagal mengambil daftar OLT online untuk sync")
		return
	}

	log.Info().Int("count", len(olts)).Msg("memulai periodic sync untuk OLT online")

	for _, olt := range olts {
		if ctx.Err() != nil {
			return
		}
		if _, err := se.syncSingleOLT(ctx, olt); err != nil {
			log.Error().Err(err).
				Str("olt_id", olt.ID).
				Str("olt_name", olt.Name).
				Msg("gagal sync OLT")
		}
	}
}

// SyncOLT menjalankan sync untuk satu OLT secara manual (trigger via API).
func (se *syncEngine) SyncOLT(ctx context.Context, oltID string) (*domain.OLTSyncResult, error) {
	olt, err := se.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return nil, err
	}
	return se.syncSingleOLT(ctx, olt)
}

// syncSingleOLT menjalankan sync lengkap untuk satu OLT.
// Langkah: buat adapter -> ambil PON ports -> per port: ONT list + traffic stats
// -> per ONT: signal -> simpan signal/traffic -> bandingkan -> perbarui total_ont_count.
func (se *syncEngine) syncSingleOLT(ctx context.Context, olt *domain.OLT) (*domain.OLTSyncResult, error) {
	adapter, err := se.createAdapter(olt)
	if err != nil {
		return nil, err
	}

	// Ambil semua PON port
	ports, err := adapter.GetAllPONPorts(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var allOLTONTs []domain.ONTPortStatus
	totalONT := 0

	for _, port := range ports {
		// Ambil daftar ONT per port
		onts, ontErr := adapter.GetONTList(ctx, port.PortIndex)
		if ontErr != nil {
			log.Warn().Err(ontErr).
				Str("olt_id", olt.ID).
				Int("port", port.PortIndex).
				Msg("gagal ambil ONT list untuk port, skip")
			continue
		}
		allOLTONTs = append(allOLTONTs, onts...)
		totalONT += len(onts)

		// Simpan signal data per ONT
		se.storeONTSignals(ctx, adapter, olt.ID, port.PortIndex, onts, now)

		// Ambil dan simpan traffic stats per port
		se.storeTrafficStats(ctx, adapter, olt.ID, port.PortIndex, now)
	}

	dbONTs, err := se.loadDBONTs(ctx, olt.ID)
	if err != nil {
		log.Warn().Err(err).Str("olt_id", olt.ID).Msg("gagal load ONT DB untuk reconciliation")
	}
	comparison := compareONTSets(allOLTONTs, dbONTs)

	if updateErr := se.oltRepo.UpdateONTCounts(ctx, olt.ID, totalONT); updateErr != nil {
		log.Error().Err(updateErr).
			Str("olt_id", olt.ID).
			Msg("gagal update total_ont_count")
	}

	result := &domain.OLTSyncResult{
		OLTID:          olt.ID,
		TotalONT:       totalONT,
		UnmanagedCount: len(comparison.Unmanaged),
		MissingCount:   len(comparison.Missing),
		UpdatedCount:   len(comparison.Updated),
		SyncedAt:       now,
	}

	log.Info().
		Str("olt_id", olt.ID).
		Int("total_ont", totalONT).
		Int("unmanaged", result.UnmanagedCount).
		Int("missing", result.MissingCount).
		Int("updated", result.UpdatedCount).
		Msg("sync OLT selesai")

	return result, nil
}

func (se *syncEngine) loadDBONTs(ctx context.Context, oltID string) ([]domain.ONTPortStatus, error) {
	if se.ontRepo == nil {
		return nil, nil
	}
	result, err := se.ontRepo.List(ctx, domain.ONTListParams{
		OLTID:    oltID,
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, err
	}
	onts := make([]domain.ONTPortStatus, 0, len(result.Data))
	for _, ont := range result.Data {
		onts = append(onts, domain.ONTPortStatus{
			ONTIndex:     ont.ONTIndex,
			PONPortIndex: ont.PONPortIndex,
			SerialNumber: ont.SerialNumber,
			Name:         ont.Description,
			Status:       string(ont.Status),
		})
	}
	return onts, nil
}

// createAdapter mendekripsi kredensial dan membuat OLTAdapter via factory.
func (se *syncEngine) createAdapter(olt *domain.OLT) (domain.OLTAdapter, error) {
	snmpCfg, err := se.buildSNMPConfig(olt)
	if err != nil {
		return nil, err
	}
	cliCfg := domain.CLIConfig{} // sync hanya butuh SNMP, CLI tidak diperlukan
	return se.factory.CreateAdapter(olt.Brand, snmpCfg, cliCfg)
}
