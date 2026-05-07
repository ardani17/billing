package usecase

import (
	"context"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 19.1**
//
// Untuk sembarang koleksi OLT statuses:
// total = online + offline + maintenance
// Tidak ada OLT yang dihitung di lebih dari satu bucket status.
// =============================================================================

func statusCountsGen() *rapid.Generator[map[domain.OLTStatus]int64] {
	return rapid.Custom(func(t *rapid.T) map[domain.OLTStatus]int64 {
		return map[domain.OLTStatus]int64{
			domain.OLTStatusOnline:      int64(rapid.IntRange(0, 1000).Draw(t, "online")),
			domain.OLTStatusOffline:     int64(rapid.IntRange(0, 1000).Draw(t, "offline")),
			domain.OLTStatusMaintenance: int64(rapid.IntRange(0, 1000).Draw(t, "maintenance")),
		}
	})
}

// TestProperty_OLTStatusSummaryInvariant memverifikasi bahwa untuk sembarang
// koleksi OLT statuses, total_olts == online + offline + maintenance.
//
// **Memvalidasi: Kebutuhan 19.1**
func TestProperty_OLTStatusSummaryInvariant(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		counts := statusCountsGen().Draw(rt, "counts")
		alarmCount := int64(rapid.IntRange(0, 500).Draw(rt, "alarms"))

		repo := newMockOLTRepo()
		repo.statusCounts = counts
		alarmRepo := &mockAlarmRepo{activeByTenant: alarmCount}

		mgr := &oltManager{
			oltRepo:   repo,
			alarmRepo: alarmRepo,
		}

		summary, err := mgr.GetStatusSummary(context.Background())
		if err != nil {
			t.Fatalf("GetStatusSummary gagal: %v", err)
		}

		// Invariant utama: total = online + offline + maintenance
		expectedTotal := counts[domain.OLTStatusOnline] +
			counts[domain.OLTStatusOffline] +
			counts[domain.OLTStatusMaintenance]

		if summary.TotalOLTs != expectedTotal {
			t.Errorf("total=%d, want online(%d)+offline(%d)+maintenance(%d)=%d",
				summary.TotalOLTs,
				summary.OnlineCount, summary.OfflineCount, summary.MaintenanceCount,
				expectedTotal)
		}

		if summary.OnlineCount != counts[domain.OLTStatusOnline] {
			t.Errorf("online=%d, want %d", summary.OnlineCount, counts[domain.OLTStatusOnline])
		}
		if summary.OfflineCount != counts[domain.OLTStatusOffline] {
			t.Errorf("offline=%d, want %d", summary.OfflineCount, counts[domain.OLTStatusOffline])
		}
		if summary.MaintenanceCount != counts[domain.OLTStatusMaintenance] {
			t.Errorf("maintenance=%d, want %d",
				summary.MaintenanceCount, counts[domain.OLTStatusMaintenance])
		}

		// Verifikasi alarm count diteruskan dengan benar
		if summary.ActiveAlarmCount != alarmCount {
			t.Errorf("alarm_count=%d, want %d", summary.ActiveAlarmCount, alarmCount)
		}
	})
}
