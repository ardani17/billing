package domain

import "time"

// =============================================================================
// Konstanta Foto — batasan dan tipe file yang diizinkan untuk foto node
// =============================================================================

const (
	// MaxPhotosPerNode adalah jumlah maksimal foto yang dapat di-upload per node.
	MaxPhotosPerNode = 5

	// MaxPhotoSizeBytes adalah ukuran maksimal file foto dalam bytes (1 MB).
	MaxPhotoSizeBytes = 1 * 1024 * 1024
)

// AllowedPhotoTypes berisi daftar MIME type foto yang diizinkan untuk upload.
var AllowedPhotoTypes = []string{
	"image/jpeg",
	"image/png",
	"image/webp",
}

// IsAllowedPhotoType memeriksa apakah MIME type foto diizinkan untuk upload.
func IsAllowedPhotoType(mimeType string) bool {
	for _, t := range AllowedPhotoTypes {
		if t == mimeType {
			return true
		}
	}
	return false
}

// =============================================================================
// NodePhoto Entity — foto dokumentasi yang di-upload per node di peta
// =============================================================================

// NodePhoto merepresentasikan foto yang di-upload per node untuk dokumentasi
// instalasi. Setiap node dapat memiliki maksimal 5 foto (MaxPhotosPerNode).
// File disimpan di path tenant-isolated: uploads/{tenant_id}/map-photos/{node_id}/{photo_id}.{ext}
// Data diisolasi per tenant via RLS di PostgreSQL.
type NodePhoto struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	MapNodeID     string     `json:"map_node_id"`
	FilePath      string     `json:"file_path"`
	FileSizeBytes int        `json:"file_size_bytes"`
	Caption       *string    `json:"caption,omitempty"`
	UploadedBy    string     `json:"uploaded_by"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
