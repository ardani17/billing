package adapter

import (
	"context"
	"fmt"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// --- Tes OID Constants ---

func TestZTEOIDConstants(t *testing.T) {
	// Verifikasi OID standar
	tests := []struct {
		name     string
		oid      string
		expected string
	}{
		{"sysDescr", zteSysDescr, "1.3.6.1.2.1.1.1.0"},
		{"sysUpTime", zteSysUpTime, "1.3.6.1.2.1.1.3.0"},
		{"sysName", zteSysName, "1.3.6.1.2.1.1.5.0"},
		{"ifAdminStatus", zteIfAdminStatus, "1.3.6.1.2.1.2.2.1.7"},
		{"ifOperStatus", zteIfOperStatus, "1.3.6.1.2.1.2.2.1.8"},
		{"ONUSerialNumber", zteONUSerialNumber, "1.3.6.1.4.1.3902.1012.3.28.1.1.5"},
		{"ONUName", zteONUName, "1.3.6.1.4.1.3902.1012.3.28.1.1.2"},
		{"ONUDistance", zteONUDistance, "1.3.6.1.4.1.3902.1012.3.11.4.1.2"},
		{"PONRxOctets", ztePONRxOctets, "1.3.6.1.4.1.3902.1015.1010.5.4.1.2"},
		{"PONTxOctets", ztePONTxOctets, "1.3.6.1.4.1.3902.1015.1010.5.4.1.17"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.oid != tc.expected {
				t.Errorf("OID %s: got %q, want %q", tc.name, tc.oid, tc.expected)
			}
		})
	}
}

// --- Tes ZTE Index Calculation ---

func TestZTECalculateOLTIndex(t *testing.T) {
	tests := []struct {
		board, pon int
		expected   int
	}{
		{0, 0, 1 << 28},                          // 268435456
		{0, 1, (1 << 28) | (1 << 8)},             // 268435712
		{1, 0, (1 << 28) | (1 << 16)},            // 268500992
		{1, 3, (1 << 28) | (1 << 16) | (3 << 8)}, // 268501760
	}
	for _, tc := range tests {
		name := fmt.Sprintf("board%d_pon%d", tc.board, tc.pon)
		t.Run(name, func(t *testing.T) {
			got := zteCalculateOLTIndex(tc.board, tc.pon)
			if got != tc.expected {
				t.Errorf("zteCalculateOLTIndex(%d, %d) = %d, want %d",
					tc.board, tc.pon, got, tc.expected)
			}
		})
	}
}

func TestZTEAddressMapper_ResearchExamples(t *testing.T) {
	tests := []struct {
		name      string
		board     int
		pon       int
		expected  int
		portIndex int
	}{
		{name: "board1_pon1", board: 1, pon: 1, expected: 268501248, portIndex: 0},
		{name: "board1_pon2", board: 1, pon: 2, expected: 268501504, portIndex: 1},
		{name: "board2_pon1", board: 2, pon: 1, expected: 268566784, portIndex: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := zteCalculateOLTIndex(tc.board, tc.pon); got != tc.expected {
				t.Fatalf("zteCalculateOLTIndex(%d, %d) = %d, want %d", tc.board, tc.pon, got, tc.expected)
			}
			if tc.portIndex >= 0 {
				got, err := zteOLTIndexForPort(tc.portIndex)
				if err != nil {
					t.Fatalf("zteOLTIndexForPort(%d) error: %v", tc.portIndex, err)
				}
				if got != tc.expected {
					t.Fatalf("zteOLTIndexForPort(%d) = %d, want %d", tc.portIndex, got, tc.expected)
				}
			}
		})
	}
}

// --- Tes sysDescr Parsing ---

func TestZTEParseSystemDescr(t *testing.T) {
	tests := []struct {
		sysDescr         string
		expectedModel    string
		expectedFirmware string
	}{
		{
			"ZTE ZXA10 C320 Version V2.1.0 Software",
			"C320", "V2.1.0",
		},
		{
			"ZTE ZXA10 C300 V1.2.5P3",
			"C300", "V1.2.5P3",
		},
		{
			"ZTE ZXA10 C600 Version V3.0.0",
			"C600", "V3.0.0",
		},
		{
			"ZTE Unknown OLT V4.0.0",
			"", "V4.0.0",
		},
		{
			"Some random string without version",
			"", "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.sysDescr, func(t *testing.T) {
			model, firmware := zteParseSystemDescr(tc.sysDescr)
			if model != tc.expectedModel {
				t.Errorf("model: got %q, want %q", model, tc.expectedModel)
			}
			if firmware != tc.expectedFirmware {
				t.Errorf("firmware: got %q, want %q", firmware, tc.expectedFirmware)
			}
		})
	}
}

func TestIfStatusToString(t *testing.T) {
	tests := []struct {
		val      int64
		expected string
	}{
		{1, "up"},
		{2, "down"},
		{3, "testing"},
		{0, "unknown"},
		{99, "unknown"},
	}
	for _, tc := range tests {
		got := ifStatusToString(tc.val)
		if got != tc.expected {
			t.Errorf("ifStatusToString(%d) = %q, want %q", tc.val, got, tc.expected)
		}
	}
}

