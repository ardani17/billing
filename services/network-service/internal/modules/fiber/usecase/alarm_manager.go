// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan AlarmManager untuk manajemen alarm OLT (trap receiver + polling).
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// alarmPurgeThreshold adalah durasi retensi alarm sebelum di-purge (90 hari).
const alarmPurgeThreshold = 90 * 24 * time.Hour

// Compile-time cek: alarmManager harus mengimplementasikan domain.AlarmManager.
var _ domain.AlarmManager = (*alarmManager)(nil)

// alarmManager mengimplementasikan domain.AlarmManager.
// Mengelola alarm OLT via SNMP trap receiver dan polling.
type alarmManager struct {
	alarmRepo    domain.AlarmRepository
	oltRepo      domain.OLTRepository
	factory      domain.OLTAdapterFactory
	encryptor    domain.CredentialEncryptor
	eventPub     domain.OLTEventPublisher
	trapPort     int
	trapListener *gosnmp.TrapListener
	stopChan     chan struct{}
}

// NewAlarmManager membuat instance AlarmManager baru dengan semua dependensi.
func NewAlarmManager(
	alarmRepo domain.AlarmRepository,
	oltRepo domain.OLTRepository,
	factory domain.OLTAdapterFactory,
	encryptor domain.CredentialEncryptor,
	eventPub domain.OLTEventPublisher,
	trapPort int,
) domain.AlarmManager {
	if trapPort <= 0 {
		trapPort = 162
	}
	return &alarmManager{
		alarmRepo: alarmRepo,
		oltRepo:   oltRepo,
		factory:   factory,
		encryptor: encryptor,
		eventPub:  eventPub,
		trapPort:  trapPort,
		stopChan:  make(chan struct{}),
	}
}

// StartTrapReceiver memulai SNMP trap receiver pada port yang dikonfigurasi.
// Listener berjalan di goroutine terpisah dan memproses trap yang masuk.
func (am *alarmManager) StartTrapReceiver(ctx context.Context) error {
	am.trapListener = gosnmp.NewTrapListener()
	am.trapListener.OnNewTrap = am.handleTrap

	addr := fmt.Sprintf("0.0.0.0:%d", am.trapPort)

	go func() {
		log.Info().Str("addr", addr).Msg("memulai SNMP trap receiver")
		if err := am.trapListener.Listen(addr); err != nil {
			select {
			case <-am.stopChan:
				// Listener dihentikan secara normal
				return
			default:
				log.Error().Err(err).Msg("SNMP trap receiver gagal")
			}
		}
	}()

	// Tunggu listener siap atau context dibatalkan
	select {
	case <-am.trapListener.Listening():
		log.Info().Int("port", am.trapPort).Msg("SNMP trap receiver aktif")
		return nil
	case <-ctx.Done():
		am.trapListener.Close()
		return ctx.Err()
	case <-time.After(5 * time.Second):
		am.trapListener.Close()
		return domain.ErrTrapReceiverFailed
	}
}

// StopTrapReceiver menghentikan SNMP trap receiver.
func (am *alarmManager) StopTrapReceiver() {
	close(am.stopChan)
	if am.trapListener != nil {
		am.trapListener.Close()
		log.Info().Msg("SNMP trap receiver dihentikan")
	}
}

// PollAlarms mengambil alarm dari OLT via adapter dan menyimpan ke database.
// Mengembalikan daftar alarm baru yang ditemukan.
func (am *alarmManager) PollAlarms(ctx context.Context, oltID string) ([]domain.OLTAlarm, error) {
	olt, err := am.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return nil, err
	}

	adapter, err := am.createAdapter(olt)
	if err != nil {
		return nil, err
	}

	alarms, err := adapter.GetAlarms(ctx)
	if err != nil {
		log.Error().Err(err).Str("olt_id", oltID).Msg("gagal polling alarm dari OLT")
		return nil, err
	}

	// Simpan setiap alarm baru ke database dan terbitkan event
	for i := range alarms {
		alarms[i].Source = domain.AlarmSourcePolling
		am.saveAndPublishAlarm(ctx, olt, &alarms[i])
	}

	log.Info().Str("olt_id", oltID).Int("count", len(alarms)).Msg("alarm polling selesai")
	return alarms, nil
}

// GetAlarms mengambil daftar alarm dari database dengan filter dan paginasi.
func (am *alarmManager) GetAlarms(ctx context.Context, oltID string, params domain.AlarmListParams) (*domain.AlarmListResult, error) {
	return am.alarmRepo.List(ctx, oltID, params)
}

// PurgeOldAlarms menghapus alarm yang lebih tua dari 90 hari.
func (am *alarmManager) PurgeOldAlarms(ctx context.Context) (int64, error) {
	before := time.Now().Add(-alarmPurgeThreshold)
	count, err := am.alarmRepo.PurgeOlderThan(ctx, before)
	if err != nil {
		log.Error().Err(err).Msg("gagal purge alarm lama")
		return 0, err
	}
	if count > 0 {
		log.Info().Int64("count", count).Msg("alarm lama berhasil di-purge")
	}
	return count, nil
}
