package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// RouteType Constants - tipe jalur kabel yang didukung di peta FTTH
// =============================================================================

const (
	// RouteTypeBackbone merepresentasikan jalur kabel backbone (OLT -> ODP).
	RouteTypeBackbone = "backbone"

	// RouteTypeDrop merepresentasikan jalur kabel drop (ODP -> ONT).
	RouteTypeDrop = "drop"
)

// ValidRouteTypes berisi daftar tipe route yang valid untuk validasi input.
var ValidRouteTypes = []string{RouteTypeBackbone, RouteTypeDrop}

// IsValidRouteType memeriksa apakah tipe route valid.
func IsValidRouteType(routeType string) bool {
	for _, t := range ValidRouteTypes {
		if t == routeType {
			return true
		}
	}
	return false
}

// =============================================================================
// CableRoute Entitas - jalur kabel fiber antara dua node di peta
// =============================================================================

// CableRoute merepresentasikan jalur kabel fiber antara dua node di peta.
// Setiap CableRoute menghubungkan FromNodeID ke ToNodeID dengan koordinat waypoints.
// Jarak (DistanceMeters) dihitung otomatis dari koordinat menggunakan formula Haversine.
// Data diisolasi per tenant via RLS di PostgreSQL.
type CableRoute struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	FromNodeID     string          `json:"from_node_id"`
	ToNodeID       string          `json:"to_node_id"`
	RouteType      string          `json:"route_type"`  // "backbone", "drop"
	Coordinates    json.RawMessage `json:"coordinates"` // [[lat,lng], ...]
	DistanceMeters float64         `json:"distance_meters"`
	CoreCount      *int            `json:"core_count,omitempty"`
	Description    *string         `json:"description,omitempty"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// =============================================================================
// CableRouteListParams - parameter kueri untuk list cable route berdasarkan
// bounding box dan filter opsional
// =============================================================================

// CableRouteListParams berisi parameter untuk kueri list cable route di peta.
// Bounds digunakan untuk membatasi area peta yang ditampilkan.
// Filter digunakan untuk menyaring route berdasarkan kriteria tertentu.
type CableRouteListParams struct {
	// Bounding box - area peta yang visible
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLng float64 `json:"max_lng"`

	// Filter opsional
	RouteType  string `json:"route_type,omitempty"`
	FromNodeID string `json:"from_node_id,omitempty"`
	ToNodeID   string `json:"to_node_id,omitempty"`

	// Tenant context - diisi dari auth middleware
	TenantID string `json:"-"`
}
