package domain

import "time"

// --- ONT Status ---

// ONTStatus mendefinisikan status lifecycle ONT.
type ONTStatus string

const (
	// ONTStatusRegistered menandakan ONT terdaftar di DB, belum di OLT.
	ONTStatusRegistered ONTStatus = "registered"

	// ONTStatusProvisioned menandakan ONT aktif di OLT.
	ONTStatusProvisioned ONTStatus = "provisioned"

	// ONTStatusUnregistered menandakan ONT terdeteksi di OLT tapi belum di DB.
	ONTStatusUnregistered ONTStatus = "unregistered"

	// ONTStatusMissing menandakan ONT ada di DB tapi tidak di OLT.
	ONTStatusMissing ONTStatus = "missing"

	// ONTStatusDecommissioned menandakan ONT dihapus dari OLT.
	ONTStatusDecommissioned ONTStatus = "decommissioned"
)

// --- Provisioning State ---

// ProvisioningState mendefinisikan state proses provisioning.
type ProvisioningState string

const (
	// ProvisioningStatePending menandakan menunggu provisioning.
	ProvisioningStatePending ProvisioningState = "pending"

	// ProvisioningStateInProgress menandakan sedang diproses.
	ProvisioningStateInProgress ProvisioningState = "in_progress"

	// ProvisioningStateCompleted menandakan provisioning berhasil.
	ProvisioningStateCompleted ProvisioningState = "completed"

	// ProvisioningStateFailed menandakan provisioning gagal.
	ProvisioningStateFailed ProvisioningState = "failed"
)

// --- ONT Status Transitions ---

// ValidONTTransitions mendefinisikan transisi status ONT yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidONTTransitions = map[ONTStatus][]ONTStatus{
	ONTStatusRegistered:     {ONTStatusProvisioned, ONTStatusDecommissioned},
	ONTStatusProvisioned:    {ONTStatusDecommissioned, ONTStatusMissing},
	ONTStatusUnregistered:   {ONTStatusRegistered, ONTStatusProvisioned},
	ONTStatusMissing:        {ONTStatusProvisioned, ONTStatusDecommissioned},
	ONTStatusDecommissioned: {ONTStatusRegistered},
}

// CanTransitionONT memeriksa apakah transisi status ONT valid.
func CanTransitionONT(current, target ONTStatus) bool {
	targets, ok := ValidONTTransitions[current]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// --- ONT Entitas ---

// ONT merepresentasikan entitas ONT per tenant.
// Setiap tenant memiliki daftar ONT sendiri yang diisolasi via RLS.
type ONT struct {
	ID                   string            `json:"id"`
	TenantID             string            `json:"tenant_id"`
	OLTID                string            `json:"olt_id"`
	PONPortIndex         int               `json:"pon_port_index"`
	ONTIndex             int               `json:"ont_index"`
	SerialNumber         string            `json:"serial_number"`
	CustomerID           *string           `json:"customer_id,omitempty"`
	ODPID                *string           `json:"odp_id,omitempty"`
	VLANID               *string           `json:"vlan_id,omitempty"`
	ServiceProfileID     *string           `json:"service_profile_id,omitempty"`
	Status               ONTStatus         `json:"status"`
	ProvisioningState    ProvisioningState `json:"provisioning_state"`
	Description          string            `json:"description,omitempty"`
	LastProvisionedAt    *time.Time        `json:"last_provisioned_at,omitempty"`
	LastDecommissionedAt *time.Time        `json:"last_decommissioned_at,omitempty"`
	DeletedAt            *time.Time        `json:"deleted_at,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}
