package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// --- Sync Status ---

// SyncStatus mendefinisikan status sinkronisasi PPPoE user antara database dan router.
type SyncStatus string

const (
	// SyncStatusSynced menandakan data sudah sinkron antara DB dan router.
	SyncStatusSynced SyncStatus = "synced"

	// SyncStatusPendingCreate menandakan user belum dibuat di router.
	SyncStatusPendingCreate SyncStatus = "pending_create"

	// SyncStatusPendingUpdate menandakan perubahan belum diterapkan di router.
	SyncStatusPendingUpdate SyncStatus = "pending_update"

	// SyncStatusPendingDelete menandakan user belum dihapus dari router.
	SyncStatusPendingDelete SyncStatus = "pending_delete"

	// SyncStatusOutOfSync menandakan data berbeda antara DB dan router.
	SyncStatusOutOfSync SyncStatus = "out_of_sync"

	// SyncStatusError menandakan terjadi error saat sinkronisasi.
	SyncStatusError SyncStatus = "error"
)

// --- PPPoE User Entitas ---

// PPPoEUser merepresentasikan user PPPoE yang dikelola ISPBoss di router.
// Setiap tenant memiliki daftar PPPoE user sendiri yang diisolasi via RLS.
type PPPoEUser struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	CustomerID        string     `json:"customer_id"`
	RouterID          string     `json:"router_id"`
	Username          string     `json:"username"`
	PasswordEncrypted string     `json:"-"`
	ProfileName       string     `json:"profile_name"`
	Service           string     `json:"service"`
	RemoteAddress     string     `json:"remote_address,omitempty"`
	Comment           string     `json:"comment"`
	Disabled          bool       `json:"disabled"`
	UseSimpleQueue    bool       `json:"use_simple_queue"`
	Status            string     `json:"status"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus        SyncStatus `json:"sync_status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}

// --- PPPoE Session ---

// PPPoESession merepresentasikan sesi PPPoE aktif dari router.
// Data ini diambil langsung dari router dan tidak disimpan di database.
type PPPoESession struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	CallerID string `json:"caller_id"`
	Address  string `json:"address"`
	Uptime   string `json:"uptime"`
	BytesIn  int64  `json:"bytes_in"`
	BytesOut int64  `json:"bytes_out"`
	Service  string `json:"service"`
	Encoding string `json:"encoding"`
}

// --- Comment Functions ---

// commentPrefix adalah prefix yang digunakan untuk mengidentifikasi user ISPBoss di router.
const commentPrefix = "ISPBoss:"

// BuildComment membangun comment field format "ISPBoss:{customer_id}:{tenant_id}".
// Digunakan saat membuat PPPoE secret di router untuk tracking ownership.
func BuildComment(customerID, tenantID string) string {
	return fmt.Sprintf("%s%s:%s", commentPrefix, customerID, tenantID)
}

// ParseComment mengurai comment field dan mengembalikan customer_id dan tenant_id.
// Format yang diharapkan: "ISPBoss:{customer_id}:{tenant_id}".
// Mengembalikan ErrInvalidCommentFormat jika format tidak valid.
func ParseComment(comment string) (customerID, tenantID string, err error) {
	if !strings.HasPrefix(comment, commentPrefix) {
		return "", "", ErrInvalidCommentFormat
	}

	body := strings.TrimPrefix(comment, commentPrefix)
	parts := strings.SplitN(body, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ErrInvalidCommentFormat
	}

	return parts[0], parts[1], nil
}

// IsISPBossComment memeriksa apakah comment memiliki prefix "ISPBoss:".
// Digunakan saat sync untuk membedakan user ISPBoss dengan user manual admin.
func IsISPBossComment(comment string) bool {
	return strings.HasPrefix(comment, commentPrefix)
}

// --- Profile Name Generator ---

// profileNameRegex menghapus karakter selain alfanumerik dan hyphen.
var profileNameRegex = regexp.MustCompile(`[^a-z0-9-]`)

// GenerateProfileName menghasilkan profile_name dari nama paket.
// Mengganti spasi dengan hyphen, menghapus karakter spesial, dan lowercase.
// Fungsi ini idempotent: GenerateProfileName(GenerateProfileName(x)) == GenerateProfileName(x).
func GenerateProfileName(packageName string) string {
	// Lowercase terlebih dahulu
	name := strings.ToLower(packageName)

	// Ganti spasi dengan hyphen
	name = strings.ReplaceAll(name, " ", "-")

	// Hapus karakter selain alfanumerik dan hyphen
	name = profileNameRegex.ReplaceAllString(name, "")

	// Hapus hyphen berulang
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim hyphen di awal dan akhir
	name = strings.Trim(name, "-")

	return name
}
