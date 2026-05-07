package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

func TestODPManager_Create_AutoCapacity(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	tests := []struct {
		splitterType string
		wantCapacity int
	}{
		{domain.SplitterType1x4, 4},
		{domain.SplitterType1x8, 8},
		{domain.SplitterType1x16, 16},
		{domain.SplitterType1x32, 32},
	}

	for _, tt := range tests {
		repo.odps = make(map[string]*domain.ODP)
		req := createTestODPRequest()
		req.SplitterType = tt.splitterType

		resp, err := mgr.Create(ctx, "tenant-001", req)
		if err != nil {
			t.Fatalf("Create gagal untuk %s: %v", tt.splitterType, err)
		}
		if resp.Capacity != tt.wantCapacity {
			t.Errorf("splitter %s: capacity got %d, want %d", tt.splitterType, resp.Capacity, tt.wantCapacity)
		}
		if resp.UsedPorts != 0 {
			t.Errorf("used_ports harus 0 untuk ODP baru: got %d", resp.UsedPorts)
		}
	}
}

func TestODPManager_Create_InvalidSplitterType(t *testing.T) {
	mgr, _ := newTestODPManager()
	ctx := context.Background()

	req := createTestODPRequest()
	req.SplitterType = "1:99"

	_, err := mgr.Create(ctx, "tenant-001", req)
	if !errors.Is(err, domain.ErrInvalidSplitterType) {
		t.Errorf("expected ErrInvalidSplitterType, got: %v", err)
	}
}

func TestODPManager_Create_NameConflict(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.nameExists = true
	req := createTestODPRequest()

	_, err := mgr.Create(ctx, "tenant-001", req)
	if !errors.Is(err, domain.ErrODPNameExists) {
		t.Errorf("expected ErrODPNameExists, got: %v", err)
	}
}

// =============================================================================
// Tes Cases - GetByID
// =============================================================================

func TestODPManager_GetByID_FullWarning(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.odps["odp-001"] = &domain.ODP{
		ID: "odp-001", TenantID: "tenant-001", OLTID: "olt-001",
		Name: "ODP-Full", SplitterType: domain.SplitterType1x8,
		Capacity: 8, UsedPorts: 8,
	}

	resp, err := mgr.GetByID(ctx, "odp-001")
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}
	if resp.Warning == "" {
		t.Error("expected warning untuk ODP penuh, got empty")
	}
}

func TestODPManager_GetByID_NotFull(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.odps["odp-001"] = &domain.ODP{
		ID: "odp-001", TenantID: "tenant-001", OLTID: "olt-001",
		Name: "ODP-OK", SplitterType: domain.SplitterType1x8,
		Capacity: 8, UsedPorts: 5,
	}

	resp, err := mgr.GetByID(ctx, "odp-001")
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}
	if resp.Warning != "" {
		t.Errorf("tidak seharusnya ada warning: got %q", resp.Warning)
	}
}

func TestODPManager_GetByID_NotFound(t *testing.T) {
	mgr, _ := newTestODPManager()
	ctx := context.Background()

	_, err := mgr.GetByID(ctx, "nonexistent")
	if !errors.Is(err, domain.ErrODPNotFound) {
		t.Errorf("expected ErrODPNotFound, got: %v", err)
	}
}

// =============================================================================
// =============================================================================

func TestODPManager_Update_Success(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.odps["odp-001"] = &domain.ODP{
		ID: "odp-001", TenantID: "tenant-001", OLTID: "olt-001",
		Name: "ODP-Old", SplitterType: domain.SplitterType1x8,
		Capacity: 8, Address: "Jl. Lama",
	}

	lat := -6.2088
	req := domain.UpdateODPRequest{
		Name: "ODP-New", Address: "Jl. Baru No. 5",
		Latitude: &lat, Notes: "Updated notes",
	}

	resp, err := mgr.Update(ctx, "odp-001", req)
	if err != nil {
		t.Fatalf("Update gagal: %v", err)
	}
	if resp.Name != "ODP-New" {
		t.Errorf("nama tidak terupdate: got %q, want %q", resp.Name, "ODP-New")
	}
	if resp.Address != "Jl. Baru No. 5" {
		t.Errorf("address tidak terupdate: got %q", resp.Address)
	}
	if resp.Latitude == nil || *resp.Latitude != lat {
		t.Errorf("latitude tidak terupdate")
	}
}

func TestODPManager_Update_NameConflict(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.odps["odp-001"] = &domain.ODP{
		ID: "odp-001", TenantID: "tenant-001", Name: "ODP-A",
	}
	repo.nameExists = true

	_, err := mgr.Update(ctx, "odp-001", domain.UpdateODPRequest{Name: "ODP-B"})
	if !errors.Is(err, domain.ErrODPNameExists) {
		t.Errorf("expected ErrODPNameExists, got: %v", err)
	}
}

// =============================================================================
// =============================================================================

func TestODPManager_Delete(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.odps["odp-001"] = &domain.ODP{ID: "odp-001"}

	if err := mgr.Delete(ctx, "odp-001"); err != nil {
		t.Fatalf("Delete gagal: %v", err)
	}
	if _, ok := repo.odps["odp-001"]; ok {
		t.Error("ODP masih ada di repo setelah delete")
	}
}

func TestODPManager_List(t *testing.T) {
	mgr, repo := newTestODPManager()
	ctx := context.Background()

	repo.listResult = &domain.ODPListResult{
		Data: []*domain.ODPResponse{
			{ID: "odp-001", Name: "ODP-A"},
			{ID: "odp-002", Name: "ODP-B"},
		},
		Total: 2, Page: 1, PageSize: 20, TotalPages: 1,
	}

	result, err := mgr.List(ctx, domain.ODPListParams{TenantID: "tenant-001", Page: 1, PageSize: 20})
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
