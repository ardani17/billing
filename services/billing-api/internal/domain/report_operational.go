package domain

import "time"

// =============================================================================
// Laporan aktivitas - laporan aktivitas admin/user
// =============================================================================

// UserActivity berisi aktivitas per user.
type UserActivity struct {
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name"`
	Role         string    `json:"role"`
	LoginDays    int       `json:"login_days"`
	ActionCount  int       `json:"action_count"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// ActionSummary berisi ringkasan aksi.
type ActionSummary struct {
	ActionType string  `json:"action_type"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ActivityReport berisi laporan aktivitas admin.
type ActivityReport struct {
	PerUser    []UserActivity  `json:"per_user"`
	TopActions []ActionSummary `json:"top_actions"`
}

// =============================================================================
// Laporan notifikasi - laporan statistik notifikasi dari network-service
// =============================================================================

// ChannelStats berisi statistik per channel notifikasi.
type ChannelStats struct {
	Channel        string  `json:"channel"`
	SentCount      int     `json:"sent_count"`
	DeliveredCount int     `json:"delivered_count"`
	FailedCount    int     `json:"failed_count"`
	SuccessRate    float64 `json:"success_rate"`
	Cost           int64   `json:"cost"`
}

// TemplateStats berisi statistik per template notifikasi.
type TemplateStats struct {
	TemplateName string `json:"template_name"`
	SentCount    int    `json:"sent_count"`
}

// NotificationReport berisi laporan statistik notifikasi.
type NotificationReport struct {
	TotalSent      int             `json:"total_sent"`
	TotalDelivered int             `json:"total_delivered"`
	TotalFailed    int             `json:"total_failed"`
	SuccessRate    float64         `json:"success_rate"`
	TotalCost      int64           `json:"total_cost"`
	PerChannel     []ChannelStats  `json:"per_channel"`
	PerTemplate    []TemplateStats `json:"per_template"`
	ModuleInactive bool            `json:"module_inactive"`
}

// =============================================================================
// Laporan sinkronisasi - laporan status sync MikroTik dan OLT dari network-service
// =============================================================================

// RouterSyncStatus berisi status sync per router.
type RouterSyncStatus struct {
	RouterID         string `json:"router_id"`
	RouterName       string `json:"router_name"`
	SyncOKCount      int    `json:"sync_ok_count"`
	SyncFailedCount  int    `json:"sync_failed_count"`
	OrphanUserCount  int    `json:"orphan_user_count"`
	PendingSyncCount int    `json:"pending_sync_count"`
}

// OLTSyncStatus berisi status sync per OLT.
type OLTSyncStatus struct {
	OLTID             string `json:"olt_id"`
	OLTName           string `json:"olt_name"`
	SyncOKCount       int    `json:"sync_ok_count"`
	SyncFailedCount   int    `json:"sync_failed_count"`
	UnmanagedONTCount int    `json:"unmanaged_ont_count"`
}

// SyncReport berisi laporan status sync.
type SyncReport struct {
	MikrotikSync    []RouterSyncStatus `json:"mikrotik_sync,omitempty"`
	OLTSync         []OLTSyncStatus    `json:"olt_sync,omitempty"`
	SyncSuccessRate float64            `json:"sync_success_rate"`
	ModuleInactive  map[string]bool    `json:"module_inactive,omitempty"`
}
