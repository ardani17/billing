package domain

// =============================================================================
// OLT Request DTOs — payload dari HTTP request untuk operasi OLT
// =============================================================================

// CreateOLTRequest adalah payload untuk POST /api/v1/olt/devices.
// Digunakan untuk mendaftarkan OLT baru ke sistem dengan kredensial SNMP dan CLI.
type CreateOLTRequest struct {
	Name                   string `json:"name" validate:"required,min=1,max=100"`
	Host                   string `json:"host" validate:"required,max=255"`
	SNMPVersion            string `json:"snmp_version" validate:"required,oneof=v2c v3"`
	SNMPPort               int    `json:"snmp_port,omitempty" validate:"omitempty,min=1,max=65535"`
	SNMPCommunity          string `json:"snmp_community,omitempty"`
	SNMPUsername           string `json:"snmp_username,omitempty" validate:"omitempty,max=100"`
	SNMPAuthProtocol       string `json:"snmp_auth_protocol,omitempty" validate:"omitempty,oneof=MD5 SHA"`
	SNMPAuthPassword       string `json:"snmp_auth_password,omitempty"`
	SNMPPrivProtocol       string `json:"snmp_priv_protocol,omitempty" validate:"omitempty,oneof=DES AES"`
	SNMPPrivPassword       string `json:"snmp_priv_password,omitempty"`
	CLIProtocol            string `json:"cli_protocol" validate:"required,oneof=ssh telnet"`
	CLIPort                int    `json:"cli_port" validate:"required,min=1,max=65535"`
	CLIUsername            string `json:"cli_username" validate:"required,max=100"`
	CLIPassword            string `json:"cli_password" validate:"required"`
	CLIEnablePassword      string `json:"cli_enable_password,omitempty"`
	HealthCheckIntervalSec int    `json:"health_check_interval_sec,omitempty" validate:"omitempty,min=60,max=3600"`
	Notes                  string `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// UpdateOLTRequest adalah payload untuk PUT /api/v1/olt/devices/:id.
// Semua field bersifat opsional — hanya field yang dikirim yang akan diupdate.
// CLIPort dan HealthCheckIntervalSec menggunakan pointer untuk membedakan zero value dan tidak dikirim.
type UpdateOLTRequest struct {
	Name                   string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Host                   string `json:"host,omitempty" validate:"omitempty,max=255"`
	SNMPVersion            string `json:"snmp_version,omitempty" validate:"omitempty,oneof=v2c v3"`
	SNMPCommunity          string `json:"snmp_community,omitempty"`
	SNMPUsername           string `json:"snmp_username,omitempty"`
	SNMPAuthProtocol       string `json:"snmp_auth_protocol,omitempty"`
	SNMPAuthPassword       string `json:"snmp_auth_password,omitempty"`
	SNMPPrivProtocol       string `json:"snmp_priv_protocol,omitempty"`
	SNMPPrivPassword       string `json:"snmp_priv_password,omitempty"`
	CLIProtocol            string `json:"cli_protocol,omitempty"`
	CLIPort                *int   `json:"cli_port,omitempty"`
	CLIUsername            string `json:"cli_username,omitempty"`
	CLIPassword            string `json:"cli_password,omitempty"`
	CLIEnablePassword      string `json:"cli_enable_password,omitempty"`
	HealthCheckIntervalSec *int   `json:"health_check_interval_sec,omitempty"`
	Notes                  string `json:"notes,omitempty"`
	Status                 string `json:"status,omitempty" validate:"omitempty,oneof=maintenance online offline"`
}

// OLTListParams berisi parameter untuk list OLT dengan paginasi dan filter.
// TenantID diisi dari context auth middleware, bukan dari request body.
type OLTListParams struct {
	TenantID string // diisi dari auth context
	Page     int    // halaman saat ini (default 1)
	PageSize int    // jumlah item per halaman (default 20)
	Status   string // filter berdasarkan status: online, offline, maintenance (opsional)
	Brand    string // filter berdasarkan brand: zte, huawei, dll (opsional)
	Search   string // pencarian berdasarkan name atau host (opsional)
}
