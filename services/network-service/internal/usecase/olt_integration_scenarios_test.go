// olt_integration_scenarios_test.go — skenario integration test OLT Management.
// Menguji full lifecycle OLT, ODP CRUD, health check cycle, alarm polling,
// sync engine, capacity planning, dan status summary.
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// integSignalStore — mock signal store untuk integration test.
type integSignalStore struct {
	stored int
}

func (s *integSignalStore) Store(_ context.Context, _ string, _, _ int, _ domain.ONTSignalPoint) error {
	s.stored++
	return nil
}
func (s *integSignalStore) Query(_ context.Context, _ string, _, _ int, _, _ time.Time) ([]domain.ONTSignalPoint, error) {
	return nil, nil
}
func (s *integSignalStore) GetLatest(_ context.Context, _ string, _, _ int) (*domain.ONTSignalPoint, error) {
	return nil, nil
}

// integTrafficStore — mock traffic store untuk integration test.
type integTrafficStore struct {
	stored int
}

func (s *integTrafficStore) Store(_ context.Context, _ string, _ int, _ domain.PONTrafficPoint) error {
	s.stored++
	return nil
}
func (s *integTrafficStore) Query(_ context.Context, _ string, _ int, _, _ time.Time) ([]domain.PONTrafficPoint, error) {
	return nil, nil
}
func (s *integTrafficStore) GetLatest(_ context.Context, _ string, _ int) (*domain.PONTrafficPoint, error) {
	return nil, nil
}

// newIntegSetup membuat semua komponen terintegrasi dengan mock dependencies.
func newIntegSetup() (
	*oltManager, *oltHealthChecker, *alarmManager, *syncEngine,
	*integOLTRepo, *oltIntegEventPub, *integAlarmRepo, *mockOLTAdapter,
) {
	repo := newIntegOLTRepo()
	alarmRepo := &integAlarmRepo{}
	eventPub := &oltIntegEventPub{}
	enc := &mockEncryptor{}
	adapter := &mockOLTAdapter{
		sysInfo: &domain.OLTSystemInfo{
			Brand: domain.BrandZTE, Model: "C320", FirmwareVersion: "V2.1.0",
			PONPortCount: 8, TotalONTCount: 245, SysDescr: "ZTE ZXA10 C320",
		},
		ponPorts: []domain.PONPortStatus{
			{PortIndex: 0, AdminStatus: "up", OperStatus: "up", ONTCount: 30},
			{PortIndex: 1, AdminStatus: "up", OperStatus: "up", ONTCount: 25},
		},
		ontList: []domain.ONTPortStatus{
			{ONTIndex: 0, SerialNumber: "ZTEG12345678", Status: "online", RxSignalDBm: -20.0},
		},
	}
	factory := &mockOLTAdapterFactory{adapter: adapter}
	sigStore := &integSignalStore{}
	trafStore := &integTrafficStore{}

	mgr := NewOLTManager(repo, nil, alarmRepo, factory, &mockSNMPConnector{},
		&mockCLIConnector{banner: "ZTE>"}, enc, eventPub, sigStore, trafStore).(*oltManager)

	hc := &oltHealthChecker{
		oltRepo: repo, factory: factory, encryptor: enc, eventPub: eventPub,
		workers: make(map[string]*oltWorker),
	}
	mgr.SetHealthChecker(hc)

	am := &alarmManager{
		alarmRepo: alarmRepo, oltRepo: repo, factory: &amFactory{adapter: adapter},
		encryptor: enc, eventPub: eventPub, trapPort: 16200, stopChan: make(chan struct{}),
	}

	se := &syncEngine{
		oltRepo: repo, factory: factory, encryptor: enc,
		signalStore: sigStore, trafficStore: trafStore, syncInterval: time.Minute,
	}

	return mgr, hc, am, se, repo, eventPub, alarmRepo, adapter
}

// TestIntegOLT_FullLifecycle — Create → auto-detect → GetByID → Update → Delete.
func TestIntegOLT_FullLifecycle(t *testing.T) {
	mgr, _, _, _, repo, _, _, _ := newIntegSetup()
	ctx := context.Background()

	// 1. Create OLT dengan auto-detect brand
	resp, err := mgr.Create(ctx, "t1", createTestOLTRequest())
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}
	if resp.Brand != domain.BrandZTE {
		t.Errorf("brand harus zte, dapat %q", resp.Brand)
	}
	if resp.Status != domain.OLTStatusOnline {
		t.Errorf("status harus online setelah auto-detect sukses, dapat %q", resp.Status)
	}

	// 2. GetByID — verifikasi detail
	detail, err := mgr.GetByID(ctx, resp.ID)
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}
	if detail.Name != "OLT-Test-01" {
		t.Errorf("nama salah: %q", detail.Name)
	}

	// 3. Update nama
	upd, err := mgr.Update(ctx, resp.ID, domain.UpdateOLTRequest{Name: "OLT-Updated"})
	if err != nil {
		t.Fatalf("Update gagal: %v", err)
	}
	if upd.Name != "OLT-Updated" {
		t.Errorf("nama tidak terupdate: %q", upd.Name)
	}

	// 4. Delete
	if err := mgr.Delete(ctx, resp.ID); err != nil {
		t.Fatalf("Delete gagal: %v", err)
	}
	if len(repo.olts) != 0 {
		t.Error("OLT masih ada di repo setelah delete")
	}
}

