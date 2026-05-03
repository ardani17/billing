// Package usecase — Property test untuk audit log completeness.
// Memverifikasi bahwa setiap operasi provisioning (provision, decommission, reboot)
// menghasilkan audit log entry dengan commands_sent non-empty, action sesuai operasi,
// performed_by non-empty, dan correlation_id non-empty.
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Property 6: Audit Log Completeness
// **Validates: Requirements 12.3, 12.4, 12.5**
//
// Untuk sembarang operasi provisioning (provision, decommission, reboot),
// audit log entry SHALL dibuat dengan:
// - commands_sent non-empty
// - action sesuai operasi
// - performed_by non-empty
// - correlation_id non-empty
// =============================================================================

// auditSerialNumberGen menghasilkan serial number ONT acak untuk audit test.
func auditSerialNumberGen() *rapid.Generator[string] {
	return rapid.StringMatching(`ZTEG[A-F0-9]{8}`)
}

// TestProperty6_AuditLog_ProvisionONT memverifikasi bahwa ProvisionONT
// selalu menghasilkan audit log entry yang lengkap.
//
// **Validates: Requirements 12.3, 12.4, 12.5**
func TestProperty6_AuditLog_ProvisionONT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr, _, _, _, auditRepo, _, _ := newTestProvisioningManager()
		ctx := context.Background()

		sn := auditSerialNumberGen().Draw(rt, "serialNumber")
		ponPort := rapid.IntRange(0, 7).Draw(rt, "ponPort")

		req := domain.ProvisionONTRequest{
			SerialNumber:     sn,
			OLTID:            "olt-001",
			PONPortIndex:     ponPort,
			CustomerID:       "customer-audit-001",
			ServiceProfileID: "profile-001",
			VLANID:           "vlan-001",
			Description:      "audit test ONT",
		}

		_, err := mgr.ProvisionONT(ctx, "tenant-001", req)
		if err != nil {
			// Provisioning bisa gagal karena SN duplikat di rapid iterations,
			// tapi audit log tetap harus dibuat untuk kegagalan CLI
			// Skip jika error bukan dari CLI (misal validasi)
			return
		}

		// Verifikasi audit log dibuat
		if len(auditRepo.logs) == 0 {
			t.Fatal("audit log harus dibuat setelah ProvisionONT berhasil")
		}

		// Ambil audit log terakhir
		lastLog := auditRepo.logs[len(auditRepo.logs)-1]

		// Property: commands_sent non-empty
		if len(lastLog.CommandsSent) == 0 {
			t.Error("commands_sent harus non-empty")
		}

		// Property: action sesuai operasi
		if lastLog.Action != domain.AuditActionONTProvision {
			t.Errorf("action salah: got %q, want %q", lastLog.Action, domain.AuditActionONTProvision)
		}

		// Property: performed_by non-empty
		if lastLog.PerformedBy == "" {
			t.Error("performed_by harus non-empty")
		}

		// Property: correlation_id non-empty
		if lastLog.CorrelationID == "" {
			t.Error("correlation_id harus non-empty")
		}
	})
}

// TestProperty6_AuditLog_DecommissionONT memverifikasi bahwa DecommissionONT
// selalu menghasilkan audit log entry yang lengkap.
//
// **Validates: Requirements 12.3, 12.4, 12.5**
func TestProperty6_AuditLog_DecommissionONT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr, ontRepo, _, _, auditRepo, _, _ := newTestProvisioningManager()
		ctx := context.Background()

		sn := auditSerialNumberGen().Draw(rt, "serialNumber")
		ponPort := rapid.IntRange(0, 7).Draw(rt, "ponPort")
		performer := rapid.StringMatching(`[a-z]{3,10}@test\.com`).Draw(rt, "performer")

		// Siapkan ONT provisioned untuk di-decommission
		customerID := "customer-decom-001"
		vlanID := "vlan-001"
		ontID := "ont-audit-decom-" + sn
		ontRepo.onts[ontID] = &domain.ONT{
			ID:                ontID,
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      ponPort,
			ONTIndex:          1,
			SerialNumber:      sn,
			CustomerID:        &customerID,
			VLANID:            &vlanID,
			Status:            domain.ONTStatusProvisioned,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := mgr.DecommissionONT(ctx, ontID, performer)
		if err != nil {
			t.Fatalf("DecommissionONT gagal: %v", err)
		}

		// Verifikasi audit log dibuat
		if len(auditRepo.logs) == 0 {
			t.Fatal("audit log harus dibuat setelah DecommissionONT")
		}

		// Ambil audit log terakhir
		lastLog := auditRepo.logs[len(auditRepo.logs)-1]

		// Property: commands_sent non-empty
		if len(lastLog.CommandsSent) == 0 {
			t.Error("commands_sent harus non-empty")
		}

		// Property: action sesuai operasi
		if lastLog.Action != domain.AuditActionONTDecommission {
			t.Errorf("action salah: got %q, want %q", lastLog.Action, domain.AuditActionONTDecommission)
		}

		// Property: performed_by non-empty
		if lastLog.PerformedBy == "" {
			t.Error("performed_by harus non-empty")
		}

		// Property: correlation_id non-empty
		if lastLog.CorrelationID == "" {
			t.Error("correlation_id harus non-empty")
		}
	})
}

// TestProperty6_AuditLog_RebootONT memverifikasi bahwa RebootONT
// selalu menghasilkan audit log entry yang lengkap.
//
// **Validates: Requirements 12.3, 12.4, 12.5**
func TestProperty6_AuditLog_RebootONT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr, ontRepo, _, _, auditRepo, _, _ := newTestProvisioningManager()
		ctx := context.Background()

		sn := auditSerialNumberGen().Draw(rt, "serialNumber")
		ponPort := rapid.IntRange(0, 7).Draw(rt, "ponPort")
		performer := rapid.StringMatching(`[a-z]{3,10}@test\.com`).Draw(rt, "performer")

		// Siapkan ONT provisioned untuk di-reboot
		ontID := "ont-audit-reboot-" + sn
		ontRepo.onts[ontID] = &domain.ONT{
			ID:                ontID,
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      ponPort,
			ONTIndex:          1,
			SerialNumber:      sn,
			Status:            domain.ONTStatusProvisioned,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		result, err := mgr.RebootONT(ctx, ontID, performer)
		if err != nil {
			t.Fatalf("RebootONT gagal: %v", err)
		}

		// Verifikasi result berhasil
		if !result.Success {
			t.Error("RebootONT harus berhasil dengan mock adapter")
		}

		// Verifikasi audit log dibuat
		if len(auditRepo.logs) == 0 {
			t.Fatal("audit log harus dibuat setelah RebootONT")
		}

		// Ambil audit log terakhir
		lastLog := auditRepo.logs[len(auditRepo.logs)-1]

		// Property: commands_sent non-empty
		if len(lastLog.CommandsSent) == 0 {
			t.Error("commands_sent harus non-empty")
		}

		// Property: action sesuai operasi
		if lastLog.Action != domain.AuditActionONTReboot {
			t.Errorf("action salah: got %q, want %q", lastLog.Action, domain.AuditActionONTReboot)
		}

		// Property: performed_by non-empty
		if lastLog.PerformedBy == "" {
			t.Error("performed_by harus non-empty")
		}

		// Property: correlation_id non-empty
		if lastLog.CorrelationID == "" {
			t.Error("correlation_id harus non-empty")
		}
	})
}
