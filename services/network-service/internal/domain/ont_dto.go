package domain

import "time"

// =============================================================================
// ONT Request DTOs — payload dari HTTP request untuk operasi ONT
// =============================================================================

// ProvisionONTRequest adalah payload untuk POST /api/v1/olt/provisioning/ont.
// Digunakan untuk provisioning satu ONT ke OLT dengan linking ke customer,
// service profile, VLAN, dan ODP.
type ProvisionONTRequest struct {
	SerialNumber     string `json:"serial_number" validate:"required,min=1,max=50"`
	OLTID            string `json:"olt_id" validate:"required,uuid"`
	PONPortIndex     int    `json:"pon_port_index" validate:"min=0"`
	CustomerID       string `json:"customer_id" validate:"required,uuid"`
	ServiceProfileID string `json:"service_profile_id" validate:"required,uuid"`
	VLANID           string `json:"vlan_id" validate:"required,uuid"`
	ODPID            string `json:"odp_id,omitempty" validate:"omitempty,uuid"`
	Description      string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// ONTListParams berisi parameter untuk list ONT dengan paginasi dan filter.
// TenantID diisi dari context auth middleware, bukan dari request body.
type ONTListParams struct {
	TenantID          string // diisi dari auth context
	Page              int    // halaman saat ini (default 1)
	PageSize          int    // jumlah item per halaman (default 20)
	OLTID             string // filter per OLT (opsional)
	Status            string // filter per status (opsional)
	ProvisioningState string // filter per provisioning_state (opsional)
	CustomerID        string // filter per customer (opsional)
	Search            string // pencarian serial_number (opsional)
}

// =============================================================================
// ONT Response DTOs — format respons untuk operasi ONT
// =============================================================================

// ONTResponse adalah respons untuk operasi ONT (create/update/list).
// Menyertakan nama relasi (OLT, ODP, VLAN, service profile) untuk kemudahan UI.
type ONTResponse struct {
	ID                   string            `json:"id"`
	OLTID                string            `json:"olt_id"`
	OLTName              string            `json:"olt_name,omitempty"`
	PONPortIndex         int               `json:"pon_port_index"`
	ONTIndex             int               `json:"ont_index"`
	SerialNumber         string            `json:"serial_number"`
	CustomerID           *string           `json:"customer_id,omitempty"`
	ODPID                *string           `json:"odp_id,omitempty"`
	ODPName              string            `json:"odp_name,omitempty"`
	VLANID               *string           `json:"vlan_id,omitempty"`
	VLANName             string            `json:"vlan_name,omitempty"`
	ServiceProfileID     *string           `json:"service_profile_id,omitempty"`
	ServiceProfileName   string            `json:"service_profile_name,omitempty"`
	Status               ONTStatus         `json:"status"`
	ProvisioningState    ProvisioningState `json:"provisioning_state"`
	Description          string            `json:"description,omitempty"`
	LastProvisionedAt    *time.Time        `json:"last_provisioned_at,omitempty"`
	LastDecommissionedAt *time.Time        `json:"last_decommissioned_at,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// ONTDetailResponse adalah respons untuk GET /api/v1/olt/provisioning/onts/:id.
// Menyertakan audit logs terkait ONT untuk riwayat provisioning.
type ONTDetailResponse struct {
	ONTResponse
	AuditLogs []ProvisioningAuditLog `json:"audit_logs,omitempty"`
}

// ONTListResult berisi hasil list ONT dengan metadata paginasi.
type ONTListResult struct {
	Data       []*ONTResponse `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}
