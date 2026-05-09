package domain

import "time"

// =============================================================================
// Laporan uptime - laporan uptime router dari network-service
// =============================================================================

// RouterUptimeItem berisi data uptime per router.
type RouterUptimeItem struct {
	RouterID         string  `json:"router_id"`
	RouterName       string  `json:"router_name"`
	UptimePercentage float64 `json:"uptime_percentage"`
	TotalDowntimeMin int     `json:"total_downtime_minutes"`
	RebootCount      int     `json:"reboot_count"`
	StatusLabel      string  `json:"status_label"` // Sangat baik, baik, cukup, buruk
}

// DowntimeEvent berisi satu event downtime.
type DowntimeEvent struct {
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	DurationMinutes int       `json:"duration_minutes"`
	Cause           string    `json:"cause,omitempty"`
}

// UptimeReport berisi laporan uptime router.
type UptimeReport struct {
	Routers          []RouterUptimeItem `json:"routers"`
	SLATarget        *float64           `json:"sla_target,omitempty"`
	RoutersBelowSLA  []RouterUptimeItem `json:"routers_below_sla,omitempty"`
	DowntimeTimeline []DowntimeEvent    `json:"downtime_timeline,omitempty"`
	ModuleInactive   bool               `json:"module_inactive"`
	StaleData        bool               `json:"stale_data"`
	LastUpdated      *time.Time         `json:"last_updated,omitempty"`
}

// =============================================================================
// Laporan traffic - laporan traffic jaringan dari network-service
// =============================================================================

// RouterTraffic berisi traffic per router.
type RouterTraffic struct {
	RouterID      string  `json:"router_id"`
	RouterName    string  `json:"router_name"`
	DownloadBytes int64   `json:"download_bytes"`
	UploadBytes   int64   `json:"upload_bytes"`
	Percentage    float64 `json:"percentage"`
}

// CustomerTraffic berisi traffic per pelanggan (top N).
type CustomerTraffic struct {
	CustomerID    string `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	PackageName   string `json:"package_name"`
	DownloadBytes int64  `json:"download_bytes"`
	UploadBytes   int64  `json:"upload_bytes"`
	OverUseFlag   bool   `json:"over_use_flag"`
}

// TrafficReport berisi laporan traffic jaringan.
type TrafficReport struct {
	TotalDownloadBytes int64             `json:"total_download_bytes"`
	TotalUploadBytes   int64             `json:"total_upload_bytes"`
	TotalTrafficBytes  int64             `json:"total_traffic_bytes"`
	PeakTrafficBps     int64             `json:"peak_traffic_bps"`
	PeakTrafficTime    *time.Time        `json:"peak_traffic_time,omitempty"`
	AverageTrafficBps  int64             `json:"average_traffic_bps"`
	ByRouter           []RouterTraffic   `json:"by_router"`
	TopCustomers       []CustomerTraffic `json:"top_customers"`
	ModuleInactive     bool              `json:"module_inactive"`
}

// =============================================================================
// Laporan kualitas sinyal - laporan kualitas signal OLT dari network-service
// =============================================================================

// DegradingONT berisi ONT dengan signal memburuk.
type DegradingONT struct {
	CustomerName     string  `json:"customer_name"`
	CustomerID       string  `json:"customer_id"`
	CurrentSignalDBm float64 `json:"current_signal_dbm"`
	SignalChangeDB   float64 `json:"signal_change_db"`
}

// AlarmTypeSummary berisi ringkasan alarm per tipe.
type AlarmTypeSummary struct {
	AlarmType          string  `json:"alarm_type"`
	Count              int     `json:"count"`
	AvgDurationMinutes int     `json:"avg_duration_minutes"`
	ResolvedPercentage float64 `json:"resolved_percentage"`
}

// SignalQualityReport berisi laporan kualitas signal OLT.
type SignalQualityReport struct {
	NormalCount      int                `json:"normal_count"`
	WarningCount     int                `json:"warning_count"`
	WeakCount        int                `json:"weak_count"`
	CriticalCount    int                `json:"critical_count"`
	TotalONTCount    int                `json:"total_ont_count"`
	AverageSignalDBm float64            `json:"average_signal_dbm"`
	DegradingONTs    []DegradingONT     `json:"degrading_onts"`
	AlarmSummary     []AlarmTypeSummary `json:"alarm_summary"`
	ModuleInactive   bool               `json:"module_inactive"`
}

// =============================================================================
// Laporan kapasitas - laporan kapasitas jaringan dari network-service
// =============================================================================

// RouterCapacity berisi kapasitas per router.
type RouterCapacity struct {
	RouterID          string  `json:"router_id"`
	RouterName        string  `json:"router_name"`
	CurrentCustomers  int     `json:"current_customers"`
	MaxCapacity       int     `json:"max_capacity"`
	UsagePercentage   float64 `json:"usage_percentage"`
	EstimatedFullDate *string `json:"estimated_full_date,omitempty"`
}

// ODPCapacity berisi kapasitas per ODP.
type ODPCapacity struct {
	ODPID           string  `json:"odp_id"`
	ODPName         string  `json:"odp_name"`
	UsedPorts       int     `json:"used_ports"`
	TotalPorts      int     `json:"total_ports"`
	UsagePercentage float64 `json:"usage_percentage"`
	StatusLabel     string  `json:"status_label"` // OK, Hampir Penuh, Penuh
}

// CapacityReport berisi laporan kapasitas jaringan.
type CapacityReport struct {
	RouterCapacity  []RouterCapacity `json:"router_capacity,omitempty"`
	ODPCapacity     []ODPCapacity    `json:"odp_capacity,omitempty"`
	Recommendations []string         `json:"recommendations"`
	ModuleInactive  map[string]bool  `json:"module_inactive,omitempty"`
}
