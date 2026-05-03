package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// Share Link Request DTOs — payload dari HTTP request untuk operasi share link
// =============================================================================

// CreateShareLinkRequest adalah payload untuk POST /api/v1/network-map/share.
// Digunakan untuk membuat share link read-only ke peta dengan opsi
// layer visibility, expiry, dan password protection.
type CreateShareLinkRequest struct {
	VisibleLayers json.RawMessage `json:"visible_layers" validate:"required"`
	ExpiryDays    *int            `json:"expiry_days,omitempty" validate:"omitempty,min=1"`
	Password      *string         `json:"password,omitempty"`
}

// =============================================================================
// Share Link Response DTOs — format respons untuk operasi share link
// =============================================================================

// ShareLinkResponse adalah respons untuk operasi CRUD share link.
// Berisi data share link lengkap termasuk token, URL, dan embed code
// untuk berbagi peta read-only dengan pihak eksternal.
type ShareLinkResponse struct {
	ID            string          `json:"id"`
	Token         string          `json:"token"`
	URL           string          `json:"url"`
	EmbedCode     string          `json:"embed_code"`
	VisibleLayers json.RawMessage `json:"visible_layers"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
	AccessCount   int             `json:"access_count"`
	CreatedBy     string          `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

// SharedMapData adalah respons untuk GET /api/v1/network-map/share/:token.
// Berisi data peta yang difilter berdasarkan visible_layers yang ditentukan
// saat pembuatan share link. Digunakan untuk tampilan read-only publik.
type SharedMapData struct {
	Nodes         []MapNodeWithRefResponse `json:"nodes"`
	Cables        []CableRouteResponse     `json:"cables"`
	VisibleLayers json.RawMessage          `json:"visible_layers"`
}

// =============================================================================
// Shared Response DTOs — digunakan oleh beberapa DTO file lain
// =============================================================================

// NodePhotoResponse adalah respons untuk data foto node.
// Digunakan di MapNodeDetailResponse dan endpoint list photos.
type NodePhotoResponse struct {
	ID            string    `json:"id"`
	MapNodeID     string    `json:"map_node_id"`
	FilePath      string    `json:"file_path"`
	FileSizeBytes int       `json:"file_size_bytes"`
	Caption       *string   `json:"caption,omitempty"`
	UploadedBy    string    `json:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// MapChangeHistoryResponse adalah respons untuk data riwayat perubahan node.
// Digunakan di MapNodeDetailResponse dan endpoint history.
type MapChangeHistoryResponse struct {
	ID          string      `json:"id"`
	MapNodeID   string      `json:"map_node_id"`
	Action      string      `json:"action"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	PerformedBy string      `json:"performed_by"`
	CreatedAt   time.Time   `json:"created_at"`
}

// =============================================================================
// Geocoding DTOs — respons untuk reverse geocoding
// =============================================================================

// GeocodingResult adalah respons untuk GET /api/v1/network-map/geocode/reverse.
// Berisi alamat lengkap hasil reverse geocoding dari koordinat GPS.
// Jika provider gagal, field Error akan berisi pesan error dan
// hanya Latitude/Longitude yang terisi.
type GeocodingResult struct {
	Address    string   `json:"address"`
	Street     string   `json:"street,omitempty"`
	Kelurahan  string   `json:"kelurahan,omitempty"`
	Kecamatan  string   `json:"kecamatan,omitempty"`
	City       string   `json:"city,omitempty"`
	Province   string   `json:"province,omitempty"`
	PostalCode string   `json:"postal_code,omitempty"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Error      *string  `json:"error,omitempty"`
}

// =============================================================================
// Helper Functions — konversi entity ke response DTO
// =============================================================================

// ToShareLinkResponse mengkonversi MapShareLink entity ke ShareLinkResponse DTO.
// Parameter baseURL digunakan untuk membentuk URL dan embed code share link.
func ToShareLinkResponse(link *MapShareLink, baseURL string) *ShareLinkResponse {
	url := baseURL + "/share/" + link.Token
	embedCode := `<iframe src="` + url + `" width="100%" height="600" frameborder="0"></iframe>`

	return &ShareLinkResponse{
		ID:            link.ID,
		Token:         link.Token,
		URL:           url,
		EmbedCode:     embedCode,
		VisibleLayers: link.VisibleLayers,
		ExpiresAt:     link.ExpiresAt,
		AccessCount:   link.AccessCount,
		CreatedBy:     link.CreatedBy,
		CreatedAt:     link.CreatedAt,
	}
}

// ToNodePhotoResponse mengkonversi NodePhoto entity ke NodePhotoResponse DTO.
func ToNodePhotoResponse(photo *NodePhoto) *NodePhotoResponse {
	return &NodePhotoResponse{
		ID:            photo.ID,
		MapNodeID:     photo.MapNodeID,
		FilePath:      photo.FilePath,
		FileSizeBytes: photo.FileSizeBytes,
		Caption:       photo.Caption,
		UploadedBy:    photo.UploadedBy,
		CreatedAt:     photo.CreatedAt,
	}
}

// ToMapChangeHistoryResponse mengkonversi MapChangeHistory entity ke
// MapChangeHistoryResponse DTO. OldValue dan NewValue dikonversi dari
// json.RawMessage ke interface{} agar bisa di-serialize langsung ke JSON.
func ToMapChangeHistoryResponse(h *MapChangeHistory) *MapChangeHistoryResponse {
	var oldVal, newVal interface{}
	if h.OldValue != nil {
		_ = json.Unmarshal(h.OldValue, &oldVal)
	}
	if h.NewValue != nil {
		_ = json.Unmarshal(h.NewValue, &newVal)
	}

	return &MapChangeHistoryResponse{
		ID:          h.ID,
		MapNodeID:   h.MapNodeID,
		Action:      h.Action,
		OldValue:    oldVal,
		NewValue:    newVal,
		PerformedBy: h.PerformedBy,
		CreatedAt:   h.CreatedAt,
	}
}
