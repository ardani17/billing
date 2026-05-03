// Package usecase — Unit tests untuk VLANManager: CRUD, delete guard, ResolveVLAN per strategy.
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Helper untuk membuat VLAN Manager dengan mock dependencies
// =============================================================================

// mockVLANRepoForManager extends mockVLANRepo dengan kontrol tambahan untuk testing.
type mockVLANRepoForManager struct {
	vlans        map[string]*domain.VLAN
	activeONTs   int64
	vlanIDExists bool
	listResult   *domain.VLANListResult
	defaultVLAN  *domain.VLAN
}

func newMockVLANRepoForManager() *mockVLANRepoForManager {
	return &mockVLANRepoForManager{vlans: make(map[string]*domain.VLAN)}
}

func (r *mockVLANRepoForManager) Create(_ context.Context, v *domain.VLAN) (*domain.VLAN, error) {
	r.vlans[v.ID] = v
	return v, nil
}

func (r *mockVLANRepoForManager) GetByID(_ context.Context, id string) (*domain.VLAN, error) {
	v, ok := r.vlans[id]
	if !ok {
		return nil, domain.ErrVLANNotFound
	}
	return v, nil
}

func (r *mockVLANRepoForManager) Update(_ context.Context, v *domain.VLAN) (*domain.VLAN, error) {
	r.vlans[v.ID] = v
	return v, nil
}

func (r *mockVLANRepoForManager) SoftDelete(_ context.Context, id string) error {
	delete(r.vlans, id)
	return nil
}

func (r *mockVLANRepoForManager) List(_ context.Context, _ string, _ domain.VLANListParams) (*domain.VLANListResult, error) {
	if r.listResult != nil {
		return r.listResult, nil
	}
	// Bangun list dari vlans map
	var data []*domain.VLANResponse
	for _, v := range r.vlans {
		data = append(data, &domain.VLANResponse{
			ID:          v.ID,
			OLTID:       v.OLTID,
			VLANID:      v.VLANID,
			Name:        v.Name,
			VLANType:    string(v.VLANType),
			Description: v.Description,
			CreatedAt:   v.CreatedAt,
			UpdatedAt:   v.UpdatedAt,
		})
	}
	return &domain.VLANListResult{
		Data:     data,
		Total:    int64(len(data)),
		Page:     1,
		PageSize: 20,
	}, nil
}

func (r *mockVLANRepoForManager) GetByOLTAndVLANID(_ context.Context, _ string, _ int) (*domain.VLAN, error) {
	return nil, domain.ErrVLANNotFound
}

func (r *mockVLANRepoForManager) GetDefaultVLAN(_ context.Context, oltID string) (*domain.VLAN, error) {
	if r.defaultVLAN != nil {
		return r.defaultVLAN, nil
	}
	for _, v := range r.vlans {
		if v.OLTID == oltID && v.VLANType == domain.VLANTypeData {
			return v, nil
		}
	}
	return nil, domain.ErrVLANNotFound
}

func (r *mockVLANRepoForManager) VLANIDExists(_ context.Context, _ string, _ int, _ string) (bool, error) {
	return r.vlanIDExists, nil
}

func (r *mockVLANRepoForManager) CountActiveONTs(_ context.Context, _ string) (int64, error) {
	return r.activeONTs, nil
}

// newTestVLANManager membuat VLANManager dengan mock dependencies untuk testing.
func newTestVLANManager() (*vlanManager, *mockVLANRepoForManager, *mockOLTRepo) {
	vlanRepo := newMockVLANRepoForManager()
	oltRepo := newMockOLTRepo()

	// Siapkan OLT di repo
	oltRepo.olts["olt-001"] = &domain.OLT{
		ID:       "olt-001",
		TenantID: "tenant-001",
		Name:     "OLT-Test",
		Status:   domain.OLTStatusOnline,
	}

	mgr := NewVLANManager(vlanRepo, oltRepo).(*vlanManager)
	return mgr, vlanRepo, oltRepo
}

// =============================================================================
// Test Cases — VLAN CRUD
// =============================================================================

// TestVLANCreate_HappyPath memverifikasi pembuatan VLAN berhasil.
func TestVLANCreate_HappyPath(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	req := domain.CreateVLANRequest{
		OLTID:       "olt-001",
		VLANID:      100,
		Name:        "VLAN-Data",
		VLANType:    "data",
		Description: "VLAN untuk data pelanggan",
	}

	resp, err := mgr.Create(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("Create VLAN gagal: %v", err)
	}

	if resp.VLANID != 100 {
		t.Errorf("VLAN ID salah: got %d, want 100", resp.VLANID)
	}
	if resp.Name != "VLAN-Data" {
		t.Errorf("nama salah: got %q, want VLAN-Data", resp.Name)
	}
	if resp.OLTID != "olt-001" {
		t.Errorf("OLT ID salah: got %q, want olt-001", resp.OLTID)
	}
	if len(vlanRepo.vlans) != 1 {
		t.Errorf("jumlah VLAN di repo: got %d, want 1", len(vlanRepo.vlans))
	}
}

