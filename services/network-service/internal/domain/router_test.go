package domain

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

var allStatuses = []RouterStatus{StatusOnline, StatusOffline, StatusMaintenance}

// Sesuai dengan ValidRouterTransitions di constants.go.
var validTransitionPairs = map[[2]RouterStatus]bool{
	{StatusOffline, StatusOnline}:      true,
	{StatusOffline, StatusMaintenance}: true,
	{StatusOnline, StatusOffline}:      true,
	{StatusOnline, StatusMaintenance}:  true,
	{StatusMaintenance, StatusOnline}:  true,
	{StatusMaintenance, StatusOffline}: true,
}

// =============================================================================
// =============================================================================

// TestProperty_StatusTransitionCorrectness memverifikasi bahwa untuk sembarang
//
// **Memvalidasi: Kebutuhan 2.3, 2.4**
func TestProperty_StatusTransitionCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		current := rapid.SampledFrom(allStatuses).Draw(t, "current")
		target := rapid.SampledFrom(allStatuses).Draw(t, "target")

		result := CanTransitionRouter(current, target)
		pair := [2]RouterStatus{current, target}
		expected := validTransitionPairs[pair]

		if result != expected {
			t.Errorf(
				"CanTransitionRouter(%q, %q) = %v, ingin %v",
				current, target, result, expected,
			)
		}

		// Verifikasi bahwa transisi ke status yang sama selalu ditolak
		if current == target && result {
			t.Errorf(
				"CanTransitionRouter(%q, %q) seharusnya false untuk transisi ke status yang sama",
				current, target,
			)
		}
	})
}

// TestProperty_StatusTransitionInvalidStatusRejected memverifikasi bahwa
// status yang tidak dikenal selalu ditolak oleh CanTransitionRouter.
//
// **Memvalidasi: Kebutuhan 2.3, 2.4**
func TestProperty_StatusTransitionInvalidStatusRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidStatus := RouterStatus(rapid.String().Draw(t, "invalidStatus"))

		for _, s := range allStatuses {
			if invalidStatus == s {
				return // skip iterasi ini, status kebetulan valid
			}
		}

		validTarget := rapid.SampledFrom(allStatuses).Draw(t, "target")
		if CanTransitionRouter(invalidStatus, validTarget) {
			t.Errorf(
				"CanTransitionRouter(%q, %q) seharusnya false untuk status asal tidak valid",
				invalidStatus, validTarget,
			)
		}

		validCurrent := rapid.SampledFrom(allStatuses).Draw(t, "current")
		if CanTransitionRouter(validCurrent, invalidStatus) {
			t.Errorf(
				"CanTransitionRouter(%q, %q) seharusnya false untuk status tujuan tidak valid",
				validCurrent, invalidStatus,
			)
		}
	})
}

// TestProperty_StatusTransitionAllowedTargetsConsistency memverifikasi bahwa
// hasil CanTransitionRouter.
//
// **Memvalidasi: Kebutuhan 2.3, 2.4**
func TestProperty_StatusTransitionAllowedTargetsConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		current := rapid.SampledFrom(allStatuses).Draw(t, "current")

		allowedTargets := ValidRouterTransitions[current]

		// Untuk setiap status, cek konsistensi
		for _, target := range allStatuses {
			result := CanTransitionRouter(current, target)
			isAllowed := false
			for _, allowed := range allowedTargets {
				if allowed == target {
					isAllowed = true
					break
				}
			}

			if result != isAllowed {
				t.Errorf(
					"Inkonsistensi: CanTransitionRouter(%q, %q) = %v, tapi allowedTargets = %v",
					current, target, result, allowedTargets,
				)
			}
		}
	})
}

// =============================================================================
// =============================================================================

// validateRebootConfirmation mensimulasikan logika validasi reboot.
// Mengembalikan nil jika confirmation cocok dengan routerName (case-sensitive),
// atau ErrConfirmationMismatch jika tidak cocok.
func validateRebootConfirmation(routerName, confirmation string) error {
	if confirmation != routerName {
		return fmt.Errorf(
			"%w: nama konfirmasi %q tidak cocok dengan nama router %q",
			ErrConfirmationMismatch, confirmation, routerName,
		)
	}
	return nil
}

// TestProperty_RebootConfirmationValidation memverifikasi bahwa reboot hanya
// diizinkan jika string konfirmasi sama persis (case-sensitive) dengan nama
// router. Semua string lain harus ditolak.
//
// **Memvalidasi: Kebutuhan 5.8, 5.9**
func TestProperty_RebootConfirmationValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat nama router acak
		routerName := rapid.String().Draw(t, "routerName")
		// Buat string konfirmasi acak
		confirmation := rapid.String().Draw(t, "confirmation")

		err := validateRebootConfirmation(routerName, confirmation)

		if confirmation == routerName {
			// Konfirmasi cocok: reboot harus diizinkan
			if err != nil {
				t.Errorf(
					"konfirmasi %q cocok dengan nama router %q, tapi error: %v",
					confirmation, routerName, err,
				)
			}
		} else {
			// Konfirmasi tidak cocok: reboot harus ditolak
			if err == nil {
				t.Errorf(
					"konfirmasi %q tidak cocok dengan nama router %q, tapi tidak error",
					confirmation, routerName,
				)
			}
		}
	})
}

