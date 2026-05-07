// Package adapter - ZTEAdapter method untuk ONT: GetONTList dan GetONTSignal.
// Dipisah dari olt_zte_adapter.go agar setiap file di bawah 200 baris.
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetONTList mengambil daftar ONT pada satu PON port via SNMP WALK.
// Walk pada ONU management base OID untuk mendapatkan serial number, nama, dan status.
func (a *ZTEAdapter) GetONTList(ctx context.Context, portIndex int) ([]domain.ONTPortStatus, error) {
	oltIdx, err := zteOLTIndexForPort(portIndex)
	if err != nil {
		return nil, err
	}
	walkOID := fmt.Sprintf("%s.5.%d", zteONUMgmtBase, oltIdx)

	results, err := a.snmpConn.Walk(ctx, a.snmpCfg, walkOID)
	if err != nil {
		return nil, fmt.Errorf("gagal walk ont list port %d: %w", portIndex, err)
	}

	onts := make([]domain.ONTPortStatus, 0, len(results))
	for i, r := range results {
		sn := snmpResultToString(r)
		if sn == "" {
			continue
		}
		onts = append(onts, domain.ONTPortStatus{
			PONPortIndex: portIndex,
			ONTIndex:     i,
			SerialNumber: sn,
			Status:       "online",
		})
	}

	// Ambil nama ONT jika tersedia
	a.enrichONTNames(ctx, portIndex, oltIdx, onts)

	return onts, nil
}

// enrichONTNames mengisi field Name pada daftar ONT via SNMP WALK.
func (a *ZTEAdapter) enrichONTNames(ctx context.Context, portIndex, oltIdx int, onts []domain.ONTPortStatus) {
	nameOID := fmt.Sprintf("%s.2.%d", zteONUMgmtBase, oltIdx)
	nameResults, err := a.snmpConn.Walk(ctx, a.snmpCfg, nameOID)
	if err != nil {
		return // best-effort, tidak fatal jika gagal
	}
	for i, r := range nameResults {
		if i < len(onts) {
			onts[i].Name = snmpResultToString(r)
		}
	}
}

// GetONTSignal mengambil informasi signal detail ONT via SNMP GET.
// Mengambil RX power, TX power, dan distance untuk ONT tertentu.
func (a *ZTEAdapter) GetONTSignal(ctx context.Context, portIndex int, ontIndex int) (*domain.ONTSignalInfo, error) {
	oltIdx, err := zteOLTIndexForPort(portIndex)
	if err != nil {
		return nil, err
	}
	oids := []string{
		fmt.Sprintf("%s.%d.%d", zteONURxPower, oltIdx, ontIndex),
		fmt.Sprintf("%s.%d.%d", zteONUTxPower, oltIdx, ontIndex),
		fmt.Sprintf("%s.%d.%d", zteONUDistance, oltIdx, ontIndex),
	}

	results, err := a.snmpConn.Get(ctx, a.snmpCfg, oids)
	if err != nil {
		return nil, fmt.Errorf("gagal get ont signal port %d ont %d: %w", portIndex, ontIndex, err)
	}

	signal := &domain.ONTSignalInfo{ONTIndex: ontIndex}
	for _, r := range results {
		switch {
		case strings.HasPrefix(r.OID, "."+zteONURxPower) || strings.HasPrefix(r.OID, zteONURxPower):
			// ZTE mengembalikan signal dalam 0.01 dBm
			signal.RxPowerDBm = float64(snmpResultToInt64(r)) / 100.0
		case strings.HasPrefix(r.OID, "."+zteONUTxPower) || strings.HasPrefix(r.OID, zteONUTxPower):
			signal.TxPowerDBm = float64(snmpResultToInt64(r)) / 100.0
		case strings.HasPrefix(r.OID, "."+zteONUDistance) || strings.HasPrefix(r.OID, zteONUDistance):
			signal.Distance = int(snmpResultToInt64(r))
		}
	}

	signal.SignalLevel = domain.ClassifySignal(signal.RxPowerDBm)
	return signal, nil
}
