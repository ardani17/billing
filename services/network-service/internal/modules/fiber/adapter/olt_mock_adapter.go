// Package adapter - MockOLTAdapter mengembalikan data simulasi realistis
// tanpa koneksi ke OLT fisik. Digunakan saat NETWORK_MODE=mock.
package adapter

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: pastikan MockOLTAdapter mengimplementasikan domain.OLTAdapter.
var _ domain.OLTAdapter = (*MockOLTAdapter)(nil)

// MockOLTAdapter mengimplementasikan domain.OLTAdapter dengan data simulasi.
// Semua method mengembalikan data realistis tanpa koneksi jaringan.
type MockOLTAdapter struct{}

// GetSystemInfo mengembalikan informasi sistem OLT simulasi.
// Data: brand "zte", model "C320", firmware "V2.1.0", 8 PON port, 245 ONT.
func (m *MockOLTAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	return &domain.OLTSystemInfo{
		Brand:           domain.BrandZTE,
		Model:           "C320",
		FirmwareVersion: "V2.1.0",
		Uptime:          864000, // 10 hari dalam detik
		PONPortCount:    8,
		TotalONTCount:   245,
		SysDescr:        "ZTE ZXA10 C320 Version V2.1.0",
		SysName:         "OLT-MOCK-C320",
	}, nil
}

// GetPONPortStatus mengembalikan status satu PON port simulasi.
func (m *MockOLTAdapter) GetPONPortStatus(_ context.Context, portIndex int) (*domain.PONPortStatus, error) {
	status := "up"
	if portIndex == 7 { // port terakhir simulasi down
		status = "down"
	}
	return &domain.PONPortStatus{
		PortIndex:      portIndex,
		AdminStatus:    "up",
		OperStatus:     status,
		ONTCount:       30 + portIndex,
		ONTOnlineCount: 28 + portIndex,
		Description:    fmt.Sprintf("PON Port %d", portIndex),
	}, nil
}

// GetAllPONPorts mengembalikan status semua 8 PON port simulasi.
// Port 0-6 up, port 7 down - campuran realistis.
func (m *MockOLTAdapter) GetAllPONPorts(ctx context.Context) ([]domain.PONPortStatus, error) {
	ports := make([]domain.PONPortStatus, 8)
	for i := 0; i < 8; i++ {
		p, _ := m.GetPONPortStatus(ctx, i)
		ports[i] = *p
	}
	return ports, nil
}

// GetONTList mengembalikan daftar ONT simulasi pada satu PON port.
// Mengembalikan 8 ONT per port dengan serial number dan signal realistis.
func (m *MockOLTAdapter) GetONTList(_ context.Context, portIndex int) ([]domain.ONTPortStatus, error) {
	ontCount := 8
	onts := make([]domain.ONTPortStatus, ontCount)
	for i := 0; i < ontCount; i++ {
		rxSignal := -18.0 - float64(i)*1.5 // -18 sampai -28.5 dBm
		status := "online"
		if i == ontCount-1 { // ONT terakhir offline
			status = "offline"
		}
		onts[i] = domain.ONTPortStatus{
			ONTIndex:     i,
			SerialNumber: fmt.Sprintf("ZTEG%02d%02d%04d", portIndex, i, 1000+i),
			Name:         fmt.Sprintf("ONT-%d-%d", portIndex, i),
			Status:       status,
			RxSignalDBm:  rxSignal,
			SignalLevel:  domain.ClassifySignal(rxSignal),
			Distance:     500 + i*200, // 500m sampai 1900m
			Uptime:       int64(86400 * (i + 1)),
		}
	}
	return onts, nil
}

// GetONTSignal mengembalikan informasi signal detail ONT simulasi.
// Signal berkisar -18 sampai -28 dBm tergantung ontIndex.
func (m *MockOLTAdapter) GetONTSignal(_ context.Context, portIndex int, ontIndex int) (*domain.ONTSignalInfo, error) {
	rxPower := -18.0 - float64(ontIndex)*1.5
	return &domain.ONTSignalInfo{
		ONTIndex:    ontIndex,
		RxPowerDBm:  rxPower,
		TxPowerDBm:  2.5,
		SignalLevel: domain.ClassifySignal(rxPower),
		Distance:    500 + ontIndex*200,
	}, nil
}

// GetAlarms mengembalikan daftar alarm simulasi (3 alarm sample).
func (m *MockOLTAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	port2 := 2
	ont5 := 5
	port6 := 6
	return []domain.OLTAlarm{
		{
			AlarmType:    domain.AlarmTypeONTLOS,
			Severity:     domain.SeverityCritical,
			PONPortIndex: &port2,
			ONTIndex:     &ont5,
			Message:      "ONT Loss of Signal terdeteksi pada port 2 ONT 5",
			Source:       domain.AlarmSourcePolling,
		},
		{
			AlarmType:    domain.AlarmTypeONTSignalDegraded,
			Severity:     domain.SeverityWarning,
			PONPortIndex: &port6,
			ONTIndex:     nil,
			Message:      "Signal degradasi terdeteksi pada port 6",
			Source:       domain.AlarmSourcePolling,
		},
		{
			AlarmType: domain.AlarmTypeHighTemperature,
			Severity:  domain.SeverityMinor,
			Message:   "Suhu OLT mencapai 52°C",
			Source:    domain.AlarmSourcePolling,
		},
	}, nil
}

// GetSFPInfo mengembalikan informasi SFP module simulasi pada satu PON port.
func (m *MockOLTAdapter) GetSFPInfo(_ context.Context, portIndex int) (*domain.SFPInfo, error) {
	status := "normal"
	temp := 35.0 + float64(portIndex)*2.0
	if temp > 45.0 {
		status = "warm"
	}
	return &domain.SFPInfo{
		PortIndex:   portIndex,
		TxPowerDBm:  3.5,
		RxPowerDBm:  -15.2 - float64(portIndex)*0.5,
		Temperature: temp,
		SFPType:     "GPON C+",
		Status:      status,
	}, nil
}

// GetTrafficStats mengembalikan statistik traffic simulasi pada satu PON port.
func (m *MockOLTAdapter) GetTrafficStats(_ context.Context, portIndex int) (*domain.PONTrafficStats, error) {
	base := int64(portIndex+1) * 1000000
	return &domain.PONTrafficStats{
		PortIndex: portIndex,
		RxBytes:   base * 1024,
		RxPackets: base * 10,
		TxBytes:   base * 512,
		TxPackets: base * 5,
	}, nil
}

// Ping selalu berhasil (mengembalikan nil) karena mock tidak memerlukan koneksi jaringan.
func (m *MockOLTAdapter) Ping(_ context.Context) error {
	return nil
}
