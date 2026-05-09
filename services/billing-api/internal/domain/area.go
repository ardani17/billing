package domain

import (
	"errors"
	"time"
)

// Area merepresentasikan wilayah/area geografis pelanggan.
type Area struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	ODPID         string    `json:"odp_id,omitempty"`
	CenterLat     *float64  `json:"center_lat,omitempty"`
	CenterLng     *float64  `json:"center_lng,omitempty"`
	CustomerCount int       `json:"customer_count,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateAreaRequest adalah payload untuk POST /v1/areas.
type CreateAreaRequest struct {
	Name        string   `json:"name" validate:"required,min=2,max=255"`
	Description string   `json:"description" validate:"omitempty"`
	ODPID       string   `json:"odp_id" validate:"omitempty"`
	CenterLat   *float64 `json:"center_lat" validate:"omitempty,min=-90,max=90"`
	CenterLng   *float64 `json:"center_lng" validate:"omitempty,min=-180,max=180"`
}

// UpdateAreaRequest adalah payload untuk PUT /v1/areas/:id.
type UpdateAreaRequest struct {
	Name        string   `json:"name" validate:"omitempty,min=2,max=255"`
	Description string   `json:"description" validate:"omitempty"`
	ODPID       string   `json:"odp_id" validate:"omitempty"`
	CenterLat   *float64 `json:"center_lat" validate:"omitempty,min=-90,max=90"`
	CenterLng   *float64 `json:"center_lng" validate:"omitempty,min=-180,max=180"`
}

// --- Area Variabel error domain ---

var (
	// ErrAreaNotFound dikembalikan saat area tidak ditemukan
	ErrAreaNotFound = errors.New("area tidak ditemukan")

	// ErrAreaNameDuplicate dikembalikan saat nama area sudah ada di tenant yang sama
	ErrAreaNameDuplicate = errors.New("nama area sudah terdaftar")

	// ErrAreaHasCustomers dikembalikan saat area masih memiliki pelanggan
	ErrAreaHasCustomers = errors.New("area masih memiliki pelanggan")
)
