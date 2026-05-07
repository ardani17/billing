// Package usecase - unit tests untuk sync engine.
package usecase

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

// mockSyncOLTRepo merekam panggilan untuk verifikasi di test.
type mockSyncOLTRepo struct {
	mu             sync.Mutex
	olts           map[string]*domain.OLT
	onlineOLTs     []*domain.OLT
	lastONTCountID string
	lastONTCount   int
}

func newMockSyncOLTRepo() *mockSyncOLTRepo {
	return &mockSyncOLTRepo{olts: make(map[string]*domain.OLT)}
}

func (r *mockSyncOLTRepo) Create(_ context.Context, o *domain.OLT) (*domain.OLT, error) {
	return o, nil
}
func (r *mockSyncOLTRepo) GetByID(_ context.Context, id string) (*domain.OLT, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.olts[id]; ok {
		return o, nil
	}
	return nil, domain.ErrOLTNotFound
}
func (r *mockSyncOLTRepo) Update(_ context.Context, o *domain.OLT) (*domain.OLT, error) {
	return o, nil
}
func (r *mockSyncOLTRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (r *mockSyncOLTRepo) List(_ context.Context, _ domain.OLTListParams) (*domain.OLTListResult, error) {
	return nil, nil
}
func (r *mockSyncOLTRepo) CountByStatus(_ context.Context) (map[domain.OLTStatus]int64, error) {
	return nil, nil
}
func (r *mockSyncOLTRepo) GetActiveOLTs(_ context.Context) ([]*domain.OLT, error) {
	return nil, nil
}
func (r *mockSyncOLTRepo) GetOnlineOLTs(_ context.Context) ([]*domain.OLT, error) {
	return r.onlineOLTs, nil
}
func (r *mockSyncOLTRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (r *mockSyncOLTRepo) UpdateHealthCheck(_ context.Context, _ string, _ domain.OLTHealthCheckUpdate) error {
	return nil
}
func (r *mockSyncOLTRepo) UpdateONTCounts(_ context.Context, id string, count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastONTCountID = id
	r.lastONTCount = count
	return nil
}

// mockSyncAdapter merekam panggilan adapter untuk verifikasi.
type mockSyncAdapter struct {
	ponPorts     []domain.PONPortStatus
	ontLists     map[int][]domain.ONTPortStatus // key: portIndex
	signals      map[string]*domain.ONTSignalInfo
	trafficStats map[int]*domain.PONTrafficStats
}

func newMockSyncAdapter() *mockSyncAdapter {
	return &mockSyncAdapter{
		ontLists:     make(map[int][]domain.ONTPortStatus),
		signals:      make(map[string]*domain.ONTSignalInfo),
		trafficStats: make(map[int]*domain.PONTrafficStats),
	}
}

func (a *mockSyncAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	return nil, nil
}
func (a *mockSyncAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, nil
}
func (a *mockSyncAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return a.ponPorts, nil
}
func (a *mockSyncAdapter) GetONTList(_ context.Context, portIndex int) ([]domain.ONTPortStatus, error) {
	return a.ontLists[portIndex], nil
}
func (a *mockSyncAdapter) GetONTSignal(_ context.Context, portIndex, ontIndex int) (*domain.ONTSignalInfo, error) {
	key := signalKey(portIndex, ontIndex)
	if s, ok := a.signals[key]; ok {
		return s, nil
	}
	return &domain.ONTSignalInfo{RxPowerDBm: -20.0, SignalLevel: domain.SignalNormal}, nil
}
func (a *mockSyncAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	return nil, nil
}
func (a *mockSyncAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) {
	return nil, nil
}
func (a *mockSyncAdapter) GetTrafficStats(_ context.Context, portIndex int) (*domain.PONTrafficStats, error) {
	if s, ok := a.trafficStats[portIndex]; ok {
		return s, nil
	}
	return &domain.PONTrafficStats{PortIndex: portIndex, RxBytes: 1000, TxBytes: 500}, nil
}
func (a *mockSyncAdapter) Ping(_ context.Context) error { return nil }

// --- Provisioning method stubs (diperlukan oleh interface OLTAdapter) ---
func (a *mockSyncAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *mockSyncAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *mockSyncAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *mockSyncAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *mockSyncAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *mockSyncAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, nil
}

func signalKey(port, ont int) string {
	return fmt.Sprintf("%d:%d", port, ont)
}

// mockSyncFactory mengembalikan adapter yang sudah dikonfigurasi.
type mockSyncFactory struct {
	adapter domain.OLTAdapter
}

func (f *mockSyncFactory) CreateAdapter(_ domain.OLTBrand, _ domain.SNMPConfig, _ domain.CLIConfig) (domain.OLTAdapter, error) {
	return f.adapter, nil
}

// mockSignalStore merekam panggilan Store untuk verifikasi.
type mockSignalStore struct {
	mu     sync.Mutex
	stored []signalStoreCall
}

type signalStoreCall struct {
	oltID     string
	portIndex int
	ontIndex  int
}

func (s *mockSignalStore) Store(_ context.Context, oltID string, portIndex, ontIndex int, _ domain.ONTSignalPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stored = append(s.stored, signalStoreCall{oltID, portIndex, ontIndex})
	return nil
}
func (s *mockSignalStore) Query(_ context.Context, _ string, _, _ int, _, _ time.Time) ([]domain.ONTSignalPoint, error) {
	return nil, nil
}
func (s *mockSignalStore) GetLatest(_ context.Context, _ string, _, _ int) (*domain.ONTSignalPoint, error) {
	return nil, nil
}

// mockTrafficStore merekam panggilan Store untuk verifikasi.
type mockTrafficStore struct {
	mu     sync.Mutex
	stored []trafficStoreCall
}

type trafficStoreCall struct {
	oltID     string
	portIndex int
}

func (s *mockTrafficStore) Store(_ context.Context, oltID string, portIndex int, _ domain.PONTrafficPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stored = append(s.stored, trafficStoreCall{oltID, portIndex})
	return nil
}
func (s *mockTrafficStore) Query(_ context.Context, _ string, _ int, _, _ time.Time) ([]domain.PONTrafficPoint, error) {
	return nil, nil
}
func (s *mockTrafficStore) GetLatest(_ context.Context, _ string, _ int) (*domain.PONTrafficPoint, error) {
	return nil, nil
}

type mockSyncEncryptor struct{}

func (e *mockSyncEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}
func (e *mockSyncEncryptor) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

// =============================================================================
// =============================================================================

func newTestSyncEngine() (*syncEngine, *mockSyncOLTRepo, *mockSyncAdapter, *mockSignalStore, *mockTrafficStore) {
	repo := newMockSyncOLTRepo()
	adapter := newMockSyncAdapter()
	signalStore := &mockSignalStore{}
	trafficStore := &mockTrafficStore{}

	se := &syncEngine{
		oltRepo:      repo,
		factory:      &mockSyncFactory{adapter: adapter},
		encryptor:    &mockSyncEncryptor{},
		signalStore:  signalStore,
		trafficStore: trafficStore,
		syncInterval: defaultSyncInterval,
	}
	return se, repo, adapter, signalStore, trafficStore
}

// makeSyncTestOLT membuat OLT entity untuk testing sync.
func makeSyncTestOLT(id string) *domain.OLT {
	return &domain.OLT{
		ID:                     id,
		TenantID:               "tenant-001",
		Name:                   "OLT-Sync-Test",
		Host:                   "192.168.1.100",
		SNMPVersion:            domain.SNMPv2c,
		SNMPPort:               161,
		SNMPCommunityEncrypted: "enc:public",
		CLIProtocol:            domain.CLIProtocolSSH,
		CLIPort:                22,
		CLIPasswordEncrypted:   "enc:secret",
		Brand:                  domain.BrandZTE,
		Status:                 domain.OLTStatusOnline,
	}
}

// =============================================================================
// =============================================================================

func TestSyncEngine_SyncOLT_Success(t *testing.T) {
	se, repo, adapter, signalStore, trafficStore := newTestSyncEngine()
	ctx := context.Background()

	olt := makeSyncTestOLT("olt-001")
	repo.olts["olt-001"] = olt

	// Setup adapter: 2 PON ports, masing-masing 3 ONT
	adapter.ponPorts = []domain.PONPortStatus{
		{PortIndex: 0, AdminStatus: "up", OperStatus: "up", ONTCount: 3},
		{PortIndex: 1, AdminStatus: "up", OperStatus: "up", ONTCount: 3},
	}
	adapter.ontLists[0] = []domain.ONTPortStatus{
		{ONTIndex: 0, SerialNumber: "SN-0-0", Status: "online"},
		{ONTIndex: 1, SerialNumber: "SN-0-1", Status: "online"},
		{ONTIndex: 2, SerialNumber: "SN-0-2", Status: "offline"},
	}
	adapter.ontLists[1] = []domain.ONTPortStatus{
		{ONTIndex: 0, SerialNumber: "SN-1-0", Status: "online"},
		{ONTIndex: 1, SerialNumber: "SN-1-1", Status: "online"},
		{ONTIndex: 2, SerialNumber: "SN-1-2", Status: "online"},
	}

	result, err := se.SyncOLT(ctx, "olt-001")
	if err != nil {
		t.Fatalf("SyncOLT gagal: %v", err)
	}

	// Verifikasi hasil sync
	if result.OLTID != "olt-001" {
		t.Errorf("OLT ID salah: got %q, want %q", result.OLTID, "olt-001")
	}
	if result.TotalONT != 6 {
		t.Errorf("total ONT salah: got %d, want 6", result.TotalONT)
	}
	// Semua ONT unmanaged karena DB kosong
	if result.UnmanagedCount != 6 {
		t.Errorf("unmanaged count salah: got %d, want 6", result.UnmanagedCount)
	}
	if result.MissingCount != 0 {
		t.Errorf("missing count salah: got %d, want 0", result.MissingCount)
	}
	if result.SyncedAt.IsZero() {
		t.Error("synced_at tidak boleh zero")
	}

	// Verifikasi UpdateONTCounts dipanggil
	repo.mu.Lock()
	if repo.lastONTCountID != "olt-001" {
		t.Errorf("UpdateONTCounts OLT ID salah: got %q", repo.lastONTCountID)
	}
	if repo.lastONTCount != 6 {
		t.Errorf("UpdateONTCounts count salah: got %d, want 6", repo.lastONTCount)
	}
	repo.mu.Unlock()

	// Verifikasi signal store dipanggil untuk setiap ONT (6 ONT total)
	signalStore.mu.Lock()
	if len(signalStore.stored) != 6 {
		t.Errorf("signal store calls salah: got %d, want 6", len(signalStore.stored))
	}
	signalStore.mu.Unlock()

	// Verifikasi traffic store dipanggil untuk setiap port (2 port)
	trafficStore.mu.Lock()
	if len(trafficStore.stored) != 2 {
		t.Errorf("traffic store calls salah: got %d, want 2", len(trafficStore.stored))
	}
	trafficStore.mu.Unlock()
}

// =============================================================================
// Unit Tes: SyncOLT - OLT tidak ditemukan
// =============================================================================

func TestSyncEngine_SyncOLT_NotFound(t *testing.T) {
	se, _, _, _, _ := newTestSyncEngine()
	ctx := context.Background()

	_, err := se.SyncOLT(ctx, "nonexistent")
	if err != domain.ErrOLTNotFound {
		t.Errorf("expected ErrOLTNotFound, got: %v", err)
	}
}

// =============================================================================
// Unit Tes: Signal dan traffic store calls
// =============================================================================

func TestSyncEngine_StoreSignalAndTraffic(t *testing.T) {
	se, repo, adapter, signalStore, trafficStore := newTestSyncEngine()
	ctx := context.Background()

	olt := makeSyncTestOLT("olt-002")
	repo.olts["olt-002"] = olt

	// Setup: 1 port dengan 2 ONT
	adapter.ponPorts = []domain.PONPortStatus{
		{PortIndex: 0, AdminStatus: "up", OperStatus: "up", ONTCount: 2},
	}
	adapter.ontLists[0] = []domain.ONTPortStatus{
		{ONTIndex: 0, SerialNumber: "SN-A", Status: "online"},
		{ONTIndex: 1, SerialNumber: "SN-B", Status: "online"},
	}
	adapter.trafficStats[0] = &domain.PONTrafficStats{
		PortIndex: 0, RxBytes: 5000, TxBytes: 3000, RxPackets: 100, TxPackets: 50,
	}

	_, err := se.SyncOLT(ctx, "olt-002")
	if err != nil {
		t.Fatalf("SyncOLT gagal: %v", err)
	}

	// Verifikasi signal store: 2 ONT pada port 0
	signalStore.mu.Lock()
	if len(signalStore.stored) != 2 {
		t.Errorf("signal store calls: got %d, want 2", len(signalStore.stored))
	}
	for _, call := range signalStore.stored {
		if call.oltID != "olt-002" {
			t.Errorf("signal store OLT ID salah: got %q", call.oltID)
		}
		if call.portIndex != 0 {
			t.Errorf("signal store port index salah: got %d", call.portIndex)
		}
	}
	signalStore.mu.Unlock()

	// Verifikasi traffic store: 1 port
	trafficStore.mu.Lock()
	if len(trafficStore.stored) != 1 {
		t.Errorf("traffic store calls: got %d, want 1", len(trafficStore.stored))
	}
	if len(trafficStore.stored) > 0 && trafficStore.stored[0].oltID != "olt-002" {
		t.Errorf("traffic store OLT ID salah: got %q", trafficStore.stored[0].oltID)
	}
	trafficStore.mu.Unlock()
}

// =============================================================================
// Unit Tes: compareONTSets - contoh spesifik
// =============================================================================

func TestCompareONTSets_EmptySets(t *testing.T) {
	result := compareONTSets(nil, nil)
	if len(result.Unmanaged) != 0 || len(result.Missing) != 0 ||
		len(result.Updated) != 0 || len(result.Synced) != 0 {
		t.Error("semua kategori harus kosong untuk input kosong")
	}
}

func TestCompareONTSets_AllUnmanaged(t *testing.T) {
	oltONTs := []domain.ONTPortStatus{
		{SerialNumber: "SN-001", Status: "online", ONTIndex: 0, Name: "ONT-1"},
		{SerialNumber: "SN-002", Status: "online", ONTIndex: 1, Name: "ONT-2"},
	}
	result := compareONTSets(oltONTs, nil)
	if len(result.Unmanaged) != 2 {
		t.Errorf("unmanaged: got %d, want 2", len(result.Unmanaged))
	}
	if len(result.Missing) != 0 {
		t.Errorf("missing: got %d, want 0", len(result.Missing))
	}
}

func TestCompareONTSets_AllMissing(t *testing.T) {
	dbONTs := []domain.ONTPortStatus{
		{SerialNumber: "SN-001", Status: "online", ONTIndex: 0, Name: "ONT-1"},
	}
	result := compareONTSets(nil, dbONTs)
	if len(result.Missing) != 1 {
		t.Errorf("missing: got %d, want 1", len(result.Missing))
	}
	if len(result.Unmanaged) != 0 {
		t.Errorf("unmanaged: got %d, want 0", len(result.Unmanaged))
	}
}

func TestCompareONTSets_MixedCategories(t *testing.T) {
	oltONTs := []domain.ONTPortStatus{
		{SerialNumber: "SN-001", Status: "online", ONTIndex: 0, Name: "ONT-1"}, // synced
		{SerialNumber: "SN-002", Status: "online", ONTIndex: 1, Name: "ONT-2"}, // updated (status berbeda)
		{SerialNumber: "SN-003", Status: "online", ONTIndex: 2, Name: "ONT-3"}, // unmanaged
	}
	dbONTs := []domain.ONTPortStatus{
		{SerialNumber: "SN-001", Status: "online", ONTIndex: 0, Name: "ONT-1"},  // synced
		{SerialNumber: "SN-002", Status: "offline", ONTIndex: 1, Name: "ONT-2"}, // updated
		{SerialNumber: "SN-004", Status: "online", ONTIndex: 3, Name: "ONT-4"},  // missing
	}

	result := compareONTSets(oltONTs, dbONTs)

	if len(result.Synced) != 1 {
		t.Errorf("synced: got %d, want 1", len(result.Synced))
	}
	if len(result.Updated) != 1 {
		t.Errorf("updated: got %d, want 1", len(result.Updated))
	}
	if len(result.Unmanaged) != 1 {
		t.Errorf("unmanaged: got %d, want 1", len(result.Unmanaged))
	}
	if len(result.Missing) != 1 {
		t.Errorf("missing: got %d, want 1", len(result.Missing))
	}

	// Verifikasi serial number di kategori yang benar
	if result.Unmanaged[0].SerialNumber != "SN-003" {
		t.Errorf("unmanaged SN salah: got %q", result.Unmanaged[0].SerialNumber)
	}
	if result.Missing[0].SerialNumber != "SN-004" {
		t.Errorf("missing SN salah: got %q", result.Missing[0].SerialNumber)
	}
}
