// Package usecase - unit tests untuk OLT pemeriksa kesehatan.
// Menguji perilaku handleOLTSuccess dan handleOLTFailure secara langsung
package usecase

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

// mockOLTHealthRepo merekam panggilan UpdateHealthCheck untuk OLT tes pemeriksa kesehatan.
type mockOLTHealthRepo struct {
	mu              sync.Mutex
	olts            map[string]*domain.OLT
	lastHealthCheck *domain.OLTHealthCheckUpdate
	lastOLTID       string
}

func newMockOLTHealthRepo() *mockOLTHealthRepo {
	return &mockOLTHealthRepo{olts: make(map[string]*domain.OLT)}
}

func (r *mockOLTHealthRepo) Create(_ context.Context, o *domain.OLT) (*domain.OLT, error) {
	return o, nil
}
func (r *mockOLTHealthRepo) GetByID(_ context.Context, id string) (*domain.OLT, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.olts[id]; ok {
		return o, nil
	}
	return nil, domain.ErrOLTNotFound
}
func (r *mockOLTHealthRepo) Update(_ context.Context, o *domain.OLT) (*domain.OLT, error) {
	return o, nil
}
func (r *mockOLTHealthRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (r *mockOLTHealthRepo) List(_ context.Context, _ domain.OLTListParams) (*domain.OLTListResult, error) {
	return nil, nil
}
func (r *mockOLTHealthRepo) CountByStatus(_ context.Context) (map[domain.OLTStatus]int64, error) {
	return nil, nil
}
func (r *mockOLTHealthRepo) GetActiveOLTs(_ context.Context) ([]*domain.OLT, error) {
	return nil, nil
}
func (r *mockOLTHealthRepo) GetOnlineOLTs(_ context.Context) ([]*domain.OLT, error) {
	return nil, nil
}
func (r *mockOLTHealthRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (r *mockOLTHealthRepo) UpdateHealthCheck(_ context.Context, id string, params domain.OLTHealthCheckUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastOLTID = id
	r.lastHealthCheck = &params
	return nil
}
func (r *mockOLTHealthRepo) UpdateONTCounts(_ context.Context, _ string, _ int) error {
	return nil
}

// mockOLTHealthEventPub merekam panggilan terbitkan event OLT.
type mockOLTHealthEventPub struct {
	mu            sync.Mutex
	offlineCalled bool
	onlineCalled  bool
	offlineOLTID  string
	onlineOLTID   string
}

func (p *mockOLTHealthEventPub) PublishDeviceOffline(_ context.Context, payload domain.OLTDeviceOfflinePayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offlineCalled = true
	p.offlineOLTID = payload.OLTID
	return nil
}
func (p *mockOLTHealthEventPub) PublishDeviceOnline(_ context.Context, payload domain.OLTDeviceOnlinePayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onlineCalled = true
	p.onlineOLTID = payload.OLTID
	return nil
}
func (p *mockOLTHealthEventPub) PublishAlarm(_ context.Context, _ domain.OLTAlarmPayload) error {
	return nil
}

// --- Provisioning event stubs (diperlukan oleh interface OLTEventPublisher) ---
func (p *mockOLTHealthEventPub) PublishONTProvisioned(_ context.Context, _ domain.ONTProvisionedPayload) error {
	return nil
}
func (p *mockOLTHealthEventPub) PublishONTDecommissioned(_ context.Context, _ domain.ONTDecommissionedPayload) error {
	return nil
}
func (p *mockOLTHealthEventPub) PublishONTAutoProvisioned(_ context.Context, _ domain.ONTAutoProvisionedPayload) error {
	return nil
}
func (p *mockOLTHealthEventPub) PublishONTAutoProvisionFailed(_ context.Context, _ domain.ONTAutoProvisionFailedPayload) error {
	return nil
}
func (p *mockOLTHealthEventPub) PublishONTPortMigrated(_ context.Context, _ domain.ONTPortMigratedPayload) error {
	return nil
}

type mockOLTHealthEncryptor struct{}

func (e *mockOLTHealthEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}
func (e *mockOLTHealthEncryptor) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

type mockOLTHealthFactory struct {
	adapter domain.OLTAdapter
}

func (f *mockOLTHealthFactory) CreateAdapter(_ domain.OLTBrand, _ domain.SNMPConfig, _ domain.CLIConfig) (domain.OLTAdapter, error) {
	return f.adapter, nil
}

// =============================================================================
// =============================================================================

func newTestOLTHealthChecker() (*oltHealthChecker, *mockOLTHealthRepo, *mockOLTHealthEventPub) {
	repo := newMockOLTHealthRepo()
	eventPub := &mockOLTHealthEventPub{}
	crypto := &mockOLTHealthEncryptor{}
	factory := &mockOLTHealthFactory{adapter: &mockOLTAdapter{}}

	hc := &oltHealthChecker{
		oltRepo:   repo,
		factory:   factory,
		encryptor: crypto,
		eventPub:  eventPub,
		workers:   make(map[string]*oltWorker),
	}
	return hc, repo, eventPub
}

// makeTestOLT membuat OLT entity untuk testing dengan parameter yang diberikan.
func makeTestOLT(id string, status domain.OLTStatus, failureCount int) *domain.OLT {
	return &domain.OLT{
		ID:           id,
		TenantID:     "tenant-001",
		Name:         "OLT-Test",
		Host:         "192.168.1.100",
		SNMPVersion:  domain.SNMPv2c,
		SNMPPort:     161,
		Status:       status,
		FailureCount: failureCount,
	}
}

// =============================================================================
// =============================================================================

func TestOLTHealthChecker_SuccessResetsFailureCount(t *testing.T) {
	hc, repo, _ := newTestOLTHealthChecker()
	ctx := context.Background()

	olt := makeTestOLT("olt-001", domain.OLTStatusOnline, 2)
	hc.handleOLTSuccess(ctx, olt)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.lastHealthCheck == nil {
		t.Fatal("UpdateHealthCheck tidak dipanggil")
	}
	if repo.lastHealthCheck.FailureCount != 0 {
		t.Fatalf("failure_count harus 0, dapat %d", repo.lastHealthCheck.FailureCount)
	}
	if repo.lastHealthCheck.LastCheckedAt == nil {
		t.Fatal("last_checked_at harus di-update")
	}
	if repo.lastHealthCheck.LastOnlineAt == nil {
		t.Fatal("last_online_at harus di-update")
	}
	// Status tidak boleh berubah jika sudah online
	if repo.lastHealthCheck.Status != nil {
		t.Fatal("status tidak boleh berubah saat sudah online")
	}
}

// =============================================================================
// =============================================================================

func TestOLTHealthChecker_FailureIncrementsFailureCount(t *testing.T) {
	hc, repo, _ := newTestOLTHealthChecker()
	ctx := context.Background()

	olt := makeTestOLT("olt-001", domain.OLTStatusOnline, 0)
	hc.handleOLTFailure(ctx, olt)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.lastHealthCheck == nil {
		t.Fatal("UpdateHealthCheck tidak dipanggil")
	}
	if repo.lastHealthCheck.FailureCount != 1 {
		t.Fatalf("failure_count harus 1, dapat %d", repo.lastHealthCheck.FailureCount)
	}
	if repo.lastHealthCheck.LastCheckedAt == nil {
		t.Fatal("last_checked_at harus di-update")
	}
	// Belum mencapai threshold - status tidak boleh berubah
	if repo.lastHealthCheck.Status != nil {
		t.Fatal("status tidak boleh berubah saat failure_count < 3")
	}
}

// =============================================================================
// =============================================================================

func TestOLTHealthChecker_ThreeFailuresSetOffline(t *testing.T) {
	hc, repo, eventPub := newTestOLTHealthChecker()
	ctx := context.Background()

	// OLT online dengan failure_count=2 (satu lagi -> threshold)
	olt := makeTestOLT("olt-001", domain.OLTStatusOnline, 2)
	hc.handleOLTFailure(ctx, olt)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.lastHealthCheck == nil {
		t.Fatal("UpdateHealthCheck tidak dipanggil")
	}
	if repo.lastHealthCheck.FailureCount != 3 {
		t.Fatalf("failure_count harus 3, dapat %d", repo.lastHealthCheck.FailureCount)
	}
	if repo.lastHealthCheck.Status == nil {
		t.Fatal("status harus di-set ke offline saat failure_count >= 3")
	}
	if *repo.lastHealthCheck.Status != domain.OLTStatusOffline {
		t.Fatalf("status harus offline, dapat %s", *repo.lastHealthCheck.Status)
	}

	eventPub.mu.Lock()
	defer eventPub.mu.Unlock()

	if !eventPub.offlineCalled {
		t.Fatal("PublishDeviceOffline harus dipanggil saat transisi ke offline")
	}
	if eventPub.offlineOLTID != "olt-001" {
		t.Fatalf("offline event OLT ID salah: got %q", eventPub.offlineOLTID)
	}
}

// =============================================================================
// =============================================================================

func TestOLTHealthChecker_MaintenanceSkip(t *testing.T) {
	hc, repo, _ := newTestOLTHealthChecker()
	ctx := context.Background()

	olt := makeTestOLT("olt-001", domain.OLTStatusMaintenance, 0)
	repo.olts["olt-001"] = olt

	hc.checkOLT(ctx, "olt-001")

	repo.mu.Lock()
	defer repo.mu.Unlock()

	// Tidak boleh ada UpdateHealthCheck dipanggil
	if repo.lastHealthCheck != nil {
		t.Fatal("UpdateHealthCheck tidak boleh dipanggil saat OLT dalam maintenance")
	}
}

// =============================================================================
// Tes: Recovery (offline -> online) + event published
// =============================================================================

func TestOLTHealthChecker_RecoveryOfflineToOnline(t *testing.T) {
	hc, repo, eventPub := newTestOLTHealthChecker()
	ctx := context.Background()

	lastOnline := time.Now().Add(-10 * time.Minute)
	olt := makeTestOLT("olt-001", domain.OLTStatusOffline, 3)
	olt.LastOnlineAt = &lastOnline

	beforeCall := time.Now()
	hc.handleOLTSuccess(ctx, olt)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.lastHealthCheck == nil {
		t.Fatal("UpdateHealthCheck tidak dipanggil")
	}
	if repo.lastHealthCheck.FailureCount != 0 {
		t.Fatalf("failure_count harus 0 setelah recovery, dapat %d", repo.lastHealthCheck.FailureCount)
	}
	if repo.lastHealthCheck.Status == nil {
		t.Fatal("status harus di-set ke online saat recovery")
	}
	if *repo.lastHealthCheck.Status != domain.OLTStatusOnline {
		t.Fatalf("status harus online, dapat %s", *repo.lastHealthCheck.Status)
	}
	if repo.lastHealthCheck.LastCheckedAt.Before(beforeCall) {
		t.Fatal("last_checked_at harus >= waktu sebelum panggilan")
	}

	eventPub.mu.Lock()
	defer eventPub.mu.Unlock()

	if !eventPub.onlineCalled {
		t.Fatal("PublishDeviceOnline harus dipanggil saat recovery offline→online")
	}
	if eventPub.onlineOLTID != "olt-001" {
		t.Fatalf("online event OLT ID salah: got %q", eventPub.onlineOLTID)
	}
}

// =============================================================================
// Tes: Already offline - no duplicate offline event
// =============================================================================

func TestOLTHealthChecker_AlreadyOfflineNoDuplicateEvent(t *testing.T) {
	hc, repo, eventPub := newTestOLTHealthChecker()
	ctx := context.Background()

	// OLT sudah offline dengan failure_count tinggi
	olt := makeTestOLT("olt-001", domain.OLTStatusOffline, 5)
	hc.handleOLTFailure(ctx, olt)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.lastHealthCheck.FailureCount != 6 {
		t.Fatalf("failure_count harus 6, dapat %d", repo.lastHealthCheck.FailureCount)
	}
	// Status tidak boleh di-atur ulang karena sudah offline
	if repo.lastHealthCheck.Status != nil {
		t.Fatal("status tidak boleh di-set ulang saat sudah offline")
	}

	eventPub.mu.Lock()
	defer eventPub.mu.Unlock()

	if eventPub.offlineCalled {
		t.Fatal("PublishDeviceOffline tidak boleh dipanggil saat sudah offline")
	}
}
