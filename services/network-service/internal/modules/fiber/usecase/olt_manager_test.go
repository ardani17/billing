package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

type mockOLTRepo struct {
	olts         map[string]*domain.OLT
	nameExists   bool
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
	listResult   *domain.OLTListResult
	statusCounts map[domain.OLTStatus]int64
}

func newMockOLTRepo() *mockOLTRepo {
	return &mockOLTRepo{
		olts:         make(map[string]*domain.OLT),
		statusCounts: make(map[domain.OLTStatus]int64),
	}
}

func (r *mockOLTRepo) Create(_ context.Context, olt *domain.OLT) (*domain.OLT, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	olt.ID = "olt-test-001"
	olt.CreatedAt = time.Now()
	olt.UpdatedAt = time.Now()
	r.olts[olt.ID] = olt
	return olt, nil
}

func (r *mockOLTRepo) GetByID(_ context.Context, id string) (*domain.OLT, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	olt, ok := r.olts[id]
	if !ok {
		return nil, domain.ErrOLTNotFound
	}
	return olt, nil
}

func (r *mockOLTRepo) Update(_ context.Context, olt *domain.OLT) (*domain.OLT, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	olt.UpdatedAt = time.Now()
	r.olts[olt.ID] = olt
	return olt, nil
}

func (r *mockOLTRepo) SoftDelete(_ context.Context, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.olts, id)
	return nil
}

func (r *mockOLTRepo) List(_ context.Context, _ domain.OLTListParams) (*domain.OLTListResult, error) {
	if r.listResult != nil {
		return r.listResult, nil
	}
	return &domain.OLTListResult{Data: []*domain.OLTResponse{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

func (r *mockOLTRepo) CountByStatus(_ context.Context) (map[domain.OLTStatus]int64, error) {
	return r.statusCounts, nil
}

func (r *mockOLTRepo) GetActiveOLTs(_ context.Context) ([]*domain.OLT, error) { return nil, nil }
func (r *mockOLTRepo) GetOnlineOLTs(_ context.Context) ([]*domain.OLT, error) { return nil, nil }

func (r *mockOLTRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return r.nameExists, nil
}

func (r *mockOLTRepo) UpdateHealthCheck(_ context.Context, _ string, _ domain.OLTHealthCheckUpdate) error {
	return nil
}

func (r *mockOLTRepo) UpdateONTCounts(_ context.Context, _ string, _ int) error { return nil }

type mockAlarmRepo struct {
	activeCount    int64
	activeByTenant int64
}

func (r *mockAlarmRepo) Create(_ context.Context, a *domain.OLTAlarmRecord) (*domain.OLTAlarmRecord, error) {
	return a, nil
}

func (r *mockAlarmRepo) List(_ context.Context, _ string, _ domain.AlarmListParams) (*domain.AlarmListResult, error) {
	return &domain.AlarmListResult{}, nil
}

func (r *mockAlarmRepo) CountActive(_ context.Context, _ string) (int64, error) {
	return r.activeCount, nil
}

func (r *mockAlarmRepo) CountActiveByTenant(_ context.Context) (int64, error) {
	return r.activeByTenant, nil
}

func (r *mockAlarmRepo) ClearAlarm(_ context.Context, _ string) error { return nil }

func (r *mockAlarmRepo) PurgeOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

type mockEncryptor struct {
	failEncrypt bool
	failDecrypt bool
}

func (e *mockEncryptor) Encrypt(plaintext string) (string, error) {
	if e.failEncrypt {
		return "", domain.ErrEncryptionFailed
	}
	return "enc:" + plaintext, nil
}

func (e *mockEncryptor) Decrypt(ciphertext string) (string, error) {
	if e.failDecrypt {
		return "", domain.ErrDecryptionFailed
	}
	if len(ciphertext) > 4 && ciphertext[:4] == "enc:" {
		return ciphertext[4:], nil
	}
	return ciphertext, nil
}

type mockOLTAdapter struct {
	sysInfo    *domain.OLTSystemInfo
	sysInfoErr error
	ponPorts   []domain.PONPortStatus
	ontList    []domain.ONTPortStatus
	sfpInfo    *domain.SFPInfo
	alarms     []domain.OLTAlarm
}

func (a *mockOLTAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	if a.sysInfoErr != nil {
		return nil, a.sysInfoErr
	}
	return a.sysInfo, nil
}

func (a *mockOLTAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, nil
}

func (a *mockOLTAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return a.ponPorts, nil
}

func (a *mockOLTAdapter) GetONTList(_ context.Context, _ int) ([]domain.ONTPortStatus, error) {
	return a.ontList, nil
}

func (a *mockOLTAdapter) GetONTSignal(_ context.Context, _, _ int) (*domain.ONTSignalInfo, error) {
	return &domain.ONTSignalInfo{RxPowerDBm: -20.0, SignalLevel: "normal"}, nil
}

func (a *mockOLTAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	return a.alarms, nil
}

func (a *mockOLTAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) {
	if a.sfpInfo != nil {
		return a.sfpInfo, nil
	}
	return &domain.SFPInfo{Status: "normal"}, nil
}

func (a *mockOLTAdapter) GetTrafficStats(_ context.Context, _ int) (*domain.PONTrafficStats, error) {
	return &domain.PONTrafficStats{RxBytes: 1000, TxBytes: 2000, RxPackets: 100, TxPackets: 200}, nil
}

func (a *mockOLTAdapter) Ping(_ context.Context) error { return nil }

// --- Provisioning methods ---

func (a *mockOLTAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"onu add"}, Responses: []string{"ok"}}, nil
}

func (a *mockOLTAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"onu delete"}, Responses: []string{"ok"}}, nil
}

