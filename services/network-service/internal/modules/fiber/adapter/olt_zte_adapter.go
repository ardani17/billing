// Package adapter - ZTEAdapter mengimplementasikan domain.OLTAdapter
// untuk OLT brand ZTE (C300, C320, C600) menggunakan SNMP + CLI.
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: pastikan ZTEAdapter mengimplementasikan domain.OLTAdapter.
var _ domain.OLTAdapter = (*ZTEAdapter)(nil)

// ZTEAdapter mengimplementasikan domain.OLTAdapter untuk brand ZTE.
// Menggunakan SNMP untuk pemantauan dan CLI untuk provisioning.
type ZTEAdapter struct {
	snmpConn domain.SNMPConnector
	cliConn  domain.CLIConnector
	snmpCfg  domain.SNMPConfig
	cliCfg   domain.CLIConfig
}

// NewZTEAdapter membuat instance baru ZTEAdapter.
func NewZTEAdapter(
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
	snmpCfg domain.SNMPConfig,
	cliCfg domain.CLIConfig,
) *ZTEAdapter {
	return &ZTEAdapter{
		snmpConn: snmpConn,
		cliConn:  cliConn,
		snmpCfg:  snmpCfg,
		cliCfg:   cliCfg,
	}
}

// GetSystemInfo mengambil informasi sistem OLT via SNMP GET sysDescr, sysUpTime, sysName.
// Parsing brand/model/firmware dari sysDescr string.
func (a *ZTEAdapter) GetSystemInfo(ctx context.Context) (*domain.OLTSystemInfo, error) {
	results, err := a.snmpConn.Get(ctx, a.snmpCfg, []string{
		zteSysDescr, zteSysUpTime, zteSysName,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal get system info: %w", err)
	}

	info := &domain.OLTSystemInfo{Brand: domain.BrandZTE}
	for _, r := range results {
		switch r.OID {
		case zteSysDescr, "." + zteSysDescr:
			info.SysDescr = snmpResultToString(r)
			info.Model, info.FirmwareVersion = zteParseSystemDescr(info.SysDescr)
		case zteSysUpTime, "." + zteSysUpTime:
			// sysUpTime dalam timeticks (1/100 detik)
			info.Uptime = snmpResultToInt64(r) / 100
		case zteSysName, "." + zteSysName:
			info.SysName = snmpResultToString(r)
		}
	}
	return info, nil
}

// Ping memeriksa konektivitas OLT via SNMP ping (GET sysUpTime).
func (a *ZTEAdapter) Ping(ctx context.Context) error {
	return a.snmpConn.Ping(ctx, a.snmpCfg)
}

// GetPONPortStatus mengambil status satu PON port via SNMP.
func (a *ZTEAdapter) GetPONPortStatus(ctx context.Context, portIndex int) (*domain.PONPortStatus, error) {
	ports, err := a.GetAllPONPorts(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range ports {
		if p.PortIndex == portIndex {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("pon port %d tidak ditemukan", portIndex)
}

// GetAllPONPorts mengambil status semua PON port via SNMP WALK ifAdminStatus/ifOperStatus.
func (a *ZTEAdapter) GetAllPONPorts(ctx context.Context) ([]domain.PONPortStatus, error) {
	adminResults, err := a.snmpConn.Walk(ctx, a.snmpCfg, zteIfAdminStatus)
	if err != nil {
		return nil, fmt.Errorf("gagal walk ifAdminStatus: %w", err)
	}
	operResults, err := a.snmpConn.Walk(ctx, a.snmpCfg, zteIfOperStatus)
	if err != nil {
		return nil, fmt.Errorf("gagal walk ifOperStatus: %w", err)
	}

	// Buat map operStatus berdasarkan OID suffix
	operMap := make(map[string]int64, len(operResults))
	for _, r := range operResults {
		suffix := oidSuffix(r.OID, zteIfOperStatus)
		operMap[suffix] = snmpResultToInt64(r)
	}

	ports := make([]domain.PONPortStatus, 0, len(adminResults))
	for i, r := range adminResults {
		suffix := oidSuffix(r.OID, zteIfAdminStatus)
		adminVal := snmpResultToInt64(r)
		operVal := operMap[suffix]

		ports = append(ports, domain.PONPortStatus{
			PortIndex:   i,
			AdminStatus: ifStatusToString(adminVal),
			OperStatus:  ifStatusToString(operVal),
			Description: fmt.Sprintf("PON Port %d", i),
		})
	}
	return ports, nil
}

// GetAlarms mengambil daftar alarm aktif dari OLT via SNMP WALK.
func (a *ZTEAdapter) GetAlarms(ctx context.Context) ([]domain.OLTAlarm, error) {
	results, err := a.snmpConn.Walk(ctx, a.snmpCfg, zteAlarmBase)
	if err != nil {
		return nil, fmt.Errorf("gagal walk alarm: %w", err)
	}

	alarms := make([]domain.OLTAlarm, 0, len(results))
	for _, r := range results {
		msg := snmpResultToString(r)
		if msg == "" {
			continue
		}
		alarms = append(alarms, domain.OLTAlarm{
			AlarmType: domain.AlarmTypeONTLOS,
			Severity:  domain.SeverityMajor,
			Message:   msg,
			Source:    domain.AlarmSourcePolling,
		})
	}
	return alarms, nil
}

// GetSFPInfo mengambil informasi SFP module pada satu PON port via SNMP.
func (a *ZTEAdapter) GetSFPInfo(ctx context.Context, portIndex int) (*domain.SFPInfo, error) {
	oltIdx, err := zteOLTIndexForPort(portIndex)
	if err != nil {
		return nil, err
	}
	oids := []string{
		fmt.Sprintf("%s.%d", zteSFPTxPower, oltIdx),
		fmt.Sprintf("%s.%d", zteSFPRxPower, oltIdx),
		fmt.Sprintf("%s.%d", zteSFPTemperature, oltIdx),
	}

	results, err := a.snmpConn.Get(ctx, a.snmpCfg, oids)
	if err != nil {
		return nil, fmt.Errorf("gagal get sfp info port %d: %w", portIndex, err)
	}

	sfp := &domain.SFPInfo{PortIndex: portIndex, SFPType: "GPON C+", Status: "normal"}
	for _, r := range results {
		switch {
		case strings.HasPrefix(r.OID, "."+zteSFPTxPower) || strings.HasPrefix(r.OID, zteSFPTxPower):
			sfp.TxPowerDBm = float64(snmpResultToInt64(r)) / 100.0
		case strings.HasPrefix(r.OID, "."+zteSFPRxPower) || strings.HasPrefix(r.OID, zteSFPRxPower):
			sfp.RxPowerDBm = float64(snmpResultToInt64(r)) / 100.0
		case strings.HasPrefix(r.OID, "."+zteSFPTemperature) || strings.HasPrefix(r.OID, zteSFPTemperature):
			sfp.Temperature = float64(snmpResultToInt64(r)) / 100.0
		}
	}

	// Klasifikasi status SFP berdasarkan suhu
	sfp.Status = classifySFPStatus(sfp.Temperature)
	return sfp, nil
}

// GetTrafficStats mengambil statistik traffic PON port via SNMP GET.
func (a *ZTEAdapter) GetTrafficStats(ctx context.Context, portIndex int) (*domain.PONTrafficStats, error) {
	oltIdx, err := zteOLTIndexForPort(portIndex)
	if err != nil {
		return nil, err
	}
	oids := []string{
		fmt.Sprintf("%s.%d", ztePONRxOctets, oltIdx),
		fmt.Sprintf("%s.%d", ztePONRxPkts, oltIdx),
		fmt.Sprintf("%s.%d", ztePONTxOctets, oltIdx),
		fmt.Sprintf("%s.%d", ztePONTxPkts, oltIdx),
	}

	results, err := a.snmpConn.Get(ctx, a.snmpCfg, oids)
	if err != nil {
		return nil, fmt.Errorf("gagal get traffic stats port %d: %w", portIndex, err)
	}

	stats := &domain.PONTrafficStats{PortIndex: portIndex}
	for _, r := range results {
		switch {
		case strings.HasPrefix(r.OID, "."+ztePONRxOctets) || strings.HasPrefix(r.OID, ztePONRxOctets):
			stats.RxBytes = snmpResultToInt64(r)
		case strings.HasPrefix(r.OID, "."+ztePONRxPkts) || strings.HasPrefix(r.OID, ztePONRxPkts):
			stats.RxPackets = snmpResultToInt64(r)
		case strings.HasPrefix(r.OID, "."+ztePONTxOctets) || strings.HasPrefix(r.OID, ztePONTxOctets):
			stats.TxBytes = snmpResultToInt64(r)
		case strings.HasPrefix(r.OID, "."+ztePONTxPkts) || strings.HasPrefix(r.OID, ztePONTxPkts):
			stats.TxPackets = snmpResultToInt64(r)
		}
	}
	return stats, nil
}
