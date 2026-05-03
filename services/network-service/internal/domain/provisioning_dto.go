package domain

import "time"

// =============================================================================
// Bulk Provisioning DTOs — preview dan result untuk bulk provisioning via CSV
// =============================================================================

// BulkPreview berisi hasil validasi CSV sebelum eksekusi.
// Dikembalikan oleh POST /api/v1/olt/provisioning/bulk.
type BulkPreview struct {
	BulkID     string           `json:"bulk_id"`
	OLTID      string           `json:"olt_id"`
	TotalRows  int              `json:"total_rows"`
	ValidCount int              `json:"valid_count"`
	ErrorCount int              `json:"error_count"`
	Rows       []BulkRowPreview `json:"rows"`
}

// BulkRowPreview berisi status validasi per baris CSV.
// Valid=true jika baris lolos semua validasi, ErrorMessage berisi alasan jika gagal.
type BulkRowPreview struct {
	RowNumber    int    `json:"row_number"`
	SerialNumber string `json:"serial_number"`
	CustomerID   string `json:"customer_id"`
	PONPort      int    `json:"pon_port"`
	VLAN         string `json:"vlan"`
	ODP          string `json:"odp"`
	Description  string `json:"description"`
	Valid        bool   `json:"valid"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// BulkResult berisi hasil eksekusi bulk provisioning.
// Dikembalikan oleh POST /api/v1/olt/provisioning/bulk/execute.
type BulkResult struct {
	BulkID       string          `json:"bulk_id"`
	Total        int             `json:"total"`
	SuccessCount int             `json:"success_count"`
	FailureCount int             `json:"failure_count"`
	Rows         []BulkRowResult `json:"rows"`
}

// BulkRowResult berisi hasil provisioning per baris.
// Success=true jika provisioning berhasil, ONTID berisi ID ONT yang dibuat.
type BulkRowResult struct {
	RowNumber    int    `json:"row_number"`
	SerialNumber string `json:"serial_number"`
	Success      bool   `json:"success"`
	ONTID        string `json:"ont_id,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// =============================================================================
// Provisioning Settings Request DTO
// =============================================================================

// UpdateSettingsRequest adalah payload untuk PUT /api/v1/olt/provisioning/settings.
// Semua field menggunakan pointer untuk membedakan antara tidak dikirim dan zero value.
type UpdateSettingsRequest struct {
	AutoProvisioningEnabled  *bool  `json:"auto_provisioning_enabled,omitempty"`
	AutoPortMigrationEnabled *bool  `json:"auto_port_migration_enabled,omitempty"`
	VLANStrategy             string `json:"vlan_strategy,omitempty" validate:"omitempty,oneof=single per_paket per_odp per_pelanggan"`
}

// =============================================================================
// Audit Log List DTOs — parameter dan result untuk list audit log
// =============================================================================

// AuditLogListParams berisi parameter untuk list audit log dengan paginasi dan filter.
// TenantID diisi dari context auth middleware, bukan dari request body.
type AuditLogListParams struct {
	TenantID string     // diisi dari auth context
	Page     int        // halaman saat ini (default 1)
	PageSize int        // jumlah item per halaman (default 20)
	OLTID    string     // filter per OLT (opsional)
	ONTID    string     // filter per ONT (opsional)
	Action   string     // filter per action (opsional)
	DateFrom *time.Time // filter tanggal mulai (opsional)
	DateTo   *time.Time // filter tanggal akhir (opsional)
}

// AuditLogListResult berisi hasil list audit log dengan metadata paginasi.
type AuditLogListResult struct {
	Data       []*ProvisioningAuditLog `json:"data"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}
