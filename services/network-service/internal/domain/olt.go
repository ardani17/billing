package domain

import (
	"strings"
	"time"
)

// --- OLT Status ---

// OLTStatus mendefinisikan status konektivitas OLT.
type OLTStatus string

const (
	// OLTStatusOnline menandakan OLT aktif dan dapat dijangkau.
	OLTStatusOnline OLTStatus = "online"

	// OLTStatusOffline menandakan OLT tidak dapat dijangkau.
	OLTStatusOffline OLTStatus = "offline"

	// OLTStatusMaintenance menandakan OLT sedang dalam pemeliharaan.
	OLTStatusMaintenance OLTStatus = "maintenance"
)

// ValidOLTTransitions mendefinisikan transisi status OLT yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidOLTTransitions = map[OLTStatus][]OLTStatus{
	OLTStatusOffline:     {OLTStatusOnline, OLTStatusMaintenance},
	OLTStatusOnline:      {OLTStatusOffline, OLTStatusMaintenance},
	OLTStatusMaintenance: {OLTStatusOnline, OLTStatusOffline},
}

// CanTransitionOLT memeriksa apakah transisi status OLT valid.
func CanTransitionOLT(current, target OLTStatus) bool {
	targets, ok := ValidOLTTransitions[current]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// --- OLT Brand ---

// OLTBrand mendefinisikan brand OLT yang didukung.
type OLTBrand string

const (
	BrandZTE       OLTBrand = "zte"
	BrandHuawei    OLTBrand = "huawei"
	BrandFiberHome OLTBrand = "fiberhome"
	BrandVSOL      OLTBrand = "vsol"
	BrandHSGQ      OLTBrand = "hsgq"
)

// DetectBrand mendeteksi brand OLT dari sysDescr string.
// Mengembalikan string kosong jika brand tidak dikenali.
func DetectBrand(sysDescr string) OLTBrand {
	lower := strings.ToLower(sysDescr)
	switch {
	case strings.Contains(lower, "zte") || strings.Contains(lower, "zxa10"):
		return BrandZTE
	case strings.Contains(lower, "huawei") || strings.Contains(lower, "ma56"):
		return BrandHuawei
	case strings.Contains(lower, "fiberhome") || strings.Contains(lower, "an5516"):
		return BrandFiberHome
	case strings.Contains(lower, "vsol") || strings.Contains(lower, "v1600"):
		return BrandVSOL
	case strings.Contains(lower, "hsgq"):
		return BrandHSGQ
	default:
		return ""
	}
}

// --- SNMP Version ---

// SNMPVersion mendefinisikan versi SNMP yang didukung.
type SNMPVersion string

const (
	SNMPv2c SNMPVersion = "v2c"
	SNMPv3  SNMPVersion = "v3"
)

// --- CLI Protocol ---

// CLIProtocol mendefinisikan protokol CLI yang didukung.
type CLIProtocol string

const (
	CLIProtocolSSH    CLIProtocol = "ssh"
	CLIProtocolTelnet CLIProtocol = "telnet"
)

// --- OLT Entity ---

// OLT merepresentasikan perangkat OLT yang terdaftar per tenant.
// Setiap tenant memiliki daftar OLT sendiri yang diisolasi via RLS.
type OLT struct {
	ID                        string      `json:"id"`
	TenantID                  string      `json:"tenant_id"`
	Name                      string      `json:"name"`
	Host                      string      `json:"host"`
	SNMPVersion               SNMPVersion `json:"snmp_version"`
	SNMPPort                  int         `json:"snmp_port"`
	SNMPCommunityEncrypted    string      `json:"-"`
	SNMPUsername              string      `json:"snmp_username,omitempty"`
	SNMPAuthProtocol          string      `json:"snmp_auth_protocol,omitempty"`
	SNMPAuthPasswordEncrypted string      `json:"-"`
	SNMPPrivProtocol          string      `json:"snmp_priv_protocol,omitempty"`
	SNMPPrivPasswordEncrypted string      `json:"-"`
	CLIProtocol               CLIProtocol `json:"cli_protocol"`
	CLIPort                   int         `json:"cli_port"`
	CLIUsername               string      `json:"cli_username"`
	CLIPasswordEncrypted      string      `json:"-"`
	CLIEnablePasswordEncrypted string     `json:"-"`
	Brand                     OLTBrand    `json:"brand,omitempty"`
	Model                     string      `json:"model,omitempty"`
	FirmwareVersion           string      `json:"firmware_version,omitempty"`
	PONPortCount              int         `json:"pon_port_count"`
	TotalONTCount             int         `json:"total_ont_count"`
	Status                    OLTStatus   `json:"status"`
	HealthCheckIntervalSec    int         `json:"health_check_interval_sec"`
	LastOnlineAt              *time.Time  `json:"last_online_at,omitempty"`
	LastCheckedAt             *time.Time  `json:"last_checked_at,omitempty"`
	FailureCount              int         `json:"failure_count"`
	Notes                     string      `json:"notes,omitempty"`
	DeletedAt                 *time.Time  `json:"deleted_at,omitempty"`
	CreatedAt                 time.Time   `json:"created_at"`
	UpdatedAt                 time.Time   `json:"updated_at"`
}

// --- OLT Health Check Update ---

// OLTHealthCheckUpdate berisi field yang diupdate saat health check OLT selesai.
type OLTHealthCheckUpdate struct {
	LastCheckedAt *time.Time
	LastOnlineAt  *time.Time
	FailureCount  int
	Status        *OLTStatus
}
