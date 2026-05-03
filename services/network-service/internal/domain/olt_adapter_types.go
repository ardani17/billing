package domain

import "time"

// --- Adapter Response Types ---
// Tipe-tipe berikut digunakan sebagai response dari OLTAdapter
// dan connector (SNMP/CLI) untuk komunikasi antar layer.

// OLTSystemInfo berisi informasi sistem OLT dari auto-detect.
type OLTSystemInfo struct {
	Brand           OLTBrand `json:"brand"`
	Model           string   `json:"model"`
	FirmwareVersion string   `json:"firmware_version"`
	Uptime          int64    `json:"uptime_seconds"`
	PONPortCount    int      `json:"pon_port_count"`
	TotalONTCount   int      `json:"total_ont_count"`
	SysDescr        string   `json:"sys_descr"`
	SysName         string   `json:"sys_name"`
}

// PONPortStatus berisi status satu PON port.
type PONPortStatus struct {
	PortIndex      int    `json:"port_index"`
	AdminStatus    string `json:"admin_status"`    // up / down
	OperStatus     string `json:"oper_status"`     // up / down
	ONTCount       int    `json:"ont_count"`
	ONTOnlineCount int    `json:"ont_online_count"`
	Description    string `json:"description,omitempty"`
}

// ONTPortStatus berisi status satu ONT pada PON port.
// Digunakan sebagai response dari adapter GetONTList.
type ONTPortStatus struct {
	ONTIndex     int         `json:"ont_index"`
	SerialNumber string      `json:"serial_number"`
	Name         string      `json:"name,omitempty"`
	Status       string      `json:"status"`         // online / offline
	RxSignalDBm  float64     `json:"rx_signal_dbm"`
	SignalLevel  SignalLevel  `json:"signal_level"`
	Distance     int         `json:"distance_meters"`
	Uptime       int64       `json:"uptime_seconds"`
}

// ONTSignalInfo berisi informasi signal detail ONT.
type ONTSignalInfo struct {
	ONTIndex    int         `json:"ont_index"`
	RxPowerDBm  float64     `json:"rx_power_dbm"`
	TxPowerDBm  float64     `json:"tx_power_dbm,omitempty"`
	SignalLevel SignalLevel  `json:"signal_level"`
	Distance    int         `json:"distance_meters"`
}

// SFPInfo berisi informasi SFP module pada satu PON port.
type SFPInfo struct {
	PortIndex   int     `json:"port_index"`
	TxPowerDBm  float64 `json:"tx_power_dbm"`
	RxPowerDBm  float64 `json:"rx_power_dbm"`
	Temperature float64 `json:"temperature_celsius"`
	SFPType     string  `json:"sfp_type"` // contoh: "GPON C+", "GPON B+"
	Status      string  `json:"status"`   // normal, warm, degraded, failed, empty
}

// PONTrafficStats berisi statistik traffic satu PON port.
type PONTrafficStats struct {
	PortIndex int   `json:"port_index"`
	RxBytes   int64 `json:"rx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxBytes   int64 `json:"tx_bytes"`
	TxPackets int64 `json:"tx_packets"`
}

// --- Time-Series Data Points ---

// ONTSignalPoint berisi data point signal untuk time-series storage.
type ONTSignalPoint struct {
	Timestamp   time.Time   `json:"timestamp"`
	RxPowerDBm  float64     `json:"rx_power_dbm"`
	SignalLevel SignalLevel  `json:"signal_level"`
}

// PONTrafficPoint berisi data point traffic untuk time-series storage.
type PONTrafficPoint struct {
	Timestamp time.Time `json:"timestamp"`
	RxBytes   int64     `json:"rx_bytes"`
	RxPackets int64     `json:"rx_packets"`
	TxBytes   int64     `json:"tx_bytes"`
	TxPackets int64     `json:"tx_packets"`
}

// --- Konfigurasi Koneksi ---

// SNMPConfig berisi konfigurasi koneksi SNMP ke OLT.
type SNMPConfig struct {
	Host         string        // alamat IP atau hostname OLT
	Port         int           // default 161
	Version      SNMPVersion   // v2c atau v3
	Community    string        // untuk v2c
	Username     string        // untuk v3
	AuthProtocol string        // MD5 atau SHA (v3)
	AuthPassword string        // untuk v3
	PrivProtocol string        // DES atau AES (v3)
	PrivPassword string        // untuk v3
	Timeout      time.Duration // default 5s connect, 10s request
}

// CLIConfig berisi konfigurasi koneksi CLI (SSH/Telnet) ke OLT.
type CLIConfig struct {
	Host           string        // alamat IP atau hostname OLT
	Port           int           // default 22 (SSH) atau 23 (Telnet)
	Protocol       CLIProtocol   // ssh atau telnet
	Username       string
	Password       string
	EnablePassword string        // opsional, untuk privileged mode
	ConnTimeout    time.Duration // default 10s
	CmdTimeout     time.Duration // default 30s
}

// --- SNMP Result Types ---

// SNMPValueType mendefinisikan tipe nilai SNMP.
type SNMPValueType string

const (
	// SNMPValueInteger untuk tipe integer SNMP.
	SNMPValueInteger SNMPValueType = "integer"

	// SNMPValueString untuk tipe string SNMP.
	SNMPValueString SNMPValueType = "string"

	// SNMPValueCounter32 untuk tipe counter 32-bit SNMP.
	SNMPValueCounter32 SNMPValueType = "counter32"

	// SNMPValueCounter64 untuk tipe counter 64-bit SNMP.
	SNMPValueCounter64 SNMPValueType = "counter64"

	// SNMPValueGauge32 untuk tipe gauge 32-bit SNMP.
	SNMPValueGauge32 SNMPValueType = "gauge32"

	// SNMPValueTimeTicks untuk tipe timeticks SNMP.
	SNMPValueTimeTicks SNMPValueType = "timeticks"
)

// SNMPResult berisi hasil satu SNMP operation.
type SNMPResult struct {
	OID   string        // OID yang di-query
	Type  SNMPValueType // tipe nilai hasil
	Value interface{}   // nilai hasil SNMP
}
