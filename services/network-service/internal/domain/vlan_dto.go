package domain

import "time"

// =============================================================================
// VLAN Request DTOs — payload dari HTTP request untuk operasi VLAN
// =============================================================================

// CreateVLANRequest adalah payload untuk POST /api/v1/olt/devices/:id/vlans.
// Digunakan untuk membuat VLAN baru pada OLT tertentu.
// OLTID diisi oleh handler dari URL path parameter (:id).
type CreateVLANRequest struct {
	OLTID       string `json:"-"`                                                    // diisi dari URL path
	VLANID      int    `json:"vlan_id" validate:"required,min=1,max=4094"`
	Name        string `json:"name" validate:"required,min=1,max=100"`
	VLANType    string `json:"vlan_type" validate:"required,oneof=data voice management"`
	Description string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// UpdateVLANRequest adalah payload untuk PUT /api/v1/olt/vlans/:id.
// Semua field bersifat opsional — hanya field yang dikirim yang akan diupdate.
type UpdateVLANRequest struct {
	Name        string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	VLANType    string `json:"vlan_type,omitempty" validate:"omitempty,oneof=data voice management"`
	Description string `json:"description,omitempty"`
}

// VLANListParams berisi parameter untuk list VLAN dengan paginasi.
type VLANListParams struct {
	Page     int // halaman saat ini (default 1)
	PageSize int // jumlah item per halaman (default 20)
}

// =============================================================================
// VLAN Response DTOs — format respons untuk operasi VLAN
// =============================================================================

// VLANResponse adalah respons untuk operasi VLAN (create/update/list).
// Menyertakan jumlah ONT aktif yang menggunakan VLAN ini.
type VLANResponse struct {
	ID          string    `json:"id"`
	OLTID       string    `json:"olt_id"`
	VLANID      int       `json:"vlan_id"`
	Name        string    `json:"name"`
	VLANType    string    `json:"vlan_type"`
	Description string    `json:"description,omitempty"`
	ActiveONTs  int64     `json:"active_onts"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// VLANListResult berisi hasil list VLAN dengan metadata paginasi.
type VLANListResult struct {
	Data       []*VLANResponse `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}
