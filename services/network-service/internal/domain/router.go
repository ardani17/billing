package domain

import "time"

// --- Router Entity ---

// Router merepresentasikan perangkat MikroTik yang terdaftar per tenant.
// Setiap tenant memiliki daftar router sendiri yang diisolasi via RLS.
type Router struct {
	ID                     string       `json:"id"`
	TenantID               string       `json:"tenant_id"`
	Name                   string       `json:"name"`
	Host                   string       `json:"host"`
	Port                   int          `json:"port"`
	Username               string       `json:"username"`
	PasswordEncrypted      string       `json:"-"`
	UseSSL                 bool         `json:"use_ssl"`
	ServiceTypes           []string     `json:"service_types"`
	RouterOSVersion        string       `json:"router_os_version,omitempty"`
	BoardName              string       `json:"board_name,omitempty"`
	CPUCount               int          `json:"cpu_count,omitempty"`
	TotalRAMMB             int          `json:"total_ram_mb,omitempty"`
	Identity               string       `json:"identity,omitempty"`
	Status                 RouterStatus `json:"status"`
	HealthCheckIntervalSec int          `json:"health_check_interval_sec"`
	LastOnlineAt           *time.Time   `json:"last_online_at,omitempty"`
	LastCheckedAt          *time.Time   `json:"last_checked_at,omitempty"`
	LastUptimeSec          *int64       `json:"last_uptime_sec,omitempty"`
	FailureCount           int          `json:"failure_count"`
	Notes                  string       `json:"notes,omitempty"`
	DeletedAt              *time.Time   `json:"deleted_at,omitempty"`
	CreatedAt              time.Time    `json:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at"`
}

// --- Connection Config ---

// ConnectionConfig berisi konfigurasi koneksi ke router MikroTik.
type ConnectionConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	UseSSL         bool
	ConnectTimeout time.Duration
	CommandTimeout time.Duration
}

// --- System Resource ---

// SystemResource berisi informasi sistem yang diambil dari router.
// Digunakan untuk auto-detect info saat create/test connection.
type SystemResource struct {
	Version      string `json:"version"`
	BoardName    string `json:"board_name"`
	CPUCount     int    `json:"cpu_count"`
	CPULoad      int    `json:"cpu_load"`
	TotalRAM     int64  `json:"total_ram"`
	FreeRAM      int64  `json:"free_ram"`
	Uptime       int64  `json:"uptime"`
	Architecture string `json:"architecture"`
	Identity     string `json:"identity"`
}

// --- Router Metrics ---

// RouterMetrics berisi metrik yang dikumpulkan dari router saat health check.
type RouterMetrics struct {
	CPULoad         int   `json:"cpu_load"`
	RAMUsagePercent int   `json:"ram_usage_percent"`
	UptimeSeconds   int64 `json:"uptime_seconds"`
	ActiveSessions  int   `json:"active_sessions"`
}

// RouterMetricsPoint berisi metrik dengan timestamp untuk time-series storage.
type RouterMetricsPoint struct {
	Timestamp time.Time     `json:"timestamp"`
	Metrics   RouterMetrics `json:"metrics"`
}

// --- Pool Stats ---

// PoolStats berisi statistik connection pool untuk satu router.
type PoolStats struct {
	Active int `json:"active"`
	Idle   int `json:"idle"`
	Total  int `json:"total"`
}

// --- Status Summary ---

// StatusSummary berisi ringkasan status router untuk dashboard tenant.
// Invariant: TotalRouters == OnlineCount + OfflineCount + MaintenanceCount.
type StatusSummary struct {
	TotalRouters     int64 `json:"total_routers"`
	OnlineCount      int64 `json:"online_count"`
	OfflineCount     int64 `json:"offline_count"`
	MaintenanceCount int64 `json:"maintenance_count"`
}

// --- Health Check Update ---

// HealthCheckUpdate berisi field yang diupdate saat health check selesai.
type HealthCheckUpdate struct {
	LastCheckedAt *time.Time
	LastOnlineAt  *time.Time
	LastUptimeSec *int64
	FailureCount  int
	Status        *RouterStatus
}
