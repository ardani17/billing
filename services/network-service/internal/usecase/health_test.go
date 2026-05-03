// Package usecase — property-based tests untuk health checker.
// Menguji perilaku handleSuccess dan handleFailure secara langsung
// menggunakan mock implementations dari domain interfaces.
package usecase

import (
	"context"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock implementations — merekam panggilan untuk verifikasi di test
// =============================================================================

// mockRouterRepo merekam panggilan UpdateHealthCheck.
type mockRouterRepo struct {
	mu              sync.Mutex
	lastHealthCheck *domain.HealthCheckUpdate
	lastRouterID    string
}

func (m *mockRouterRepo) Create(_ context.Context, r *domain.Router) (*domain.Router, error) {
	return r, nil
}
func (m *mockRouterRepo) GetByID(_ context.Context, id string) (*domain.Router, error) {
	return nil, nil
}
func (m *mockRouterRepo) Update(_ context.Context, r *domain.Router) (*domain.Router, error) {
	return r, nil
}
func (m *mockRouterRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *mockRouterRepo) List(_ context.Context, _ domain.RouterListParams) (*domain.RouterListResult, error) {
	return nil, nil
}
func (m *mockRouterRepo) CountByStatus(_ context.Context) (map[domain.RouterStatus]int64, error) {
	return nil, nil
}
func (m *mockRouterRepo) GetActiveRouters(_ context.Context) ([]*domain.Router, error) {
	return nil, nil
}
func (m *mockRouterRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRouterRepo) UpdateHealthCheck(_ context.Context, id string, params domain.HealthCheckUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastRouterID = id
	m.lastHealthCheck = &params
	return nil
}

// mockMetricsStore merekam panggilan Store.
type mockMetricsStore struct {
	mu       sync.Mutex
	stored   bool
	routerID string
	metrics  *domain.RouterMetrics
}

func (m *mockMetricsStore) Store(_ context.Context, routerID string, metrics domain.RouterMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stored = true
	m.routerID = routerID
	m.metrics = &metrics
	return nil
}
func (m *mockMetricsStore) Query(_ context.Context, _ string, _, _ time.Time) ([]domain.RouterMetricsPoint, error) {
	return nil, nil
}
func (m *mockMetricsStore) GetLatest(_ context.Context, _ string) (*domain.RouterMetricsPoint, error) {
	return nil, nil
}

// mockEventPublisher merekam panggilan publish event.
type mockEventPublisher struct {
	mu                    sync.Mutex
	offlineCalled         bool
	onlineCalled          bool
	rebootCalled          bool
	rebootPrevUptime      int64
	rebootCurrUptime      int64
	offlineRouter         *domain.Router
	onlineRouter          *domain.Router
	onlineDowntimeDur     time.Duration
}

func (m *mockEventPublisher) PublishRouterOffline(_ context.Context, router *domain.Router) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offlineCalled = true
	m.offlineRouter = router
	return nil
}
func (m *mockEventPublisher) PublishRouterOnline(_ context.Context, router *domain.Router, downtime time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onlineCalled = true
	m.onlineRouter = router
	m.onlineDowntimeDur = downtime
	return nil
}
func (m *mockEventPublisher) PublishUnexpectedReboot(_ context.Context, _ *domain.Router, prev, curr int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rebootCalled = true
	m.rebootPrevUptime = prev
	m.rebootCurrUptime = curr
	return nil
}

// mockCredentialEncryptor — tidak digunakan langsung oleh handleSuccess/handleFailure.
type mockCredentialEncryptor struct{}

func (m *mockCredentialEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}
func (m *mockCredentialEncryptor) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

// =============================================================================
// Helper — membuat healthChecker instance untuk testing
// =============================================================================

// newTestHealthChecker membuat healthChecker dengan mock dependencies.
func newTestHealthChecker() (*healthChecker, *mockRouterRepo, *mockMetricsStore, *mockEventPublisher) {
	repo := &mockRouterRepo{}
	metrics := &mockMetricsStore{}
	events := &mockEventPublisher{}
	crypto := &mockCredentialEncryptor{}

	hc := &healthChecker{
		repo:    repo,
		metrics: metrics,
		events:  events,
		crypto:  crypto,
		workers: make(map[string]*routerWorker),
	}
	return hc, repo, metrics, events
}

