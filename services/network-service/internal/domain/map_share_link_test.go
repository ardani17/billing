package domain

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestGenerateShareToken_PanjangDanFormat(t *testing.T) {
	token, err := GenerateShareToken()
	if err != nil {
		t.Fatalf("GenerateShareToken() error: %v", err)
	}

	// Token harus 64 karakter hex (32 bytes * 2)
	expectedLen := ShareTokenLength * 2
	if len(token) != expectedLen {
		t.Errorf("panjang token = %d, expected %d", len(token), expectedLen)
	}

	// Token harus valid hex string
	_, err = hex.DecodeString(token)
	if err != nil {
		t.Errorf("token bukan valid hex: %v", err)
	}
}

func TestGenerateShareToken_Unik(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateShareToken()
		if err != nil {
			t.Fatalf("GenerateShareToken() error pada iterasi %d: %v", i, err)
		}
		if tokens[token] {
			t.Fatalf("token duplikat ditemukan pada iterasi %d: %s", i, token)
		}
		tokens[token] = true
	}
}

func TestMapShareLink_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		link     MapShareLink
		expected bool
	}{
		{
			name:     "tanpa expiry — tidak expired",
			link:     MapShareLink{ExpiresAt: nil},
			expected: false,
		},
		{
			name: "expiry di masa depan — tidak expired",
			link: MapShareLink{ExpiresAt: func() *time.Time {
				t := time.Now().Add(24 * time.Hour)
				return &t
			}()},
			expected: false,
		},
		{
			name: "expiry di masa lalu — expired",
			link: MapShareLink{ExpiresAt: func() *time.Time {
				t := time.Now().Add(-24 * time.Hour)
				return &t
			}()},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.link.IsExpired()
			if got != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestShareTokenLength_Konstanta(t *testing.T) {
	// Verifikasi konstanta ShareTokenLength = 32 bytes
	if ShareTokenLength != 32 {
		t.Errorf("ShareTokenLength = %d, expected 32", ShareTokenLength)
	}
}
