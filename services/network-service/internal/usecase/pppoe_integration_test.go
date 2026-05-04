// pppoe_integration_test.go — integration test (unit-level) untuk PPPoE Manager.
// Menguji flow end-to-end: event masuk → manager → mock adapter → DB → event keluar.
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/adapter"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// --- Mock implementations (prefix "integ" untuk menghindari konflik dengan health_test.go) ---

type integRouterRepo struct{ router *domain.Router }

func (m *integRouterRepo) GetByID(_ context.Context, _ string) (*domain.Router, error) {
	return m.router, nil
}
func (m *integRouterRepo) Create(_ context.Context, r *domain.Router) (*domain.Router, error) {
	return r, nil
}
func (m *integRouterRepo) Update(_ context.Context, r *domain.Router) (*domain.Router, error) {
	return r, nil
}
func (m *integRouterRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *integRouterRepo) List(_ context.Context, _ domain.RouterListParams) (*domain.RouterListResult, error) {
	return nil, nil
}
func (m *integRouterRepo) CountByStatus(_ context.Context) (map[domain.RouterStatus]int64, error) {
	return nil, nil
}
func (m *integRouterRepo) GetActiveRouters(_ context.Context) ([]*domain.Router, error) {
	return nil, nil
}
func (m *integRouterRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *integRouterRepo) UpdateHealthCheck(_ context.Context, _ string, _ domain.HealthCheckUpdate) error {
	return nil
}

type integUserRepo struct{ created []*domain.PPPoEUser }

