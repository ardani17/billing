// Package usecase — Integration test end-to-end untuk provisioning lifecycle.
// Memverifikasi full flow: create VLAN → create service profile → provision ONT →
// verify status → decommission ONT → verify status. Juga test bulk provisioning
// dan port migration detection + confirmation.
package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Integration Test 1: Full Provisioning Lifecycle
// **Validates: Requirements 1.2, 3.1, 7.1, 9.1, 10.1, 14.12**
//
// Flow: create VLAN → create service profile → provision ONT →
// verify status=provisioned → decommission ONT → verify status=decommissioned
// =============================================================================

// TestIntegration_FullProvisioningLifecycle memverifikasi lifecycle lengkap
// provisioning ONT dari awal sampai decommission menggunakan mock dependencies.
func TestIntegration_FullProvisioningLifecycle(t *testing.T) {
	mgr, ontRepo, vlanRepo, profileRepo, auditRepo, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// --- Step 1: Siapkan VLAN ---
	vlanRepo.vlans["vlan-int-001"] = &domain.VLAN{
		ID:       "vlan-int-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   200,
		Name:     "VLAN-Integration",
		VLANType: domain.VLANTypeData,
	}

	// --- Step 2: Siapkan Service Profile ---
	profileRepo.profiles["profile-int-001"] = &domain.ServiceProfile{
		ID:               "profile-int-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-Integration",
		LineProfileID:    10,
		ServiceProfileID: 20,
	}

	// --- Step 3: Provision ONT ---
	req := domain.ProvisionONTRequest{
		SerialNumber:     "ZTEGINT00001",
		OLTID:            "olt-001",
		PONPortIndex:     2,
		CustomerID:       "customer-int-001",
		ServiceProfileID: "profile-int-001",
		VLANID:           "vlan-int-001",
		Description:      "ONT integration test",
	}

	resp, err := mgr.ProvisionONT(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("ProvisionONT gagal: %v", err)
	}

	// --- Step 4: Verifikasi status provisioned ---
	if resp.Status != domain.ONTStatusProvisioned {
		t.Errorf("status ONT salah: got %q, want provisioned", resp.Status)
	}
	if resp.ProvisioningState != domain.ProvisioningStateCompleted {
		t.Errorf("provisioning state salah: got %q, want completed", resp.ProvisioningState)
	}
	if resp.SerialNumber != "ZTEGINT00001" {
		t.Errorf("serial number salah: got %q", resp.SerialNumber)
	}

	// Verifikasi ONT tersimpan di repo
	ontID := resp.ID
	ont, ok := ontRepo.onts[ontID]
	if !ok {
		t.Fatal("ONT tidak ditemukan di repo setelah provisioning")
	}
	if ont.LastProvisionedAt == nil {
		t.Error("last_provisioned_at harus terisi setelah provisioning")
	}

	// Verifikasi audit log dibuat untuk provisioning
	if len(auditRepo.logs) == 0 {
		t.Error("audit log harus dibuat setelah provisioning")
	}

	// Verifikasi event ont.provisioned dipublish
	if len(eventPub.provisionedEvents) != 1 {
		t.Errorf("jumlah event provisioned: got %d, want 1", len(eventPub.provisionedEvents))
	}

	// --- Step 5: Decommission ONT ---
	err = mgr.DecommissionONT(ctx, ontID, "admin@integration-test.com")
	if err != nil {
		t.Fatalf("DecommissionONT gagal: %v", err)
	}

	// --- Step 6: Verifikasi status decommissioned ---
	ont = ontRepo.onts[ontID]
	if ont.Status != domain.ONTStatusDecommissioned {
		t.Errorf("status ONT setelah decommission: got %q, want decommissioned", ont.Status)
	}
	if ont.CustomerID != nil {
		t.Error("customer_id harus nil setelah decommission")
	}
	if ont.LastDecommissionedAt == nil {
		t.Error("last_decommissioned_at harus terisi setelah decommission")
	}

	// Verifikasi event ont.decommissioned dipublish
	if len(eventPub.decommissionedEvents) != 1 {
		t.Errorf("jumlah event decommissioned: got %d, want 1", len(eventPub.decommissionedEvents))
	}

	// Verifikasi audit log bertambah (provisioning + decommission)
	if len(auditRepo.logs) < 2 {
		t.Errorf("audit log harus minimal 2 (provision + decommission): got %d", len(auditRepo.logs))
	}
}

// =============================================================================
// Integration Test 2: Bulk Provisioning Flow
// **Validates: Requirements 5.1, 14.12**
//
// Flow: CSV upload → validate → execute → verify results
// =============================================================================

