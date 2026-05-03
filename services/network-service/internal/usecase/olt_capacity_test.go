package usecase

import (
	"fmt"
	"math"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Property 6: Capacity Calculation Correctness
// **Validates: Requirements 15.1, 15.2, 15.3, 15.4, 15.5**
//
// Untuk sembarang set PON port ONT counts dan max_ont_per_port:
// - available = total - used
// - utilization = (used / total) * 100
// - Warning untuk port > 90%
// =============================================================================

// ponPortGen menghasilkan slice PONPortStatus acak untuk property test.
func ponPortGen() *rapid.Generator[[]domain.PONPortStatus] {
	return rapid.Custom(func(t *rapid.T) []domain.PONPortStatus {
		count := rapid.IntRange(1, 16).Draw(t, "portCount")
		ports := make([]domain.PONPortStatus, count)
		for i := 0; i < count; i++ {
			operStatus := rapid.SampledFrom([]string{"up", "down"}).Draw(t, fmt.Sprintf("oper_%d", i))
			ontCount := rapid.IntRange(0, maxONTPerPort).Draw(t, fmt.Sprintf("ont_%d", i))
			ports[i] = domain.PONPortStatus{
				PortIndex:   i,
				AdminStatus: "up",
				OperStatus:  operStatus,
				ONTCount:    ontCount,
			}
		}
		return ports
	})
}

// TestProperty_CapacityCalculationCorrectness memverifikasi bahwa kalkulasi
// kapasitas OLT memenuhi invariant: available = total - used,
// utilization = (used/total)*100, dan warning untuk port > 90%.
//
// **Validates: Requirements 15.1, 15.2, 15.3, 15.4, 15.5**
func TestProperty_CapacityCalculationCorrectness(t *testing.T) {
	mgr := &oltManager{}

	rapid.Check(t, func(rt *rapid.T) {
		ports := ponPortGen().Draw(rt, "ports")
		capacity := mgr.calculateCapacity(ports)

		// Hitung expected values secara independen
		expectedTotal := len(ports) * maxONTPerPort
		expectedUsed := 0
		expectedActive := 0
		for _, p := range ports {
			expectedUsed += p.ONTCount
			if p.OperStatus == "up" {
				expectedActive++
			}
		}
		expectedAvailable := expectedTotal - expectedUsed
		if expectedAvailable < 0 {
			expectedAvailable = 0
		}

		// Invariant 1: available = total - used
		if capacity.AvailableONTSlots != expectedAvailable {
			t.Errorf("available=%d, want total(%d)-used(%d)=%d",
				capacity.AvailableONTSlots, expectedTotal, expectedUsed, expectedAvailable)
		}

		// Invariant 2: total_ont_slots = total_pon_ports * maxONTPerPort
		if capacity.TotalONTSlots != expectedTotal {
			t.Errorf("total_ont_slots=%d, want %d", capacity.TotalONTSlots, expectedTotal)
		}

		// Invariant 3: used_ont_slots = sum of ont_count per port
		if capacity.UsedONTSlots != expectedUsed {
			t.Errorf("used_ont_slots=%d, want %d", capacity.UsedONTSlots, expectedUsed)
		}

		// Invariant 4: utilization = (used / total) * 100
		expectedUtil := 0.0
		if expectedTotal > 0 {
			expectedUtil = float64(expectedUsed) / float64(expectedTotal) * 100
		}
		if math.Abs(capacity.UtilizationPercent-expectedUtil) > 0.001 {
			t.Errorf("utilization=%.4f, want %.4f", capacity.UtilizationPercent, expectedUtil)
		}

		// Invariant 5: active_pon_ports = jumlah port dengan oper_status=up
		if capacity.ActivePONPorts != expectedActive {
			t.Errorf("active_pon_ports=%d, want %d", capacity.ActivePONPorts, expectedActive)
		}

		// Invariant 6: port breakdown length = jumlah port
		if len(capacity.PortBreakdown) != len(ports) {
			t.Errorf("port_breakdown length=%d, want %d", len(capacity.PortBreakdown), len(ports))
		}

		// Invariant 7: warning untuk port > 90% utilisasi
		for i, pc := range capacity.PortBreakdown {
			portUtil := float64(ports[i].ONTCount) / float64(maxONTPerPort) * 100
			if portUtil > warningThreshold && pc.Warning == "" {
				t.Errorf("port %d utilisasi %.1f%% > 90%% tapi warning kosong", i, portUtil)
			}
			if portUtil <= warningThreshold && pc.Warning != "" {
				t.Errorf("port %d utilisasi %.1f%% <= 90%% tapi ada warning: %q", i, portUtil, pc.Warning)
			}
		}
	})
}

// TestCapacity_EmptyPorts memverifikasi kapasitas dengan 0 port (edge case).
func TestCapacity_EmptyPorts(t *testing.T) {
	mgr := &oltManager{}
	capacity := mgr.calculateCapacity([]domain.PONPortStatus{})

	if capacity.TotalPONPorts != 0 {
		t.Errorf("total_pon_ports=%d, want 0", capacity.TotalPONPorts)
	}
	if capacity.TotalONTSlots != 0 {
		t.Errorf("total_ont_slots=%d, want 0", capacity.TotalONTSlots)
	}
	if capacity.UtilizationPercent != 0 {
		t.Errorf("utilization=%.2f, want 0", capacity.UtilizationPercent)
	}
}

// TestCapacity_FullPort memverifikasi warning muncul saat port penuh.
func TestCapacity_FullPort(t *testing.T) {
	mgr := &oltManager{}
	ports := []domain.PONPortStatus{
		{PortIndex: 0, OperStatus: "up", ONTCount: maxONTPerPort},
	}
	capacity := mgr.calculateCapacity(ports)

	if capacity.AvailableONTSlots != 0 {
		t.Errorf("available=%d, want 0", capacity.AvailableONTSlots)
	}
	if capacity.PortBreakdown[0].Warning == "" {
		t.Error("port penuh (100%) harus memiliki warning")
	}
}
