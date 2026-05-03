// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi logika health check OLT: checkOLT, handleOLTSuccess, handleOLTFailure, buildSNMPConfig.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// checkOLT melakukan satu kali health check untuk OLT tertentu.
// Skip jika status maintenance, decrypt credentials, buat adapter, SNMP Ping.
func (hc *oltHealthChecker) checkOLT(ctx context.Context, oltID string) {
	// Ambil data OLT terbaru dari database
	olt, err := hc.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		log.Error().Err(err).Str("olt_id", oltID).Msg("gagal ambil data OLT untuk health check")
		return
	}

	// Skip jika status maintenance
	if olt.Status == domain.OLTStatusMaintenance {
		log.Debug().Str("olt_id", oltID).Msg("skip health check: OLT dalam maintenance")
		return
	}

	// Bangun SNMP config (decrypt credentials)
	snmpCfg, err := hc.buildSNMPConfig(olt)
	if err != nil {
		log.Error().Err(err).Str("olt_id", oltID).Msg("gagal build SNMP config untuk health check")
		hc.handleOLTFailure(ctx, olt)
		return
	}

	// Buat adapter via factory
	adapter, err := hc.factory.CreateAdapter(olt.Brand, snmpCfg, domain.CLIConfig{})
	if err != nil {
		log.Error().Err(err).Str("olt_id", oltID).Msg("gagal membuat adapter untuk health check")
		hc.handleOLTFailure(ctx, olt)
		return
	}

	// SNMP Ping untuk cek konektivitas
	if pingErr := adapter.Ping(ctx); pingErr != nil {
		log.Warn().Err(pingErr).Str("olt_id", oltID).Msg("OLT health check gagal: SNMP ping gagal")
		hc.handleOLTFailure(ctx, olt)
		return
	}

	// Health check berhasil
	hc.handleOLTSuccess(ctx, olt)
}

// handleOLTSuccess memproses hasil health check yang berhasil.
// Reset failure_count, update last_checked_at, deteksi recovery offline→online.
func (hc *oltHealthChecker) handleOLTSuccess(ctx context.Context, olt *domain.OLT) {
	now := time.Now()

	update := domain.OLTHealthCheckUpdate{
		LastCheckedAt: &now,
		LastOnlineAt:  &now,
		FailureCount:  0,
	}

	// Jika sebelumnya offline, transisi ke online dan publish event
	wasOffline := olt.Status == domain.OLTStatusOffline
	if wasOffline {
		onlineStatus := domain.OLTStatusOnline
		update.Status = &onlineStatus

		var downtimeDuration time.Duration
		if olt.LastOnlineAt != nil {
			downtimeDuration = now.Sub(*olt.LastOnlineAt)
		}

		payload := domain.OLTDeviceOnlinePayload{
			OLTID:            olt.ID,
			OLTName:          olt.Name,
			TenantID:         olt.TenantID,
			Brand:            string(olt.Brand),
			DowntimeDuration: downtimeDuration,
		}
		_ = hc.eventPub.PublishDeviceOnline(ctx, payload)

		log.Info().
			Str("olt_id", olt.ID).
			Str("olt_name", olt.Name).
			Dur("downtime", downtimeDuration).
			Msg("OLT kembali online")
	}

	if err := hc.oltRepo.UpdateHealthCheck(ctx, olt.ID, update); err != nil {
		log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal update health check setelah sukses")
	}

	log.Debug().Str("olt_id", olt.ID).Msg("OLT health check berhasil")
}

// handleOLTFailure memproses hasil health check yang gagal.
// Increment failure_count, jika >= 3 set offline dan publish event.
func (hc *oltHealthChecker) handleOLTFailure(ctx context.Context, olt *domain.OLT) {
	now := time.Now()
	newFailureCount := olt.FailureCount + 1

	update := domain.OLTHealthCheckUpdate{
		LastCheckedAt: &now,
		FailureCount:  newFailureCount,
	}

	// Jika sudah mencapai threshold, set status offline
	if newFailureCount >= oltFailureThreshold && olt.Status != domain.OLTStatusOffline {
		offlineStatus := domain.OLTStatusOffline
		update.Status = &offlineStatus

		var lastOnline time.Time
		if olt.LastOnlineAt != nil {
			lastOnline = *olt.LastOnlineAt
		}

		payload := domain.OLTDeviceOfflinePayload{
			OLTID:        olt.ID,
			OLTName:      olt.Name,
			TenantID:     olt.TenantID,
			Brand:        string(olt.Brand),
			LastOnlineAt: lastOnline,
		}
		_ = hc.eventPub.PublishDeviceOffline(ctx, payload)

		log.Warn().
			Str("olt_id", olt.ID).
			Str("olt_name", olt.Name).
			Int("failure_count", newFailureCount).
			Msg("OLT dianggap offline setelah kegagalan berturut-turut")
	}

	if err := hc.oltRepo.UpdateHealthCheck(ctx, olt.ID, update); err != nil {
		log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal update health check setelah gagal")
	}

	log.Debug().
		Str("olt_id", olt.ID).
		Int("failure_count", newFailureCount).
		Msg("OLT health check gagal, failure count bertambah")
}

// buildSNMPConfig mendekripsi kredensial SNMP dan membangun SNMPConfig.
func (hc *oltHealthChecker) buildSNMPConfig(olt *domain.OLT) (domain.SNMPConfig, error) {
	cfg := domain.SNMPConfig{
		Host:    olt.Host,
		Port:    olt.SNMPPort,
		Version: olt.SNMPVersion,
		Timeout: 10 * time.Second,
	}

	if olt.SNMPVersion == domain.SNMPv2c {
		community, err := hc.encryptor.Decrypt(olt.SNMPCommunityEncrypted)
		if err != nil {
			return cfg, domain.ErrDecryptionFailed
		}
		cfg.Community = community
	} else {
		cfg.Username = olt.SNMPUsername
		cfg.AuthProtocol = olt.SNMPAuthProtocol
		cfg.PrivProtocol = olt.SNMPPrivProtocol
		if olt.SNMPAuthPasswordEncrypted != "" {
			authPass, err := hc.encryptor.Decrypt(olt.SNMPAuthPasswordEncrypted)
			if err != nil {
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.AuthPassword = authPass
		}
		if olt.SNMPPrivPasswordEncrypted != "" {
			privPass, err := hc.encryptor.Decrypt(olt.SNMPPrivPasswordEncrypted)
			if err != nil {
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.PrivPassword = privPass
		}
	}

	return cfg, nil
}
