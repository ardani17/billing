package domain

import "time"

// --- Alarm Type Constants ---

const (
	// AlarmTypeONTLOS menandakan alarm Loss of Signal pada ONT.
	AlarmTypeONTLOS = "ont_los"

	// AlarmTypeONTDyingGasp menandakan alarm dying gasp (kegagalan daya) pada ONT.
	AlarmTypeONTDyingGasp = "ont_dying_gasp"

	// AlarmTypePONPortDown menandakan alarm PON port down.
	AlarmTypePONPortDown = "pon_port_down"

	// AlarmTypePowerFailure menandakan alarm kegagalan daya pada OLT.
	AlarmTypePowerFailure = "power_failure"

	// AlarmTypeHighTemperature menandakan alarm suhu tinggi pada OLT.
	AlarmTypeHighTemperature = "high_temperature"

	// AlarmTypeONTSignalDegraded menandakan alarm degradasi signal pada ONT.
	AlarmTypeONTSignalDegraded = "ont_signal_degraded"
)

// --- Severity Constants ---

const (
	// SeverityCritical menandakan alarm dengan tingkat keparahan kritis.
	SeverityCritical = "critical"

	// SeverityMajor menandakan alarm dengan tingkat keparahan mayor.
	SeverityMajor = "major"

	// SeverityMinor menandakan alarm dengan tingkat keparahan minor.
	SeverityMinor = "minor"

	// SeverityWarning menandakan alarm dengan tingkat keparahan peringatan.
	SeverityWarning = "warning"

	// SeverityClear menandakan alarm telah dibersihkan/clear.
	SeverityClear = "clear"
)

// --- Alarm Source Constants ---

const (
	// AlarmSourceTrap menandakan alarm diterima via SNMP trap (push).
	AlarmSourceTrap = "trap"

	// AlarmSourcePolling menandakan alarm diterima via SNMP polling (pull).
	AlarmSourcePolling = "polling"
)

// --- Alarm Status Constants ---

const (
	// AlarmStatusActive menandakan alarm masih aktif.
	AlarmStatusActive = "active"

	// AlarmStatusCleared menandakan alarm sudah dibersihkan/clear.
	AlarmStatusCleared = "cleared"
)

// --- Alarm Structs ---

// OLTAlarm berisi satu alarm dari OLT (dari adapter).
// Struct ini digunakan sebagai output dari adapter saat polling atau parsing trap.
type OLTAlarm struct {
	AlarmType    string `json:"alarm_type"`              // ont_los, ont_dying_gasp, pon_port_down, dll.
	Severity     string `json:"severity"`                // critical, major, minor, warning, clear
	PONPortIndex *int   `json:"pon_port_index,omitempty"` // indeks PON port terkait (opsional)
	ONTIndex     *int   `json:"ont_index,omitempty"`      // indeks ONT terkait (opsional)
	Message      string `json:"message"`                 // pesan deskriptif alarm
	Source       string `json:"source"`                  // trap / polling
}

// OLTAlarmRecord berisi alarm yang disimpan di database.
// Struct ini merepresentasikan baris pada tabel olt_alarms.
type OLTAlarmRecord struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	OLTID        string     `json:"olt_id"`
	PONPortIndex *int       `json:"pon_port_index,omitempty"`
	ONTIndex     *int       `json:"ont_index,omitempty"`
	AlarmType    string     `json:"alarm_type"`
	Severity     string     `json:"severity"`
	Message      string     `json:"message"`
	Source       string     `json:"source"`
	Status       string     `json:"status"`                // active / cleared
	ClearedAt    *time.Time `json:"cleared_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
