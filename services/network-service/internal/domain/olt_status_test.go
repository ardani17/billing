package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// allOLTStatuses berisi semua status OLT yang valid.
var allOLTStatuses = []OLTStatus{OLTStatusOnline, OLTStatusOffline, OLTStatusMaintenance}

// validOLTTransitionPairs berisi semua pasangan transisi status OLT yang valid.
// Sesuai dengan ValidOLTTransitions di olt.go.
var validOLTTransitionPairs = map[[2]OLTStatus]bool{
	{OLTStatusOffline, OLTStatusOnline}:      true,
	{OLTStatusOffline, OLTStatusMaintenance}: true,
	{OLTStatusOnline, OLTStatusOffline}:      true,
	{OLTStatusOnline, OLTStatusMaintenance}:  true,
	{OLTStatusMaintenance, OLTStatusOnline}:  true,
	{OLTStatusMaintenance, OLTStatusOffline}: true,
}

// =============================================================================
// Feature: olt-management, Property 1: OLT Status Transition Validation
// =============================================================================

// TestProperty_OLTStatusTransitionValidation memverifikasi bahwa untuk sembarang
// pasangan OLTStatus (current, target), CanTransitionOLT mengembalikan true
// jika dan hanya jika pasangan tersebut ada di ValidOLTTransitions.
// Untuk semua pasangan yang tidak valid, fungsi harus mengembalikan false.
//
// **Validates: Requirements 2.3, 2.4**
func TestProperty_OLTStatusTransitionValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih status asal dan tujuan secara acak dari semua status valid
		current := rapid.SampledFrom(allOLTStatuses).Draw(t, "current")
		target := rapid.SampledFrom(allOLTStatuses).Draw(t, "target")

		result := CanTransitionOLT(current, target)
		pair := [2]OLTStatus{current, target}
		expected := validOLTTransitionPairs[pair]

		if result != expected {
			t.Errorf(
				"CanTransitionOLT(%q, %q) = %v, ingin %v",
				current, target, result, expected,
			)
		}
	})
}

// TestProperty_OLTStatusTransitionInvalidPairs memverifikasi bahwa untuk
// sembarang pasangan yang TIDAK ada di ValidOLTTransitions, CanTransitionOLT
// mengembalikan false. Termasuk status yang tidak dikenal.
//
// **Validates: Requirements 2.3, 2.4**
func TestProperty_OLTStatusTransitionInvalidPairs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate string acak sebagai status tidak valid
		invalidStatus := OLTStatus(rapid.String().Draw(t, "invalidStatus"))

		// Pastikan bukan salah satu status valid
		for _, s := range allOLTStatuses {
			if invalidStatus == s {
				return // skip iterasi ini, status kebetulan valid
			}
		}

		// Transisi dari status tidak valid harus selalu ditolak
		validTarget := rapid.SampledFrom(allOLTStatuses).Draw(t, "target")
		if CanTransitionOLT(invalidStatus, validTarget) {
			t.Errorf(
				"CanTransitionOLT(%q, %q) seharusnya false untuk status asal tidak valid",
				invalidStatus, validTarget,
			)
		}

		// Transisi ke status tidak valid juga harus ditolak
		validCurrent := rapid.SampledFrom(allOLTStatuses).Draw(t, "current")
		if CanTransitionOLT(validCurrent, invalidStatus) {
			t.Errorf(
				"CanTransitionOLT(%q, %q) seharusnya false untuk status tujuan tidak valid",
				validCurrent, invalidStatus,
			)
		}
	})
}

// TestProperty_OLTSelfTransitionAlwaysInvalid memverifikasi bahwa transisi
// ke status yang sama (self-transition) selalu ditolak oleh CanTransitionOLT.
//
// **Validates: Requirements 2.3, 2.4**
func TestProperty_OLTSelfTransitionAlwaysInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih status acak
		status := rapid.SampledFrom(allOLTStatuses).Draw(t, "status")

		// Self-transition harus selalu false
		if CanTransitionOLT(status, status) {
			t.Errorf(
				"CanTransitionOLT(%q, %q) seharusnya false untuk self-transition",
				status, status,
			)
		}
	})
}

// =============================================================================
// Example-based tests untuk transisi status OLT yang diketahui
// =============================================================================

// TestOLTStatusTransition_KnownValidPairs memverifikasi transisi valid yang
// sudah diketahui secara eksplisit.
//
// **Validates: Requirements 2.3, 2.4**
func TestOLTStatusTransition_KnownValidPairs(t *testing.T) {
	// Daftar transisi valid sesuai Requirements 2.3
	validPairs := []struct {
		from OLTStatus
		to   OLTStatus
	}{
		{OLTStatusOffline, OLTStatusOnline},
		{OLTStatusOffline, OLTStatusMaintenance},
		{OLTStatusOnline, OLTStatusOffline},
		{OLTStatusOnline, OLTStatusMaintenance},
		{OLTStatusMaintenance, OLTStatusOnline},
		{OLTStatusMaintenance, OLTStatusOffline},
	}

	for _, pair := range validPairs {
		t.Run(string(pair.from)+"→"+string(pair.to), func(t *testing.T) {
			if !CanTransitionOLT(pair.from, pair.to) {
				t.Errorf(
					"CanTransitionOLT(%q, %q) seharusnya true untuk transisi valid",
					pair.from, pair.to,
				)
			}
		})
	}
}

// TestOLTStatusTransition_KnownInvalidPairs memverifikasi transisi yang
// tidak valid ditolak, termasuk self-transition dan status tidak dikenal.
//
// **Validates: Requirements 2.3, 2.4**
func TestOLTStatusTransition_KnownInvalidPairs(t *testing.T) {
	invalidPairs := []struct {
		name string
		from OLTStatus
		to   OLTStatus
	}{
		{"self: online→online", OLTStatusOnline, OLTStatusOnline},
		{"self: offline→offline", OLTStatusOffline, OLTStatusOffline},
		{"self: maintenance→maintenance", OLTStatusMaintenance, OLTStatusMaintenance},
		{"unknown→online", OLTStatus("unknown"), OLTStatusOnline},
		{"online→unknown", OLTStatusOnline, OLTStatus("unknown")},
		{"empty→online", OLTStatus(""), OLTStatusOnline},
		{"online→empty", OLTStatusOnline, OLTStatus("")},
	}

	for _, pair := range invalidPairs {
		t.Run(pair.name, func(t *testing.T) {
			if CanTransitionOLT(pair.from, pair.to) {
				t.Errorf(
					"CanTransitionOLT(%q, %q) seharusnya false untuk transisi tidak valid",
					pair.from, pair.to,
				)
			}
		})
	}
}
