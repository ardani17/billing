package domain

import "time"

// --- Audit Action ---

// AuditAction mendefinisikan tipe aksi provisioning untuk audit trail.
type AuditAction string

const (
	// AuditActionONTProvision untuk aksi provisioning ONT.
	AuditActionONTProvision AuditAction = "ont_provision"

	// AuditActionONTDecommission untuk aksi decommission ONT.
	AuditActionONTDecommission AuditAction = "ont_decommission"

	// AuditActionONTReboot untuk aksi reboot ONT.
	AuditActionONTReboot AuditAction = "ont_reboot"

	// AuditActionServicePortAdd untuk aksi menambahkan service-port.
	AuditActionServicePortAdd AuditAction = "service_port_add"

	// AuditActionServicePortRemove untuk aksi menghapus service-port.
	AuditActionServicePortRemove AuditAction = "service_port_remove"

	// AuditActionBulkProvision untuk aksi bulk provisioning.
	AuditActionBulkProvision AuditAction = "bulk_provision"

	// AuditActionAutoProvision untuk aksi auto-provisioning.
	AuditActionAutoProvision AuditAction = "auto_provision"
)

// --- Provisioning Audit Log Entity ---

// ProvisioningAuditLog merepresentasikan satu record audit trail provisioning.
// Tabel ini append-only — tidak ada operasi update atau delete.
// Setiap command yang dikirim ke OLT dicatat beserta response-nya.
type ProvisioningAuditLog struct {
	ID               string      `json:"id"`
	TenantID         string      `json:"tenant_id"`
	OLTID            string      `json:"olt_id"`
	ONTID            *string     `json:"ont_id,omitempty"`
	Action           AuditAction `json:"action"`
	CommandsSent     []string    `json:"commands_sent"`
	CommandResponses []string    `json:"command_responses"`
	Status           string      `json:"status"`
	ErrorMessage     string      `json:"error_message,omitempty"`
	PerformedBy      string      `json:"performed_by"`
	CorrelationID    string      `json:"correlation_id"`
	CreatedAt        time.Time   `json:"created_at"`
}
