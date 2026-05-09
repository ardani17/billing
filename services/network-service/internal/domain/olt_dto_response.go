package domain

import "time"

// =============================================================================
// OLT Respons DTOs - format respons untuk operasi OLT
// =============================================================================

// OLTResponse adalah respons untuk operasi buat/perbarui/list OLT.
// Tidak menyertakan kredensial (SNMP community, password) untuk keamanan.
type OLTResponse struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	Host                   string     `json:"host"`
	Brand                  OLTBrand   `json:"brand,omitempty"`
	Model                  string     `json:"model,omitempty"`
	FirmwareVersion        string     `json:"firmware_version,omitempty"`
	PONPortCount           int        `json:"pon_port_count"`
	TotalONTCount          int        `json:"total_ont_count"`
	Status                 OLTStatus  `json:"status"`
	HealthCheckIntervalSec int        `json:"health_check_interval_sec"`
	LastOnlineAt           *time.Time `json:"last_online_at,omitempty"`
	Notes                  string     `json:"notes,omitempty"`
	Warning                string     `json:"warning,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// OLTDetailResponse adalah respons untuk GET /api/v1/olt/devices/:id.
// Menyertakan informasi tambahan: versi SNMP, protokol CLI, dan jumlah alarm aktif.
type OLTDetailResponse struct {
	OLTResponse
	SNMPVersion      SNMPVersion `json:"snmp_version"`
	CLIProtocol      CLIProtocol `json:"cli_protocol"`
	CLIPort          int         `json:"cli_port"`
	ActiveAlarmCount int64       `json:"active_alarm_count"`
}

// OLTListResult berisi hasil list OLT dengan metadata paginasi.
type OLTListResult struct {
	Data       []*OLTResponse `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// =============================================================================
// OLT Status Summary - ringkasan status untuk dashboard widget
// =============================================================================

// OLTStatusSummary berisi ringkasan status semua OLT tenant untuk dashboard.
// Dihitung dari data di database tanpa koneksi langsung ke perangkat OLT.
type OLTStatusSummary struct {
	TotalOLTs        int64 `json:"total_olts"`
	OnlineCount      int64 `json:"online_count"`
	OfflineCount     int64 `json:"offline_count"`
	MaintenanceCount int64 `json:"maintenance_count"`
	ActiveAlarmCount int64 `json:"active_alarm_count"`
}

// =============================================================================
// CLI Tes Result - hasil test koneksi CLI (SSH/Telnet)
// =============================================================================

// CLITestResult berisi hasil test koneksi CLI ke OLT.
// Success=true jika koneksi berhasil, Banner berisi prompt/banner dari OLT.
type CLITestResult struct {
	Success bool   `json:"success"`
	Banner  string `json:"banner,omitempty"`
	Error   string `json:"error,omitempty"`
}

// =============================================================================
// Capacity Planning - data kapasitas OLT dan per PON port
// =============================================================================

// OLTCapacity berisi data capacity planning untuk satu OLT.
// Digunakan untuk endpoint GET /api/v1/olt/devices/:id/capacity.
type OLTCapacity struct {
	TotalPONPorts            int            `json:"total_pon_ports"`
	ActivePONPorts           int            `json:"active_pon_ports"`
	TotalONTSlots            int            `json:"total_ont_slots"`
	UsedONTSlots             int            `json:"used_ont_slots"`
	AvailableONTSlots        int            `json:"available_ont_slots"`
	UtilizationPercent       float64        `json:"utilization_percent"`
	GrowthRatePerMonth       float64        `json:"growth_rate_per_month"`
	EstimatedMonthsRemaining float64        `json:"estimated_months_remaining"`
	PortBreakdown            []PortCapacity `json:"port_breakdown"`
}

// PortCapacity berisi kapasitas per PON port.
// Warning diisi jika utilisasi port melebihi 90%.
type PortCapacity struct {
	PortIndex          int     `json:"port_index"`
	ONTCount           int     `json:"ont_count"`
	MaxONTPerPort      int     `json:"max_ont_per_port"` // bawaan 64 untuk GPON
	UtilizationPercent float64 `json:"utilization_percent"`
	Warning            string  `json:"warning,omitempty"` // diisi jika utilisasi > 90%
}
