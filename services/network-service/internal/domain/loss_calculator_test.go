package domain

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

var validSplitterTypeGen = rapid.SampledFrom(ValidSplitterTypes)

// Jarak 0-100km, count 0-20, power -10 sampai 10 dBm, sensitivity -30 sampai -20 dBm.
func drawValidLossInput(t *rapid.T) LossCalculatorInput {
	return LossCalculatorInput{
		DistanceOLTtoODPKm: rapid.Float64Range(0, 100).Draw(t, "distOltOdp"),
		DistanceODPtoONTKm: rapid.Float64Range(0, 100).Draw(t, "distOdpOnt"),
		SplitterCount:      rapid.IntRange(0, 20).Draw(t, "splitterCount"),
		SplitterType:       validSplitterTypeGen.Draw(t, "splitterType"),
		ConnectorCount:     rapid.IntRange(0, 20).Draw(t, "connectorCount"),
		SpliceCount:        rapid.IntRange(0, 20).Draw(t, "spliceCount"),
		SFPTxPowerDBm:      rapid.Float64Range(-10, 10).Draw(t, "sfpTxPower"),
		ONTSensitivityDBm:  rapid.Float64Range(-30, -20).Draw(t, "ontSensitivity"),
	}
}

// =============================================================================
// =============================================================================

// TestLossCalculatorDecomposition memverifikasi bahwa total_loss_db selalu
// sama dengan penjumlahan komponen: fiber + splitter + connector + splice + safety_margin,
// dan total_loss_db selalu >= 3.0 dB (karena safety margin selalu disertakan).
//
// **Memvalidasi: Kebutuhan 11.2, 11.3, 11.4**
func TestLossCalculatorDecomposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := drawValidLossInput(t)
		result := CalculateLoss(input)

		// Hitung penjumlahan komponen secara manual
		expectedTotal := result.FiberLossDB +
			result.SplitterLossDB +
			result.ConnectorLossDB +
			result.SpliceLossDB +
			result.SafetyMarginDB

		// Verifikasi dekomposisi: total_loss == fiber + splitter + connector + splice + safety_margin
		if math.Abs(result.TotalLossDB-expectedTotal) > 1e-9 {
			t.Fatalf(
				"dekomposisi loss gagal: TotalLossDB=%.15f, sum komponen=%.15f, selisih=%.15e\n"+
					"  fiber=%.6f, splitter=%.6f, connector=%.6f, splice=%.6f, safety=%.6f",
				result.TotalLossDB, expectedTotal, math.Abs(result.TotalLossDB-expectedTotal),
				result.FiberLossDB, result.SplitterLossDB, result.ConnectorLossDB,
				result.SpliceLossDB, result.SafetyMarginDB,
			)
		}

		// Verifikasi total_loss >= 3.0 dB (safety margin selalu disertakan)
		if result.TotalLossDB < SafetyMargin {
			t.Fatalf(
				"total loss di bawah safety margin: TotalLossDB=%.6f, SafetyMargin=%.6f",
				result.TotalLossDB, SafetyMargin,
			)
		}
	})
}

// =============================================================================
// =============================================================================

// TestLossCalculatorSignalFormula memverifikasi bahwa estimated_signal_at_ont
// selalu sama dengan sfp_tx_power - (total_loss - safety_margin).
//
// **Memvalidasi: Kebutuhan 11.5**
func TestLossCalculatorSignalFormula(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := drawValidLossInput(t)
		result := CalculateLoss(input)

		// Formula: estimated_signal = sfp_tx_power - (total_loss - safety_margin)
		expectedSignal := input.SFPTxPowerDBm - (result.TotalLossDB - SafetyMargin)

		if math.Abs(result.EstimatedSignalAtONT-expectedSignal) > 1e-9 {
			t.Fatalf(
				"formula signal gagal: EstimatedSignalAtONT=%.15f, expected=%.15f, selisih=%.15e\n"+
					"  sfp_tx_power=%.6f, total_loss=%.6f, safety_margin=%.6f",
				result.EstimatedSignalAtONT, expectedSignal,
				math.Abs(result.EstimatedSignalAtONT-expectedSignal),
				input.SFPTxPowerDBm, result.TotalLossDB, SafetyMargin,
			)
		}
	})
}

// =============================================================================
// Unit Tes: Contoh kalkulasi spesifik
// =============================================================================

// distance_olt_odp=5km, distance_odp_ont=0.5km, 1 splitter 1:8, 4 konektor,
// 2 splice, SFP=5dBm, ONT sensitivity=-28dBm.
//
//   - fiber_loss = (5 + 0.5) * 0.35 = 1.925
//   - splitter_loss = 10.5 * 1 = 10.5
//   - connector_loss = 4 * 0.5 = 2.0
//   - splice_loss = 2 * 0.1 = 0.2
//   - safety_margin = 3.0
//   - total_loss = 1.925 + 10.5 + 2.0 + 0.2 + 3.0 = 17.625
//   - budget_available = 5 - (-28) = 33
//   - remaining_margin = 33 - 17.625 = 15.375
//   - estimated_signal = 5 - (17.625 - 3.0) = -9.625
//   - feasible = true (remaining_margin > 0)
func TestCalculateLossSpecificExample(t *testing.T) {
	input := LossCalculatorInput{
		DistanceOLTtoODPKm: 5.0,
		DistanceODPtoONTKm: 0.5,
		SplitterCount:      1,
		SplitterType:       "1:8",
		ConnectorCount:     4,
		SpliceCount:        2,
		SFPTxPowerDBm:      5.0,
		ONTSensitivityDBm:  -28.0,
	}

	result := CalculateLoss(input)

	// Toleransi floating point
	const tol = 1e-9

	// Verifikasi setiap komponen loss
	if math.Abs(result.FiberLossDB-1.925) > tol {
		t.Errorf("fiber_loss: got %.6f, want 1.925", result.FiberLossDB)
	}
	if math.Abs(result.SplitterLossDB-10.5) > tol {
		t.Errorf("splitter_loss: got %.6f, want 10.5", result.SplitterLossDB)
	}
	if math.Abs(result.ConnectorLossDB-2.0) > tol {
		t.Errorf("connector_loss: got %.6f, want 2.0", result.ConnectorLossDB)
	}
	if math.Abs(result.SpliceLossDB-0.2) > tol {
		t.Errorf("splice_loss: got %.6f, want 0.2", result.SpliceLossDB)
	}
	if math.Abs(result.SafetyMarginDB-3.0) > tol {
		t.Errorf("safety_margin: got %.6f, want 3.0", result.SafetyMarginDB)
	}

	// Verifikasi total loss
	if math.Abs(result.TotalLossDB-17.625) > tol {
		t.Errorf("total_loss: got %.6f, want 17.625", result.TotalLossDB)
	}

	// Verifikasi budget dan margin
	if math.Abs(result.BudgetAvailableDB-33.0) > tol {
		t.Errorf("budget_available: got %.6f, want 33.0", result.BudgetAvailableDB)
	}
	if math.Abs(result.RemainingMarginDB-15.375) > tol {
		t.Errorf("remaining_margin: got %.6f, want 15.375", result.RemainingMarginDB)
	}

	// Verifikasi estimated signal
	if math.Abs(result.EstimatedSignalAtONT-(-9.625)) > tol {
		t.Errorf("estimated_signal: got %.6f, want -9.625", result.EstimatedSignalAtONT)
	}

	// Verifikasi feasibility
	if !result.Feasible {
		t.Errorf("feasible: got false, want true (remaining_margin=%.6f > 0)", result.RemainingMarginDB)
	}
}
