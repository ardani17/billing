package usecase

import (
	"testing"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Unit Tests — SyncScheduler
// =============================================================================

// TestNewSyncScheduler memverifikasi bahwa SyncScheduler dapat dibuat dengan parameter valid.
func TestNewSyncScheduler(t *testing.T) {
	logger := zerolog.Nop()

	scheduler := NewSyncScheduler(nil, nil, 15, logger)
	if scheduler == nil {
		t.Fatal("NewSyncScheduler mengembalikan nil")
	}
	if scheduler.interval.Minutes() != 15 {
		t.Errorf("interval = %v, want 15m", scheduler.interval)
	}

	// Default interval jika <= 0
	scheduler2 := NewSyncScheduler(nil, nil, 0, logger)
	if scheduler2.interval.Minutes() != 15 {
		t.Errorf("interval default = %v, want 15m", scheduler2.interval)
	}

	scheduler3 := NewSyncScheduler(nil, nil, -5, logger)
	if scheduler3.interval.Minutes() != 15 {
		t.Errorf("interval negatif = %v, want 15m", scheduler3.interval)
	}
}

// TestSyncSchedulerStopBeforeStart memverifikasi bahwa Stop() tidak panic
// jika dipanggil sebelum Start().
func TestSyncSchedulerStopBeforeStart(t *testing.T) {
	logger := zerolog.Nop()
	scheduler := NewSyncScheduler(nil, nil, 15, logger)

	// Stop tanpa Start seharusnya tidak panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panic: %v", r)
		}
	}()
	scheduler.Stop()
}

// TestFilterPPPoERouters memverifikasi bahwa filterPPPoERouters hanya mengembalikan
// router dengan status online dan service_type "pppoe".
func TestFilterPPPoERouters(t *testing.T) {
	routers := []*domain.Router{
		{
			ID:           "r1",
			Name:         "Router Online PPPoE",
			Status:       domain.StatusOnline,
			ServiceTypes: []string{"pppoe", "hotspot"},
		},
		{
			ID:           "r2",
			Name:         "Router Offline PPPoE",
			Status:       domain.StatusOffline,
			ServiceTypes: []string{"pppoe"},
		},
		{
			ID:           "r3",
			Name:         "Router Maintenance PPPoE",
			Status:       domain.StatusMaintenance,
			ServiceTypes: []string{"pppoe"},
		},
		{
			ID:           "r4",
			Name:         "Router Online Hotspot Only",
			Status:       domain.StatusOnline,
			ServiceTypes: []string{"hotspot"},
		},
		{
			ID:           "r5",
			Name:         "Router Online PPPoE+DHCP",
			Status:       domain.StatusOnline,
			ServiceTypes: []string{"pppoe", "dhcp_binding"},
		},
		{
			ID:           "r6",
			Name:         "Router Online No Service",
			Status:       domain.StatusOnline,
			ServiceTypes: []string{},
		},
	}

	result := filterPPPoERouters(routers)

	// Hanya r1 dan r5 yang online + pppoe
	if len(result) != 2 {
		t.Fatalf("filterPPPoERouters count = %d, want 2", len(result))
	}

	expectedIDs := map[string]bool{"r1": true, "r5": true}
	for _, r := range result {
		if !expectedIDs[r.ID] {
			t.Errorf("unexpected router in result: %s (%s)", r.ID, r.Name)
		}
	}
}

// TestFilterPPPoERoutersEmpty memverifikasi bahwa filterPPPoERouters mengembalikan
// slice kosong jika tidak ada router yang memenuhi kriteria.
func TestFilterPPPoERoutersEmpty(t *testing.T) {
	// Semua offline
	routers := []*domain.Router{
		{ID: "r1", Status: domain.StatusOffline, ServiceTypes: []string{"pppoe"}},
		{ID: "r2", Status: domain.StatusMaintenance, ServiceTypes: []string{"pppoe"}},
	}
	result := filterPPPoERouters(routers)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d routers", len(result))
	}

	// Nil input
	result2 := filterPPPoERouters(nil)
	if len(result2) != 0 {
		t.Errorf("expected empty result for nil input, got %d routers", len(result2))
	}
}
