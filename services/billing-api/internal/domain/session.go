package domain

import "time"

// Session merepresentasikan satu sesi login aktif dari satu device.
type Session struct {
	// ID adalah UUID unik untuk session
	ID string `json:"id"`

	// UserID adalah UUID user pemilik session
	UserID string `json:"user_id"`

	// TokenHash adalah hash SHA-256 dari refresh token (tidak di-expose ke JSON)
	TokenHash string `json:"-"`

	// DeviceInfo adalah informasi device/browser yang digunakan (opsional)
	DeviceInfo string `json:"device_info,omitempty"`

	// IPAddress adalah alamat IP saat session dibuat (opsional)
	IPAddress string `json:"ip_address,omitempty"`

	// ExpiresAt adalah waktu kedaluwarsa session
	ExpiresAt time.Time `json:"expires_at"`

	// CreatedAt adalah waktu pembuatan session
	CreatedAt time.Time `json:"created_at"`

	// IsCurrent menunjukkan apakah session ini adalah session yang sedang aktif (opsional, computed)
	IsCurrent bool `json:"is_current,omitempty"`
}
