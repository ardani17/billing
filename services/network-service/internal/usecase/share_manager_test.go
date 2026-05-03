// share_manager_test.go — unit test dan property test untuk ShareManager.
// Menggunakan mock in-memory repository dan pgregory.net/rapid untuk PBT.
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock Repository: ShareLinkRepository — in-memory untuk testing
// =============================================================================

// mockShareLinkRepo adalah implementasi in-memory dari domain.ShareLinkRepository.
type mockShareLinkRepo struct {
	mu    sync.Mutex
	links map[string]*domain.MapShareLink // key: Token
}

func newMockShareLinkRepo() *mockShareLinkRepo {
	return &mockShareLinkRepo{links: make(map[string]*domain.MapShareLink)}
}

func (r *mockShareLinkRepo) Create(_ context.Context, link *domain.MapShareLink) (*domain.MapShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	link.CreatedAt = time.Now()
	r.links[link.Token] = link
	return link, nil
}

func (r *mockShareLinkRepo) GetByToken(_ context.Context, token string) (*domain.MapShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	link, ok := r.links[token]
	if !ok {
		return nil, domain.ErrShareLinkNotFound
	}
	return link, nil
}

func (r *mockShareLinkRepo) Delete(_ context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.links[token]; !ok {
		return domain.ErrShareLinkNotFound
	}
	delete(r.links, token)
	return nil
}

func (r *mockShareLinkRepo) ListByTenant(_ context.Context, tenantID string) ([]*domain.MapShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var results []*domain.MapShareLink
	for _, l := range r.links {
		if l.TenantID == tenantID {
			results = append(results, l)
		}
	}
	return results, nil
}

func (r *mockShareLinkRepo) IncrementAccessCount(_ context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	link, ok := r.links[token]
	if !ok {
		return domain.ErrShareLinkNotFound
	}
	link.AccessCount++
	return nil
}

// =============================================================================
// Helper: membuat ShareManager dengan mock dependencies
// =============================================================================

// newTestShareManager membuat instance ShareManager dengan mock repository.
func newTestShareManager() (domain.ShareManager, *mockShareLinkRepo, *mockMapNodeRepo, *mockCableRouteRepo) {
	shareRepo := newMockShareLinkRepo()
	nodeRepo := newMockMapNodeRepo()
	cableRepo := newMockCableRouteRepo()
	mgr := NewShareManager(shareRepo, nodeRepo, cableRepo)
	return mgr, shareRepo, nodeRepo, cableRepo
}

// =============================================================================
// Unit Test 1: TestCreateShareLink — buat share link baru
// =============================================================================

// TestCreateShareLink memverifikasi bahwa CreateShareLink menghasilkan
// share link dengan token unik dan data yang benar.
func TestCreateShareLink(t *testing.T) {
	mgr, _, _, _ := newTestShareManager()
	ctx := context.Background()

	layers, _ := json.Marshal([]string{"olt", "odp", "ont"})
	expiryDays := 7

	req := domain.CreateShareLinkRequest{
		VisibleLayers: layers,
		ExpiryDays:    &expiryDays,
	}

	resp, err := mgr.CreateShareLink(ctx, "tenant-share", "admin-1", req)
	if err != nil {
		t.Fatalf("CreateShareLink gagal: %v", err)
	}

	// Verifikasi response
	if resp.Token == "" {
		t.Error("Token seharusnya tidak kosong")
	}
	if resp.URL == "" {
		t.Error("URL seharusnya tidak kosong")
	}
	if resp.EmbedCode == "" {
		t.Error("EmbedCode seharusnya tidak kosong")
	}
	if resp.ExpiresAt == nil {
		t.Error("ExpiresAt seharusnya terisi untuk link dengan expiry")
	}
	if resp.CreatedBy != "admin-1" {
		t.Errorf("CreatedBy: got %q, want %q", resp.CreatedBy, "admin-1")
	}
}

// =============================================================================
// Unit Test 2: TestGetSharedMap_Success — akses share link valid
// =============================================================================

