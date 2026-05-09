package domain

import "time"

// --- PPPoE Profile Entitas ---

// PPPoEProfile merepresentasikan profil bandwidth PPPoE di router.
// Setiap profil terkait dengan satu paket (package_id) dan berisi konfigurasi
// rate limit, burst, address pool, dan parameter PPPoE lainnya.
// Profil disinkronkan ke semua router yang memiliki service_type pppoe.
type PPPoEProfile struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	PackageID              string    `json:"package_id"`
	ProfileName            string    `json:"profile_name"`
	DownloadLimit          string    `json:"download_limit"`
	UploadLimit            string    `json:"upload_limit"`
	BurstDownload          string    `json:"burst_download,omitempty"`
	BurstUpload            string    `json:"burst_upload,omitempty"`
	BurstThresholdDownload string    `json:"burst_threshold_download,omitempty"`
	BurstThresholdUpload   string    `json:"burst_threshold_upload,omitempty"`
	BurstTime              string    `json:"burst_time,omitempty"`
	AddressPool            string    `json:"address_pool,omitempty"`
	LocalAddress           string    `json:"local_address"`
	OnlyOne                bool      `json:"only_one"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// PackageProfilePayload membawa metadata paket dari billing-api untuk cadangan
// saat tabel pppoe_profiles network-service belum punya mapping package_id.
type PackageProfilePayload struct {
	TenantID            string
	PackageID           string
	MikrotikProfileName string
	DownloadMbps        int
	UploadMbps          int
	AddressPool         string
}
