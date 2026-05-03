// olt_integration_advanced_test.go — skenario lanjutan integration test OLT.
// Menguji alarm polling, sync engine, capacity planning, dan status summary.
package usecase

import (
	"context"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// TestIntegOLT_AlarmPollingFlow — Poll alarm → simpan ke DB → publish event.
func TestIntegOLT_AlarmPollingFlow(t *testing.T) {
	_, _, am, _, repo, eventPub, alarmRepo, adapter := newIntegSetup()
	ctx := context.Background()

	// Siapkan OLT di repo untuk alarm manager
	repo.olts["olt-alarm"] = &domain.OLT{
		ID: "olt-alarm", TenantID: "t1", Name: "OLT-Alarm", Host: "10.0.0.2",
		SNMPVersion: domain.SNMPv2c, SNMPPort: 161, Brand: domain.BrandZTE,
		Status: domain.OLTStatusOnline, CLIProtocol: domain.CLIProtocolSSH,
		CLIPort: 22, CLIUsername: "admin", CLIPasswordEncrypted: "enc:pass",
		SNMPCommunityEncrypted: "enc:public",
	}

	// Konfigurasi adapter untuk mengembalikan alarm
	portIdx := 0
	adapter.alarms = []domain.OLTAlarm{
		{AlarmType: domain.AlarmTypeONTLOS, Severity: domain.SeverityCritical,
			PONPortIndex: &portIdx, Message: "ONT LOS pada port 0"},
		{AlarmType: domain.AlarmTypeHighTemperature, Severity: domain.SeverityWarning,
			Message: "Suhu OLT tinggi: 55°C"},
	}

	// Poll alarm
	alarms, err := am.PollAlarms(ctx, "olt-alarm")
	if err != nil {
		t.Fatalf("PollAlarms gagal: %v", err)
	}
	if len(alarms) != 2 {
		t.Fatalf("harus ada 2 alarm, dapat %d", len(alarms))
	}

	// Verifikasi alarm tersimpan di DB
	alarmRepo.mu.Lock()
	savedCount := len(alarmRepo.records)
	alarmRepo.mu.Unlock()
	if savedCount != 2 {
		t.Errorf("harus ada 2 alarm di DB, dapat %d", savedCount)
	}

	// Verifikasi event alarm dipublish
	eventPub.mu.Lock()
	alarmEvents := len(eventPub.alarms)
	eventPub.mu.Unlock()
	if alarmEvents != 2 {
		t.Errorf("harus ada 2 alarm event, dapat %d", alarmEvents)
	}

	// Verifikasi source = polling
	for _, a := range alarms {
		if a.Source != domain.AlarmSourcePolling {
			t.Errorf("alarm source harus polling, dapat %q", a.Source)
		}
	}
}

// TestIntegOLT_SyncEngineCycle — Sync OLT → store signal/traffic → update ONT counts.
func TestIntegOLT_SyncEngineCycle(t *testing.T) {
	_, _, _, se, repo, _, _, adapter := newIntegSetup()
	ctx := context.Background()

	// Siapkan OLT online di repo
	repo.olts["olt-sync"] = &domain.OLT{
		ID: "olt-sync", TenantID: "t1", Name: "OLT-Sync", Host: "10.0.0.3",
		SNMPVersion: domain.SNMPv2c, SNMPPort: 161, Brand: domain.BrandZTE,
		Status: domain.OLTStatusOnline, SNMPCommunityEncrypted: "enc:public",
		CLIProtocol: domain.CLIProtocolSSH, CLIPort: 22,
		CLIUsername: "admin", CLIPasswordEncrypted: "enc:pass",
	}

	// Konfigurasi adapter: 2 port, 1 ONT per port
	adapter.ponPorts = []domain.PONPortStatus{
		{PortIndex: 0, AdminStatus: "up", OperStatus: "up", ONTCount: 1},
		{PortIndex: 1, AdminStatus: "up", OperStatus: "up", ONTCount: 1},
	}
	adapter.ontList = []domain.ONTPortStatus{
		{ONTIndex: 0, SerialNumber: "ZTEG00000001", Status: "online", RxSignalDBm: -18.5},
	}

	// Jalankan sync
	result, err := se.SyncOLT(ctx, "olt-sync")
	if err != nil {
		t.Fatalf("SyncOLT gagal: %v", err)
	}

	// Verifikasi hasil sync: 2 port × 1 ONT = 2 total ONT
	if result.TotalONT != 2 {
		t.Errorf("total ONT harus 2, dapat %d", result.TotalONT)
	}
	// Semua ONT dari OLT, tidak ada di DB → semua unmanaged
	if result.UnmanagedCount != 2 {
		t.Errorf("unmanaged harus 2, dapat %d", result.UnmanagedCount)
	}

	// Verifikasi total_ont_count diupdate di repo
	repo.mu.Lock()
	ontCount := repo.olts["olt-sync"].TotalONTCount
	repo.mu.Unlock()
	if ontCount != 2 {
		t.Errorf("total_ont_count di repo harus 2, dapat %d", ontCount)
	}
}

// TestIntegOLT_CapacityPlanning — Hitung kapasitas dari PON port data → verifikasi invariant.
func TestIntegOLT_CapacityPlanning(t *testing.T) {
	mgr, _, _, _, repo, _, _, adapter := newIntegSetup()
	ctx := context.Background()

	// Siapkan OLT di repo
	repo.olts["olt-cap"] = &domain.OLT{
		ID: "olt-cap", TenantID: "t1", Name: "OLT-Cap", Host: "10.0.0.4",
		SNMPVersion: domain.SNMPv2c, SNMPPort: 161, Brand: domain.BrandZTE,
		Status: domain.OLTStatusOnline, PONPortCount: 4,
		SNMPCommunityEncrypted: "enc:public", CLIProtocol: domain.CLIProtocolSSH,
		CLIPort: 22, CLIUsername: "admin", CLIPasswordEncrypted: "enc:pass",
	}

	// Konfigurasi adapter: 4 port dengan utilisasi berbeda
	adapter.ponPorts = []domain.PONPortStatus{
		{PortIndex: 0, OperStatus: "up", ONTCount: 60},  // 93.75% — harus ada warning
		{PortIndex: 1, OperStatus: "up", ONTCount: 30},  // 46.88%
		{PortIndex: 2, OperStatus: "up", ONTCount: 10},  // 15.63%
		{PortIndex: 3, OperStatus: "down", ONTCount: 0}, // 0%
	}

	capacity, err := mgr.GetCapacity(ctx, "olt-cap")
	if err != nil {
		t.Fatalf("GetCapacity gagal: %v", err)
	}

	// Invariant: available = total - used
	expectedTotal := 4 * maxONTPerPort // 256
	expectedUsed := 60 + 30 + 10 + 0  // 100
	if capacity.TotalONTSlots != expectedTotal {
		t.Errorf("total slots=%d, want %d", capacity.TotalONTSlots, expectedTotal)
	}
	if capacity.UsedONTSlots != expectedUsed {
		t.Errorf("used slots=%d, want %d", capacity.UsedONTSlots, expectedUsed)
	}
	if capacity.AvailableONTSlots != expectedTotal-expectedUsed {
		t.Errorf("available=%d, want %d", capacity.AvailableONTSlots, expectedTotal-expectedUsed)
	}

	// Port 0 harus ada warning (>90%)
	if capacity.PortBreakdown[0].Warning == "" {
		t.Error("port 0 (93.75%) harus memiliki warning")
	}
	// Port 1 tidak boleh ada warning
	if capacity.PortBreakdown[1].Warning != "" {
		t.Errorf("port 1 (46.88%%) tidak boleh ada warning: %q", capacity.PortBreakdown[1].Warning)
	}
	// Active ports = 3 (port 3 down)
	if capacity.ActivePONPorts != 3 {
		t.Errorf("active ports=%d, want 3", capacity.ActivePONPorts)
	}
}

// TestIntegOLT_StatusSummary — Verifikasi total = online + offline + maintenance.
func TestIntegOLT_StatusSummary(t *testing.T) {
	mgr, _, _, _, repo, _, alarmRepo, _ := newIntegSetup()
	ctx := context.Background()

	// Siapkan beberapa OLT dengan status berbeda
	statuses := []domain.OLTStatus{
		domain.OLTStatusOnline, domain.OLTStatusOnline, domain.OLTStatusOnline,
		domain.OLTStatusOffline, domain.OLTStatusOffline,
		domain.OLTStatusMaintenance,
	}
	for i, s := range statuses {
		id := string(rune('a' + i))
		repo.olts["olt-"+id] = &domain.OLT{ID: "olt-" + id, Status: s}
	}

	// Tambahkan beberapa alarm
	alarmRepo.records = append(alarmRepo.records,
		&domain.OLTAlarmRecord{ID: "a1"}, &domain.OLTAlarmRecord{ID: "a2"},
	)

	summary, err := mgr.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary gagal: %v", err)
	}

	// Invariant utama: total = online + offline + maintenance
	if summary.TotalOLTs != summary.OnlineCount+summary.OfflineCount+summary.MaintenanceCount {
		t.Errorf("total(%d) != online(%d)+offline(%d)+maintenance(%d)",
			summary.TotalOLTs, summary.OnlineCount, summary.OfflineCount, summary.MaintenanceCount)
	}
	if summary.OnlineCount != 3 {
		t.Errorf("online=%d, want 3", summary.OnlineCount)
	}
	if summary.OfflineCount != 2 {
		t.Errorf("offline=%d, want 2", summary.OfflineCount)
	}
	if summary.MaintenanceCount != 1 {
		t.Errorf("maintenance=%d, want 1", summary.MaintenanceCount)
	}
	if summary.ActiveAlarmCount != 2 {
		t.Errorf("alarm count=%d, want 2", summary.ActiveAlarmCount)
	}
}
