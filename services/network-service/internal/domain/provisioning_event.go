package domain

// =============================================================================
// Provisioning Event Type Constants
// =============================================================================

const (
	// EventONTProvisioned adalah tipe event saat ONT berhasil di-provision.
	EventONTProvisioned = "ont.provisioned"

	// EventONTDecommissioned adalah tipe event saat ONT berhasil di-decommission.
	EventONTDecommissioned = "ont.decommissioned"

	// EventONTAutoProvisioned adalah tipe event saat ONT berhasil di-auto-provision.
	EventONTAutoProvisioned = "ont.auto_provisioned"

	// EventONTAutoProvisionFail adalah tipe event saat auto-provisioning gagal.
	EventONTAutoProvisionFail = "ont.auto_provision_failed"

	// EventONTPortMigrated adalah tipe event saat port migration terdeteksi.
	EventONTPortMigrated = "ont.port_migrated"
)

// =============================================================================
// Provisioning Event Payloads
// =============================================================================

// ONTProvisionedPayload adalah payload event ont.provisioned.
// Dipublikasikan saat ONT berhasil di-provision ke OLT.
type ONTProvisionedPayload struct {
	CorrelationID string `json:"correlation_id"`
	ONTID         string `json:"ont_id"`
	SerialNumber  string `json:"serial_number"`
	CustomerID    string `json:"customer_id"`
	OLTID         string `json:"olt_id"`
	OLTName       string `json:"olt_name"`
	PONPortIndex  int    `json:"pon_port_index"`
	VLANID        string `json:"vlan_id"`
	TenantID      string `json:"tenant_id"`
}

// ONTDecommissionedPayload adalah payload event ont.decommissioned.
// Dipublikasikan saat ONT berhasil di-decommission dari OLT.
type ONTDecommissionedPayload struct {
	CorrelationID string `json:"correlation_id"`
	ONTID         string `json:"ont_id"`
	SerialNumber  string `json:"serial_number"`
	CustomerID    string `json:"customer_id"`
	OLTID         string `json:"olt_id"`
	OLTName       string `json:"olt_name"`
	PONPortIndex  int    `json:"pon_port_index"`
	TenantID      string `json:"tenant_id"`
}

// ONTAutoProvisionedPayload adalah payload event ont.auto_provisioned.
// Dipublikasikan saat ONT berhasil di-auto-provision berdasarkan SN match.
type ONTAutoProvisionedPayload struct {
	CorrelationID string `json:"correlation_id"`
	ONTID         string `json:"ont_id"`
	SerialNumber  string `json:"serial_number"`
	CustomerID    string `json:"customer_id"`
	OLTID         string `json:"olt_id"`
	PONPortIndex  int    `json:"pon_port_index"`
	TenantID      string `json:"tenant_id"`
}

// ONTAutoProvisionFailedPayload adalah payload event ont.auto_provision_failed.
// Dipublikasikan saat auto-provisioning gagal, ONT tetap berstatus unregistered.
type ONTAutoProvisionFailedPayload struct {
	CorrelationID string `json:"correlation_id"`
	SerialNumber  string `json:"serial_number"`
	OLTID         string `json:"olt_id"`
	PONPortIndex  int    `json:"pon_port_index"`
	ErrorMessage  string `json:"error_message"`
	TenantID      string `json:"tenant_id"`
}

// ONTPortMigratedPayload adalah payload event ont.port_migrated.
// Dipublikasikan saat ONT terdeteksi pindah dari satu PON port ke port lain.
type ONTPortMigratedPayload struct {
	CorrelationID string `json:"correlation_id"`
	ONTID         string `json:"ont_id"`
	SerialNumber  string `json:"serial_number"`
	OLTID         string `json:"olt_id"`
	OldPortIndex  int    `json:"old_port_index"`
	NewPortIndex  int    `json:"new_port_index"`
	OldONTIndex   int    `json:"old_ont_index"`
	NewONTIndex   int    `json:"new_ont_index"`
	TenantID      string `json:"tenant_id"`
}