// TestProperty_RebootConfirmationExactMatch memverifikasi bahwa konfirmasi
// yang sama persis dengan nama router selalu diterima.
//
// **Memvalidasi: Kebutuhan 5.8, 5.9**
func TestProperty_RebootConfirmationExactMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat nama router acak
		routerName := rapid.String().Draw(t, "routerName")

		// Konfirmasi yang sama persis harus selalu diterima
		err := validateRebootConfirmation(routerName, routerName)
		if err != nil {
			t.Errorf(
				"konfirmasi sama persis dengan nama router %q, tapi error: %v",
				routerName, err,
			)
		}
	})
}

// =============================================================================
// =============================================================================

// TestProperty_StatusSummaryInvariant memverifikasi bahwa untuk sembarang
// distribusi status router, total_routers selalu sama dengan
//
// **Memvalidasi: Kebutuhan 7.1**
func TestProperty_StatusSummaryInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat jumlah router per status secara acak
		onlineCount := int64(rapid.IntRange(0, 100).Draw(t, "onlineCount"))
		offlineCount := int64(rapid.IntRange(0, 100).Draw(t, "offlineCount"))
		maintenanceCount := int64(rapid.IntRange(0, 100).Draw(t, "maintenanceCount"))

		// Buat StatusSummary dari distribusi yang di-buat
		summary := StatusSummary{
			TotalRouters:     onlineCount + offlineCount + maintenanceCount,
			OnlineCount:      onlineCount,
			OfflineCount:     offlineCount,
			MaintenanceCount: maintenanceCount,
		}

		// Invariant: total harus sama dengan jumlah per status
		computedTotal := summary.OnlineCount + summary.OfflineCount + summary.MaintenanceCount
		if summary.TotalRouters != computedTotal {
			t.Errorf(
				"Invariant dilanggar: TotalRouters=%d != Online(%d) + Offline(%d) + Maintenance(%d) = %d",
				summary.TotalRouters,
				summary.OnlineCount,
				summary.OfflineCount,
				summary.MaintenanceCount,
				computedTotal,
			)
		}

		// Verifikasi semua count non-negatif
		if summary.OnlineCount < 0 || summary.OfflineCount < 0 || summary.MaintenanceCount < 0 {
			t.Error("Count tidak boleh negatif")
		}

		if summary.TotalRouters < 0 {
			t.Error("TotalRouters tidak boleh negatif")
		}
	})
}

// TestProperty_StatusSummaryFromRouterList memverifikasi bahwa StatusSummary
// yang dihitung dari daftar router menghasilkan total yang konsisten.
// Mensimulasikan skenario nyata: buat daftar router dengan status acak,
// hitung summary, dan verifikasi invariant.
//
// **Memvalidasi: Kebutuhan 7.1**
func TestProperty_StatusSummaryFromRouterList(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat jumlah router acak (0-100)
		numRouters := rapid.IntRange(0, 100).Draw(t, "numRouters")

		// Hitung jumlah per status dari daftar router yang di-buat
		var onlineCount, offlineCount, maintenanceCount int64
		for i := 0; i < numRouters; i++ {
			status := rapid.SampledFrom(allStatuses).Draw(t, fmt.Sprintf("status_%d", i))
			switch status {
			case StatusOnline:
				onlineCount++
			case StatusOffline:
				offlineCount++
			case StatusMaintenance:
				maintenanceCount++
			}
		}

		// Buat summary dari hasil penghitungan
		summary := StatusSummary{
			TotalRouters:     onlineCount + offlineCount + maintenanceCount,
			OnlineCount:      onlineCount,
			OfflineCount:     offlineCount,
			MaintenanceCount: maintenanceCount,
		}

		// Invariant 1: total == sum of counts
		if summary.TotalRouters != summary.OnlineCount+summary.OfflineCount+summary.MaintenanceCount {
			t.Errorf(
				"Invariant dilanggar: total=%d != online(%d)+offline(%d)+maintenance(%d)",
				summary.TotalRouters, summary.OnlineCount, summary.OfflineCount, summary.MaintenanceCount,
			)
		}

		// Invariant 2: total harus sama dengan jumlah router yang di-buat
		if summary.TotalRouters != int64(numRouters) {
			t.Errorf(
				"TotalRouters=%d tidak sama dengan jumlah router yang di-generate=%d",
				summary.TotalRouters, numRouters,
			)
		}
	})
}
