package domain

import "time"

// --- Event Type Constants ---

const (
	// EventOLTDeviceOffline adalah tipe event saat OLT berubah status menjadi offline.
	EventOLTDeviceOffline = "olt.device_offline"

	// EventOLTDeviceOnline adalah tipe event saat OLT berubah status menjadi online.
	EventOLTDeviceOnline = "olt.device_online"

	// EventOLTAlarm adalah tipe event saat alarm diterima dari OLT.
	EventOLTAlarm = "olt.alarm"
)

// --- Payload event ---

// OLTDeviceOfflinePayload adalah payload event olt.device_offline.
// Dipublikasikan saat OLT terdeteksi offline setelah 3x kegagalan health cek berturut-turut.
type OLTDeviceOfflinePayload struct {
	CorrelationID string    `json:"correlation_id"`
	OLTID         string    `json:"olt_id"`
	OLTName       string    `json:"olt_name"`
	TenantID      string    `json:"tenant_id"`
	Brand         string    `json:"brand"`
	LastOnlineAt  time.Time `json:"last_online_at"`
}

// OLTDeviceOnlinePayload adalah payload event olt.device_online.
// Dipublikasikan saat OLT yang sebelumnya offline kembali merespons health cek.
type OLTDeviceOnlinePayload struct {
	CorrelationID    string        `json:"correlation_id"`
	OLTID            string        `json:"olt_id"`
	OLTName          string        `json:"olt_name"`
	TenantID         string        `json:"tenant_id"`
	Brand            string        `json:"brand"`
	DowntimeDuration time.Duration `json:"downtime_duration"`
}

// OLTAlarmPayload adalah payload event olt.alarm.
// Dipublikasikan saat alarm diterima dari OLT, baik via SNMP trap maupun polling.
type OLTAlarmPayload struct {
	CorrelationID string `json:"correlation_id"`
	OLTID         string `json:"olt_id"`
	OLTName       string `json:"olt_name"`
	TenantID      string `json:"tenant_id"`
	AlarmType     string `json:"alarm_type"`
	Severity      string `json:"severity"`
	PONPortIndex  *int   `json:"pon_port_index,omitempty"`
	ONTIndex      *int   `json:"ont_index,omitempty"`
	Message       string `json:"message"`
}

// --- Alarm List Parameters dan Result ---

// AlarmListParams berisi parameter untuk list alarm dengan paginasi dan filter.
type AlarmListParams struct {
	Page     int
	PageSize int
	Severity string // filter berdasarkan severity (opsional)
	Status   string // filter berdasarkan status: active, cleared (opsional)
}

// AlarmListResult berisi hasil list alarm dengan metadata paginasi.
type AlarmListResult struct {
	Data       []*OLTAlarmRecord `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}