// TestVLANCreate_DuplicateVLANID memverifikasi error saat VLAN ID sudah ada.
func TestVLANCreate_DuplicateVLANID(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	vlanRepo.vlanIDExists = true

	req := domain.CreateVLANRequest{
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: "data",
	}

	_, err := mgr.Create(ctx, "tenant-001", req)
	if err != domain.ErrVLANIDExists {
		t.Errorf("expected ErrVLANIDExists, got: %v", err)
	}
}

// TestVLANCreate_OLTNotFound memverifikasi error saat OLT tidak ditemukan.
func TestVLANCreate_OLTNotFound(t *testing.T) {
	mgr, _, _ := newTestVLANManager()
	ctx := context.Background()

	req := domain.CreateVLANRequest{
		OLTID:    "nonexistent",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: "data",
	}

	_, err := mgr.Create(ctx, "tenant-001", req)
	if err != domain.ErrOLTNotFound {
		t.Errorf("expected ErrOLTNotFound, got: %v", err)
	}
}

// TestVLANGetByID_HappyPath memverifikasi pengambilan VLAN berdasarkan ID.
func TestVLANGetByID_HappyPath(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	vlanRepo.vlans["vlan-001"] = &domain.VLAN{
		ID:       "vlan-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: domain.VLANTypeData,
	}

	resp, err := mgr.GetByID(ctx, "vlan-001")
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}
	if resp.VLANID != 100 {
		t.Errorf("VLAN ID salah: got %d, want 100", resp.VLANID)
	}
}

// TestVLANGetByID_NotFound memverifikasi error saat VLAN tidak ditemukan.
func TestVLANGetByID_NotFound(t *testing.T) {
	mgr, _, _ := newTestVLANManager()
	ctx := context.Background()

	_, err := mgr.GetByID(ctx, "nonexistent")
	if err != domain.ErrVLANNotFound {
		t.Errorf("expected ErrVLANNotFound, got: %v", err)
	}
}

// TestVLANUpdate_HappyPath memverifikasi update VLAN berhasil.
func TestVLANUpdate_HappyPath(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	vlanRepo.vlans["vlan-001"] = &domain.VLAN{
		ID:       "vlan-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: domain.VLANTypeData,
	}

	req := domain.UpdateVLANRequest{
		Name:     "VLAN-Data-Updated",
		VLANType: "voice",
	}

	resp, err := mgr.Update(ctx, "vlan-001", req)
	if err != nil {
		t.Fatalf("Update VLAN gagal: %v", err)
	}
	if resp.Name != "VLAN-Data-Updated" {
		t.Errorf("nama salah: got %q, want VLAN-Data-Updated", resp.Name)
	}
	if resp.VLANType != "voice" {
		t.Errorf("tipe salah: got %q, want voice", resp.VLANType)
	}
}

// =============================================================================
// Test Cases — Delete Guard (VLAN in use)
// =============================================================================

// TestVLANDelete_HappyPath memverifikasi delete VLAN berhasil saat tidak ada ONT aktif.
func TestVLANDelete_HappyPath(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	vlanRepo.vlans["vlan-001"] = &domain.VLAN{
		ID:       "vlan-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: domain.VLANTypeData,
	}

	err := mgr.Delete(ctx, "vlan-001")
	if err != nil {
		t.Fatalf("Delete VLAN gagal: %v", err)
	}

	if len(vlanRepo.vlans) != 0 {
		t.Errorf("VLAN seharusnya sudah dihapus, masih ada %d", len(vlanRepo.vlans))
	}
}

// TestVLANDelete_InUse memverifikasi error saat VLAN masih digunakan ONT aktif.
func TestVLANDelete_InUse(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	vlanRepo.vlans["vlan-001"] = &domain.VLAN{
		ID:       "vlan-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: domain.VLANTypeData,
	}
	vlanRepo.activeONTs = 3 // Ada 3 ONT aktif menggunakan VLAN ini

	err := mgr.Delete(ctx, "vlan-001")
	if err != domain.ErrVLANInUse {
		t.Errorf("expected ErrVLANInUse, got: %v", err)
	}

	// VLAN tidak boleh terhapus
	if len(vlanRepo.vlans) != 1 {
		t.Error("VLAN seharusnya tidak terhapus saat masih digunakan")
	}
}

// TestVLANDelete_NotFound memverifikasi error saat VLAN tidak ditemukan.
func TestVLANDelete_NotFound(t *testing.T) {
	mgr, _, _ := newTestVLANManager()
	ctx := context.Background()

	err := mgr.Delete(ctx, "nonexistent")
	if err != domain.ErrVLANNotFound {
		t.Errorf("expected ErrVLANNotFound, got: %v", err)
	}
}

// =============================================================================
// Test Cases — ResolveVLAN per Strategy
// =============================================================================

