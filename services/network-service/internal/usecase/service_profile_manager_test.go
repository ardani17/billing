// Package usecase — Unit tests untuk ServiceProfileManager: CRUD, delete guard, ResolveProfile.
package usecase

import (
	"context"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock untuk ServiceProfileManager tests
// =============================================================================

// mockServiceProfileRepoForManager extends mockServiceProfileRepo dengan kontrol tambahan.
type mockServiceProfileRepoForManager struct {
	profiles      map[string]*domain.ServiceProfile
	activeONTs    int64
	profileExists bool
	packageMap    map[string]*domain.ServiceProfile // key: oltID+packageID
}

func newMockServiceProfileRepoForManager() *mockServiceProfileRepoForManager {
	return &mockServiceProfileRepoForManager{
		profiles:   make(map[string]*domain.ServiceProfile),
		packageMap: make(map[string]*domain.ServiceProfile),
	}
}

func (r *mockServiceProfileRepoForManager) Create(_ context.Context, p *domain.ServiceProfile) (*domain.ServiceProfile, error) {
	r.profiles[p.ID] = p
	return p, nil
}

func (r *mockServiceProfileRepoForManager) GetByID(_ context.Context, id string) (*domain.ServiceProfile, error) {
	p, ok := r.profiles[id]
	if !ok {
		return nil, domain.ErrServiceProfileNotFound
	}
	return p, nil
}

func (r *mockServiceProfileRepoForManager) Update(_ context.Context, p *domain.ServiceProfile) (*domain.ServiceProfile, error) {
	r.profiles[p.ID] = p
	return p, nil
}

func (r *mockServiceProfileRepoForManager) SoftDelete(_ context.Context, id string) error {
	delete(r.profiles, id)
	return nil
}

func (r *mockServiceProfileRepoForManager) List(_ context.Context, _ string, _ domain.ServiceProfileListParams) (*domain.ServiceProfileListResult, error) {
	return &domain.ServiceProfileListResult{}, nil
}

func (r *mockServiceProfileRepoForManager) GetByPackageAndOLT(_ context.Context, oltID, packageID string) (*domain.ServiceProfile, error) {
	key := oltID + ":" + packageID
	p, ok := r.packageMap[key]
	if !ok {
		return nil, domain.ErrNoProfileMapping
	}
	return p, nil
}

func (r *mockServiceProfileRepoForManager) ProfileExists(_ context.Context, _ string, _, _ int, _ string) (bool, error) {
	return r.profileExists, nil
}

func (r *mockServiceProfileRepoForManager) CountActiveONTs(_ context.Context, _ string) (int64, error) {
	return r.activeONTs, nil
}

// newTestServiceProfileManager membuat ServiceProfileManager dengan mock dependencies.
func newTestServiceProfileManager() (*serviceProfileManager, *mockServiceProfileRepoForManager, *mockOLTRepo) {
	profileRepo := newMockServiceProfileRepoForManager()
	oltRepo := newMockOLTRepo()

	// Siapkan OLT di repo
	oltRepo.olts["olt-001"] = &domain.OLT{
		ID:       "olt-001",
		TenantID: "tenant-001",
		Name:     "OLT-Test",
		Status:   domain.OLTStatusOnline,
	}

	mgr := NewServiceProfileManager(profileRepo, oltRepo).(*serviceProfileManager)
	return mgr, profileRepo, oltRepo
}

// =============================================================================
// Test Cases — Service Profile CRUD
// =============================================================================

// TestServiceProfileCreate_HappyPath memverifikasi pembuatan service profile berhasil.
func TestServiceProfileCreate_HappyPath(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	req := domain.CreateServiceProfileRequest{
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
		PackageID:        "pkg-001",
		Description:      "Profile untuk paket 10 Mbps",
	}

	resp, err := mgr.Create(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("Create service profile gagal: %v", err)
	}

	if resp.Name != "Profile-10M" {
		t.Errorf("nama salah: got %q, want Profile-10M", resp.Name)
	}
	if resp.LineProfileID != 1 {
		t.Errorf("line_profile_id salah: got %d, want 1", resp.LineProfileID)
	}
	if resp.ServiceProfileID != 1 {
		t.Errorf("service_profile_id salah: got %d, want 1", resp.ServiceProfileID)
	}
	if resp.OLTID != "olt-001" {
		t.Errorf("OLT ID salah: got %q, want olt-001", resp.OLTID)
	}
	if len(profileRepo.profiles) != 1 {
		t.Errorf("jumlah profile di repo: got %d, want 1", len(profileRepo.profiles))
	}
}

// TestServiceProfileCreate_DuplicateProfile memverifikasi error saat kombinasi profile sudah ada.
func TestServiceProfileCreate_DuplicateProfile(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	profileRepo.profileExists = true

	req := domain.CreateServiceProfileRequest{
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
	}

	_, err := mgr.Create(ctx, "tenant-001", req)
	if err != domain.ErrServiceProfileExists {
		t.Errorf("expected ErrServiceProfileExists, got: %v", err)
	}
}

// TestServiceProfileCreate_OLTNotFound memverifikasi error saat OLT tidak ditemukan.
func TestServiceProfileCreate_OLTNotFound(t *testing.T) {
	mgr, _, _ := newTestServiceProfileManager()
	ctx := context.Background()

	req := domain.CreateServiceProfileRequest{
		OLTID:            "nonexistent",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
	}

	_, err := mgr.Create(ctx, "tenant-001", req)
	if err != domain.ErrOLTNotFound {
		t.Errorf("expected ErrOLTNotFound, got: %v", err)
	}
}

// TestServiceProfileGetByID_HappyPath memverifikasi pengambilan profile berdasarkan ID.
func TestServiceProfileGetByID_HappyPath(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	pkgID := "pkg-001"
	profileRepo.profiles["profile-001"] = &domain.ServiceProfile{
		ID:               "profile-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
		PackageID:        &pkgID,
	}

	resp, err := mgr.GetByID(ctx, "profile-001")
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}
	if resp.Name != "Profile-10M" {
		t.Errorf("nama salah: got %q, want Profile-10M", resp.Name)
	}
}

