package domain

import "time"

// --- Splitter Type Constants ---

// Tipe splitter yang didukung untuk ODP.
const (
	// SplitterType1x4 adalah splitter dengan rasio 1:4 (kapasitas 4 port).
	SplitterType1x4 = "1:4"

	// SplitterType1x8 adalah splitter dengan rasio 1:8 (kapasitas 8 port).
	SplitterType1x8 = "1:8"

	// SplitterType1x16 adalah splitter dengan rasio 1:16 (kapasitas 16 port).
	SplitterType1x16 = "1:16"

	// SplitterType1x32 adalah splitter dengan rasio 1:32 (kapasitas 32 port).
	SplitterType1x32 = "1:32"
)

// --- ODP Entity ---

// ODP merepresentasikan Optical Distribution Point (splitter) per tenant.
// Setiap ODP terhubung ke satu OLT pada PON port tertentu dan memiliki
// kapasitas port sesuai tipe splitter yang digunakan.
type ODP struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	OLTID        string     `json:"olt_id"`
	PONPortIndex int        `json:"pon_port_index"`
	Name         string     `json:"name"`
	SplitterType string     `json:"splitter_type"`
	Capacity     int        `json:"capacity"`
	UsedPorts    int        `json:"used_ports"`
	Address      string     `json:"address,omitempty"`
	Latitude     *float64   `json:"latitude,omitempty"`
	Longitude    *float64   `json:"longitude,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// --- Splitter Capacity Helper ---

// SplitterCapacity mengembalikan kapasitas port berdasarkan tipe splitter.
// Mengembalikan 0 jika tipe splitter tidak dikenali.
func SplitterCapacity(splitterType string) int {
	switch splitterType {
	case SplitterType1x4:
		return 4
	case SplitterType1x8:
		return 8
	case SplitterType1x16:
		return 16
	case SplitterType1x32:
		return 32
	default:
		return 0
	}
}
