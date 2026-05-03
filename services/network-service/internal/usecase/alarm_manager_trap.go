// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi handler trap SNMP dan helper saveAndPublishAlarm untuk alarm manager.
package usecase

import (
	"context"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// handleTrap memproses SNMP trap yang diterima dari OLT.
// Parse PDU untuk menentukan tipe alarm, severity, dan informasi terkait.
func (am *alarmManager) handleTrap(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
	ctx := context.Background()
	sourceIP := addr.IP.String()

	alarm := parseTrapPDU(packet.Variables, sourceIP)
	if alarm == nil {
		log.Debug().Str("source", sourceIP).Msg("trap diterima tapi tidak dapat di-parse")
		return
	}

	// Simpan alarm ke database (tanpa tenant/olt_id karena trap tidak membawa info tersebut)
	record := &domain.OLTAlarmRecord{
		ID:           uuid.New().String(),
		AlarmType:    alarm.AlarmType,
		Severity:     alarm.Severity,
		PONPortIndex: alarm.PONPortIndex,
		ONTIndex:     alarm.ONTIndex,
		Message:      alarm.Message,
		Source:       domain.AlarmSourceTrap,
		Status:       domain.AlarmStatusActive,
		CreatedAt:    time.Now(),
	}

	if _, err := am.alarmRepo.Create(ctx, record); err != nil {
		log.Error().Err(err).Str("source", sourceIP).Msg("gagal menyimpan alarm dari trap")
		return
	}

	// Publish event alarm (best-effort)
	payload := domain.OLTAlarmPayload{
		CorrelationID: uuid.New().String(),
		AlarmType:     alarm.AlarmType,
		Severity:      alarm.Severity,
		PONPortIndex:  alarm.PONPortIndex,
		ONTIndex:      alarm.ONTIndex,
		Message:       alarm.Message,
	}
	_ = am.eventPub.PublishAlarm(ctx, payload)

	log.Info().
		Str("source", sourceIP).
		Str("alarm_type", alarm.AlarmType).
		Str("severity", alarm.Severity).
		Msg("alarm dari trap berhasil diproses")
}

// saveAndPublishAlarm menyimpan alarm ke database dan publish event.
func (am *alarmManager) saveAndPublishAlarm(ctx context.Context, olt *domain.OLT, alarm *domain.OLTAlarm) {
	record := &domain.OLTAlarmRecord{
		ID:           uuid.New().String(),
		TenantID:     olt.TenantID,
		OLTID:        olt.ID,
		AlarmType:    alarm.AlarmType,
		Severity:     alarm.Severity,
		PONPortIndex: alarm.PONPortIndex,
		ONTIndex:     alarm.ONTIndex,
		Message:      alarm.Message,
		Source:       alarm.Source,
		Status:       domain.AlarmStatusActive,
		CreatedAt:    time.Now(),
	}

	if _, err := am.alarmRepo.Create(ctx, record); err != nil {
		log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal menyimpan alarm")
		return
	}

	payload := domain.OLTAlarmPayload{
		CorrelationID: uuid.New().String(),
		OLTID:         olt.ID,
		OLTName:       olt.Name,
		TenantID:      olt.TenantID,
		AlarmType:     alarm.AlarmType,
		Severity:      alarm.Severity,
		PONPortIndex:  alarm.PONPortIndex,
		ONTIndex:      alarm.ONTIndex,
		Message:       alarm.Message,
	}
	_ = am.eventPub.PublishAlarm(ctx, payload)
}

// createAdapter mendekripsi kredensial dan membuat OLTAdapter dari OLT entity.
func (am *alarmManager) createAdapter(olt *domain.OLT) (domain.OLTAdapter, error) {
	snmpCfg, err := am.buildSNMPConfig(olt)
	if err != nil {
		return nil, err
	}
	cliCfg, err := am.buildCLIConfig(olt)
	if err != nil {
		return nil, err
	}
	return am.factory.CreateAdapter(olt.Brand, snmpCfg, cliCfg)
}

// buildSNMPConfig mendekripsi kredensial SNMP dan membangun SNMPConfig.
func (am *alarmManager) buildSNMPConfig(olt *domain.OLT) (domain.SNMPConfig, error) {
	cfg := domain.SNMPConfig{
		Host:    olt.Host,
		Port:    olt.SNMPPort,
		Version: olt.SNMPVersion,
		Timeout: 10 * time.Second,
	}
	if olt.SNMPVersion == domain.SNMPv2c {
		community, err := am.encryptor.Decrypt(olt.SNMPCommunityEncrypted)
		if err != nil {
			return cfg, domain.ErrDecryptionFailed
		}
		cfg.Community = community
	} else {
		cfg.Username = olt.SNMPUsername
		cfg.AuthProtocol = olt.SNMPAuthProtocol
		cfg.PrivProtocol = olt.SNMPPrivProtocol
		if olt.SNMPAuthPasswordEncrypted != "" {
			authPass, err := am.encryptor.Decrypt(olt.SNMPAuthPasswordEncrypted)
			if err != nil {
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.AuthPassword = authPass
		}
		if olt.SNMPPrivPasswordEncrypted != "" {
			privPass, err := am.encryptor.Decrypt(olt.SNMPPrivPasswordEncrypted)
			if err != nil {
				return cfg, domain.ErrDecryptionFailed
			}
			cfg.PrivPassword = privPass
		}
	}
	return cfg, nil
}

// buildCLIConfig mendekripsi kredensial CLI dan membangun CLIConfig.
func (am *alarmManager) buildCLIConfig(olt *domain.OLT) (domain.CLIConfig, error) {
	password, err := am.encryptor.Decrypt(olt.CLIPasswordEncrypted)
	if err != nil {
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
		enablePass, err := am.encryptor.Decrypt(olt.CLIEnablePasswordEncrypted)
		if err != nil {
			return domain.CLIConfig{}, domain.ErrDecryptionFailed
		}
		cfg.EnablePassword = enablePass
	}
	return cfg, nil
}