// TestIntegODP_CRUDFlow — Create ODP → GetByID (cek warning) → Update → Delete.
func TestIntegODP_CRUDFlow(t *testing.T) {
	odpRepo := newMockODPRepo()
	mgr := NewODPManager(odpRepo, nil).(*odpManager)
	ctx := context.Background()

	// 1. Create ODP — kapasitas otomatis dari splitter_type
	resp, err := mgr.Create(ctx, "t1", domain.CreateODPRequest{
		OLTID: "olt-001", PONPortIndex: 0, Name: "ODP-01",
		SplitterType: domain.SplitterType1x8, Address: "Jl. Merdeka",
	})
	if err != nil {
		t.Fatalf("Create ODP gagal: %v", err)
	}
	if resp.Capacity != 8 {
		t.Errorf("kapasitas harus 8 untuk 1:8, dapat %d", resp.Capacity)
	}

	// 2. Simulasikan ODP penuh, lalu GetByID — harus ada warning
	odpRepo.odps[resp.ID].UsedPorts = 8
	detail, err := mgr.GetByID(ctx, resp.ID)
	if err != nil {
		t.Fatalf("GetByID ODP gagal: %v", err)
	}
	if detail.Warning == "" {
		t.Error("ODP penuh harus memiliki warning")
	}

	// 3. Update nama
	updated, err := mgr.Update(ctx, resp.ID, domain.UpdateODPRequest{Name: "ODP-Updated"})
	if err != nil {
		t.Fatalf("Update ODP gagal: %v", err)
	}
	if updated.Name != "ODP-Updated" {
		t.Errorf("nama ODP tidak terupdate: %q", updated.Name)
	}

	// 4. Delete
	if err := mgr.Delete(ctx, resp.ID); err != nil {
		t.Fatalf("Delete ODP gagal: %v", err)
	}
}

// TestIntegOLT_HealthCheckCycle — Online → 3x failure → offline + event → recovery → online + event.
func TestIntegOLT_HealthCheckCycle(t *testing.T) {
	_, hc, _, _, repo, eventPub, _, adapter := newIntegSetup()
	ctx := context.Background()

	// Siapkan OLT online di repo
	now := time.Now()
	olt := &domain.OLT{
		ID: "olt-hc", TenantID: "t1", Name: "OLT-HC", Host: "10.0.0.1",
		SNMPVersion: domain.SNMPv2c, SNMPPort: 161, Status: domain.OLTStatusOnline,
		FailureCount: 0, LastOnlineAt: &now,
	}
	repo.olts["olt-hc"] = olt

	// Simulasikan 3x failure berturut-turut
	for i := 0; i < 3; i++ {
		hc.handleOLTFailure(ctx, repo.olts["olt-hc"])
		// Refresh dari repo karena handleOLTFailure mengupdate via repo
	}

	// Verifikasi: status offline dan event offline dipublish
	if repo.olts["olt-hc"].Status != domain.OLTStatusOffline {
		t.Errorf("status harus offline setelah 3x failure, dapat %q", repo.olts["olt-hc"].Status)
	}
	eventPub.mu.Lock()
	if len(eventPub.offline) != 1 {
		t.Errorf("harus ada 1 offline event, dapat %d", len(eventPub.offline))
	}
	eventPub.mu.Unlock()

	// Recovery: simulasikan ping sukses saat offline
	adapter.sysInfoErr = nil // pastikan adapter bisa di-ping
	hc.handleOLTSuccess(ctx, repo.olts["olt-hc"])

	if repo.olts["olt-hc"].Status != domain.OLTStatusOnline {
		t.Errorf("status harus online setelah recovery, dapat %q", repo.olts["olt-hc"].Status)
	}
	eventPub.mu.Lock()
	if len(eventPub.online) != 1 {
		t.Errorf("harus ada 1 online event setelah recovery, dapat %d", len(eventPub.online))
	}
	eventPub.mu.Unlock()
}
