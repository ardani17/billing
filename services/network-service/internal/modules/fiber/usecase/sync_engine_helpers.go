// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi helper methods untuk SyncEngine: penyimpanan signal/traffic
// dan pembangunan konfigurasi SNMP dari kredensial terenkripsi.
package usecase

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// storeONTSignals menyimpan signal data untuk setiap ONT pada satu port.
func (se *syncEngine) storeONTSignals(
	ctx context.Context,
	adapter domain.OLTAdapter,
	oltID string,
	portIndex int,
	onts []domain.ONTPortStatus,
	now time.Time,
) {
	if se.signalStore == nil {
		return
	}
	for _, ont := range onts {
		signal, err := adapter.GetONTSignal(ctx, portIndex, ont.ONTIndex)
		if err != nil {
			continue // skip ONT yang gagal ambil signal
		}
		point := domain.ONTSignalPoint{
			Timestamp:   now,
			RxPowerDBm:  signal.RxPowerDBm,
			SignalLevel: signal.SignalLevel,
		}
		_ = se.signalStore.Store(ctx, oltID, portIndex, ont.ONTIndex, point)
	}
}

// storeTrafficStats mengambil dan menyimpan traffic stats untuk satu port.
func (se *syncEngine) storeTrafficStats(
	ctx context.Context,
	adapter domain.OLTAdapter,
	oltID string,
	portIndex int,
	now time.Time,
) {
	if se.trafficStore == nil {
		return
	}
	stats, err := adapter.GetTrafficStats(ctx, portIndex)
	if err != nil {
		return // skip port yang gagal ambil traffic
	}
	point := domain.PONTrafficPoint{
		Timestamp: now,
		RxBytes:   stats.RxBytes,
		RxPackets: stats.RxPackets,
		TxBytes:   stats.TxBytes,
		TxPackets: stats.TxPackets,
	}
	_ = se.trafficStore.Store(ctx, oltID, portIndex, point)
}

// buildSNMPConfig mendekripsi kredensial SNMP dan membangun SNMPConfig.
func (se *syncEngine) buildSNMPConfig(olt *domain.OLT) (domain.SNMPConfig, error) {
	cfg := domain.SNMPConfig{
		Host:    olt.Host,
		Port:    olt.SNMPPort,
		Version: olt.SNMPVersion,
		Timeout: 10 * time.Second,
	}

	if olt.SNMPVersion == domain.SNMPv2c {
		community, err := se.encryptor.Decrypt(olt.SNMPCommunityEncrypted)
		if err != nil {
			return cfg, domain.ErrDecryptionFailed
		}
		cfg.Community = community
	} else {
		cfg.Username = olt.SNMPUsername
		cfg.AuthProtocol = olt.SNMPAuthProtocol
		cfg.PrivProtocol = olt.SNMPPrivProtocol
		if olt.SNMPAuthPasswordEncrypted != "" {
			authPass, err := se.encryptor.Decrypt(olt.SNMPAuthPasswordEncrypted)
			if err != nil {
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.AuthPassword = authPass
		}
		if olt.SNMPPrivPasswordEncrypted != "" {
			privPass, err := se.encryptor.Decrypt(olt.SNMPPrivPasswordEncrypted)
			if err != nil {
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.PrivPassword = privPass
		}
	}

	return cfg, nil
}