func (m *integUserRepo) Create(_ context.Context, u *domain.PPPoEUser) (*domain.PPPoEUser, error) {
	u.ID = "generated-id"
	m.created = append(m.created, u)
	return u, nil
}
func (m *integUserRepo) GetByID(_ context.Context, _ string) (*domain.PPPoEUser, error) {
	return nil, domain.ErrPPPoEUserNotFound
}
func (m *integUserRepo) GetByUsername(_ context.Context, _, _ string) (*domain.PPPoEUser, error) {
	return nil, domain.ErrPPPoEUserNotFound
}
func (m *integUserRepo) GetByCustomerID(_ context.Context, cid string) (*domain.PPPoEUser, error) {
	for _, u := range m.created {
		if u.CustomerID == cid {
			return u, nil
		}
	}
	return nil, domain.ErrPPPoEUserNotFound
}
func (m *integUserRepo) Update(_ context.Context, u *domain.PPPoEUser) (*domain.PPPoEUser, error) {
	return u, nil
}
func (m *integUserRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (m *integUserRepo) List(_ context.Context, _ domain.PPPoEUserListParams) (*domain.PPPoEUserListResult, error) {
	return nil, nil
}
func (m *integUserRepo) GetByRouterID(_ context.Context, _ string) ([]*domain.PPPoEUser, error) {
	return nil, nil
}
func (m *integUserRepo) GetSyncStatusSummary(_ context.Context, _ string) (*domain.SyncStatusSummary, error) {
	return nil, nil
}
func (m *integUserRepo) UpdateSyncStatus(_ context.Context, _ string, _ domain.SyncStatus, _ *time.Time) error {
	return nil
}
func (m *integUserRepo) BulkUpdateSyncStatus(_ context.Context, _ []string, _ domain.SyncStatus, _ *time.Time) error {
	return nil
}

type integProfileRepo struct{ profile *domain.PPPoEProfile }

func (m *integProfileRepo) GetByPackageID(_ context.Context, _ string) (*domain.PPPoEProfile, error) {
	return m.profile, nil
}
func (m *integProfileRepo) Create(_ context.Context, p *domain.PPPoEProfile) (*domain.PPPoEProfile, error) {
	return p, nil
}
func (m *integProfileRepo) GetByID(_ context.Context, _ string) (*domain.PPPoEProfile, error) {
	return m.profile, nil
}
func (m *integProfileRepo) GetByProfileName(_ context.Context, _, _ string) (*domain.PPPoEProfile, error) {
	return m.profile, nil
}
func (m *integProfileRepo) Update(_ context.Context, p *domain.PPPoEProfile) (*domain.PPPoEProfile, error) {
	return p, nil
}
func (m *integProfileRepo) ListByTenant(_ context.Context, _ string) ([]*domain.PPPoEProfile, error) {
	return nil, nil
}

// integAdapter merekam perintah yang dieksekusi.
type integAdapter struct{ commands []string }

func (m *integAdapter) Execute(_ context.Context, cmd string, _ map[string]string) ([]map[string]string, error) {
	m.commands = append(m.commands, cmd)
	if cmd == "/ppp/profile/print" {
		return []map[string]string{{".id": "*1", "name": "10mbps"}}, nil
	}
	return nil, nil
}
func (m *integAdapter) Connect(_ context.Context, _ domain.ConnectionConfig) error { return nil }
func (m *integAdapter) Close() error                                               { return nil }
func (m *integAdapter) GetSystemResource(_ context.Context) (*domain.SystemResource, error) {
	return nil, nil
}
func (m *integAdapter) Ping(_ context.Context) error { return nil }

type integConnPool struct{ adapter *integAdapter }

func (m *integConnPool) Get(_ context.Context, _ domain.CommandPriority) (domain.RouterOSAdapter, error) {
	return m.adapter, nil
}
func (m *integConnPool) Put(_ domain.RouterOSAdapter)   {}
func (m *integConnPool) Close() error                   { return nil }
func (m *integConnPool) Stats() domain.PoolStats        { return domain.PoolStats{} }
func (m *integConnPool) WarmUp(_ context.Context) error { return nil }

type integPoolManager struct{ pool *integConnPool }

func (m *integPoolManager) GetPool(_ string, _ domain.ConnectionConfig) domain.ConnPool {
	return m.pool
}
func (m *integPoolManager) ClosePool(_ string) {}
func (m *integPoolManager) CloseAll()          {}

type integEncryptor struct{}

func (m *integEncryptor) Encrypt(p string) (string, error) { return p, nil }
func (m *integEncryptor) Decrypt(c string) (string, error) { return c, nil }

type integEventPub struct{ results []domain.CommandResultPayload }

func (m *integEventPub) PublishCommandResult(_ context.Context, r domain.CommandResultPayload) error {
	m.results = append(m.results, r)
	return nil
}
func (m *integEventPub) PublishSyncFailed(_ context.Context, _ domain.SyncFailedPayload) error {
	return nil
}

// --- Helper: buat manager dengan semua mock ---

func newIntegTestManager() (PPPoEManager, *integAdapter, *integUserRepo, *integEventPub) {
	adpt := &integAdapter{}
	userRepo := &integUserRepo{}
	eventPub := &integEventPub{}
	router := &domain.Router{ID: "r1", TenantID: "t1", Host: "10.0.0.1", Port: 8728,
		Username: "admin", PasswordEncrypted: "secret", RouterOSVersion: "6.49.10"}
	profile := &domain.PPPoEProfile{ID: "p1", TenantID: "t1", PackageID: "pkg1",
		ProfileName: "10mbps", DownloadLimit: "10M", UploadLimit: "5M", LocalAddress: "gateway"}

	mgr := NewPPPoEManager(userRepo, &integProfileRepo{profile: profile},
		&integRouterRepo{router: router}, &integPoolManager{pool: &integConnPool{adapter: adpt}},
		&integEncryptor{}, eventPub, adapter.NewCommandBuilder, zerolog.Nop())
	return mgr, adpt, userRepo, eventPub
}

// --- Integration Tests ---

// TestIntegration_HandleCustomerActivated_Success — full flow: event → manager → adapter → DB → event keluar.
func TestIntegration_HandleCustomerActivated_Success(t *testing.T) {
	mgr, adpt, userRepo, eventPub := newIntegTestManager()
	payload := domain.CustomerActivatedPayload{
		CustomerID: "cust-1", TenantID: "t1", Name: "Budi", PackageID: "pkg1",
		ConnectionMethod: "pppoe", PPPoEUsername: "budi-pppoe", PPPoEPassword: "rahasia", RouterID: "r1",
	}
	if err := mgr.HandleCustomerActivated(context.Background(), payload); err != nil {
		t.Fatalf("HandleCustomerActivated gagal: %v", err)
	}
	// Verifikasi: adapter mengecek profile lebih dulu lalu membuat secret.
	if len(adpt.commands) < 2 {
		t.Fatal("adapter tidak menerima command apapun")
	}
	if adpt.commands[0] != "/ppp/profile/print" {
		t.Fatalf("expected /ppp/profile/print, got %s", adpt.commands[0])
	}
	if adpt.commands[1] != "/ppp/secret/add" {
		t.Fatalf("expected /ppp/secret/add, got %s", adpt.commands[1])
	}
	// Verifikasi: user tersimpan di DB mock
	if len(userRepo.created) != 1 {
		t.Fatalf("expected 1 user created, got %d", len(userRepo.created))
	}
	u := userRepo.created[0]
	if u.Username != "budi-pppoe" || u.ProfileName != "10mbps" || u.SyncStatus != domain.SyncStatusSynced {
		t.Fatalf("user data tidak sesuai: username=%s profile=%s sync=%s", u.Username, u.ProfileName, u.SyncStatus)
	}
	// Verifikasi: event publisher menerima command_result sukses
	if len(eventPub.results) != 1 || eventPub.results[0].Status != "success" {
		t.Fatalf("expected 1 success event, got %d events", len(eventPub.results))
	}
}

// TestIntegration_HandleCustomerActivated_SkipNonPPPoE — event non-PPPoE di-skip.
func TestIntegration_HandleCustomerActivated_SkipNonPPPoE(t *testing.T) {
	mgr, adpt, userRepo, eventPub := newIntegTestManager()
	payload := domain.CustomerActivatedPayload{
		CustomerID: "cust-2", TenantID: "t1", ConnectionMethod: "static", RouterID: "r1",
	}
	if err := mgr.HandleCustomerActivated(context.Background(), payload); err != nil {
		t.Fatalf("expected nil error for non-pppoe, got: %v", err)
	}
	if len(adpt.commands) != 0 {
		t.Fatal("adapter seharusnya tidak menerima command untuk non-pppoe")
	}
	if len(userRepo.created) != 0 {
		t.Fatal("user seharusnya tidak dibuat untuk non-pppoe")
	}
	if len(eventPub.results) != 0 {
		t.Fatal("event seharusnya tidak dipublish untuk non-pppoe")
	}
}

// TestIntegration_HandleIsolir_Success — verifikasi sequence: disable + disconnect + firewall rule.
func TestIntegration_HandleIsolir_Success(t *testing.T) {
	mgr, adpt, userRepo, eventPub := newIntegTestManager()
	// Buat user dulu agar GetByCustomerID bisa menemukan
	userRepo.created = append(userRepo.created, &domain.PPPoEUser{
		ID: "u1", CustomerID: "cust-3", TenantID: "t1", RouterID: "r1",
		Username: "andi-pppoe", RemoteAddress: "10.10.10.5", Disabled: false,
	})
	payload := domain.CustomerIsolirPayload{
		CustomerID: "cust-3", TenantID: "t1", RouterID: "r1", PPPoEUsername: "andi-pppoe",
		ConnectionMethod: "pppoe", IsolirMethod: "firewall_nat_redirect", WalledGardenIP: "192.168.1.1",
	}
	if err := mgr.HandleIsolir(context.Background(), payload); err != nil {
		t.Fatalf("HandleIsolir gagal: %v", err)
	}
	// Verifikasi sequence: minimal 3 commands (set secret, print active, add nat rule)
	if len(adpt.commands) < 3 {
		t.Fatalf("expected >= 3 commands, got %d: %v", len(adpt.commands), adpt.commands)
	}
	if adpt.commands[0] != "/ppp/secret/set" {
		t.Fatalf("step 1 expected /ppp/secret/set, got %s", adpt.commands[0])
	}
	// Command terakhir harus NAT rule
	last := adpt.commands[len(adpt.commands)-1]
	if last != "/ip/firewall/nat/add" {
		t.Fatalf("step terakhir expected /ip/firewall/nat/add, got %s", last)
	}
	// Verifikasi event publisher menerima command_result sukses
	if len(eventPub.results) != 1 || eventPub.results[0].Status != "success" {
		t.Fatalf("expected 1 success event, got %v", eventPub.results)
	}
	if eventPub.results[0].Operation != "isolir" {
		t.Fatalf("expected operation=isolir, got %s", eventPub.results[0].Operation)
	}
}