// genRouterStatus menghasilkan status online atau offline (bukan maintenance).
func genRouterStatus(t *rapid.T) domain.RouterStatus {
	statuses := []domain.RouterStatus{domain.StatusOnline, domain.StatusOffline}
	return statuses[rapid.IntRange(0, len(statuses)-1).Draw(t, "status_idx")]
}

// =============================================================================
// Property 5: Successful health check resets failure state
// =============================================================================

// **Validates: Requirements 6.2, 6.5**
//
// Untuk router dengan failure_count >= 0 dan status online/offline,
// setelah health check berhasil:
// - failure_count HARUS 0
// - last_checked_at HARUS di-update
// - Jika sebelumnya offline, status HARUS transisi ke online
func TestProperty5_SuccessfulHealthCheckResetsFailureState(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hc, repo, _, events := newTestHealthChecker()

		// Generate failure_count acak (0-10)
		failureCount := rapid.IntRange(0, 10).Draw(t, "failure_count")
		status := genRouterStatus(t)

		// Buat router dengan state yang di-generate
		uptime := rapid.Int64Range(1000, 999999).Draw(t, "current_uptime")
		prevUptime := rapid.Int64Range(uptime, uptime+100000).Draw(t, "prev_uptime")

		router := &domain.Router{
			ID:            "router-test-1",
			TenantID:      "tenant-1",
			Name:          "Router Test",
			Status:        status,
			FailureCount:  failureCount,
			LastUptimeSec: &prevUptime,
		}

		// System resource dari health check yang berhasil
		sysRes := &domain.SystemResource{
			Version:   "6.49.10",
			BoardName: "RB750Gr3",
			CPUCount:  2,
			CPULoad:   rapid.IntRange(0, 100).Draw(t, "cpu_load"),
			TotalRAM:  268435456, // 256MB
			FreeRAM:   134217728, // 128MB
			Uptime:    uptime,
		}

		beforeCall := time.Now()
		hc.handleSuccess(context.Background(), router, sysRes)

		// Verifikasi: failure_count harus 0
		repo.mu.Lock()
		defer repo.mu.Unlock()

		if repo.lastHealthCheck == nil {
			t.Fatal("UpdateHealthCheck tidak dipanggil")
		}
		if repo.lastHealthCheck.FailureCount != 0 {
			t.Fatalf("failure_count harus 0, dapat %d", repo.lastHealthCheck.FailureCount)
		}

		// Verifikasi: last_checked_at harus di-update (tidak nil dan >= beforeCall)
		if repo.lastHealthCheck.LastCheckedAt == nil {
			t.Fatal("last_checked_at harus di-update, dapat nil")
		}
		if repo.lastHealthCheck.LastCheckedAt.Before(beforeCall) {
			t.Fatal("last_checked_at harus >= waktu sebelum panggilan")
		}

		// Verifikasi: jika sebelumnya offline, status harus transisi ke online
		events.mu.Lock()
		defer events.mu.Unlock()

		if status == domain.StatusOffline {
			if repo.lastHealthCheck.Status == nil {
				t.Fatal("status harus di-set ke online saat sebelumnya offline")
			}
			if *repo.lastHealthCheck.Status != domain.StatusOnline {
				t.Fatalf("status harus online, dapat %s", *repo.lastHealthCheck.Status)
			}
			if !events.onlineCalled {
				t.Fatal("PublishRouterOnline harus dipanggil saat transisi offline→online")
			}
		}
	})
}

// =============================================================================
// Property 6: Failed health check increments failure count
// =============================================================================

