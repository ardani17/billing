package domain

import "time"

// --- Service Profile Entity ---

// ServiceProfile merepresentasikan mapping antara paket ISPBoss dan OLT profile.
// Digunakan saat provisioning untuk menentukan line profile dan service profile
// yang akan diterapkan ke ONT berdasarkan paket pelanggan.
type ServiceProfile struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	OLTID            string     `json:"olt_id"`
	Name             string     `json:"name"`
	LineProfileID    int        `json:"line_profile_id"`
	ServiceProfileID int        `json:"service_profile_id"`
	PackageID        *string    `json:"package_id,omitempty"`
	Description      string     `json:"description,omitempty"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
