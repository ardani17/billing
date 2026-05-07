package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Tes Cases - DecommissionONT
// =============================================================================

// TestDecommissionONT_HappyPath memverifikasi decommission ONT berhasil.
func TestDecommissionONT_HappyPath(t *testing.T) {
	mgr, ontRepo, _, _, auditRepo, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Siapkan ONT yang sudah provisioned
	customerID := "customer-001"
	vlanID := "vlan-001"
	ontRepo.onts["ont-001"] = &domain.ONT{
		ID:                "ont-001",
		TenantID:          "tenant-001",
		OLTID:             "olt-001",
		PONPortIndex:      0,
		ONTIndex:          1,
		SerialNumber:      "ZTEG12345678",
		CustomerID:        &customerID,
		VLANID:            &vlanID,
		Status:            domain.ONTStatusProvisioned,
		ProvisioningState: domain.ProvisioningStateCompleted,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := mgr.DecommissionONT(ctx, "ont-001", "admin@test.com")
	if err != nil {
		t.Fatalf("DecommissionONT gagal: %v", err)
	}

	// Verifikasi status berubah ke decommissioned
	ont := ontRepo.onts["ont-001"]
	if ont.Status != domain.ONTStatusDecommissioned {
		t.Errorf("status salah: got %q, want decommissioned", ont.Status)
	}
	if ont.CustomerID != nil {
		t.Error("customer_id harus nil setelah decommission")
	}
	if ont.LastDecommissionedAt == nil {
		t.Error("last_decommissioned_at harus terisi")
	}

	// Verifikasi audit log
	if len(auditRepo.logs) == 0 {
		t.Error("audit log tidak dibuat")
	}

	// Verifikasi event
	if len(eventPub.decommissionedEvents) != 1 {
		t.Errorf("jumlah event decommissioned: got %d, want 1", len(eventPub.decommissionedEvents))
	}
}

func TestDecommissionONT_NotFound(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	err := mgr.DecommissionONT(ctx, "nonexistent", "admin@test.com")
	if !errors.Is(err, domain.ErrONTNotFound) {
		t.Errorf("expected ErrONTNotFound, got: %v", err)
	}
}

// TestDecommissionONT_CLIFailure memverifikasi handling saat CLI command gagal.
func TestDecommissionONT_CLIFailure(t *testing.T) {
	mgr, ontRepo, _, _, auditRepo, _, adapter := newTestProvisioningManager()
	ctx := context.Background()

	vlanID := "vlan-001"
	ontRepo.onts["ont-001"] = &domain.ONT{
		ID:                "ont-001",
		TenantID:          "tenant-001",
		OLTID:             "olt-001",
		PONPortIndex:      0,
		ONTIndex:          1,
		SerialNumber:      "ZTEG12345678",
		VLANID:            &vlanID,
		Status:            domain.ONTStatusProvisioned,
		ProvisioningState: domain.ProvisioningStateCompleted,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Simulasikan RemoveServicePort gagal
	adapter.removeSPResult = &domain.ProvisioningResult{
		Success:      false,
		CommandsSent: []string{"service-port delete"},
		Responses:    []string{"Error: timeout"},
		ErrorMessage: "CLI timeout",
	}

	err := mgr.DecommissionONT(ctx, "ont-001", "admin@test.com")
	if !errors.Is(err, domain.ErrDecommissionFailed) {
		t.Errorf("expected ErrDecommissionFailed, got: %v", err)
	}

	// Verifikasi audit log mencatat kegagalan
	if len(auditRepo.logs) == 0 {
		t.Error("audit log harus dibuat meski decommission gagal")
	}
}

// TestHandleCustomerTerminated_WithONT memverifikasi event-driven decommission.
func TestHandleCustomerTerminated_WithONT(t *testing.T) {
	mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	customerID := "customer-001"
	vlanID := "vlan-001"
	ontRepo.onts["ont-001"] = &domain.ONT{
		ID:                "ont-001",
		TenantID:          "tenant-001",
		OLTID:             "olt-001",
		PONPortIndex:      0,
		ONTIndex:          1,
		SerialNumber:      "ZTEG12345678",
		CustomerID:        &customerID,
		VLANID:            &vlanID,
		Status:            domain.ONTStatusProvisioned,
		ProvisioningState: domain.ProvisioningStateCompleted,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	ontRepo.customerONT = ontRepo.onts["ont-001"]

	err := mgr.HandleCustomerTerminated(ctx, "customer-001", "tenant-001")
	if err != nil {
		t.Fatalf("HandleCustomerTerminated gagal: %v", err)
	}

	// Verifikasi ONT di-decommission
	if len(eventPub.decommissionedEvents) != 1 {
		t.Errorf("event decommissioned harus dipublish: got %d", len(eventPub.decommissionedEvents))
	}
}

// TestHandleCustomerTerminated_NoONT memverifikasi skip saat customer tidak punya ONT.
func TestHandleCustomerTerminated_NoONT(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	err := mgr.HandleCustomerTerminated(ctx, "customer-no-ont", "tenant-001")
	if err != nil {
		t.Fatalf("HandleCustomerTerminated harus berhasil saat tidak ada ONT: %v", err)
	}
}
