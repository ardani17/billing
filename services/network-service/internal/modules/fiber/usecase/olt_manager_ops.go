// File ini berisi implementasi operasi pemantauan dan test connection untuk OLT Manager:
// TestSNMP, TestCLI, GetStatusSummary, GetPONPorts, GetONTList, GetSFPStatus, GetCapacity.
package usecase

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// TestSNMP menguji koneksi SNMP ke OLT dan mengembalikan system info.
// Decrypt credentials -> buat adapter via factory -> GetSystemInfo.
func (m *oltManager) TestSNMP(ctx context.Context, id string) (*domain.OLTSystemInfo, error) {
	olt, err := m.oltRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	snmpCfg, err := m.buildSNMPConfig(olt)
	if err != nil {
		return nil, err
	}

	if olt.Brand == "" {
		sysInfo, err := m.probeOLTSystemInfo(ctx, snmpCfg)
		if err != nil {
			adapter, adapterErr := m.factory.CreateAdapter(domain.BrandZTE, snmpCfg, domain.CLIConfig{})
			if adapterErr != nil {
				log.Error().Err(err).Str("olt_id", id).Msg("gagal generic SNMP probe")
				return nil, domain.ErrSNMPConnectionFailed
			}
			return adapter.GetSystemInfo(ctx)
		}
		return sysInfo, nil
	}

	cliCfg, err := m.buildCLIConfig(olt)
	if err != nil {
		return nil, err
	}

	adapter, err := m.factory.CreateAdapter(olt.Brand, snmpCfg, cliCfg)
	if err != nil {
		log.Error().Err(err).Str("olt_id", id).Msg("gagal buat adapter untuk test SNMP")
		return nil, err
	}

	sysInfo, err := adapter.GetSystemInfo(ctx)
	if err != nil {
		log.Error().Err(err).Str("olt_id", id).Msg("gagal test SNMP connection")
		return nil, domain.ErrSNMPConnectionFailed
	}

	return sysInfo, nil
}

// TestCLI menguji koneksi CLI (SSH/Telnet) ke OLT.
// Decrypt credentials -> panggil CLIConnector.TestConnection.
func (m *oltManager) TestCLI(ctx context.Context, id string) (*domain.CLITestResult, error) {
	olt, err := m.oltRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	cliCfg, err := m.buildCLIConfig(olt)
	if err != nil {
		return nil, err
	}

	banner, connErr := m.cliConn.TestConnection(ctx, cliCfg)
	if connErr != nil {
		log.Warn().Err(connErr).Str("olt_id", id).Msg("gagal test CLI connection")
		return &domain.CLITestResult{
			Success: false,
			Error:   connErr.Error(),
		}, nil
	}

	return &domain.CLITestResult{
		Success: true,
		Banner:  banner,
	}, nil
}

// GetStatusSummary mengembalikan ringkasan status semua OLT tenant.
// Menghitung total dari penjumlahan semua status counts + alarm aktif.
func (m *oltManager) GetStatusSummary(ctx context.Context) (*domain.OLTStatusSummary, error) {
	counts, err := m.oltRepo.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	alarmCount, err := m.alarmRepo.CountActiveByTenant(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("gagal hitung alarm aktif untuk summary")
		alarmCount = 0
	}

	summary := &domain.OLTStatusSummary{
		OnlineCount:      counts[domain.OLTStatusOnline],
		OfflineCount:     counts[domain.OLTStatusOffline],
		MaintenanceCount: counts[domain.OLTStatusMaintenance],
		ActiveAlarmCount: alarmCount,
	}
	summary.TotalOLTs = summary.OnlineCount + summary.OfflineCount + summary.MaintenanceCount

	return summary, nil
}

// GetPONPorts mengambil status semua PON port untuk satu OLT.
// Decrypt credentials -> buat adapter -> GetAllPONPorts.
func (m *oltManager) GetPONPorts(ctx context.Context, oltID string) ([]domain.PONPortStatus, error) {
	olt, err := m.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return nil, err
	}

	adapter, err := m.createAdapter(ctx, olt)
	if err != nil {
		return nil, err
	}

	ports, err := adapter.GetAllPONPorts(ctx)
	if err != nil {
		log.Error().Err(err).Str("olt_id", oltID).Msg("gagal ambil PON ports dari adapter")
		return nil, domain.ErrSNMPConnectionFailed
	}

	return ports, nil
}

// GetONTList mengambil daftar ONT pada satu PON port.
// Decrypt credentials -> buat adapter -> GetONTList.
func (m *oltManager) GetONTList(ctx context.Context, oltID string, portIndex int) ([]domain.ONTPortStatus, error) {
	olt, err := m.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return nil, err
	}

	adapter, err := m.createAdapter(ctx, olt)
	if err != nil {
		return nil, err
	}

	onts, err := adapter.GetONTList(ctx, portIndex)
	if err != nil {
		log.Error().Err(err).Str("olt_id", oltID).Int("port", portIndex).Msg("gagal ambil ONT list dari adapter")
		return nil, domain.ErrSNMPConnectionFailed
	}

	return onts, nil
}

// GetSFPStatus mengambil status SFP module semua PON port.
// Decrypt credentials -> buat adapter -> GetSFPInfo per port.
func (m *oltManager) GetSFPStatus(ctx context.Context, oltID string) ([]domain.SFPInfo, error) {
	olt, err := m.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return nil, err
	}

	adapter, err := m.createAdapter(ctx, olt)
	if err != nil {
		return nil, err
	}

	// Ambil SFP info untuk setiap PON port
	var sfpList []domain.SFPInfo
	for i := 0; i < olt.PONPortCount; i++ {
		sfp, sfpErr := adapter.GetSFPInfo(ctx, i)
		if sfpErr != nil {
			log.Warn().Err(sfpErr).Str("olt_id", oltID).Int("port", i).Msg("gagal ambil SFP info")
			continue
		}
		sfpList = append(sfpList, *sfp)
	}

	return sfpList, nil
}
