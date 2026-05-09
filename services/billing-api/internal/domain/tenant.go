// Package domain berisi entity dan interface bisnis untuk billing-api.
// Layer domain tidak boleh mengimpor package dari layer lain (handler, repositori).
package domain

import "time"

// Tenant merepresentasikan operator ISP atau RT/RW Net yang berlangganan ISPBoss.
// Setiap tenant memiliki data terisolasi melalui mekanisme RLS di PostgreSQL.
type Tenant struct {
	// ID adalah UUID unik untuk tenant
	ID string `json:"id"`

	// Name adalah nama perusahaan atau organisasi tenant
	Name string `json:"name"`

	// Domain adalah domain kustom untuk white-label (opsional)
	Domain string `json:"domain,omitempty"`

	// Plan adalah paket langganan tenant (starter, pro, enterprise)
	Plan string `json:"plan"`

	// Status menunjukkan status tenant (active, suspended, cancelled)
	Status string `json:"status"`

	// CreatedAt adalah waktu pembuatan record
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt adalah waktu terakhir record diperbarui
	UpdatedAt time.Time `json:"updated_at"`
}