func TestClassifySFPStatus(t *testing.T) {
	tests := []struct {
		temp     float64
		expected string
	}{
		{0, "empty"},
		{-1, "empty"},
		{30, "normal"},
		{44.9, "normal"},
		{45, "warm"},
		{59.9, "warm"},
		{60, "degraded"},
		{80, "degraded"},
	}
	for _, tc := range tests {
		got := classifySFPStatus(tc.temp)
		if got != tc.expected {
			t.Errorf("classifySFPStatus(%.1f) = %q, want %q", tc.temp, got, tc.expected)
		}
	}
}

func TestOIDSuffix(t *testing.T) {
	tests := []struct {
		oid, prefix, expected string
	}{
		{".1.3.6.1.2.1.2.2.1.7.100", "1.3.6.1.2.1.2.2.1.7", "100"},
		{"1.3.6.1.2.1.2.2.1.7.200", "1.3.6.1.2.1.2.2.1.7", "200"},
		{".1.3.6.1.2.1.2.2.1.7.100", ".1.3.6.1.2.1.2.2.1.7", "100"},
		{"unrelated.oid", "1.3.6.1.2.1", "unrelated.oid"},
	}
	for _, tc := range tests {
		got := oidSuffix(tc.oid, tc.prefix)
		if got != tc.expected {
			t.Errorf("oidSuffix(%q, %q) = %q, want %q", tc.oid, tc.prefix, got, tc.expected)
		}
	}
}

