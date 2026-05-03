package domain

import "time"

// PasswordReset merepresentasikan token reset password.
type PasswordReset struct {
	// ID adalah UUID unik untuk token reset
	ID string `json:"id"`

	// UserID adalah UUID user yang meminta reset password
	UserID string `json:"user_id"`

	// TokenHash adalah hash SHA-256 dari token (tidak di-expose ke JSON)
	TokenHash string `json:"-"`

	// ExpiresAt adalah waktu kedaluwarsa token (1 jam dari pembuatan)
	ExpiresAt time.Time `json:"expires_at"`

	// Used menunjukkan apakah token sudah digunakan
	Used bool `json:"used"`

	// CreatedAt adalah waktu pembuatan token
	CreatedAt time.Time `json:"created_at"`
}

// EmailVerification merepresentasikan token verifikasi email.
type EmailVerification struct {
	// ID adalah UUID unik untuk token verifikasi
	ID string `json:"id"`

	// UserID adalah UUID user yang perlu verifikasi email
	UserID string `json:"user_id"`

	// TokenHash adalah hash SHA-256 dari token (tidak di-expose ke JSON)
	TokenHash string `json:"-"`

	// ExpiresAt adalah waktu kedaluwarsa token (24 jam dari pembuatan)
	ExpiresAt time.Time `json:"expires_at"`

	// Used menunjukkan apakah token sudah digunakan
	Used bool `json:"used"`

	// CreatedAt adalah waktu pembuatan token
	CreatedAt time.Time `json:"created_at"`
}
