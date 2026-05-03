// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan struct oltManager, constructor, dan helper methods.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time check: oltManager harus mengimplementasikan domain.OLTManager.
var _ domain.OLTManager = (*oltManager)(nil)

// oltManager mengimplementasikan domain.OLTManager.
// Mengelola business logic CRUD OLT, auto-detect, test connection, dan status summary.
type oltManager struct {
	oltRepo       domain.OLTRepository
	odpRepo       domain.ODPRepository
	alarmRepo     domain.AlarmRepository
	factory       domain.OLTAdapterFactory
	snmpConn      domain.SNMPConnector
	cliConn       domain.CLIConnector
	encryptor     domain.CredentialEncryptor
	eventPub      domain.OLTEventPublisher
	signalStore   domain.SignalStore
	trafficStore  domain.TrafficStore
	healthChecker domain.OLTHealthChecker // opsional, di-set setelah konstruksi
}

// NewOLTManager membuat instance OLTManager baru dengan semua dependensi.
func NewOLTManager(
	oltRepo domain.OLTRepository,
	odpRepo domain.ODPRepository,
	alarmRepo domain.AlarmRepository,
	factory domain.OLTAdapterFactory,
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
	encryptor domain.CredentialEncryptor,
	eventPub domain.OLTEventPublisher,
	signalStore domain.SignalStore,
	trafficStore domain.TrafficStore,
) domain.OLTManager {
	return &oltManager{
		oltRepo:      oltRepo,
		odpRepo:      odpRepo,
		alarmRepo:    alarmRepo,
		factory:      factory,
		snmpConn:     snmpConn,
		cliConn:      cliConn,
		encryptor:    encryptor,
		eventPub:     eventPub,
		signalStore:  signalStore,
		trafficStore: trafficStore,
	}
}

// SetHealthChecker mengatur health checker setelah konstruksi.
// Dipanggil dari wiring di main.go karena circular dependency.
func (m *oltManager) SetHealthChecker(hc domain.OLTHealthChecker) {
	m.healthChecker = hc
}

// buildSNMPConfig mendekripsi kredensial SNMP dan membangun SNMPConfig.
func (m *oltManager) buildSNMPConfig(olt *domain.OLT) (domain.SNMPConfig, error) {
	cfg := domain.SNMPConfig{
		Host:    olt.Host,
		Port:    olt.SNMPPort,
		Version: olt.SNMPVersion,
		Timeout: 10 * time.Second,
	}

	if olt.SNMPVersion == domain.SNMPv2c {
		community, err := m.encryptor.Decrypt(olt.SNMPCommunityEncrypted)
		if err != nil {
			log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal dekripsi SNMP community")
			return cfg, domain.ErrDecryptionFailed
		}
		cfg.Community = community
	} else {
		cfg.Username = olt.SNMPUsername
		cfg.AuthProtocol = olt.SNMPAuthProtocol
		cfg.PrivProtocol = olt.SNMPPrivProtocol
		if olt.SNMPAuthPasswordEncrypted != "" {
			authPass, err := m.encryptor.Decrypt(olt.SNMPAuthPasswordEncrypted)
			if err != nil {
				log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal dekripsi SNMP auth password")
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.AuthPassword = authPass
		}
		if olt.SNMPPrivPasswordEncrypted != "" {
			privPass, err := m.encryptor.Decrypt(olt.SNMPPrivPasswordEncrypted)
			if err != nil {
				log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal dekripsi SNMP priv password")
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.PrivPassword = privPass
		}
	}

	return cfg, nil
}

// buildCLIConfig mendekripsi kredensial CLI dan membangun CLIConfig.
func (m *oltManager) buildCLIConfig(olt *domain.OLT) (domain.CLIConfig, error) {
	password, err := m.encryptor.Decrypt(olt.CLIPasswordEncrypted)
	if err != nil {
		log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal dekripsi CLI password")
		return domain.CLIConfig{}, domain.ErrDecryptionFailed
	}

	cfg := domain.CLIConfig{
		Host:        olt.Host,
		Port:        olt.CLIPort,
		Protocol:    olt.CLIProtocol,
		Username:    olt.CLIUsername,
		Password:    password,
		ConnTimeout: 10 * time.Second,
		CmdTimeout:  30 * time.Second,
	}

	if olt.CLIEnablePasswordEncrypted != "" {
		enablePass, err := m.encryptor.Decrypt(olt.CLIEnablePasswordEncrypted)
		if err != nil {
			log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal dekripsi CLI enable password")
			return domain.CLIConfig{}, domain.ErrDecryptionFailed
		}
		cfg.EnablePassword = enablePass
	}

	return cfg, nil
}

// oltToResponse mengkonversi entity OLT ke OLTResponse (tanpa kredensial).
func oltToResponse(olt *domain.OLT) *domain.OLTResponse {
	return &domain.OLTResponse{
		ID:                     olt.ID,
		Name:                   olt.Name,
		Host:                   olt.Host,
		Brand:                  olt.Brand,
		Model:                  olt.Model,
		FirmwareVersion:        olt.FirmwareVersion,
		PONPortCount:           olt.PONPortCount,
		TotalONTCount:          olt.TotalONTCount,
		Status:                 olt.Status,
		HealthCheckIntervalSec: olt.HealthCheckIntervalSec,
		LastOnlineAt:           olt.LastOnlineAt,
		Notes:                  olt.Notes,
		CreatedAt:              olt.CreatedAt,
		UpdatedAt:              olt.UpdatedAt,
	}
}

// createAdapter membuat OLTAdapter dari OLT entity (decrypt + factory).
func (m *oltManager) createAdapter(ctx context.Context, olt *domain.OLT) (domain.OLTAdapter, error) {
	snmpCfg, err := m.buildSNMPConfig(olt)
	if err != nil {
		return nil, err
	}
	cliCfg, err := m.buildCLIConfig(olt)
	if err != nil {
		return nil, err
	}
	adapter, err := m.factory.CreateAdapter(olt.Brand, snmpCfg, cliCfg)
	if err != nil {
		log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal membuat adapter OLT")
		return nil, err
	}
	return adapter, nil
}
