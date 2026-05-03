package domain

import "fmt"

// =============================================================================
// Konstanta Loss Calculator — parameter standar untuk kalkulasi optical loss
// =============================================================================

const (
	// FiberLossPerKm adalah loss fiber optik per kilometer dalam dB.
	FiberLossPerKm = 0.35

	// ConnectorLossEach adalah loss per konektor dalam dB.
	ConnectorLossEach = 0.5

	// SpliceLossEach adalah loss per splice (sambungan) dalam dB.
	SpliceLossEach = 0.1

	// SafetyMargin adalah margin keamanan dalam dB yang selalu ditambahkan.
	SafetyMargin = 3.0
)

// SplitterLoss berisi loss dB berdasarkan tipe splitter.
// Tipe yang didukung: "1:4", "1:8", "1:16", "1:32".
var SplitterLoss = map[string]float64{
	"1:4":  7.0,
	"1:8":  10.5,
	"1:16": 13.5,
	"1:32": 17.0,
}

// ValidSplitterTypes berisi daftar tipe splitter yang valid untuk validasi.
var ValidSplitterTypes = []string{"1:4", "1:8", "1:16", "1:32"}

// =============================================================================
// LossCalculatorInput — parameter input untuk kalkulasi optical loss budget
// =============================================================================

// LossCalculatorInput berisi parameter untuk kalkulasi optical loss budget.
// Semua jarak dalam kilometer, power dalam dBm, count dalam integer.
type LossCalculatorInput struct {
	DistanceOLTtoODPKm float64 `json:"distance_olt_to_odp_km"`
	DistanceODPtoONTKm float64 `json:"distance_odp_to_ont_km"`
	SplitterCount      int     `json:"splitter_count"`
	SplitterType       string  `json:"splitter_type"`
	ConnectorCount     int     `json:"connector_count"`
	SpliceCount        int     `json:"splice_count"`
	SFPTxPowerDBm      float64 `json:"sfp_tx_power_dbm"`
	ONTSensitivityDBm  float64 `json:"ont_sensitivity_dbm"`
}

// =============================================================================
// LossCalculatorResult — hasil kalkulasi optical loss budget
// =============================================================================

// LossCalculatorResult berisi hasil kalkulasi optical loss budget.
// Semua nilai loss dalam dB, signal dalam dBm.
type LossCalculatorResult struct {
	TotalLossDB          float64 `json:"total_loss_db"`
	BudgetAvailableDB    float64 `json:"budget_available_db"`
	RemainingMarginDB    float64 `json:"remaining_margin_db"`
	EstimatedSignalAtONT float64 `json:"estimated_signal_at_ont_dbm"`
	Feasible             bool    `json:"feasible"`
	FiberLossDB          float64 `json:"fiber_loss_db"`
	SplitterLossDB       float64 `json:"splitter_loss_db"`
	ConnectorLossDB      float64 `json:"connector_loss_db"`
	SpliceLossDB         float64 `json:"splice_loss_db"`
	SafetyMarginDB       float64 `json:"safety_margin_db"`
}

// =============================================================================
// ValidateLossInput — validasi input loss calculator
// =============================================================================

// ValidateLossInput memvalidasi input loss calculator.
// Mengembalikan error jika splitter_type tidak valid atau jarak/count negatif.
func ValidateLossInput(input LossCalculatorInput) error {
	if _, ok := SplitterLoss[input.SplitterType]; !ok {
		return fmt.Errorf("%w: tipe splitter '%s' tidak valid, gunakan 1:4, 1:8, 1:16, atau 1:32",
			ErrInvalidLossInput, input.SplitterType)
	}
	if input.DistanceOLTtoODPKm < 0 {
		return fmt.Errorf("%w: jarak OLT ke ODP tidak boleh negatif (%.2f)",
			ErrInvalidLossInput, input.DistanceOLTtoODPKm)
	}
	if input.DistanceODPtoONTKm < 0 {
		return fmt.Errorf("%w: jarak ODP ke ONT tidak boleh negatif (%.2f)",
			ErrInvalidLossInput, input.DistanceODPtoONTKm)
	}
	if input.SplitterCount < 0 {
		return fmt.Errorf("%w: jumlah splitter tidak boleh negatif (%d)",
			ErrInvalidLossInput, input.SplitterCount)
	}
	if input.ConnectorCount < 0 {
		return fmt.Errorf("%w: jumlah konektor tidak boleh negatif (%d)",
			ErrInvalidLossInput, input.ConnectorCount)
	}
	if input.SpliceCount < 0 {
		return fmt.Errorf("%w: jumlah splice tidak boleh negatif (%d)",
			ErrInvalidLossInput, input.SpliceCount)
	}
	return nil
}

// =============================================================================
// CalculateLoss — menghitung optical loss budget (pure function)
// =============================================================================

// CalculateLoss menghitung optical loss budget berdasarkan parameter input.
//
// Formula:
//   - fiber_loss = (distance_olt_to_odp + distance_odp_to_ont) * FiberLossPerKm
//   - splitter_loss = SplitterLoss[splitter_type] * splitter_count
//   - connector_loss = connector_count * ConnectorLossEach
//   - splice_loss = splice_count * SpliceLossEach
//   - total_loss = fiber_loss + splitter_loss + connector_loss + splice_loss + SafetyMargin
//   - budget_available = sfp_tx_power - ont_sensitivity
//   - remaining_margin = budget_available - total_loss
//   - estimated_signal = sfp_tx_power - (total_loss - SafetyMargin)
//   - feasible = remaining_margin > 0
//
// Fungsi pure — tidak ada side effect, cocok untuk property-based testing.
func CalculateLoss(input LossCalculatorInput) LossCalculatorResult {
	// Hitung loss per komponen
	totalDistanceKm := input.DistanceOLTtoODPKm + input.DistanceODPtoONTKm
	fiberLoss := totalDistanceKm * FiberLossPerKm

	splitterLossPerUnit := SplitterLoss[input.SplitterType]
	splitterLoss := splitterLossPerUnit * float64(input.SplitterCount)

	connectorLoss := float64(input.ConnectorCount) * ConnectorLossEach
	spliceLoss := float64(input.SpliceCount) * SpliceLossEach

	// Hitung total loss (termasuk safety margin)
	totalLoss := fiberLoss + splitterLoss + connectorLoss + spliceLoss + SafetyMargin

	// Hitung budget dan margin
	budgetAvailable := input.SFPTxPowerDBm - input.ONTSensitivityDBm
	remainingMargin := budgetAvailable - totalLoss

	// Hitung estimasi signal di ONT (tanpa safety margin)
	estimatedSignal := input.SFPTxPowerDBm - (totalLoss - SafetyMargin)

	return LossCalculatorResult{
		TotalLossDB:          totalLoss,
		BudgetAvailableDB:    budgetAvailable,
		RemainingMarginDB:    remainingMargin,
		EstimatedSignalAtONT: estimatedSignal,
		Feasible:             remainingMargin > 0,
		FiberLossDB:          fiberLoss,
		SplitterLossDB:       splitterLoss,
		ConnectorLossDB:      connectorLoss,
		SpliceLossDB:         spliceLoss,
		SafetyMarginDB:       SafetyMargin,
	}
}
