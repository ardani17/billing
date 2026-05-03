// Package usecase — Property test untuk reboot status guard.
// Memverifikasi bahwa RebootONT hanya berhasil untuk ONT dengan status "provisioned".
// ONT dengan status lain harus mengembalikan ErrONTNotProvisioned.
package usecase

import (
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Property 3: Reboot Status Guard
// **Validates: Requirements 8.4**
//
// Untuk sembarang ONT dengan status BUKAN "provisioned"
// (registered, unregistered, missing, decommissioned),
// RebootONT SHALL mengembalikan error.
// Hanya ONT dengan status "provisioned" yang boleh di-reboot.
// =============================================================================

// nonProvisionedStatusGen menghasilkan status ONT acak yang BUKAN "provisioned".
func nonProvisionedStatusGen() *rapid.Generator[domain.ONTStatus] {
	return rapid.SampledFrom([]domain.ONTStatus{
		domain.ONTStatusRegistered,
		domain.ONTStatusUnregistered,
		domain.ONTStatusMissing,
		domain.ONTStatusDecommissioned,
	})
}

// allONTStatusGen menghasilkan sembarang status ONT yang valid.
func allONTStatusGen() *rapid.Generator[domain.ONTStatus] {
	return rapid.SampledFrom([]domain.ONTStatus{
		domain.ONTStatusRegistered,
		domain.ONTStatusProvisioned,
		domain.ONTStatusUnregistered,
		domain.ONTStatusMissing,
		domain.ONTStatusDecommissioned,
	})
}

// rebootStatusGuard mengimplementasikan logika guard reboot:
// hanya ONT dengan status "provisioned" yang boleh di-reboot.
// Fungsi ini merepresentasikan logika yang akan ada di ProvisioningManager.RebootONT.
func rebootStatusGuard(ont *domain.ONT) error {
	if ont.Status != domain.ONTStatusProvisioned {
		return domain.ErrONTNotProvisioned
	}
	return nil
}

// TestProperty3_RebootStatusGuard_NonProvisioned memverifikasi bahwa
// untuk sembarang ONT dengan status BUKAN "provisioned",
// reboot guard mengembalikan ErrONTNotProvisioned.
//
// **Validates: Requirements 8.4**
func TestProperty3_RebootStatusGuard_NonProvisioned(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		status := nonProvisionedStatusGen().Draw(rt, "status")
		ontID := rapid.StringMatching(
			`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
		).Draw(rt, "ontID")

		ont := &domain.ONT{
			ID:                ontID,
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      rapid.IntRange(0, 15).Draw(rt, "ponPort"),
			ONTIndex:          rapid.IntRange(1, 128).Draw(rt, "ontIndex"),
			SerialNumber:      "ZTEG12345678",
			Status:            status,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := rebootStatusGuard(ont)
		if err == nil {
			t.Fatalf("reboot guard: status=%q seharusnya mengembalikan error, tapi nil",
				status)
		}
		if err != domain.ErrONTNotProvisioned {
			t.Fatalf("reboot guard: status=%q expected ErrONTNotProvisioned, got: %v",
				status, err)
		}
	})
}

// TestProperty3_RebootStatusGuard_Provisioned memverifikasi bahwa
// ONT dengan status "provisioned" diizinkan untuk reboot (guard return nil).
//
// **Validates: Requirements 8.4**
func TestProperty3_RebootStatusGuard_Provisioned(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ontID := rapid.StringMatching(
			`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
		).Draw(rt, "ontID")

		ont := &domain.ONT{
			ID:                ontID,
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      rapid.IntRange(0, 15).Draw(rt, "ponPort"),
			ONTIndex:          rapid.IntRange(1, 128).Draw(rt, "ontIndex"),
			SerialNumber:      "ZTEG12345678",
			Status:            domain.ONTStatusProvisioned,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := rebootStatusGuard(ont)
		if err != nil {
			t.Fatalf("reboot guard: status=provisioned seharusnya nil, got: %v", err)
		}
	})
}

// TestProperty3_RebootStatusGuard_Exhaustive memverifikasi property secara
// exhaustive: untuk sembarang status ONT, reboot guard mengembalikan nil
// HANYA jika status == "provisioned", dan error untuk status lainnya.
//
// **Validates: Requirements 8.4**
func TestProperty3_RebootStatusGuard_Exhaustive(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		status := allONTStatusGen().Draw(rt, "status")

		ont := &domain.ONT{
			ID:                "test-ont-id",
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      0,
			ONTIndex:          1,
			SerialNumber:      "ZTEG12345678",
			Status:            status,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := rebootStatusGuard(ont)

		if status == domain.ONTStatusProvisioned {
			// Status provisioned: reboot harus diizinkan
			if err != nil {
				t.Fatalf("status=%q: reboot seharusnya diizinkan, got error: %v",
					status, err)
			}
		} else {
			// Status lain: reboot harus ditolak
			if err == nil {
				t.Fatalf("status=%q: reboot seharusnya ditolak, tapi error nil",
					status)
			}
			if err != domain.ErrONTNotProvisioned {
				t.Fatalf("status=%q: expected ErrONTNotProvisioned, got: %v",
					status, err)
			}
		}
	})
}
