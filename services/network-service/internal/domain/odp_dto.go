package domain

import "time"

// =============================================================================
// ODP Request DTOs — payload dari HTTP request untuk operasi ODP
// =============================================================================

// CreateODPRequest adalah payload untuk POST /api/v1/olt/odp.
// Digunakan untuk membuat ODP baru yang terhubung ke OLT pada PON port tertentu.
type CreateODPRequest struct {
	OLTID        string   `json:"olt_id" validate:"required,uuid"`
	PONPortIndex int      `json:"pon_port_index" validate:"required,min=0"`
	Name         string   `json:"name" validate:"required,min=1,max=100"`
	SplitterType string   `json:"splitter_type" validate:"required,oneof=1:4 1:8 1:16 1:32"`
	Address      string   `json:"address,omitempty" validate:"omitempty,max=500"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Notes        string   `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// UpdateODPRequest adalah payload untuk PUT /api/v1/olt/odp/:id.
// Semua field bersifat opsional — hanya field yang dikirim yang akan diupdate.
type UpdateODPRequest struct {
	Name      string   `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Address   string   `json:"address,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Notes     string   `json:"notes,omitempty"`
}

// ODPListParams berisi parameter untuk list ODP dengan paginasi dan filter.
// TenantID diisi dari context auth middleware, bukan dari request body.
type ODPListParams struct {
	TenantID     string // diisi dari auth context
	Page         int    // halaman saat ini (default 1)
	PageSize     int    // jumlah item per halaman (default 20)
	OLTID        string // filter berdasarkan olt_id (opsional)
	PONPortIndex *int   // filter berdasarkan pon_port (opsional)
}

// =============================================================================
// ODP Response DTOs — format respons untuk operasi ODP
// =============================================================================

// ODPResponse adalah respons untuk operasi create/update/list ODP.
// Menyertakan informasi kapasitas (capacity, used_ports) dan lokasi GPS.
type ODPResponse struct {
	ID           string    `json:"id"`
	OLTID        string    `json:"olt_id"`
	OLTName      string    `json:"olt_name,omitempty"`
	PONPortIndex int       `json:"pon_port_index"`
	Name         string    `json:"name"`
	SplitterType string    `json:"splitter_type"`
	Capacity     int       `json:"capacity"`
	UsedPorts    int       `json:"used_ports"`
	Address      string    `json:"address,omitempty"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ODPDetailResponse adalah respons untuk GET /api/v1/olt/odp/:id.
// Menyertakan warning jika ODP sudah penuh (used_ports == capacity).
type ODPDetailResponse struct {
	ODPResponse
	Warning string `json:"warning,omitempty"` // diisi jika ODP penuh
}

// ODPListResult berisi hasil list ODP dengan metadata paginasi.
type ODPListResult struct {
	Data       []*ODPResponse `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}
