package domain

import "time"

// --- VLAN Strategy ---

// VLANStrategy mendefinisikan strategi assignment VLAN saat provisioning.
type VLANStrategy string

const (
	// VLANStrategySingle - semua pelanggan menggunakan 1 VLAN (bawaan).
	VLANStrategySingle VLANStrategy = "single"

	// VLANStrategyPerPaket - VLAN berbeda per paket internet.
	VLANStrategyPerPaket VLANStrategy = "per_paket"

	// VLANStrategyPerODP - VLAN berbeda per ODP/splitter.
	VLANStrategyPerODP VLANStrategy = "per_odp"

	// VLANStrategyPerPelanggan - VLAN unik per pelanggan.
	VLANStrategyPerPelanggan VLANStrategy = "per_pelanggan"
)

// --- VLAN Type ---

// VLANType mendefinisikan tipe VLAN.
type VLANType string

const (
	// VLANTypeData untuk VLAN data pelanggan.
	VLANTypeData VLANType = "data"

	// VLANTypeVoice untuk VLAN voice/VoIP.
	VLANTypeVoice VLANType = "voice"

	// VLANTypeManagement untuk VLAN management OLT.
	VLANTypeManagement VLANType = "management"
)

// --- VLAN Resolve Context ---

// VLANResolveContext berisi konteks untuk resolusi VLAN berdasarkan strategy.
type VLANResolveContext struct {
	PackageID  string // untuk strategy per_paket
	ODPID      string // untuk strategy per_odp
	CustomerID string // untuk strategy per_pelanggan
}

// --- VLAN Entitas ---

// VLAN merepresentasikan VLAN per OLT per tenant.
// Setiap OLT memiliki daftar VLAN sendiri yang diisolasi via RLS.
type VLAN struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	OLTID       string     `json:"olt_id"`
	VLANID      int        `json:"vlan_id"`
	Name        string     `json:"name"`
	VLANType    VLANType   `json:"vlan_type"`
	Description string     `json:"description,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
