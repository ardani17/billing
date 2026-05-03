package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// Cable Route Request DTOs — payload dari HTTP request untuk operasi cable route
// =============================================================================

// CreateCableRouteRequest adalah payload untuk POST /api/v1/network-map/cables.
// Digunakan untuk membuat cable route baru yang menghubungkan dua map node.
// Coordinates berisi array waypoints [[lat,lng], ...] untuk polyline di peta.
// DistanceMeters dihitung otomatis dari coordinates menggunakan formula Haversine.
type CreateCableRouteRequest struct {
	FromNodeID  string          `json:"from_node_id" validate:"required,uuid"`
	ToNodeID    string          `json:"to_node_id" validate:"required,uuid"`
	RouteType   string          `json:"route_type" validate:"required,oneof=backbone drop"`
	Coordinates json.RawMessage `json:"coordinates" validate:"required"`
	CoreCount   *int            `json:"core_count,omitempty" validate:"omitempty,min=1"`
	Description *string         `json:"description,omitempty"`
}

// UpdateCableRouteRequest adalah payload untuk PUT /api/v1/network-map/cables/:id.
// Semua field bersifat opsional — hanya field yang dikirim yang akan diupdate.
// Jika Coordinates diupdate, DistanceMeters akan dihitung ulang secara otomatis.
type UpdateCableRouteRequest struct {
	Coordinates json.RawMessage `json:"coordinates,omitempty"`
	CoreCount   *int            `json:"core_count,omitempty" validate:"omitempty,min=1"`
	Description *string         `json:"description,omitempty"`
}

// =============================================================================
// Cable Route Response DTOs — format respons untuk operasi cable route
// =============================================================================

// CableRouteResponse adalah respons untuk operasi CRUD cable route.
// Berisi data cable route lengkap termasuk jarak yang dihitung otomatis.
type CableRouteResponse struct {
	ID             string          `json:"id"`
	FromNodeID     string          `json:"from_node_id"`
	ToNodeID       string          `json:"to_node_id"`
	RouteType      string          `json:"route_type"`
	Coordinates    json.RawMessage `json:"coordinates"`
	DistanceMeters float64         `json:"distance_meters"`
	CoreCount      *int            `json:"core_count,omitempty"`
	Description    *string         `json:"description,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// =============================================================================
// Helper Functions — konversi entity ke response DTO
// =============================================================================

// ToCableRouteResponse mengkonversi CableRoute entity ke CableRouteResponse DTO.
func ToCableRouteResponse(route *CableRoute) *CableRouteResponse {
	return &CableRouteResponse{
		ID:             route.ID,
		FromNodeID:     route.FromNodeID,
		ToNodeID:       route.ToNodeID,
		RouteType:      route.RouteType,
		Coordinates:    route.Coordinates,
		DistanceMeters: route.DistanceMeters,
		CoreCount:      route.CoreCount,
		Description:    route.Description,
		CreatedAt:      route.CreatedAt,
		UpdatedAt:      route.UpdatedAt,
	}
}
