// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi handler trap SNMP dan helper saveAndPublishAlarm untuk alarm manager.
package usecase

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/gosnmp/gosnmp"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// handleTrap memproses SNMP trap yang diterima dari OLT.
// Parsing PDU untuk menentukan tipe alarm, severity, dan informasi terkait.
func (am *alarmManager) handleTrap(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
	ctx := context.Background()
	sourceIP := addr.IP.String()

	alarm := parseTrapPDU(packet.Variables, sourceIP)
	if alarm == nil {
		log.Debug().Str("source", sourceIP).Msg("trap diterima tapi tidak dapat di-parse")
		return
	}

	olt, err := am.findOLTByTrapSource(ctx, sourceIP)
	if err != nil {
		log.Warn().Err(err).Str("source", sourceIP).Msg("trap diterima dari source yang tidak terdaftar")
		return
	}

	if alarm.Severity == domain.SeverityClear {
		am.clearMatchingTrapAlarm(ctx, olt, alarm)
		return
	}

	if am.hasActiveTrapAlarm(ctx, olt.ID, alarm) {
		log.Info().
			Str("source", sourceIP).
			Str("olt_id", olt.ID).
			Str("alarm_type", alarm.AlarmType).
			Msg("trap alarm aktif sudah ada, skip duplikat")
		return
	}

	record := &domain.OLTAlarmRecord{
		ID:           uuid.New().String(),
		TenantID:     olt.TenantID,
		OLTID:        olt.ID,
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

	// Terbitkan event alarm (best-effort)
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

	log.Info().
		Str("source", sourceIP).
		Str("olt_id", olt.ID).
		Str("alarm_type", alarm.AlarmType).
		Str("severity", alarm.Severity).
		Msg("alarm dari trap berhasil diproses")
}

func (am *alarmManager) findOLTByTrapSource(ctx context.Context, sourceIP string) (*domain.OLT, error) {
	if lookup, ok := am.oltRepo.(interface {
		GetByHost(context.Context, string) (*domain.OLT, error)
	}); ok {
		return lookup.GetByHost(ctx, sourceIP)
	}
	olts, err := am.oltRepo.GetActiveOLTs(ctx)
	if err != nil {
		return nil, err
	}
	for _, olt := range olts {
		if olt.Host == sourceIP {
			return olt, nil
		}
	}
	return nil, domain.ErrOLTNotFound
}

func (am *alarmManager) hasActiveTrapAlarm(ctx context.Context, oltID string, alarm *domain.OLTAlarm) bool {
	result, err := am.alarmRepo.List(ctx, oltID, domain.AlarmListParams{
		Page:     1,
		PageSize: 100,
		Status:   domain.AlarmStatusActive,
	})
	if err != nil {
		log.Warn().Err(err).Str("olt_id", oltID).Msg("gagal cek duplikat trap alarm")
		return false
	}
	for _, existing := range result.Data {
		if sameAlarmFingerprint(existing, alarm) {
			return true
		}
	}
	return false
}

func (am *alarmManager) clearMatchingTrapAlarm(ctx context.Context, olt *domain.OLT, alarm *domain.OLTAlarm) {
	result, err := am.alarmRepo.List(ctx, olt.ID, domain.AlarmListParams{
		Page:     1,
		PageSize: 100,
		Status:   domain.AlarmStatusActive,
	})
	if err != nil {
		log.Warn().Err(err).Str("olt_id", olt.ID).Msg("gagal mencari alarm aktif untuk clear trap")
		return
	}
	for _, existing := range result.Data {
		if sameAlarmFingerprint(existing, alarm) {
			if err := am.alarmRepo.ClearAlarm(ctx, existing.ID); err != nil {
				log.Warn().Err(err).Str("alarm_id", existing.ID).Msg("gagal clear trap alarm")
				return
			}
			_ = am.eventPub.PublishAlarm(ctx, domain.OLTAlarmPayload{
				CorrelationID: uuid.New().String(),
				OLTID:         olt.ID,
				OLTName:       olt.Name,
				TenantID:      olt.TenantID,
				AlarmType:     alarm.AlarmType,
				Severity:      alarm.Severity,
				PONPortIndex:  alarm.PONPortIndex,
				ONTIndex:      alarm.ONTIndex,
				Message:       alarm.Message,
			})
			log.Info().Str("olt_id", olt.ID).Str("alarm_id", existing.ID).Msg("trap alarm berhasil di-clear")
			return
		}
	}
	log.Info().Str("olt_id", olt.ID).Str("alarm_type", alarm.AlarmType).Msg("clear trap diterima tanpa alarm aktif")
}

func sameAlarmFingerprint(record *domain.OLTAlarmRecord, alarm *domain.OLTAlarm) bool {
	if record.AlarmType != alarm.AlarmType {
		return false
	}
	if !sameOptionalInt(record.PONPortIndex, alarm.PONPortIndex) {
		return false
	}
	return sameOptionalInt(record.ONTIndex, alarm.ONTIndex)
}

func sameOptionalInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// saveAndPublishAlarm menyimpan alarm ke database dan terbitkan event.
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