func (a *mockOLTAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"service-port add"}, Responses: []string{"ok"}}, nil
}

func (a *mockOLTAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"service-port delete"}, Responses: []string{"ok"}}, nil
}

func (a *mockOLTAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"onu reset"}, Responses: []string{"ok"}}, nil
}

func (a *mockOLTAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, nil
}

type mockOLTAdapterFactory struct {
	adapter   domain.OLTAdapter
	createErr error
}

func (f *mockOLTAdapterFactory) CreateAdapter(_ domain.OLTBrand, _ domain.SNMPConfig, _ domain.CLIConfig) (domain.OLTAdapter, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.adapter, nil
}

type mockCLIConnector struct {
	banner  string
	testErr error
}

func (c *mockCLIConnector) Execute(_ context.Context, _ domain.CLIConfig, _ string) (string, error) {
	return "", nil
}

func (c *mockCLIConnector) ExecuteMultiple(_ context.Context, _ domain.CLIConfig, _ []string) ([]string, error) {
	return nil, nil
}

func (c *mockCLIConnector) TestConnection(_ context.Context, _ domain.CLIConfig) (string, error) {
	if c.testErr != nil {
		return "", c.testErr
	}
	return c.banner, nil
}

type mockSNMPConnector struct{}

func (c *mockSNMPConnector) Get(_ context.Context, _ domain.SNMPConfig, _ []string) ([]domain.SNMPResult, error) {
	return nil, nil
}

func (c *mockSNMPConnector) Walk(_ context.Context, _ domain.SNMPConfig, _ string) ([]domain.SNMPResult, error) {
	return nil, nil
}

func (c *mockSNMPConnector) GetBulk(_ context.Context, _ domain.SNMPConfig, _ []string, _ int) ([]domain.SNMPResult, error) {
	return nil, nil
}

func (c *mockSNMPConnector) Ping(_ context.Context, _ domain.SNMPConfig) error { return nil }

type mockOLTEventPublisher struct {
	provisionedEvents    []domain.ONTProvisionedPayload
	decommissionedEvents []domain.ONTDecommissionedPayload
	autoProvEvents       []domain.ONTAutoProvisionedPayload
	autoProvFailEvents   []domain.ONTAutoProvisionFailedPayload
	portMigratedEvents   []domain.ONTPortMigratedPayload
}

func (p *mockOLTEventPublisher) PublishDeviceOffline(_ context.Context, _ domain.OLTDeviceOfflinePayload) error {
	return nil
}

func (p *mockOLTEventPublisher) PublishDeviceOnline(_ context.Context, _ domain.OLTDeviceOnlinePayload) error {
	return nil
}

func (p *mockOLTEventPublisher) PublishAlarm(_ context.Context, _ domain.OLTAlarmPayload) error {
	return nil
}

func (p *mockOLTEventPublisher) PublishONTProvisioned(_ context.Context, payload domain.ONTProvisionedPayload) error {
	p.provisionedEvents = append(p.provisionedEvents, payload)
	return nil
}

func (p *mockOLTEventPublisher) PublishONTDecommissioned(_ context.Context, payload domain.ONTDecommissionedPayload) error {
	p.decommissionedEvents = append(p.decommissionedEvents, payload)
	return nil
}

