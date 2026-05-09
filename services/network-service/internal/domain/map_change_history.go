package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// ChangeAction Constants - aksi perubahan yang dicatat di riwayat node
// =============================================================================

const (
	// ChangeActionCreated dicatat saat node baru dibuat di peta.
	ChangeActionCreated = "created"

	// ChangeActionLocationMoved dicatat saat lokasi (latitude/longitude) node dipindahkan.
	ChangeActionLocationMoved = "location_moved"

	// ChangeActionCustomFieldsUpdated dicatat saat kustom field node diperbarui.
	ChangeActionCustomFieldsUpdated = "custom_fields_updated"

	// ChangeActionPhotoAdded dicatat saat foto baru di-upload ke node.
	ChangeActionPhotoAdded = "photo_added"

	// ChangeActionPhotoRemoved dicatat saat foto dihapus dari node.
	ChangeActionPhotoRemoved = "photo_removed"

	// ChangeActionDeleted dicatat saat node di-hapus lunak (masuk trash).
	ChangeActionDeleted = "deleted"

	// ChangeActionRestored dicatat saat node di-restore dari trash.
	ChangeActionRestored = "restored"
)

// ValidChangeActions berisi daftar aksi perubahan yang valid untuk validasi input.
var ValidChangeActions = []string{
	ChangeActionCreated,
	ChangeActionLocationMoved,
	ChangeActionCustomFieldsUpdated,
	ChangeActionPhotoAdded,
	ChangeActionPhotoRemoved,
	ChangeActionDeleted,
	ChangeActionRestored,
}

// IsValidChangeAction memeriksa apakah aksi perubahan valid.
func IsValidChangeAction(action string) bool {
	for _, a := range ValidChangeActions {
		if a == action {
			return true
		}
	}
	return false
}

// =============================================================================
// MapChangeHistory Entitas - riwayat perubahan per node di peta (append-only)
// =============================================================================

// MapChangeHistory merepresentasikan satu entri riwayat perubahan pada node di peta.
// Tabel ini bersifat append-only: tidak ada operasi perbarui atau hapus.
// Setiap modifikasi pada lokasi, kustom field, atau foto node akan menghasilkan
// entri baru dengan old_value dan new_value dalam format JSONB.
// Data diisolasi per tenant via RLS di PostgreSQL.
type MapChangeHistory struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	MapNodeID   string          `json:"map_node_id"`
	Action      string          `json:"action"`
	OldValue    json.RawMessage `json:"old_value,omitempty"`
	NewValue    json.RawMessage `json:"new_value,omitempty"`
	PerformedBy string          `json:"performed_by"`
	CreatedAt   time.Time       `json:"created_at"`
}