// TestGetSharedMap_Success memverifikasi bahwa GetSharedMap mengembalikan
// data peta yang benar untuk share link yang valid.
func TestGetSharedMap_Success(t *testing.T) {
	mgr, _, nodeRepo, _ := newTestShareManager()
	ctx := context.Background()

	// Seed node
	seedNodeForExport(nodeRepo, "node-share-1", "tenant-share", domain.NodeTypeODP, -6.2, 106.8)

	layers, _ := json.Marshal([]string{"odp"})
	req := domain.CreateShareLinkRequest{
		VisibleLayers: layers,
	}

	created, err := mgr.CreateShareLink(ctx, "tenant-share", "admin-1", req)
	if err != nil {
		t.Fatalf("CreateShareLink gagal: %v", err)
	}

	// Akses shared map
	data, err := mgr.GetSharedMap(ctx, created.Token, "")
	if err != nil {
		t.Fatalf("GetSharedMap gagal: %v", err)
	}

	if data == nil {
		t.Fatal("SharedMapData seharusnya tidak nil")
	}
}

// =============================================================================
// Unit Test 3: TestGetSharedMap_Expired — akses link yang sudah expired
// =============================================================================

// TestGetSharedMap_Expired memverifikasi bahwa GetSharedMap mengembalikan
// ErrShareLinkExpired untuk link yang sudah kedaluwarsa.
func TestGetSharedMap_Expired(t *testing.T) {
	mgr, shareRepo, _, _ := newTestShareManager()
	ctx := context.Background()

	// Buat link yang sudah expired
	layers, _ := json.Marshal([]string{"olt"})
	expiryDays := 1
	req := domain.CreateShareLinkRequest{
		VisibleLayers: layers,
		ExpiryDays:    &expiryDays,
	}

	created, err := mgr.CreateShareLink(ctx, "tenant-share", "admin-1", req)
	if err != nil {
		t.Fatalf("CreateShareLink gagal: %v", err)
	}

	// Manipulasi expiry ke masa lalu
	shareRepo.mu.Lock()
	link := shareRepo.links[created.Token]
	pastTime := time.Now().Add(-48 * time.Hour)
	link.ExpiresAt = &pastTime
	shareRepo.mu.Unlock()

	// Akses link yang expired
	_, err = mgr.GetSharedMap(ctx, created.Token, "")
	if err != domain.ErrShareLinkExpired {
		t.Errorf("error: got %v, want %v", err, domain.ErrShareLinkExpired)
	}
}

// =============================================================================
// Unit Test 4: TestGetSharedMap_WrongPassword — password salah
// =============================================================================

// TestGetSharedMap_WrongPassword memverifikasi bahwa GetSharedMap mengembalikan
// ErrShareLinkPassword saat password yang diberikan salah.
func TestGetSharedMap_WrongPassword(t *testing.T) {
	mgr, _, _, _ := newTestShareManager()
	ctx := context.Background()

	layers, _ := json.Marshal([]string{"olt"})
	password := "rahasia123"
	req := domain.CreateShareLinkRequest{
		VisibleLayers: layers,
		Password:      &password,
	}

	created, err := mgr.CreateShareLink(ctx, "tenant-share", "admin-1", req)
	if err != nil {
		t.Fatalf("CreateShareLink gagal: %v", err)
	}

	// Akses dengan password salah
	_, err = mgr.GetSharedMap(ctx, created.Token, "salah")
	if err != domain.ErrShareLinkPassword {
		t.Errorf("error: got %v, want %v", err, domain.ErrShareLinkPassword)
	}

	// Akses dengan password benar
	_, err = mgr.GetSharedMap(ctx, created.Token, "rahasia123")
	if err != nil {
		t.Fatalf("GetSharedMap dengan password benar gagal: %v", err)
	}
}

// =============================================================================
// Unit Test 5: TestDeleteShareLink — hapus share link
// =============================================================================

// TestDeleteShareLink memverifikasi bahwa DeleteShareLink menghapus link
// dan link tidak bisa diakses lagi setelah dihapus.
func TestDeleteShareLink(t *testing.T) {
	mgr, _, _, _ := newTestShareManager()
	ctx := context.Background()

	layers, _ := json.Marshal([]string{"olt"})
	req := domain.CreateShareLinkRequest{VisibleLayers: layers}

	created, err := mgr.CreateShareLink(ctx, "tenant-share", "admin-1", req)
	if err != nil {
		t.Fatalf("CreateShareLink gagal: %v", err)
	}

	// Hapus link
	err = mgr.DeleteShareLink(ctx, created.Token)
	if err != nil {
		t.Fatalf("DeleteShareLink gagal: %v", err)
	}

	// Verifikasi link tidak bisa diakses
	_, err = mgr.GetSharedMap(ctx, created.Token, "")
	if err != domain.ErrShareLinkNotFound {
		t.Errorf("error: got %v, want %v", err, domain.ErrShareLinkNotFound)
	}
}

