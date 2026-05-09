package domain

import "time"

// =============================================================================
// Service Profile DTO permintaan - payload dari HTTP permintaan
// =============================================================================

// CreateServiceProfileRequest adalah payload untuk POST /api/v1/olt/devices/:id/service-profiles.
// Digunakan untuk membuat mapping antara paket ISPBoss dan OLT line/service profile.
// OLTID diisi oleh handler dari URL path parameter (:id).
type CreateServiceProfileRequest struct {
	OLTID            string `json:"-"` // diisi dari URL path
	Name             string `json:"name" validate:"required,min=1,max=100"`
	LineProfileID    int    `json:"line_profile_id" validate:"required,min=0"`
	ServiceProfileID int    `json:"service_profile_id" validate:"required,min=0"`
	PackageID        string `json:"package_id,omitempty" validate:"omitempty,uuid"`
	Description      string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// UpdateServiceProfileRequest adalah payload untuk PUT /api/v1/olt/service-profiles/:id.
// Semua field bersifat opsional - hanya field yang dikirim yang akan diupdate.
// LineProfileID dan ServiceProfileID menggunakan pointer untuk membedakan zero value.
type UpdateServiceProfileRequest struct {
	Name             string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	LineProfileID    *int   `json:"line_profile_id,omitempty"`
	ServiceProfileID *int   `json:"service_profile_id,omitempty"`
	PackageID        string `json:"package_id,omitempty" validate:"omitempty,uuid"`
	Description      string `json:"description,omitempty"`
}

// ServiceProfileListParams berisi parameter untuk list service profile dengan paginasi.
type ServiceProfileListParams struct {
	Page     int // halaman saat ini (bawaan 1)
	PageSize int // jumlah item per halaman (bawaan 20)
}

// =============================================================================
// Service Profile Respons DTOs - format respons untuk operasi service profile
// =============================================================================

// ServiceProfileResponse adalah respons untuk operasi service profile.
// Menyertakan jumlah ONT aktif yang menggunakan profile ini.
type ServiceProfileResponse struct {
	ID               string    `json:"id"`
	OLTID            string    `json:"olt_id"`
	Name             string    `json:"name"`
	LineProfileID    int       `json:"line_profile_id"`
	ServiceProfileID int       `json:"service_profile_id"`
	PackageID        *string   `json:"package_id,omitempty"`
	Description      string    `json:"description,omitempty"`
	ActiveONTs       int64     `json:"active_onts"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ServiceProfileListResult berisi hasil list service profile dengan metadata paginasi.
type ServiceProfileListResult struct {
	Data       []*ServiceProfileResponse `json:"data"`
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
}
