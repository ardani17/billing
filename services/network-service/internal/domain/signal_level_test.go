package domain

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

// allSignalLevels berisi semua signal level yang valid.
var allSignalLevels = []SignalLevel{SignalNormal, SignalWarning, SignalWeak, SignalCritical}

// =============================================================================
// Feature: olt-management, Property 2: Signal Level Classification
// =============================================================================

// TestProperty_SignalLevelExhaustive memverifikasi bahwa untuk sembarang
// float64 rx_power dalam dBm, ClassifySignal selalu mengembalikan tepat satu
// dari 4 signal level yang valid. Tidak ada input yang menghasilkan level
// di luar set yang didefinisikan (exhaustive).
//
// **Validates: Requirements 2.8, 10.2**
func TestProperty_SignalLevelExhaustive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate float64 acak termasuk nilai ekstrem
		rxPower := rapid.Float64().Draw(t, "rxPower")

		// Skip NaN karena bukan nilai dBm yang valid
		if math.IsNaN(rxPower) {
			return
		}

		result := ClassifySignal(rxPower)

		// Pastikan hasil adalah salah satu dari 4 level yang valid
		valid := false
		for _, level := range allSignalLevels {
			if result == level {
				valid = true
				break
			}
		}

		if !valid {
			t.Errorf(
				"ClassifySignal(%v) = %q, bukan salah satu dari signal level yang valid",
				rxPower, result,
			)
		}
	})
}

// TestProperty_SignalLevelMatchesThresholds memverifikasi bahwa untuk sembarang
// float64 rx_power dalam dBm, ClassifySignal mengembalikan level yang sesuai
// dengan aturan threshold:
//   - Normal:   >= -25 dBm
//   - Warning:  >= -27 dBm dan < -25 dBm
//   - Weak:     >= -30 dBm dan < -27 dBm
//   - Critical: < -30 dBm
//
// **Validates: Requirements 2.8, 10.2**
func TestProperty_SignalLevelMatchesThresholds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate float64 dalam rentang realistis dBm (-80 sampai +10)
		rxPower := rapid.Float64Range(-80.0, 10.0).Draw(t, "rxPower")

		result := ClassifySignal(rxPower)

		// Tentukan level yang diharapkan berdasarkan threshold
		var expected SignalLevel
		switch {
		case rxPower >= SignalThresholdWarning: // >= -25
			expected = SignalNormal
		case rxPower >= SignalThresholdWeak: // >= -27 dan < -25
			expected = SignalWarning
		case rxPower >= SignalThresholdCritical: // >= -30 dan < -27
			expected = SignalWeak
		default: // < -30
			expected = SignalCritical
		}

		if result != expected {
			t.Errorf(
				"ClassifySignal(%v) = %q, ingin %q",
				rxPower, result, expected,
			)
		}
	})
}

// TestProperty_SignalLevelBoundaryValues memverifikasi bahwa nilai tepat di
// batas threshold (-25, -27, -30 dBm) diklasifikasikan dengan benar.
// Juga menguji nilai sedikit di atas dan di bawah setiap batas menggunakan
// math.Nextafter untuk mendapatkan float64 terdekat yang benar-benar berbeda.
//
// **Validates: Requirements 2.8, 10.2**
func TestProperty_SignalLevelBoundaryValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih salah satu threshold secara acak
		thresholds := []float64{
			SignalThresholdWarning,  // -25.0
			SignalThresholdWeak,     // -27.0
			SignalThresholdCritical, // -30.0
		}
		threshold := rapid.SampledFrom(thresholds).Draw(t, "threshold")

		// Gunakan math.Nextafter untuk mendapatkan float64 tepat di bawah threshold.
		// Ini menghindari masalah presisi floating-point dimana threshold - epsilon
		// bisa tetap sama dengan threshold jika epsilon terlalu kecil.
		justBelow := math.Nextafter(threshold, math.Inf(-1))

		// Test tepat di batas (inklusif ke level atas)
		atBoundary := ClassifySignal(threshold)
		// Test float64 terdekat di bawah batas
		belowBoundary := ClassifySignal(justBelow)

		switch threshold {
		case SignalThresholdWarning: // -25.0
			if atBoundary != SignalNormal {
				t.Errorf("ClassifySignal(%v) = %q, ingin %q (tepat di batas warning)",
					threshold, atBoundary, SignalNormal)
			}
			if belowBoundary != SignalWarning {
				t.Errorf("ClassifySignal(%v) = %q, ingin %q (sedikit di bawah batas warning)",
					justBelow, belowBoundary, SignalWarning)
			}

		case SignalThresholdWeak: // -27.0
			if atBoundary != SignalWarning {
				t.Errorf("ClassifySignal(%v) = %q, ingin %q (tepat di batas weak)",
					threshold, atBoundary, SignalWarning)
			}
			if belowBoundary != SignalWeak {
				t.Errorf("ClassifySignal(%v) = %q, ingin %q (sedikit di bawah batas weak)",
					justBelow, belowBoundary, SignalWeak)
			}

		case SignalThresholdCritical: // -30.0
			if atBoundary != SignalWeak {
				t.Errorf("ClassifySignal(%v) = %q, ingin %q (tepat di batas critical)",
					threshold, atBoundary, SignalWeak)
			}
			if belowBoundary != SignalCritical {
				t.Errorf("ClassifySignal(%v) = %q, ingin %q (sedikit di bawah batas critical)",
					justBelow, belowBoundary, SignalCritical)
			}
		}
	})
}

// =============================================================================
// Example-based tests untuk signal level classification yang diketahui
// =============================================================================

// TestSignalLevel_KnownValues memverifikasi klasifikasi signal untuk
// nilai-nilai dBm yang sudah diketahui secara eksplisit.
//
// **Validates: Requirements 2.8, 10.2**
func TestSignalLevel_KnownValues(t *testing.T) {
	cases := []struct {
		name     string
		rxPower  float64
		expected SignalLevel
	}{
		// Normal: >= -25 dBm
		{"signal kuat -8 dBm", -8.0, SignalNormal},
		{"signal normal -20 dBm", -20.0, SignalNormal},
		{"tepat di batas normal -25 dBm", -25.0, SignalNormal},
		{"signal positif +5 dBm", 5.0, SignalNormal},
		{"signal nol 0 dBm", 0.0, SignalNormal},

		// Warning: >= -27 dan < -25 dBm
		{"sedikit di bawah normal -25.001", -25.001, SignalWarning},
		{"warning -26 dBm", -26.0, SignalWarning},
		{"tepat di batas warning -27 dBm", -27.0, SignalWarning},

		// Weak: >= -30 dan < -27 dBm
		{"sedikit di bawah warning -27.001", -27.001, SignalWeak},
		{"weak -28 dBm", -28.0, SignalWeak},
		{"weak -29 dBm", -29.0, SignalWeak},
		{"tepat di batas weak -30 dBm", -30.0, SignalWeak},

		// Critical: < -30 dBm
		{"sedikit di bawah weak -30.001", -30.001, SignalCritical},
		{"critical -35 dBm", -35.0, SignalCritical},
		{"critical -50 dBm", -50.0, SignalCritical},
		{"LOS sangat lemah -80 dBm", -80.0, SignalCritical},

		// Nilai ekstrem
		{"infinity positif", math.Inf(1), SignalNormal},
		{"infinity negatif", math.Inf(-1), SignalCritical},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ClassifySignal(tc.rxPower)
			if result != tc.expected {
				t.Errorf(
					"ClassifySignal(%v) = %q, ingin %q",
					tc.rxPower, result, tc.expected,
				)
			}
		})
	}
}