func TestSNMPResultToString(t *testing.T) {
	tests := []struct {
		name     string
		result   domain.SNMPResult
		expected string
	}{
		{"string value", domain.SNMPResult{Value: "hello"}, "hello"},
		{"byte value", domain.SNMPResult{Value: []byte("world")}, "world"},
		{"int value", domain.SNMPResult{Value: int64(42)}, "42"},
		{"nil value", domain.SNMPResult{Value: nil}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := snmpResultToString(tc.result)
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestSNMPResultToInt64(t *testing.T) {
	tests := []struct {
		name     string
		result   domain.SNMPResult
		expected int64
	}{
		{"int64", domain.SNMPResult{Value: int64(100)}, 100},
		{"int", domain.SNMPResult{Value: int(50)}, 50},
		{"int32", domain.SNMPResult{Value: int32(25)}, 25},
		{"uint32", domain.SNMPResult{Value: uint32(75)}, 75},
		{"float64", domain.SNMPResult{Value: float64(3.14)}, 3},
		{"nil", domain.SNMPResult{Value: nil}, 0},
		{"string", domain.SNMPResult{Value: "not a number"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := snmpResultToInt64(tc.result)
			if got != tc.expected {
				t.Errorf("got %d, want %d", got, tc.expected)
			}
		})
	}
}

type mockSNMPConnector struct {
	getResults  []domain.SNMPResult
	walkResults []domain.SNMPResult
	getErr      error
	walkErr     error
	pingErr     error
}

func (m *mockSNMPConnector) Get(_ context.Context, _ domain.SNMPConfig, _ []string) ([]domain.SNMPResult, error) {
	return m.getResults, m.getErr
}

func (m *mockSNMPConnector) Walk(_ context.Context, _ domain.SNMPConfig, _ string) ([]domain.SNMPResult, error) {
	return m.walkResults, m.walkErr
}

func (m *mockSNMPConnector) GetBulk(_ context.Context, _ domain.SNMPConfig, _ []string, _ int) ([]domain.SNMPResult, error) {
	return m.getResults, m.getErr
}

func (m *mockSNMPConnector) Ping(_ context.Context, _ domain.SNMPConfig) error {
	return m.pingErr
}

func TestZTEAdapter_GetSystemInfo(t *testing.T) {
	mock := &mockSNMPConnector{
		getResults: []domain.SNMPResult{
			{OID: "." + zteSysDescr, Type: domain.SNMPValueString, Value: "ZTE ZXA10 C320 Version V2.1.0"},
			{OID: "." + zteSysUpTime, Type: domain.SNMPValueTimeTicks, Value: int64(8640000)}, // 86400 detik
			{OID: "." + zteSysName, Type: domain.SNMPValueString, Value: "OLT-ZTE-01"},
		},
	}

	adapter := NewZTEAdapter(mock, nil, domain.SNMPConfig{}, domain.CLIConfig{})
	info, err := adapter.GetSystemInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Brand != domain.BrandZTE {
		t.Errorf("brand: got %q, want %q", info.Brand, domain.BrandZTE)
	}
	if info.Model != "C320" {
		t.Errorf("model: got %q, want %q", info.Model, "C320")
	}
	if info.FirmwareVersion != "V2.1.0" {
		t.Errorf("firmware: got %q, want %q", info.FirmwareVersion, "V2.1.0")
	}
	if info.Uptime != 86400 {
		t.Errorf("uptime: got %d, want %d", info.Uptime, 86400)
	}
	if info.SysName != "OLT-ZTE-01" {
		t.Errorf("sysName: got %q, want %q", info.SysName, "OLT-ZTE-01")
	}
}

func TestZTEAdapter_GetSystemInfo_Error(t *testing.T) {
	mock := &mockSNMPConnector{getErr: domain.ErrSNMPTimeout}
	adapter := NewZTEAdapter(mock, nil, domain.SNMPConfig{}, domain.CLIConfig{})

	_, err := adapter.GetSystemInfo(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestZTEAdapter_Ping(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockSNMPConnector{pingErr: nil}
		adapter := NewZTEAdapter(mock, nil, domain.SNMPConfig{}, domain.CLIConfig{})
		if err := adapter.Ping(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mock := &mockSNMPConnector{pingErr: domain.ErrSNMPTimeout}
		adapter := NewZTEAdapter(mock, nil, domain.SNMPConfig{}, domain.CLIConfig{})
		if err := adapter.Ping(context.Background()); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestZTEAdapter_GetTrafficStats(t *testing.T) {
	oltIdx := zteCalculateOLTIndex(0, 1)
	mock := &mockSNMPConnector{
		getResults: []domain.SNMPResult{
			{OID: fmt.Sprintf(".%s.%d", ztePONRxOctets, oltIdx), Type: domain.SNMPValueCounter64, Value: int64(1024000)},
			{OID: fmt.Sprintf(".%s.%d", ztePONRxPkts, oltIdx), Type: domain.SNMPValueCounter64, Value: int64(5000)},
			{OID: fmt.Sprintf(".%s.%d", ztePONTxOctets, oltIdx), Type: domain.SNMPValueCounter64, Value: int64(512000)},
			{OID: fmt.Sprintf(".%s.%d", ztePONTxPkts, oltIdx), Type: domain.SNMPValueCounter64, Value: int64(2500)},
		},
	}

	adapter := NewZTEAdapter(mock, nil, domain.SNMPConfig{}, domain.CLIConfig{})
	stats, err := adapter.GetTrafficStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.PortIndex != 1 {
		t.Errorf("portIndex: got %d, want 1", stats.PortIndex)
	}
	if stats.RxBytes != 1024000 {
		t.Errorf("rxBytes: got %d, want 1024000", stats.RxBytes)
	}
	if stats.TxBytes != 512000 {
		t.Errorf("txBytes: got %d, want 512000", stats.TxBytes)
	}
}

func TestZTEAdapter_GetONTSignal(t *testing.T) {
	oltIdx := zteCalculateOLTIndex(0, 0)
	mock := &mockSNMPConnector{
		getResults: []domain.SNMPResult{
			{OID: fmt.Sprintf(".%s.%d.1", zteONURxPower, oltIdx), Type: domain.SNMPValueInteger, Value: int64(-2000)},
			{OID: fmt.Sprintf(".%s.%d.1", zteONUTxPower, oltIdx), Type: domain.SNMPValueInteger, Value: int64(250)},
			{OID: fmt.Sprintf(".%s.%d.1", zteONUDistance, oltIdx), Type: domain.SNMPValueInteger, Value: int64(1500)},
		},
	}

	adapter := NewZTEAdapter(mock, nil, domain.SNMPConfig{}, domain.CLIConfig{})
	signal, err := adapter.GetONTSignal(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.ONTIndex != 1 {
		t.Errorf("ontIndex: got %d, want 1", signal.ONTIndex)
	}
	// -2000 / 100 = -20.0 dBm
	if signal.RxPowerDBm != -20.0 {
		t.Errorf("rxPower: got %.2f, want -20.00", signal.RxPowerDBm)
	}
	// 250 / 100 = 2.5 dBm
	if signal.TxPowerDBm != 2.5 {
		t.Errorf("txPower: got %.2f, want 2.50", signal.TxPowerDBm)
	}
	if signal.Distance != 1500 {
		t.Errorf("distance: got %d, want 1500", signal.Distance)
	}
	if signal.SignalLevel != domain.SignalNormal {
		t.Errorf("signalLevel: got %q, want %q", signal.SignalLevel, domain.SignalNormal)
	}
}

// TestZTEAdapter_FactoryIntegration memverifikasi bahwa factory mengembalikan
// ZTEAdapter untuk BrandZTE di mode live.
func TestZTEAdapter_FactoryIntegration(t *testing.T) {
	mock := &mockSNMPConnector{}
	factory := NewOLTAdapterFactory("live", mock, nil)
	snmpCfg := domain.SNMPConfig{Host: "10.0.0.1", Port: 161}
	cliCfg := domain.CLIConfig{Host: "10.0.0.1", Port: 22}

	adapter, err := factory.CreateAdapter(domain.BrandZTE, snmpCfg, cliCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if _, ok := adapter.(*ZTEAdapter); !ok {
		t.Fatal("expected *ZTEAdapter")
	}
}
