package domain

import "time"

// =============================================================================
// Request/Response DTO — payload untuk PPPoE user operations
// =============================================================================

// CreatePPPoEUserRequest adalah payload untuk POST /api/v1/mikrotik/routers/:id/pppoe/users.
// Digunakan untuk membuat PPPoE user secara manual dari API.
type CreatePPPoEUserRequest struct {
	CustomerID     string `json:"customer_id" validate:"required,uuid"`
	Username       string `json:"username" validate:"required,min=1,max=100"`
	Password       string `json:"password" validate:"required"`
	ProfileName    string `json:"profile_name" validate:"required,max=100"`
	RemoteAddress  string `json:"remote_address,omitempty" validate:"omitempty,max=45"`
	UseSimpleQueue bool   `json:"use_simple_queue"`
}

// UpdatePPPoEUserRequest adalah payload untuk PUT /api/v1/mikrotik/routers/:id/pppoe/users/:user_id.
// Field pointer hanya diterapkan jika dikirim oleh client.
type UpdatePPPoEUserRequest struct {
	Password       *string `json:"password,omitempty" validate:"omitempty,min=1"`
	ProfileName    *string `json:"profile_name,omitempty" validate:"omitempty,min=1,max=100"`
	RemoteAddress  *string `json:"remote_address,omitempty" validate:"omitempty,max=45"`
	Disabled       *bool   `json:"disabled,omitempty"`
	UseSimpleQueue *bool   `json:"use_simple_queue,omitempty"`
}

// PPPoEUserListParams berisi parameter untuk list PPPoE user dengan paginasi.
type PPPoEUserListParams struct {
	RouterID   string
	TenantID   string
	Page       int
	PageSize   int
	SyncStatus string
	Search     string
}

// PPPoEUserListResult berisi hasil list PPPoE user dengan metadata paginasi.
type PPPoEUserListResult struct {
	Data       []*PPPoEUser `json:"data"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}

// SyncResult berisi hasil sinkronisasi database↔router untuk satu router.
type SyncResult struct {
	RouterID       string    `json:"router_id"`
	TotalUsers     int       `json:"total_users"`
	SyncedCount    int       `json:"synced_count"`
	OrphanCount    int       `json:"orphan_count"`
	MissingCount   int       `json:"missing_count"`
	OutOfSyncCount int       `json:"out_of_sync_count"`
	ErrorCount     int       `json:"error_count"`
	SyncedAt       time.Time `json:"synced_at"`
}

// SyncStatusSummary berisi ringkasan sync status untuk dashboard per router.
type SyncStatusSummary struct {
	SyncedCount    int        `json:"synced_count"`
	OrphanCount    int        `json:"orphan_count"`
	MissingCount   int        `json:"missing_count"`
	OutOfSyncCount int        `json:"out_of_sync_count"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
}

// =============================================================================
// Incoming Event Payloads — payload event dari Billing API via Redis queue
// =============================================================================

// CustomerActivatedPayload adalah payload event customer.activated.
// Diterima saat pelanggan baru diaktivasi dan perlu dibuatkan PPPoE user di router.
type CustomerActivatedPayload struct {
	CustomerID          string `json:"customer_id"`
	TenantID            string `json:"tenant_id"`
	Name                string `json:"name"`
	PackageID           string `json:"package_id"`
	ConnectionMethod    string `json:"connection_method"`
	PPPoEUsername       string `json:"pppoe_username"`
	PPPoEPassword       string `json:"pppoe_password"`
	RouterID            string `json:"router_id"`
	MikrotikProfileName string `json:"mikrotik_profile_name,omitempty"`
	DownloadMbps        int    `json:"download_mbps,omitempty"`
	UploadMbps          int    `json:"upload_mbps,omitempty"`
	AddressPool         string `json:"address_pool,omitempty"`
}

