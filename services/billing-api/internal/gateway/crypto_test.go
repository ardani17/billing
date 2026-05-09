package gateway

import (
	"crypto/rand"
	"errors"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Unit test untuk utilitas enkripsi AES-256-GCM dan masking API key
// =============================================================================

// generateTestKey membuat master key 32 bytes untuk testing.
func generateTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("gagal generate test key: %v", err)
	}
	return key
}

func TestEncryptDecryptAESGCM_RoundTrip(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "xnd_production_abc123xyz_secret_key"

	// Enkripsi
	ciphertext, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESGCM gagal: %v", err)
	}

	// Ciphertext harus berbeda dari plaintext
	if ciphertext == plaintext {
		t.Error("ciphertext tidak boleh sama dengan plaintext")
	}

	// Dekripsi
	result, err := DecryptAESGCM(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptAESGCM gagal: %v", err)
	}

	// Hasil dekripsi harus sama dengan plaintext asli
	if result != plaintext {
		t.Errorf("hasil dekripsi = %q, ingin %q", result, plaintext)
	}
}

func TestEncryptAESGCM_EmptyPlaintext(t *testing.T) {
	key := generateTestKey(t)
	ciphertext, err := EncryptAESGCM("", key)
	if err != nil {
		t.Fatalf("EncryptAESGCM gagal untuk string kosong: %v", err)
	}
	result, err := DecryptAESGCM(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptAESGCM gagal untuk string kosong: %v", err)
	}
	if result != "" {
		t.Errorf("hasil dekripsi = %q, ingin string kosong", result)
	}
}

func TestAESGCM_InvalidKeyLength(t *testing.T) {
	shortKey := make([]byte, 16)
	_, err := EncryptAESGCM("test", shortKey)
	if err == nil {
		t.Fatal("seharusnya error encrypt untuk key 16 bytes")
	}
	if !errors.Is(err, domain.ErrEncryptionFailed) {
		t.Errorf("error = %v, ingin ErrEncryptionFailed", err)
	}
	_, err = DecryptAESGCM("dGVzdA==", shortKey)
	if err == nil {
		t.Fatal("seharusnya error decrypt untuk key 16 bytes")
	}
	if !errors.Is(err, domain.ErrDecryptionFailed) {
		t.Errorf("error = %v, ingin ErrDecryptionFailed", err)
	}
}

func TestDecryptAESGCM_ErrorCases(t *testing.T) {
	key := generateTestKey(t)

	_, err := DecryptAESGCM("bukan-base64!!!", key)
	if !errors.Is(err, domain.ErrDecryptionFailed) {
		t.Errorf("base64 invalid: error = %v, ingin ErrDecryptionFailed", err)
	}

	// Key berbeda harus gagal
	key2 := generateTestKey(t)
	ct, _ := EncryptAESGCM("rahasia", key)
	_, err = DecryptAESGCM(ct, key2)
	if !errors.Is(err, domain.ErrDecryptionFailed) {
		t.Errorf("wrong key: error = %v, ingin ErrDecryptionFailed", err)
	}

	// Ciphertext terlalu pendek (kurang dari 12 bytes nonce)
	_, err = DecryptAESGCM("c2hvcnQ=", key)
	if !errors.Is(err, domain.ErrDecryptionFailed) {
		t.Errorf("short ciphertext: error = %v, ingin ErrDecryptionFailed", err)
	}
}

func TestEncryptAESGCM_ProducesDifferentCiphertexts(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "same-plaintext"

	// Enkripsi dua kali harus menghasilkan ciphertext berbeda (nonce acak)
	ct1, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESGCM pertama gagal: %v", err)
	}

	ct2, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESGCM kedua gagal: %v", err)
	}

	if ct1 == ct2 {
		t.Error("dua enkripsi plaintext yang sama harus menghasilkan ciphertext berbeda")
	}
}