// =============================================================================
// Unit Test 6: TestListShareLinks — daftar share link per tenant
// =============================================================================

// TestListShareLinks memverifikasi bahwa ListShareLinks mengembalikan
// semua share link untuk tenant yang diberikan.
func TestListShareLinks(t *testing.T) {
	mgr, _, _, _ := newTestShareManager()
	ctx := context.Background()

	layers, _ := json.Marshal([]string{"olt"})

	// Buat 3 link untuk tenant yang sama
	for i := 0; i < 3; i++ {
		req := domain.CreateShareLinkRequest{VisibleLayers: layers}
		_, err := mgr.CreateShareLink(ctx, "tenant-list", fmt.Sprintf("admin-%d", i), req)
		if err != nil {
			t.Fatalf("CreateShareLink ke-%d gagal: %v", i, err)
		}
	}

	// Buat 1 link untuk tenant lain
	req := domain.CreateShareLinkRequest{VisibleLayers: layers}
	_, err := mgr.CreateShareLink(ctx, "tenant-other", "admin-x", req)
	if err != nil {
		t.Fatalf("CreateShareLink gagal: %v", err)
	}

	// List untuk tenant-list
	links, err := mgr.ListShareLinks(ctx, "tenant-list")
	if err != nil {
		t.Fatalf("ListShareLinks gagal: %v", err)
	}

	if len(links) != 3 {
		t.Errorf("jumlah links: got %d, want 3", len(links))
	}
}

// =============================================================================
// Property Test 12: Share Link Expiry Enforcement
// =============================================================================

// TestPropertyShareLinkExpiryEnforcement memverifikasi bahwa share link
// dengan expires_at di masa lalu selalu ditolak, dan link tanpa expiry
// atau dengan expiry di masa depan selalu diterima.
//
// **Validates: Requirements 9.3**
func TestPropertyShareLinkExpiryEnforcement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		shareRepo := newMockShareLinkRepo()
		nodeRepo := newMockMapNodeRepo()
		cableRepo := newMockCableRouteRepo()
		mgr := NewShareManager(shareRepo, nodeRepo, cableRepo)
		ctx := context.Background()

		// Generate parameter share link
		hasExpiry := rapid.Bool().Draw(t, "hasExpiry")
		layers, _ := json.Marshal([]string{"olt", "odp"})

		// Buat share link
		req := domain.CreateShareLinkRequest{
			VisibleLayers: layers,
		}

		if hasExpiry {
			days := rapid.IntRange(1, 365).Draw(t, "expiryDays")
			req.ExpiryDays = &days
		}

		created, err := mgr.CreateShareLink(ctx, "tenant-prop", "admin", req)
		if err != nil {
			t.Fatalf("CreateShareLink gagal: %v", err)
		}

		// Tentukan apakah link expired
		isExpired := rapid.Bool().Draw(t, "isExpired")

		if hasExpiry && isExpired {
			// Manipulasi expiry ke masa lalu
			shareRepo.mu.Lock()
			link := shareRepo.links[created.Token]
			hours := rapid.IntRange(1, 720).Draw(t, "pastHours")
			pastTime := time.Now().Add(-time.Duration(hours) * time.Hour)
			link.ExpiresAt = &pastTime
			shareRepo.mu.Unlock()
		}

		// Coba akses link
		_, err = mgr.GetSharedMap(ctx, created.Token, "")

		if hasExpiry && isExpired {
			// Properti: link expired harus ditolak
			if err != domain.ErrShareLinkExpired {
				t.Fatalf("link expired seharusnya ditolak dengan ErrShareLinkExpired, got: %v", err)
			}
		} else {
			// Properti: link valid (tidak expired) harus diterima
			if err == domain.ErrShareLinkExpired {
				t.Fatalf("link valid seharusnya tidak ditolak karena expiry")
			}
		}
	})
}

// =============================================================================
// Variabel yang tidak digunakan — suppress compiler warning
// =============================================================================

// Pastikan bcrypt diimpor (digunakan oleh share_manager.go).
var _ = bcrypt.DefaultCost