// TestResolveVLAN_Single memverifikasi strategy "single" mengembalikan default VLAN.
func TestResolveVLAN_Single(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	// Siapkan default VLAN (tipe data)
	vlanRepo.vlans["vlan-default"] = &domain.VLAN{
		ID:       "vlan-default",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "Default-VLAN",
		VLANType: domain.VLANTypeData,
	}

	vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategySingle, domain.VLANResolveContext{})
	if err != nil {
		t.Fatalf("ResolveVLAN single gagal: %v", err)
	}
	if vlan.ID != "vlan-default" {
		t.Errorf("VLAN ID salah: got %q, want vlan-default", vlan.ID)
	}
}

// TestResolveVLAN_PerPaket memverifikasi strategy "per_paket" mengembalikan VLAN by package.
func TestResolveVLAN_PerPaket(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	now := time.Now()
	// VLAN untuk paket tertentu (description = package_id)
	vlanRepo.vlans["vlan-paket"] = &domain.VLAN{
		ID:          "vlan-paket",
		TenantID:    "tenant-001",
		OLTID:       "olt-001",
		VLANID:      200,
		Name:        "VLAN-Paket-10M",
		VLANType:    domain.VLANTypeData,
		Description: "pkg-001",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resolveCtx := domain.VLANResolveContext{PackageID: "pkg-001"}
	vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPaket, resolveCtx)
	if err != nil {
		t.Fatalf("ResolveVLAN per_paket gagal: %v", err)
	}
	if vlan.ID != "vlan-paket" {
		t.Errorf("VLAN ID salah: got %q, want vlan-paket", vlan.ID)
	}
}

// TestResolveVLAN_PerODP memverifikasi strategy "per_odp" mengembalikan VLAN by ODP.
func TestResolveVLAN_PerODP(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	now := time.Now()
	// VLAN untuk ODP tertentu (description = odp_id)
	vlanRepo.vlans["vlan-odp"] = &domain.VLAN{
		ID:          "vlan-odp",
		TenantID:    "tenant-001",
		OLTID:       "olt-001",
		VLANID:      300,
		Name:        "VLAN-ODP-A1",
		VLANType:    domain.VLANTypeData,
		Description: "odp-001",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resolveCtx := domain.VLANResolveContext{ODPID: "odp-001"}
	vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerODP, resolveCtx)
	if err != nil {
		t.Fatalf("ResolveVLAN per_odp gagal: %v", err)
	}
	if vlan.ID != "vlan-odp" {
		t.Errorf("VLAN ID salah: got %q, want vlan-odp", vlan.ID)
	}
}

// TestResolveVLAN_PerPelanggan memverifikasi strategy "per_pelanggan" mengembalikan VLAN unik.
func TestResolveVLAN_PerPelanggan(t *testing.T) {
	mgr, vlanRepo, _ := newTestVLANManager()
	ctx := context.Background()

	now := time.Now()
	// VLAN unik per pelanggan (description = customer_id)
	vlanRepo.vlans["vlan-cust"] = &domain.VLAN{
		ID:          "vlan-cust",
		TenantID:    "tenant-001",
		OLTID:       "olt-001",
		VLANID:      400,
		Name:        "VLAN-Cust-001",
		VLANType:    domain.VLANTypeData,
		Description: "customer-001",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resolveCtx := domain.VLANResolveContext{CustomerID: "customer-001"}
	vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPelanggan, resolveCtx)
	if err != nil {
		t.Fatalf("ResolveVLAN per_pelanggan gagal: %v", err)
	}
	if vlan.ID != "vlan-cust" {
		t.Errorf("VLAN ID salah: got %q, want vlan-cust", vlan.ID)
	}
}

// TestResolveVLAN_PerPelanggan_NotFound memverifikasi error saat VLAN pelanggan tidak ada.
func TestResolveVLAN_PerPelanggan_NotFound(t *testing.T) {
	mgr, _, _ := newTestVLANManager()
	ctx := context.Background()

	resolveCtx := domain.VLANResolveContext{CustomerID: "customer-unknown"}
	_, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPelanggan, resolveCtx)
	if err != domain.ErrVLANResolutionFailed {
		t.Errorf("expected ErrVLANResolutionFailed, got: %v", err)
	}
}

// TestResolveVLAN_InvalidStrategy memverifikasi error saat strategy tidak valid.
func TestResolveVLAN_InvalidStrategy(t *testing.T) {
	mgr, _, _ := newTestVLANManager()
	ctx := context.Background()

	_, err := mgr.ResolveVLAN(ctx, "olt-001", "invalid", domain.VLANResolveContext{})
	if err != domain.ErrInvalidVLANStrategy {
		t.Errorf("expected ErrInvalidVLANStrategy, got: %v", err)
	}
}

// TestResolveVLAN_PerPaket_EmptyPackageID memverifikasi error saat package_id kosong.
func TestResolveVLAN_PerPaket_EmptyPackageID(t *testing.T) {
	mgr, _, _ := newTestVLANManager()
	ctx := context.Background()

	resolveCtx := domain.VLANResolveContext{PackageID: ""}
	_, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPaket, resolveCtx)
	if err != domain.ErrVLANResolutionFailed {
		t.Errorf("expected ErrVLANResolutionFailed, got: %v", err)
	}
}
