package domain

import "time"

// --- Provisioning Settings Entitas ---

// ProvisioningSettings merepresentasikan settings provisioning per tenant.
// Setiap tenant memiliki satu record settings yang mengontrol perilaku
// auto-provisioning, auto-port-migration, dan strategi VLAN.
type ProvisioningSettings struct {
	ID                       string       `json:"id"`
	TenantID                 string       `json:"tenant_id"`
	AutoProvisioningEnabled  bool         `json:"auto_provisioning_enabled"`
	AutoPortMigrationEnabled bool         `json:"auto_port_migration_enabled"`
	VLANStrategy             VLANStrategy `json:"vlan_strategy"`
	CreatedAt                time.Time    `json:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at"`
}

// DefaultProvisioningSettings mengembalikan settings bawaan untuk tenant
// yang belum memiliki record di database.
// Bawaan: auto_provisioning=false, auto_port_migration=false, vlan_strategy="single".
func DefaultProvisioningSettings(tenantID string) *ProvisioningSettings {
	now := time.Now()
	return &ProvisioningSettings{
		TenantID:                 tenantID,
		AutoProvisioningEnabled:  false,
		AutoPortMigrationEnabled: false,
		VLANStrategy:             VLANStrategySingle,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}
