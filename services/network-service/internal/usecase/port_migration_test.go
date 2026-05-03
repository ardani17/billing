// Package usecase — Property test untuk port migration detection.
// Memverifikasi bahwa HandlePortMigration selalu mempublikasikan event
// ont.port_migrated dengan informasi port lama dan baru yang benar.
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Property 5: Port Migration Detection
// **Validates: Requirements 11.1**
//
// Untuk sembarang ONT di mana current port berbeda dari recorded port,
// HandlePortMigration SHALL mempublikasikan event ont.port_migrated
// dengan informasi old/new port yang benar.
// =============================================================================

// migrationSerialNumberGen menghasilkan serial number ONT acak untuk migration test.
func migrationSerialNumberGen() *rapid.Generator[string] {
	return rapid.StringMatching(`ZTEG[A-F0-9]{8}`)
}

// TestProperty5_PortMigrationDetection memverifikasi bahwa untuk sembarang ONT
// dengan port yang berbeda, HandlePortMigration mempublikasikan event
// ont.port_migrated dengan old/new port information yang benar.
//
// **Validates: Requirements 11.1**
func TestProperty5_PortMigrationDetection(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
		ctx := context.Background()

		sn := migrationSerialNumberGen().Draw(rt, "serialNumber")
		oldPort := rapid.IntRange(0, 15).Draw(rt, "oldPort")
		oldONTIdx := rapid.IntRange(1, 128).Draw(rt, "oldONTIdx")

		// Generate new port yang berbeda dari old port
		newPort := rapid.IntRange(0, 15).Draw(rt, "newPort")
		newONTIdx := rapid.IntRange(1, 128).Draw(rt, "newONTIdx")

		// Siapkan ONT di repo dengan port lama
		ontID := "ont-migrate-" + sn
		ontRepo.onts[ontID] = &domain.ONT{
			ID:                ontID,
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      oldPort,
			ONTIndex:          oldONTIdx,
			SerialNumber:      sn,
			Status:            domain.ONTStatusProvisioned,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// Panggil HandlePortMigration dengan port baru
		err := mgr.HandlePortMigration(ctx, ontID, oldPort, newPort, oldONTIdx, newONTIdx)
		if err != nil {
			t.Fatalf("HandlePortMigration gagal: %v", err)
		}

		// Property: event ont.port_migrated HARUS dipublikasikan
		if len(eventPub.portMigratedEvents) == 0 {
			t.Fatal("event ont.port_migrated harus dipublikasikan")
		}

		// Ambil event terakhir
		evt := eventPub.portMigratedEvents[len(eventPub.portMigratedEvents)-1]

		// Property: old port information benar
		if evt.OldPortIndex != oldPort {
			t.Errorf("old_port_index salah: got %d, want %d", evt.OldPortIndex, oldPort)
		}

		// Property: new port information benar
		if evt.NewPortIndex != newPort {
			t.Errorf("new_port_index salah: got %d, want %d", evt.NewPortIndex, newPort)
		}

		// Property: old ONT index benar
		if evt.OldONTIndex != oldONTIdx {
			t.Errorf("old_ont_index salah: got %d, want %d", evt.OldONTIndex, oldONTIdx)
		}

		// Property: new ONT index benar
		if evt.NewONTIndex != newONTIdx {
			t.Errorf("new_ont_index salah: got %d, want %d", evt.NewONTIndex, newONTIdx)
		}

		// Property: serial number benar
		if evt.SerialNumber != sn {
			t.Errorf("serial_number salah: got %q, want %q", evt.SerialNumber, sn)
		}

		// Property: ont_id benar
		if evt.ONTID != ontID {
			t.Errorf("ont_id salah: got %q, want %q", evt.ONTID, ontID)
		}

		// Property: olt_id benar
		if evt.OLTID != "olt-001" {
			t.Errorf("olt_id salah: got %q, want olt-001", evt.OLTID)
		}

		// Property: tenant_id benar
		if evt.TenantID != "tenant-001" {
			t.Errorf("tenant_id salah: got %q, want tenant-001", evt.TenantID)
		}

		// Property: correlation_id non-empty
		if evt.CorrelationID == "" {
			t.Error("correlation_id harus non-empty")
		}
	})
}

// TestProperty5_PortMigrationDetection_SamePort memverifikasi bahwa
// HandlePortMigration tetap mempublikasikan event meskipun port sama
// (edge case: ONT index berubah tapi port tetap).
//
// **Validates: Requirements 11.1**
func TestProperty5_PortMigrationDetection_SamePort(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr, ontRepo, _, _, _, eventPub, _ := newTestProvisioningManager()
		ctx := context.Background()

		sn := migrationSerialNumberGen().Draw(rt, "serialNumber")
		port := rapid.IntRange(0, 15).Draw(rt, "port")
		oldONTIdx := rapid.IntRange(1, 64).Draw(rt, "oldONTIdx")
		newONTIdx := rapid.IntRange(65, 128).Draw(rt, "newONTIdx")

		ontID := "ont-sameport-" + sn
		ontRepo.onts[ontID] = &domain.ONT{
			ID:                ontID,
			TenantID:          "tenant-001",
			OLTID:             "olt-001",
			PONPortIndex:      port,
			ONTIndex:          oldONTIdx,
			SerialNumber:      sn,
			Status:            domain.ONTStatusProvisioned,
			ProvisioningState: domain.ProvisioningStateCompleted,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// Port sama, tapi ONT index berbeda
		err := mgr.HandlePortMigration(ctx, ontID, port, port, oldONTIdx, newONTIdx)
		if err != nil {
			t.Fatalf("HandlePortMigration gagal: %v", err)
		}

		// Event tetap harus dipublikasikan
		if len(eventPub.portMigratedEvents) == 0 {
			t.Fatal("event ont.port_migrated harus dipublikasikan meski port sama")
		}

		evt := eventPub.portMigratedEvents[len(eventPub.portMigratedEvents)-1]

		// Verifikasi old/new ONT index benar
		if evt.OldONTIndex != oldONTIdx {
			t.Errorf("old_ont_index salah: got %d, want %d", evt.OldONTIndex, oldONTIdx)
		}
		if evt.NewONTIndex != newONTIdx {
			t.Errorf("new_ont_index salah: got %d, want %d", evt.NewONTIndex, newONTIdx)
		}
	})
}
