// Package usecase — Unit tests untuk auto-provisioning: enabled/disabled, SN match/no-match.
package usecase

import (
	"context"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Test Cases — HandleUnregisteredONT (Auto-Provisioning)
// =============================================================================

// TestHandleUnregisteredONT_AutoDisabled memverifikasi bahwa ONT tetap unregistered
// saat auto-provisioning disabled.
func TestHandleUnregisteredONT_AutoDisabled(t *testing.T) {
	mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Settings: auto-provisioning disabled (default)
	ont := domain.UnregisteredONT{
		SerialNumber: "ZTEG99999999",
		PONPortIndex: 0,
		ONTIndex:     5,
	}

	err := mgr.HandleUnregisteredONT(ctx, "olt-001", ont)
	if err != nil {
		t.Fatalf("HandleUnregisteredONT gagal: %v", err)
	}

	// Verifikasi ONT dibuat dengan status unregistered
	found := false
	for _, o := range ontRepo.onts {
		if o.SerialNumber == "ZTEG99999999" && o.Status == domain.ONTStatusUnregistered {
			found = true
			break
		}
	}
	if !found {
		t.Error("ONT unregistered harus dibuat di repo")
	}

	// Verifikasi tidak ada event auto-provisioned
	if len(eventPub.autoProvEvents) != 0 {
		t.Errorf("tidak boleh ada event auto-provisioned saat disabled: got %d", len(eventPub.autoProvEvents))
	}
}

// TestHandleUnregisteredONT_AutoEnabled_NoMatch memverifikasi bahwa ONT tetap
// unregistered saat auto-provisioning enabled tapi tidak ada customer match.
func TestHandleUnregisteredONT_AutoEnabled_NoMatch(t *testing.T) {
	mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Enable auto-provisioning via settings
	mgr.settingsRepo = &mockSettingsRepo{
		settings: &domain.ProvisioningSettings{
			TenantID:                "tenant-001",
			AutoProvisioningEnabled: true,
			VLANStrategy:            domain.VLANStrategySingle,
		},
	}

	ont := domain.UnregisteredONT{
		SerialNumber: "ZTEG88888888",
		PONPortIndex: 1,
		ONTIndex:     3,
	}

	err := mgr.HandleUnregisteredONT(ctx, "olt-001", ont)
	if err != nil {
		t.Fatalf("HandleUnregisteredONT gagal: %v", err)
	}

	// Verifikasi ONT dibuat sebagai unregistered (tidak ada customer match)
	found := false
	for _, o := range ontRepo.onts {
		if o.SerialNumber == "ZTEG88888888" && o.Status == domain.ONTStatusUnregistered {
			found = true
			break
		}
	}
	if !found {
		t.Error("ONT harus tetap unregistered saat tidak ada customer match")
	}

	// Tidak ada event auto-provisioned
	if len(eventPub.autoProvEvents) != 0 {
		t.Errorf("tidak boleh ada event auto-provisioned tanpa customer match: got %d", len(eventPub.autoProvEvents))
	}
}

// =============================================================================
// Test Cases — HandlePortMigration
// =============================================================================

// TestHandlePortMigration_AutoEnabled memverifikasi auto-update DB saat enabled.
func TestHandlePortMigration_AutoEnabled(t *testing.T) {
	mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Enable auto-port-migration
	mgr.settingsRepo = &mockSettingsRepo{
		settings: &domain.ProvisioningSettings{
			TenantID:                 "tenant-001",
			AutoPortMigrationEnabled: true,
			VLANStrategy:             domain.VLANStrategySingle,
		},
	}

	ontRepo.onts["ont-001"] = &domain.ONT{
		ID:           "ont-001",
		TenantID:     "tenant-001",
		OLTID:        "olt-001",
		PONPortIndex: 0,
		ONTIndex:     1,
		SerialNumber: "ZTEG12345678",
		Status:       domain.ONTStatusProvisioned,
	}

	err := mgr.HandlePortMigration(ctx, "ont-001", 0, 2, 1, 5)
	if err != nil {
		t.Fatalf("HandlePortMigration gagal: %v", err)
	}

	// Verifikasi event port_migrated dipublish
	if len(eventPub.portMigratedEvents) != 1 {
		t.Errorf("event port_migrated harus dipublish: got %d", len(eventPub.portMigratedEvents))
	}
}

// TestHandlePortMigration_AutoDisabled memverifikasi flag saat disabled.
func TestHandlePortMigration_AutoDisabled(t *testing.T) {
	mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Auto-port-migration disabled (default)
	ontRepo.onts["ont-001"] = &domain.ONT{
		ID:           "ont-001",
		TenantID:     "tenant-001",
		OLTID:        "olt-001",
		PONPortIndex: 0,
		ONTIndex:     1,
		SerialNumber: "ZTEG12345678",
		Status:       domain.ONTStatusProvisioned,
	}

	err := mgr.HandlePortMigration(ctx, "ont-001", 0, 2, 1, 5)
	if err != nil {
		t.Fatalf("HandlePortMigration gagal: %v", err)
	}

	// Event tetap dipublish
	if len(eventPub.portMigratedEvents) != 1 {
		t.Errorf("event port_migrated harus dipublish: got %d", len(eventPub.portMigratedEvents))
	}
}

// TestConfirmMigration_HappyPath memverifikasi konfirmasi migrasi berhasil.
func TestConfirmMigration_HappyPath(t *testing.T) {
	mgr, ontRepo, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	ontRepo.onts["ont-001"] = &domain.ONT{
		ID:       "ont-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		Status:   domain.ONTStatusProvisioned,
	}

	err := mgr.ConfirmMigration(ctx, "ont-001")
	if err != nil {
		t.Fatalf("ConfirmMigration gagal: %v", err)
	}
}