// =============================================================================
// Unit test untuk MaskAPIKey
// =============================================================================

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "key panjang standar",
			input:  "xnd_production_abc123xyz",
			expect: "********************3xyz",
		},
		{
			name:   "tepat 4 karakter",
			input:  "abcd",
			expect: "abcd",
		},
		{
			name:   "5 karakter",
			input:  "abcde",
			expect: "*bcde",
		},
		{
			name:   "kurang dari 4 karakter",
			input:  "abc",
			expect: "abc",
		},
		{
			name:   "string kosong",
			input:  "",
			expect: "",
		},
		{
			name:   "1 karakter",
			input:  "x",
			expect: "x",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskAPIKey(tc.input)
			if result != tc.expect {
				t.Errorf("MaskAPIKey(%q) = %q, ingin %q", tc.input, result, tc.expect)
			}
		})
	}
}

// =============================================================================
// =============================================================================

// TestProperty_EncryptDecryptRoundTrip memverifikasi bahwa untuk sembarang
// plaintext dan sembarang master key 32 bytes, EncryptAESGCM menghasilkan
// ciphertext yang ketika didekripsi dengan DecryptAESGCM mengembalikan
// plaintext asli. Juga memverifikasi bahwa ciphertext berbeda dari plaintext
// (enkripsi benar-benar mentransformasi data).
//
// **Memvalidasi: Kebutuhan 1.6**
func TestProperty_EncryptDecryptRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat master key acak 32 bytes
		masterKey := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "masterKey")

		// Buat plaintext acak
		plaintext := rapid.String().Draw(t, "plaintext")

		// Enkripsi plaintext
		ciphertext, err := EncryptAESGCM(plaintext, masterKey)
		if err != nil {
			t.Fatalf("EncryptAESGCM gagal: %v", err)
		}

		// Ciphertext harus berbeda dari plaintext (enkripsi mentransformasi data)
		if len(plaintext) > 0 && ciphertext == plaintext {
			t.Error("ciphertext tidak boleh sama dengan plaintext untuk non-empty string")
		}

		// Dekripsi ciphertext
		result, err := DecryptAESGCM(ciphertext, masterKey)
		if err != nil {
			t.Fatalf("DecryptAESGCM gagal: %v", err)
		}

		// Hasil dekripsi harus sama persis dengan plaintext asli
		if result != plaintext {
			t.Errorf("round-trip gagal: plaintext = %q, hasil dekripsi = %q", plaintext, result)
		}
	})
}

// =============================================================================
// =============================================================================

// TestProperty_MaskAPIKey memverifikasi bahwa untuk sembarang API key string:
//   - Jika panjang >= 4: MaskAPIKey mengembalikan string dengan 4 karakter terakhir
//     sama dengan aslinya, semua karakter sebelumnya adalah asterisk (*), dan
//     panjang total sama dengan panjang key asli.
//   - Jika panjang < 4: string dikembalikan apa adanya tanpa masking.
//
// **Memvalidasi: Kebutuhan 1.4**
func TestProperty_MaskAPIKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat API key string acak
		apiKey := rapid.String().Draw(t, "apiKey")

		// Panggil MaskAPIKey
		masked := MaskAPIKey(apiKey)

		if len(apiKey) < 4 {
			// Untuk key pendek (< 4 karakter), harus dikembalikan apa adanya
			if masked != apiKey {
				t.Errorf("key pendek (%d chars): MaskAPIKey(%q) = %q, ingin %q",
					len(apiKey), apiKey, masked, apiKey)
			}
		} else {
			// Panjang total harus sama dengan key asli
			if len(masked) != len(apiKey) {
				t.Errorf("panjang berbeda: len(MaskAPIKey(%q)) = %d, ingin %d",
					apiKey, len(masked), len(apiKey))
			}

			// 4 karakter terakhir harus sama dengan key asli
			lastFourMasked := masked[len(masked)-4:]
			lastFourOriginal := apiKey[len(apiKey)-4:]
			if lastFourMasked != lastFourOriginal {
				t.Errorf("4 karakter terakhir berbeda: masked=%q, original=%q",
					lastFourMasked, lastFourOriginal)
			}

			// Semua karakter sebelum 4 terakhir harus asterisk
			prefix := masked[:len(masked)-4]
			for i, ch := range prefix {
				if ch != '*' {
					t.Errorf("karakter di posisi %d = %q, ingin '*' (masked=%q)",
						i, string(ch), masked)
				}
			}
		}
	})
}
