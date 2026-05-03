package domain

// --- Signal Level ---

// SignalLevel mendefinisikan klasifikasi level signal ONT.
type SignalLevel string

const (
	// SignalNormal menandakan signal dalam kondisi baik (-8 sampai -25 dBm).
	SignalNormal SignalLevel = "normal"

	// SignalWarning menandakan signal mulai melemah (-25 sampai -27 dBm).
	SignalWarning SignalLevel = "warning"

	// SignalWeak menandakan signal lemah (-27 sampai -30 dBm).
	SignalWeak SignalLevel = "weak"

	// SignalCritical menandakan signal sangat lemah atau LOS (di bawah -30 dBm).
	SignalCritical SignalLevel = "critical"
)

// --- Threshold Constants ---

const (
	// SignalThresholdWarning adalah batas dBm untuk klasifikasi warning (-25 dBm).
	SignalThresholdWarning = -25.0

	// SignalThresholdWeak adalah batas dBm untuk klasifikasi weak (-27 dBm).
	SignalThresholdWeak = -27.0

	// SignalThresholdCritical adalah batas dBm untuk klasifikasi critical (-30 dBm).
	SignalThresholdCritical = -30.0
)

// ClassifySignal mengklasifikasikan signal level berdasarkan rx_power dalam dBm.
// Rentang klasifikasi:
//   - Normal:   >= -25 dBm
//   - Warning:  >= -27 dBm dan < -25 dBm
//   - Weak:     >= -30 dBm dan < -27 dBm
//   - Critical: < -30 dBm (termasuk LOS)
func ClassifySignal(rxPowerDBm float64) SignalLevel {
	switch {
	case rxPowerDBm >= SignalThresholdWarning:
		return SignalNormal
	case rxPowerDBm >= SignalThresholdWeak:
		return SignalWarning
	case rxPowerDBm >= SignalThresholdCritical:
		return SignalWeak
	default:
		return SignalCritical
	}
}
