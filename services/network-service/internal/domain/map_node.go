package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// NodeType Constants — tipe node yang didukung di peta FTTH
// =============================================================================

const (
	// NodeTypeOLT merepresentasikan node OLT (Optical Line Terminal) di peta.
	NodeTypeOLT = "olt"

	// NodeTypeODP merepresentasikan node ODP (Optical Distribution Point) di peta.
	NodeTypeODP = "odp"

	// NodeTypeONT merepresentasikan node ONT (Optical Network Terminal) di peta.
	NodeTypeONT = "ont"
)

// ValidNodeTypes berisi daftar tipe node yang valid untuk validasi input.
var ValidNodeTypes = []string{NodeTypeOLT, NodeTypeODP, NodeTypeONT}

// IsValidNodeType memeriksa apakah tipe node valid.
func IsValidNodeType(nodeType string) bool {
	for _, t := range ValidNodeTypes {
		if t == nodeType {
			return true
		}
	}
	return false
}

// =============================================================================
// MapNode Entity — titik di peta yang merepresentasikan OLT, ODP, atau ONT
// =============================================================================

// MapNode merepresentasikan titik di peta (OLT, ODP, atau ONT).
// Setiap MapNode merujuk ke entitas asli via ReferenceID dan NodeType.
// Data diisolasi per tenant via RLS di PostgreSQL.
type MapNode struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	NodeType     string          `json:"node_type"`
	ReferenceID  string          `json:"reference_id"`
	Latitude     float64         `json:"latitude"`
	Longitude    float64         `json:"longitude"`
	CustomFields json.RawMessage `json:"custom_fields"`
	DeletedAt    *time.Time      `json:"deleted_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// =============================================================================
// MapNodeWithRef — data gabungan MapNode + info dari entitas referensi
// =============================================================================

// MapNodeWithRef berisi data MapNode yang sudah di-join dengan informasi
// dari entitas referensi (OLT/ODP/ONT) untuk ditampilkan di peta.
type MapNodeWithRef struct {
	// Data dari map_nodes
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	NodeType     string          `json:"node_type"`
	ReferenceID  string          `json:"reference_id"`
	Latitude     float64         `json:"latitude"`
	Longitude    float64         `json:"longitude"`
	CustomFields json.RawMessage `json:"custom_fields"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`

	// Data join dari entitas referensi (OLT/ODP/ONT)
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	Signal        *float64 `json:"signal,omitempty"`
	CustomerName  *string `json:"customer_name,omitempty"`
	CustomerID    *string `json:"customer_id,omitempty"`
	PackageName   *string `json:"package_name,omitempty"`
	SerialNumber  *string `json:"serial_number,omitempty"`
	SplitterType  *string `json:"splitter_type,omitempty"`
	Capacity      *int    `json:"capacity,omitempty"`
	UsedPorts     *int    `json:"used_ports,omitempty"`
	Address       *string `json:"address,omitempty"`
	BillingStatus *string `json:"billing_status,omitempty"`
	PackageID     *string `json:"package_id,omitempty"`
	AreaID        *string `json:"area_id,omitempty"`
	ODPID         *string `json:"odp_id,omitempty"`
}

// =============================================================================
// MapNodeListParams — parameter query untuk list node berdasarkan bounding box
// =============================================================================

// MapNodeListParams berisi parameter untuk query list node di peta.
// Bounds digunakan untuk membatasi area peta yang ditampilkan.
// Filter digunakan untuk menyaring node berdasarkan kriteria tertentu.
type MapNodeListParams struct {
	// Bounding box — area peta yang visible
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLng float64 `json:"max_lng"`

	// Filter opsional
	NodeType      string `json:"node_type,omitempty"`
	Status        string `json:"status,omitempty"`
	BillingStatus string `json:"billing_status,omitempty"`
	PackageID     string `json:"package_id,omitempty"`
	AreaID        string `json:"area_id,omitempty"`
	ODPID         string `json:"odp_id,omitempty"`

	// Tenant context — diisi dari auth middleware
	TenantID string `json:"-"`

	// Paginasi opsional
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// =============================================================================
// MapSearchResult — hasil pencarian node di peta
// =============================================================================

// MapSearchResult berisi informasi ringkas hasil pencarian node.
// Digunakan untuk autocomplete search di frontend.
type MapSearchResult struct {
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	Identifier  string  `json:"identifier"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Description string  `json:"description"`
}
