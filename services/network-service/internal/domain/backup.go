package domain

import "time"

type RouterBackup struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id,omitempty"`
	RouterID  string    `json:"router_id"`
	FileName  string    `json:"file_name"`
	Format    string    `json:"format"`
	SizeBytes int64     `json:"size_bytes"`
	Checksum  string    `json:"checksum,omitempty"`
	Content   string    `json:"content,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RouterBackupListParams struct {
	RouterID string
	Page     int
	PageSize int
}

type RouterBackupListResult struct {
	Data       []RouterBackup `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type CreateRouterBackupInput struct {
	TenantID  string
	RouterID  string
	FileName  string
	Format    string
	SizeBytes int64
	Checksum  string
	Content   string
	CreatedBy string
}

type RouterFirmwareInfo struct {
	RouterOSVersion string            `json:"routeros_version"`
	Architecture    string            `json:"architecture,omitempty"`
	BoardName       string            `json:"board_name,omitempty"`
	FactoryFirmware string            `json:"factory_firmware,omitempty"`
	CurrentFirmware string            `json:"current_firmware,omitempty"`
	UpgradeFirmware string            `json:"upgrade_firmware,omitempty"`
	Packages        []RouterOSPackage `json:"packages"`
	Outdated        bool              `json:"outdated"`
	Warning         string            `json:"warning,omitempty"`
}

type RouterOSPackage struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Disabled  bool   `json:"disabled"`
	Scheduled string `json:"scheduled,omitempty"`
}