func (p *mockOLTEventPublisher) PublishONTAutoProvisioned(_ context.Context, payload domain.ONTAutoProvisionedPayload) error {
	p.autoProvEvents = append(p.autoProvEvents, payload)
	return nil
}

func (p *mockOLTEventPublisher) PublishONTAutoProvisionFailed(_ context.Context, payload domain.ONTAutoProvisionFailedPayload) error {
	p.autoProvFailEvents = append(p.autoProvFailEvents, payload)
	return nil
}

func (p *mockOLTEventPublisher) PublishONTPortMigrated(_ context.Context, payload domain.ONTPortMigratedPayload) error {
	p.portMigratedEvents = append(p.portMigratedEvents, payload)
	return nil
}

type mockHealthChecker struct {
	addedOLTs   []string
	removedOLTs []string
}

func (h *mockHealthChecker) Start(_ context.Context) error { return nil }
func (h *mockHealthChecker) Stop()                         {}

func (h *mockHealthChecker) AddOLT(olt *domain.OLT) {
	h.addedOLTs = append(h.addedOLTs, olt.ID)
}

func (h *mockHealthChecker) RemoveOLT(oltID string) {
	h.removedOLTs = append(h.removedOLTs, oltID)
}

func (h *mockHealthChecker) UpdateInterval(_ string, _ int) {}

// =============================================================================
// =============================================================================

func newTestOLTManager() (*oltManager, *mockOLTRepo, *mockAlarmRepo, *mockOLTAdapter, *mockHealthChecker) {
	repo := newMockOLTRepo()
	alarmRepo := &mockAlarmRepo{activeCount: 2, activeByTenant: 5}
	adapter := &mockOLTAdapter{
		sysInfo: &domain.OLTSystemInfo{
			Brand:           domain.BrandZTE,
			Model:           "C320",
			FirmwareVersion: "V2.1.0",
			PONPortCount:    8,
			TotalONTCount:   245,
			SysDescr:        "ZTE ZXA10 C320",
		},
		ponPorts: []domain.PONPortStatus{
			{PortIndex: 0, AdminStatus: "up", OperStatus: "up", ONTCount: 30},
			{PortIndex: 1, AdminStatus: "up", OperStatus: "up", ONTCount: 25},
		},
		ontList: []domain.ONTPortStatus{
			{ONTIndex: 0, SerialNumber: "ZTEG12345678", Status: "online"},
		},
	}
	factory := &mockOLTAdapterFactory{adapter: adapter}
	encryptor := &mockEncryptor{}
	cliConn := &mockCLIConnector{banner: "ZTE C320> "}
	snmpConn := &mockSNMPConnector{}
	eventPub := &mockOLTEventPublisher{}
	hc := &mockHealthChecker{}

	mgr := NewOLTManager(
		repo, nil, alarmRepo, factory, snmpConn, cliConn, encryptor, eventPub, nil, nil,
	).(*oltManager)
	mgr.SetHealthChecker(hc)

	return mgr, repo, alarmRepo, adapter, hc
}

// createTestOLTRequest membuat CreateOLTRequest standar untuk testing.
func createTestOLTRequest() domain.CreateOLTRequest {
	return domain.CreateOLTRequest{
		Name:          "OLT-Test-01",
		Host:          "192.168.1.100",
		SNMPVersion:   "v2c",
		SNMPCommunity: "public",
		CLIProtocol:   "ssh",
		CLIPort:       22,
		CLIUsername:   "admin",
		CLIPassword:   "secret123",
	}
}

// =============================================================================
// Tes Cases
// =============================================================================