// **Validates: Requirements 6.3, 6.4**
//
// Untuk router dengan failure_count N (0 <= N < 3),
// setelah health check gagal:
// - failure_count HARUS N + 1
// - Saat failure_count mencapai 3, status HARUS transisi ke offline
func TestProperty6_FailedHealthCheckIncrementsFailureCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hc, repo, _, events := newTestHealthChecker()

		// Generate failure_count 0-2 (belum mencapai threshold)
		failureCount := rapid.IntRange(0, 2).Draw(t, "failure_count")
		status := genRouterStatus(t)

		router := &domain.Router{
			ID:           "router-test-2",
			TenantID:     "tenant-1",
			Name:         "Router Fail Test",
			Status:       status,
			FailureCount: failureCount,
		}

		hc.handleFailure(context.Background(), router)

		// Verifikasi: failure_count harus N + 1
		repo.mu.Lock()
		defer repo.mu.Unlock()

		if repo.lastHealthCheck == nil {
			t.Fatal("UpdateHealthCheck tidak dipanggil")
		}

		expectedCount := failureCount + 1
		if repo.lastHealthCheck.FailureCount != expectedCount {
			t.Fatalf("failure_count harus %d, dapat %d", expectedCount, repo.lastHealthCheck.FailureCount)
		}

		// Verifikasi: saat failure_count mencapai threshold (3), status harus offline
		events.mu.Lock()
		defer events.mu.Unlock()

		if expectedCount >= failureThreshold {
			// Status harus di-set ke offline (kecuali sudah offline)
			if status != domain.StatusOffline {
				if repo.lastHealthCheck.Status == nil {
					t.Fatal("status harus di-set ke offline saat failure_count >= 3")
				}
				if *repo.lastHealthCheck.Status != domain.StatusOffline {
					t.Fatalf("status harus offline, dapat %s", *repo.lastHealthCheck.Status)
				}
				if !events.offlineCalled {
					t.Fatal("PublishRouterOffline harus dipanggil saat transisi ke offline")
				}
			}
		} else {
			// Belum mencapai threshold — status tidak boleh berubah ke offline
			if repo.lastHealthCheck.Status != nil {
				t.Fatalf("status tidak boleh berubah saat failure_count < 3, dapat %s", *repo.lastHealthCheck.Status)
			}
		}
	})
}

// =============================================================================
// Property 7: Reboot detection via uptime comparison
// =============================================================================

// **Validates: Requirements 6.6**
//
// Untuk router dengan previous uptime P > 0 dan current uptime C < P,
// health checker HARUS mendeteksi reboot dan publish event
// mikrotik.router_unexpected_reboot dengan P dan C yang benar.
func TestProperty7_RebootDetectionViaUptimeComparison(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hc, _, _, events := newTestHealthChecker()

		// Generate previous uptime > 0 dan current uptime < previous
		prevUptime := rapid.Int64Range(100, 999999).Draw(t, "prev_uptime")
		currUptime := rapid.Int64Range(1, prevUptime-1).Draw(t, "curr_uptime")

		router := &domain.Router{
			ID:            "router-test-3",
			TenantID:      "tenant-1",
			Name:          "Router Reboot Test",
			Status:        domain.StatusOnline,
			FailureCount:  0,
			LastUptimeSec: &prevUptime,
		}

		sysRes := &domain.SystemResource{
			Version:   "6.49.10",
			BoardName: "RB750Gr3",
			CPUCount:  2,
			CPULoad:   25,
			TotalRAM:  268435456,
			FreeRAM:   134217728,
			Uptime:    currUptime,
		}

		hc.handleSuccess(context.Background(), router, sysRes)

		// Verifikasi: PublishUnexpectedReboot harus dipanggil
		events.mu.Lock()
		defer events.mu.Unlock()

		if !events.rebootCalled {
			t.Fatal("PublishUnexpectedReboot harus dipanggil saat current uptime < previous uptime")
		}

		// Verifikasi: parameter prevUptime dan currUptime harus benar
		if events.rebootPrevUptime != prevUptime {
			t.Fatalf("prevUptime harus %d, dapat %d", prevUptime, events.rebootPrevUptime)
		}
		if events.rebootCurrUptime != currUptime {
			t.Fatalf("currUptime harus %d, dapat %d", currUptime, events.rebootCurrUptime)
		}
	})
}
