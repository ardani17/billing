package domain

// =============================================================================
// Provisioning Adapter Params — parameter untuk operasi provisioning via adapter
// =============================================================================

// AddONTParams berisi parameter untuk menambahkan ONT ke OLT.
// Digunakan oleh OLTAdapter.AddONT() untuk membangun CLI command per brand.
type AddONTParams struct {
	PONPortIndex     int    // indeks PON port
	ONTIndex         int    // indeks ONT pada port (auto-assign jika 0)
	SerialNumber     string // serial number ONT
	LineProfileID    int    // ID line profile di OLT
	ServiceProfileID int    // ID service profile di OLT
	Description      string // deskripsi opsional
}

// RemoveONTParams berisi parameter untuk menghapus ONT dari OLT.
// Digunakan oleh OLTAdapter.RemoveONT() saat decommission.
type RemoveONTParams struct {
	PONPortIndex int // indeks PON port
	ONTIndex     int // indeks ONT pada port
}

// AddServicePortParams berisi parameter untuk menambahkan service-port.
// Digunakan oleh OLTAdapter.AddServicePort() untuk VLAN assignment.
type AddServicePortParams struct {
	PONPortIndex int // indeks PON port
	ONTIndex     int // indeks ONT
	VLANID       int // VLAN ID untuk assignment
	GemPort      int // GEM port (default 1)
}

// RemoveServicePortParams berisi parameter untuk menghapus service-port.
// Digunakan oleh OLTAdapter.RemoveServicePort() saat decommission.
type RemoveServicePortParams struct {
	PONPortIndex int // indeks PON port
	ONTIndex     int // indeks ONT
	VLANID       int // VLAN ID yang di-remove
}

// RebootONTParams berisi parameter untuk reboot ONT.
// Digunakan oleh OLTAdapter.RebootONT() untuk troubleshooting.
type RebootONTParams struct {
	PONPortIndex int // indeks PON port
	ONTIndex     int // indeks ONT
}

// =============================================================================
// Provisioning Result — hasil eksekusi provisioning command
// =============================================================================

// ProvisioningResult berisi hasil eksekusi provisioning command ke OLT.
// CommandsSent berisi CLI commands yang dikirim, Responses berisi output dari OLT.
type ProvisioningResult struct {
	Success      bool     `json:"success"`
	CommandsSent []string `json:"commands_sent"`
	Responses    []string `json:"responses"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// =============================================================================
// Unregistered ONT — ONT terdeteksi di OLT tapi belum terdaftar di database
// =============================================================================

// UnregisteredONT berisi informasi ONT yang terdeteksi oleh sync engine
// tapi belum memiliki record di database.
type UnregisteredONT struct {
	SerialNumber string `json:"serial_number"`
	PONPortIndex int    `json:"pon_port_index"`
	ONTIndex     int    `json:"ont_index"`
}