func TestOLTManager_Create_AutoDetectSuccess(t *testing.T) {
	mgr, repo, _, _, hc := newTestOLTManager()
	ctx := context.Background()

	req := createTestOLTRequest()
	resp, err := mgr.Create(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	if resp.Name != "OLT-Test-01" {
		t.Errorf("nama OLT salah: got %q, want %q", resp.Name, "OLT-Test-01")
	}
	if resp.Brand != domain.BrandZTE {
		t.Errorf("brand salah: got %q, want %q", resp.Brand, domain.BrandZTE)
	}
	if resp.Model != "C320" {
		t.Errorf("model salah: got %q, want %q", resp.Model, "C320")
	}
	if resp.Status != domain.OLTStatusOnline {
		t.Errorf("status salah: got %q, want %q", resp.Status, domain.OLTStatusOnline)
	}
	if resp.PONPortCount != 8 {
		t.Errorf("pon_port_count salah: got %d, want %d", resp.PONPortCount, 8)
	}

	// Verifikasi OLT tersimpan di repo
	if len(repo.olts) != 1 {
		t.Errorf("jumlah OLT di repo salah: got %d, want 1", len(repo.olts))
	}

	// Verifikasi pemeriksa kesehatan dipanggil
	if len(hc.addedOLTs) != 1 {
		t.Errorf("pemeriksa kesehatan AddOLT tidak dipanggil: got %d calls", len(hc.addedOLTs))
	}

	// Verifikasi bawaan values
	olt := repo.olts["olt-test-001"]
	if olt.SNMPPort != 161 {
		t.Errorf("default SNMP port salah: got %d, want 161", olt.SNMPPort)
	}
	if olt.HealthCheckIntervalSec != 300 {
		t.Errorf("default health check interval salah: got %d, want 300", olt.HealthCheckIntervalSec)
	}
}

func TestOLTManager_Create_AutoDetectFailure(t *testing.T) {
	mgr, _, _, adapter, _ := newTestOLTManager()
	ctx := context.Background()

	// Simulasikan auto-detect gagal
	adapter.sysInfoErr = errors.New("SNMP timeout")

	req := createTestOLTRequest()
	resp, err := mgr.Create(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("Create harus berhasil meski auto-detect gagal: %v", err)
	}

	// OLT tetap disimpan sebagai offline
	if resp.Status != domain.OLTStatusOffline {
		t.Errorf("status harus offline saat auto-detect gagal: got %q", resp.Status)
	}
	if resp.Brand != "" {
		t.Errorf("brand harus kosong saat auto-detect gagal: got %q", resp.Brand)
	}
}

func TestOLTManager_GetTraffic_NoStoreReturnsEmpty(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()
	repo.olts["olt-001"] = &domain.OLT{ID: "olt-001", TenantID: "tenant-001", Brand: domain.BrandZTE}

	points, err := mgr.GetTraffic(ctx, "olt-001", 0, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("GetTraffic gagal: %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("expected empty traffic points, got %d", len(points))
	}
}

func TestOLTManager_GetSignal_NoStoreReturnsEmpty(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()
	repo.olts["olt-001"] = &domain.OLT{ID: "olt-001", TenantID: "tenant-001", Brand: domain.BrandZTE}

	points, err := mgr.GetSignal(ctx, "olt-001", 0, 1, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("GetSignal gagal: %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("expected empty signal points, got %d", len(points))
	}
}

func TestOLTManager_GetTraffic_OLTFailureState(t *testing.T) {
	mgr, _, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	if _, err := mgr.GetTraffic(ctx, "olt-missing", 0, time.Now().Add(-time.Hour), time.Now()); err != domain.ErrOLTNotFound {
		t.Fatalf("GetTraffic err = %v, want ErrOLTNotFound", err)
	}
}

func TestOLTManager_Create_NameExists(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.nameExists = true
	req := createTestOLTRequest()

	_, err := mgr.Create(ctx, "tenant-001", req)
	if !errors.Is(err, domain.ErrOLTNameExists) {
		t.Errorf("expected ErrOLTNameExists, got: %v", err)
	}
}

func TestOLTManager_Create_EncryptionFailure(t *testing.T) {
	mgr, _, _, _, _ := newTestOLTManager()
	mgr.encryptor = &mockEncryptor{failEncrypt: true}
	ctx := context.Background()

	req := createTestOLTRequest()
	_, err := mgr.Create(ctx, "tenant-001", req)
	if !errors.Is(err, domain.ErrEncryptionFailed) {
		t.Errorf("expected ErrEncryptionFailed, got: %v", err)
	}
}

func TestOLTManager_GetByID_WithAlarmCount(t *testing.T) {
	mgr, repo, alarmRepo, _, _ := newTestOLTManager()
	ctx := context.Background()

	// Siapkan OLT di repo
	repo.olts["olt-001"] = &domain.OLT{
		ID:          "olt-001",
		TenantID:    "tenant-001",
		Name:        "OLT-Test",
		Host:        "192.168.1.100",
		SNMPVersion: domain.SNMPv2c,
		CLIProtocol: domain.CLIProtocolSSH,
		CLIPort:     22,
		Status:      domain.OLTStatusOnline,
	}
	alarmRepo.activeCount = 3

	resp, err := mgr.GetByID(ctx, "olt-001")
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}

	if resp.ActiveAlarmCount != 3 {
		t.Errorf("alarm count salah: got %d, want 3", resp.ActiveAlarmCount)
	}
	if resp.SNMPVersion != domain.SNMPv2c {
		t.Errorf("snmp version salah: got %q, want %q", resp.SNMPVersion, domain.SNMPv2c)
	}
	// Harus ada warning untuk SNMP v2c
	if resp.Warning == "" {
		t.Error("expected warning untuk SNMP v2c, got empty")
	}
}

func TestOLTManager_GetByID_NotFound(t *testing.T) {
	mgr, _, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	_, err := mgr.GetByID(ctx, "nonexistent")
	if !errors.Is(err, domain.ErrOLTNotFound) {
		t.Errorf("expected ErrOLTNotFound, got: %v", err)
	}
}

func TestOLTManager_Update_WithCredentialReEncryption(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	// Siapkan OLT di repo
	repo.olts["olt-001"] = &domain.OLT{
		ID:                     "olt-001",
		TenantID:               "tenant-001",
		Name:                   "OLT-Old",
		Host:                   "192.168.1.100",
		SNMPVersion:            domain.SNMPv2c,
		SNMPCommunityEncrypted: "enc:oldcommunity",
		CLIProtocol:            domain.CLIProtocolSSH,
		CLIPort:                22,
		CLIUsername:            "admin",
		CLIPasswordEncrypted:   "enc:oldpass",
		Status:                 domain.OLTStatusOnline,
	}

	newPort := 2222
	req := domain.UpdateOLTRequest{
		Name:          "OLT-New",
		CLIPassword:   "newpass123",
		SNMPCommunity: "newcommunity",
		CLIPort:       &newPort,
	}

	resp, err := mgr.Update(ctx, "olt-001", req)
	if err != nil {
		t.Fatalf("Update gagal: %v", err)
	}

	if resp.Name != "OLT-New" {
		t.Errorf("nama tidak terupdate: got %q, want %q", resp.Name, "OLT-New")
	}

	// Verifikasi kredensial terenkripsi ulang
	olt := repo.olts["olt-001"]
	if olt.CLIPasswordEncrypted != "enc:newpass123" {
		t.Errorf("CLI password tidak terenkripsi ulang: got %q", olt.CLIPasswordEncrypted)
	}
	if olt.SNMPCommunityEncrypted != "enc:newcommunity" {
		t.Errorf("SNMP community tidak terenkripsi ulang: got %q", olt.SNMPCommunityEncrypted)
	}
	if olt.CLIPort != 2222 {
		t.Errorf("CLI port tidak terupdate: got %d, want 2222", olt.CLIPort)
	}
}

func TestOLTManager_Update_InvalidStatusTransition(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.olts["olt-001"] = &domain.OLT{
		ID:       "olt-001",
		TenantID: "tenant-001",
		Status:   domain.OLTStatusOffline,
	}

	req := domain.UpdateOLTRequest{Status: "offline"}
	_, err := mgr.Update(ctx, "olt-001", req)
	if !errors.Is(err, domain.ErrOLTInvalidStatusTransition) {
		t.Errorf("expected ErrOLTInvalidStatusTransition, got: %v", err)
	}
}

func TestOLTManager_Update_NameConflict(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.olts["olt-001"] = &domain.OLT{
		ID:       "olt-001",
		TenantID: "tenant-001",
		Name:     "OLT-A",
		Status:   domain.OLTStatusOnline,
	}
	repo.nameExists = true

	req := domain.UpdateOLTRequest{Name: "OLT-B"}
	_, err := mgr.Update(ctx, "olt-001", req)
	if !errors.Is(err, domain.ErrOLTNameExists) {
		t.Errorf("expected ErrOLTNameExists, got: %v", err)
	}
}

func TestOLTManager_Delete(t *testing.T) {
	mgr, repo, _, _, hc := newTestOLTManager()
	ctx := context.Background()

	repo.olts["olt-001"] = &domain.OLT{ID: "olt-001"}

	err := mgr.Delete(ctx, "olt-001")
	if err != nil {
		t.Fatalf("Delete gagal: %v", err)
	}

	// Verifikasi OLT dihapus dari repo
	if _, ok := repo.olts["olt-001"]; ok {
		t.Error("OLT masih ada di repo setelah delete")
	}

	// Verifikasi pemeriksa kesehatan RemoveOLT dipanggil
	if len(hc.removedOLTs) != 1 || hc.removedOLTs[0] != "olt-001" {
		t.Errorf("pemeriksa kesehatan RemoveOLT tidak dipanggil dengan benar: %v", hc.removedOLTs)
	}
}

func TestOLTManager_List(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.listResult = &domain.OLTListResult{
		Data: []*domain.OLTResponse{
			{ID: "olt-001", Name: "OLT-A"},
			{ID: "olt-002", Name: "OLT-B"},
		},
		Total:      2,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	result, err := mgr.List(ctx, domain.OLTListParams{TenantID: "tenant-001", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List gagal: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("total salah: got %d, want 2", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("jumlah data salah: got %d, want 2", len(result.Data))
	}
}

func TestOLTManager_TestSNMP(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.olts["olt-001"] = &domain.OLT{
		ID:                     "olt-001",
		TenantID:               "tenant-001",
		Host:                   "192.168.1.100",
		SNMPVersion:            domain.SNMPv2c,
		SNMPPort:               161,
		SNMPCommunityEncrypted: "enc:public",
		CLIProtocol:            domain.CLIProtocolSSH,
		CLIPort:                22,
		CLIUsername:            "admin",
		CLIPasswordEncrypted:   "enc:secret",
		Brand:                  domain.BrandZTE,
		Status:                 domain.OLTStatusOnline,
	}

	sysInfo, err := mgr.TestSNMP(ctx, "olt-001")
	if err != nil {
		t.Fatalf("TestSNMP gagal: %v", err)
	}

	if sysInfo.Brand != domain.BrandZTE {
		t.Errorf("brand salah: got %q, want %q", sysInfo.Brand, domain.BrandZTE)
	}
	if sysInfo.Model != "C320" {
		t.Errorf("model salah: got %q, want %q", sysInfo.Model, "C320")
	}
}

func TestOLTManager_TestCLI(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.olts["olt-001"] = &domain.OLT{
		ID:                   "olt-001",
		TenantID:             "tenant-001",
		Host:                 "192.168.1.100",
		CLIProtocol:          domain.CLIProtocolSSH,
		CLIPort:              22,
		CLIUsername:          "admin",
		CLIPasswordEncrypted: "enc:secret",
		Status:               domain.OLTStatusOnline,
	}

	result, err := mgr.TestCLI(ctx, "olt-001")
	if err != nil {
		t.Fatalf("TestCLI gagal: %v", err)
	}

	if !result.Success {
		t.Error("TestCLI harus berhasil")
	}
	if result.Banner != "ZTE C320> " {
		t.Errorf("banner salah: got %q, want %q", result.Banner, "ZTE C320> ")
	}
}

func TestOLTManager_TestCLI_Failure(t *testing.T) {
	mgr, repo, _, _, _ := newTestOLTManager()
	mgr.cliConn = &mockCLIConnector{testErr: errors.New("connection refused")}
	ctx := context.Background()

	repo.olts["olt-001"] = &domain.OLT{
		ID:                   "olt-001",
		CLIProtocol:          domain.CLIProtocolSSH,
		CLIPort:              22,
		CLIUsername:          "admin",
		CLIPasswordEncrypted: "enc:secret",
	}

	result, err := mgr.TestCLI(ctx, "olt-001")
	if err != nil {
		t.Fatalf("TestCLI tidak boleh return error, harus return CLITestResult: %v", err)
	}

	if result.Success {
		t.Error("TestCLI harus gagal")
	}
	if result.Error == "" {
		t.Error("TestCLI harus memiliki error message")
	}
}

func TestOLTManager_GetStatusSummary(t *testing.T) {
	mgr, repo, alarmRepo, _, _ := newTestOLTManager()
	ctx := context.Background()

	repo.statusCounts = map[domain.OLTStatus]int64{
		domain.OLTStatusOnline:      5,
		domain.OLTStatusOffline:     2,
		domain.OLTStatusMaintenance: 1,
	}
	alarmRepo.activeByTenant = 10

	summary, err := mgr.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary gagal: %v", err)
	}

	if summary.TotalOLTs != 8 {
		t.Errorf("total salah: got %d, want 8", summary.TotalOLTs)
	}
	if summary.OnlineCount != 5 {
		t.Errorf("online count salah: got %d, want 5", summary.OnlineCount)
	}
	if summary.OfflineCount != 2 {
		t.Errorf("offline count salah: got %d, want 2", summary.OfflineCount)
	}
	if summary.MaintenanceCount != 1 {
		t.Errorf("maintenance count salah: got %d, want 1", summary.MaintenanceCount)
	}
	if summary.ActiveAlarmCount != 10 {
		t.Errorf("alarm count salah: got %d, want 10", summary.ActiveAlarmCount)
	}
}