// TestIntegration_BulkProvisioningFlow memverifikasi flow bulk provisioning
// dari upload CSV sampai eksekusi dan verifikasi hasil.
func TestIntegration_BulkProvisioningFlow(t *testing.T) {
	mgr, ontRepo, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Buat CSV dengan 3 baris valid
	var sb strings.Builder
	sb.WriteString("sn_ont,pelanggan_id,pon_port,vlan,odp,deskripsi\n")
	for i := 1; i <= 3; i++ {
		sb.WriteString(fmt.Sprintf("ZTEGBULK%04d,cust-bulk-%04d,%d,vlan-001,,Bulk ONT %d\n", i, i, i%4, i))
	}
	csvData := []byte(sb.String())

	// --- Step 1: ValidateBulk ---
	preview, err := mgr.ValidateBulk(ctx, "tenant-001", "olt-001", csvData)
	if err != nil {
		t.Fatalf("ValidateBulk gagal: %v", err)
	}

	if preview.TotalRows != 3 {
		t.Errorf("TotalRows salah: got %d, want 3", preview.TotalRows)
	}
	if preview.ValidCount+preview.ErrorCount != preview.TotalRows {
		t.Errorf("ValidCount(%d) + ErrorCount(%d) != TotalRows(%d)",
			preview.ValidCount, preview.ErrorCount, preview.TotalRows)
	}

	// --- Step 2: ExecuteBulk ---
	result, err := mgr.ExecuteBulk(ctx, preview.BulkID, "admin@bulk-test.com")
	if err != nil {
		t.Fatalf("ExecuteBulk gagal: %v", err)
	}

	// Verifikasi count invariant
	if result.SuccessCount+result.FailureCount != result.Total {
		t.Errorf("SuccessCount(%d) + FailureCount(%d) != Total(%d)",
			result.SuccessCount, result.FailureCount, result.Total)
	}

	// Verifikasi ONT dibuat di repo
	if result.SuccessCount > 0 && len(ontRepo.onts) == 0 {
		t.Error("ONT harus dibuat di repo setelah bulk provisioning berhasil")
	}

	// Verifikasi setiap row result memiliki serial number
	for _, row := range result.Rows {
		if row.SerialNumber == "" {
			t.Errorf("row %d: serial number kosong di result", row.RowNumber)
		}
	}
}

// =============================================================================
// Integration Test 3: Port Migration Detection + Confirmation
// **Validates: Requirements 11.5**
//
// Flow: detect port migration → publish event → confirm migration
// =============================================================================

// TestIntegration_PortMigrationDetectionAndConfirmation memverifikasi flow
// deteksi port migration dan konfirmasi oleh admin.
func TestIntegration_PortMigrationDetectionAndConfirmation(t *testing.T) {
	mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	// Siapkan ONT yang sudah provisioned di port 0, ONT index 1
	ontRepo.onts["ont-migrate-001"] = &domain.ONT{
		ID:                "ont-migrate-001",
		TenantID:          "tenant-001",
		OLTID:             "olt-001",
		PONPortIndex:      0,
		ONTIndex:          1,
		SerialNumber:      "ZTEGMIGR0001",
		Status:            domain.ONTStatusProvisioned,
		ProvisioningState: domain.ProvisioningStateCompleted,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// --- Step 1: Deteksi port migration (pindah dari port 0 ke port 3) ---
	err := mgr.HandlePortMigration(ctx, "ont-migrate-001", 0, 3, 1, 7)
	if err != nil {
		t.Fatalf("HandlePortMigration gagal: %v", err)
	}

	// Verifikasi event ont.port_migrated dipublish
	if len(eventPub.portMigratedEvents) != 1 {
		t.Fatalf("event port_migrated harus dipublish: got %d", len(eventPub.portMigratedEvents))
	}

	// Verifikasi payload event berisi informasi port lama dan baru
	evt := eventPub.portMigratedEvents[0]
	if evt.OldPortIndex != 0 {
		t.Errorf("old_port_index salah: got %d, want 0", evt.OldPortIndex)
	}
	if evt.NewPortIndex != 3 {
		t.Errorf("new_port_index salah: got %d, want 3", evt.NewPortIndex)
	}
	if evt.OldONTIndex != 1 {
		t.Errorf("old_ont_index salah: got %d, want 1", evt.OldONTIndex)
	}
	if evt.NewONTIndex != 7 {
		t.Errorf("new_ont_index salah: got %d, want 7", evt.NewONTIndex)
	}
	if evt.SerialNumber != "ZTEGMIGR0001" {
		t.Errorf("serial_number salah: got %q", evt.SerialNumber)
	}

	// --- Step 2: Konfirmasi migration oleh admin ---
	err = mgr.ConfirmMigration(ctx, "ont-migrate-001")
	if err != nil {
		t.Fatalf("ConfirmMigration gagal: %v", err)
	}
}