// TestServiceProfileGetByID_NotFound memverifikasi error saat profile tidak ditemukan.
func TestServiceProfileGetByID_NotFound(t *testing.T) {
	mgr, _, _ := newTestServiceProfileManager()
	ctx := context.Background()

	_, err := mgr.GetByID(ctx, "nonexistent")
	if err != domain.ErrServiceProfileNotFound {
		t.Errorf("expected ErrServiceProfileNotFound, got: %v", err)
	}
}

// TestServiceProfileUpdate_HappyPath memverifikasi update profile berhasil.
func TestServiceProfileUpdate_HappyPath(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	profileRepo.profiles["profile-001"] = &domain.ServiceProfile{
		ID:               "profile-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
	}

	newLineProfile := 2
	req := domain.UpdateServiceProfileRequest{
		Name:          "Profile-20M",
		LineProfileID: &newLineProfile,
	}

	resp, err := mgr.Update(ctx, "profile-001", req)
	if err != nil {
		t.Fatalf("Update profile gagal: %v", err)
	}
	if resp.Name != "Profile-20M" {
		t.Errorf("nama salah: got %q, want Profile-20M", resp.Name)
	}
	if resp.LineProfileID != 2 {
		t.Errorf("line_profile_id salah: got %d, want 2", resp.LineProfileID)
	}
}

// =============================================================================
// Test Cases — Delete Guard (Profile in use)
// =============================================================================

// TestServiceProfileDelete_HappyPath memverifikasi delete profile berhasil.
func TestServiceProfileDelete_HappyPath(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	profileRepo.profiles["profile-001"] = &domain.ServiceProfile{
		ID:               "profile-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
	}

	err := mgr.Delete(ctx, "profile-001")
	if err != nil {
		t.Fatalf("Delete profile gagal: %v", err)
	}

	if len(profileRepo.profiles) != 0 {
		t.Errorf("profile seharusnya sudah dihapus, masih ada %d", len(profileRepo.profiles))
	}
}

// TestServiceProfileDelete_InUse memverifikasi error saat profile masih digunakan ONT aktif.
func TestServiceProfileDelete_InUse(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	profileRepo.profiles["profile-001"] = &domain.ServiceProfile{
		ID:               "profile-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
	}
	profileRepo.activeONTs = 5 // Ada 5 ONT aktif menggunakan profile ini

	err := mgr.Delete(ctx, "profile-001")
	if err != domain.ErrServiceProfileInUse {
		t.Errorf("expected ErrServiceProfileInUse, got: %v", err)
	}

	// Profile tidak boleh terhapus
	if len(profileRepo.profiles) != 1 {
		t.Error("profile seharusnya tidak terhapus saat masih digunakan")
	}
}

// TestServiceProfileDelete_NotFound memverifikasi error saat profile tidak ditemukan.
func TestServiceProfileDelete_NotFound(t *testing.T) {
	mgr, _, _ := newTestServiceProfileManager()
	ctx := context.Background()

	err := mgr.Delete(ctx, "nonexistent")
	if err != domain.ErrServiceProfileNotFound {
		t.Errorf("expected ErrServiceProfileNotFound, got: %v", err)
	}
}

// =============================================================================
// Test Cases — ResolveProfile
// =============================================================================

// TestResolveProfile_HappyPath memverifikasi resolusi profile berdasarkan package + OLT.
func TestResolveProfile_HappyPath(t *testing.T) {
	mgr, profileRepo, _ := newTestServiceProfileManager()
	ctx := context.Background()

	pkgID := "pkg-001"
	profile := &domain.ServiceProfile{
		ID:               "profile-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
		PackageID:        &pkgID,
	}
	profileRepo.profiles["profile-001"] = profile
	profileRepo.packageMap["olt-001:pkg-001"] = profile

	result, err := mgr.ResolveProfile(ctx, "olt-001", "pkg-001")
	if err != nil {
		t.Fatalf("ResolveProfile gagal: %v", err)
	}
	if result.ID != "profile-001" {
		t.Errorf("profile ID salah: got %q, want profile-001", result.ID)
	}
	if result.LineProfileID != 1 {
		t.Errorf("line_profile_id salah: got %d, want 1", result.LineProfileID)
	}
}

// TestResolveProfile_NoMapping memverifikasi error saat tidak ada mapping.
func TestResolveProfile_NoMapping(t *testing.T) {
	mgr, _, _ := newTestServiceProfileManager()
	ctx := context.Background()

	_, err := mgr.ResolveProfile(ctx, "olt-001", "pkg-unknown")
	if err != domain.ErrNoProfileMapping {
		t.Errorf("expected ErrNoProfileMapping, got: %v", err)
	}
}