// CustomerIsolirPayload adalah payload event customer.isolir.
// Diterima saat pelanggan perlu diisolir karena tunggakan pembayaran.
type CustomerIsolirPayload struct {
	CustomerID       string `json:"customer_id"`
	TenantID         string `json:"tenant_id"`
	CustomerName     string `json:"customer_name"`
	RouterID         string `json:"router_id"`
	PPPoEUsername    string `json:"pppoe_username"`
	ConnectionMethod string `json:"connection_method"`
	IsolirMethod     string `json:"isolir_method"`
	WalledGardenIP   string `json:"walled_garden_ip"`
	DNSServerIP      string `json:"dns_server_ip,omitempty"`
}

// CustomerUnIsolirPayload adalah payload event customer.un_isolir.
// Diterima saat pelanggan sudah membayar dan perlu dibuka isolirnya.
type CustomerUnIsolirPayload struct {
	CustomerID       string `json:"customer_id"`
	TenantID         string `json:"tenant_id"`
	CustomerName     string `json:"customer_name"`
	RouterID         string `json:"router_id"`
	PPPoEUsername    string `json:"pppoe_username"`
	ConnectionMethod string `json:"connection_method"`
}

// CustomerSuspendPayload adalah payload event customer.suspend.
// Diterima saat pelanggan di-suspend dan perlu dihapus dari router.
type CustomerSuspendPayload struct {
	CustomerID       string `json:"customer_id"`
	TenantID         string `json:"tenant_id"`
	CustomerName     string `json:"customer_name"`
	RouterID         string `json:"router_id"`
	PPPoEUsername    string `json:"pppoe_username"`
	ConnectionMethod string `json:"connection_method"`
}

// CustomerTerminatedPayload adalah payload event customer.terminated.
// Identik dengan CustomerSuspendPayload — keduanya menjalankan removal sequence yang sama.
type CustomerTerminatedPayload = CustomerSuspendPayload

// PackageChangedPayload adalah payload event package.changed.
// Diterima saat pelanggan upgrade/downgrade paket dan perlu update profile di router.
type PackageChangedPayload struct {
	CustomerID          string `json:"customer_id"`
	TenantID            string `json:"tenant_id"`
	OldPackageID        string `json:"old_package_id"`
	NewPackageID        string `json:"new_package_id"`
	ConnectionMethod    string `json:"connection_method"`
	RouterID            string `json:"router_id"`
	MikrotikProfileName string `json:"mikrotik_profile_name,omitempty"`
	DownloadMbps        int    `json:"download_mbps,omitempty"`
	UploadMbps          int    `json:"upload_mbps,omitempty"`
	AddressPool         string `json:"address_pool,omitempty"`
}

// =============================================================================
// Outgoing Event Payloads — payload event yang dipublikasikan ke Redis queue
// =============================================================================

// CommandResultPayload adalah payload event mikrotik.command_result.
// Dipublikasikan setelah setiap operasi PPPoE ke router selesai (sukses atau gagal).
type CommandResultPayload struct {
	CorrelationID string    `json:"correlation_id"`
	CustomerID    string    `json:"customer_id"`
	RouterID      string    `json:"router_id"`
	TenantID      string    `json:"tenant_id"`
	Operation     string    `json:"operation"`
	Status        string    `json:"status"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	ExecutedAt    time.Time `json:"executed_at"`
	DurationMs    int64     `json:"duration_ms"`
	RemoteAddress string    `json:"remote_address,omitempty"`
}

// SyncFailedPayload adalah payload event mikrotik.sync_failed.
// Dipublikasikan saat semua retry gagal atau sync job gagal untuk satu router.
type SyncFailedPayload struct {
	RouterID     string    `json:"router_id"`
	RouterName   string    `json:"router_name"`
	TenantID     string    `json:"tenant_id"`
	Operation    string    `json:"operation"`
	ErrorMessage string    `json:"error_message"`
	FailedAt     time.Time `json:"failed_at"`
}
