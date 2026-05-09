package domain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// Konstanta Share Link - konfigurasi token generation
// =============================================================================

// ShareTokenLength adalah panjang token dalam bytes sebelum di-encode ke hex.
// Token 32 bytes menghasilkan 64 karakter hex string.
const ShareTokenLength = 32

// =============================================================================
// MapShareLink Entitas - link hanya baca ke peta untuk berbagi eksternal
// =============================================================================

// MapShareLink merepresentasikan share link hanya baca ke peta.
// Admin dapat membuat link dengan opsi expiry dan password untuk
// berbagi visualisasi jaringan dengan investor, partner, atau pihak eksternal.
// Data diisolasi per tenant via RLS di PostgreSQL.
type MapShareLink struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	Token         string          `json:"token"`
	VisibleLayers json.RawMessage `json:"visible_layers"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
	PasswordHash  *string         `json:"-"`
	AccessCount   int             `json:"access_count"`
	CreatedBy     string          `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

// =============================================================================
// Token Generation - helper untuk membuat secure random token
// =============================================================================

// GenerateShareToken menghasilkan token acak yang aman secara kriptografis.
// Menggunakan crypto/rand untuk menghasilkan 32 bytes random,
// kemudian di-encode ke hex string (64 karakter).
// Token ini digunakan sebagai identifier unik untuk share link.
func GenerateShareToken() (string, error) {
	bytes := make([]byte, ShareTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("gagal generate share token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// IsExpired memeriksa apakah share link sudah kedaluwarsa.
// Mengembalikan false jika ExpiresAt nil (tidak ada expiry).
func (s *MapShareLink) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}
