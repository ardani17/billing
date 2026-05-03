package usecase

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/crypto"
)

// =============================================================================
// Feature: mikrotik-vpn, Property 2: Key encryption round-trip
// =============================================================================

// testEncryptionKey adalah kunci AES-256 (32 bytes) untuk testing.
var testEncryptionKey = []byte("01234567890123456789012345678901")

// newTestEncryptor membuat CredentialEncryptor untuk testing.
func newTestEncryptor(t *testing.T) *crypto.AESEncryptor {
	t.Helper()
	enc, err := crypto.NewAESEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("gagal membuat encryptor: %v", err)
	}
	return enc
}

// TestVPNProperty_KeyEncryptionRoundTrip memverifikasi bahwa untuk sembarang
// key string yang valid (WireGuard private key, pre-shared key, L2TP password,
// atau IPSec PSK), mengenkripsi dengan CredentialEncryptor.Encrypt() lalu
// mendekripsi dengan CredentialEncryptor.Decrypt() menghasilkan key string asli.
// Bentuk terenkripsi harus berbeda dari plaintext.
//
// **Validates: Requirements 4.2, 11.1, 11.6**
func TestVPNProperty_KeyEncryptionRoundTrip(t *testing.T) {
	gen := NewVPNKeyGenerator()
	enc := newTestEncryptor(t)

	rapid.Check(t, func(t *rapid.T) {
		// Pilih jenis key secara acak
		keyType := rapid.SampledFrom([]string{
			"wireguard_private", "preshared", "l2tp_password", "ipsec_psk",
		}).Draw(t, "keyType")

		var plaintext string

		switch keyType {
		case "wireguard_private":
			_, privKey, err := gen.GenerateWireGuardKeyPair()
			if err != nil {
				t.Fatalf("gagal generate WireGuard key pair: %v", err)
			}
			plaintext = privKey

		case "preshared":
			psk, err := gen.GeneratePreSharedKey()
			if err != nil {
				t.Fatalf("gagal generate pre-shared key: %v", err)
			}
			plaintext = psk

		case "l2tp_password":
			tunnelName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "tunnelName")
			_, password, err := gen.GenerateCredentials(tunnelName)
			if err != nil {
				t.Fatalf("gagal generate credentials: %v", err)
			}
			plaintext = password

		case "ipsec_psk":
			psk, err := gen.GenerateIPSecPSK()
			if err != nil {
				t.Fatalf("gagal generate IPSec PSK: %v", err)
			}
			plaintext = psk
		}

		// Enkripsi plaintext
		encrypted, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("gagal mengenkripsi key: %v", err)
		}

		// Properti: bentuk terenkripsi harus berbeda dari plaintext
		if encrypted == plaintext {
			t.Fatalf("hasil enkripsi sama dengan plaintext untuk keyType=%s", keyType)
		}

		// Dekripsi kembali
		decrypted, err := enc.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("gagal mendekripsi key: %v", err)
		}

		// Properti: round-trip harus menghasilkan key asli
		if decrypted != plaintext {
			t.Fatalf(
				"round-trip gagal untuk keyType=%s: plaintext=%q, decrypted=%q",
				keyType, plaintext, decrypted,
			)
		}
	})
}

// =============================================================================
// Unit Tests — WireGuard Key Pair
// =============================================================================

// TestGenerateWireGuardKeyPair_ValidBase64And32Bytes memverifikasi bahwa
// WireGuard key pair menghasilkan public key != private key, keduanya valid
// base64, dan masing-masing 32 bytes saat di-decode.
func TestGenerateWireGuardKeyPair_ValidBase64And32Bytes(t *testing.T) {
	gen := NewVPNKeyGenerator()

	rapid.Check(t, func(t *rapid.T) {
		pubKey, privKey, err := gen.GenerateWireGuardKeyPair()
		if err != nil {
			t.Fatalf("gagal generate key pair: %v", err)
		}

		// Public key != private key
		if pubKey == privKey {
			t.Fatal("public key sama dengan private key")
		}

		// Public key harus valid base64
		pubBytes, err := base64.StdEncoding.DecodeString(pubKey)
		if err != nil {
			t.Fatalf("public key bukan base64 valid: %v", err)
		}

		// Private key harus valid base64
		privBytes, err := base64.StdEncoding.DecodeString(privKey)
		if err != nil {
			t.Fatalf("private key bukan base64 valid: %v", err)
		}

		// Keduanya harus 32 bytes saat di-decode
		if len(pubBytes) != 32 {
			t.Fatalf("public key harus 32 bytes, dapat %d", len(pubBytes))
		}
		if len(privBytes) != 32 {
			t.Fatalf("private key harus 32 bytes, dapat %d", len(privBytes))
		}
	})
}

// =============================================================================
// Unit Tests — PreSharedKey
// =============================================================================

// TestGeneratePreSharedKey_ValidBase64And32Bytes memverifikasi bahwa
// pre-shared key adalah valid base64 dan 32 bytes saat di-decode.
func TestGeneratePreSharedKey_ValidBase64And32Bytes(t *testing.T) {
	gen := NewVPNKeyGenerator()

	rapid.Check(t, func(t *rapid.T) {
		psk, err := gen.GeneratePreSharedKey()
		if err != nil {
			t.Fatalf("gagal generate pre-shared key: %v", err)
		}

		// Harus valid base64
		decoded, err := base64.StdEncoding.DecodeString(psk)
		if err != nil {
			t.Fatalf("pre-shared key bukan base64 valid: %v", err)
		}

		// Harus 32 bytes
		if len(decoded) != 32 {
			t.Fatalf("pre-shared key harus 32 bytes, dapat %d", len(decoded))
		}
	})
}

// =============================================================================
// Unit Tests — GenerateCredentials
// =============================================================================

// TestGenerateCredentials_UsernameContainsTunnelName memverifikasi bahwa
// username mengandung tunnel name dan password adalah hex-encoded.
func TestGenerateCredentials_UsernameContainsTunnelName(t *testing.T) {
	gen := NewVPNKeyGenerator()

	rapid.Check(t, func(t *rapid.T) {
		tunnelName := rapid.StringMatching(`[a-z]{3,20}`).Draw(t, "tunnelName")

		username, password, err := gen.GenerateCredentials(tunnelName)
		if err != nil {
			t.Fatalf("gagal generate credentials: %v", err)
		}

		// Username harus mengandung tunnel name
		if !strings.Contains(username, tunnelName) {
			t.Fatalf("username %q tidak mengandung tunnel name %q", username, tunnelName)
		}

		// Username harus diawali "vpn-"
		if !strings.HasPrefix(username, "vpn-") {
			t.Fatalf("username %q tidak diawali 'vpn-'", username)
		}

		// Password harus hex-encoded (hanya karakter 0-9, a-f)
		_, err = hex.DecodeString(password)
		if err != nil {
			t.Fatalf("password %q bukan hex valid: %v", password, err)
		}

		// Password hex-decoded harus 32 bytes (64 karakter hex)
		if len(password) != 64 {
			t.Fatalf("password harus 64 karakter hex (32 bytes), dapat %d", len(password))
		}
	})
}

// =============================================================================
// Unit Tests — GenerateIPSecPSK
// =============================================================================

// TestGenerateIPSecPSK_ValidBase64And32Bytes memverifikasi bahwa
// IPSec PSK adalah valid base64 dan 32 bytes saat di-decode.
func TestGenerateIPSecPSK_ValidBase64And32Bytes(t *testing.T) {
	gen := NewVPNKeyGenerator()

	rapid.Check(t, func(t *rapid.T) {
		psk, err := gen.GenerateIPSecPSK()
		if err != nil {
			t.Fatalf("gagal generate IPSec PSK: %v", err)
		}

		// Harus valid base64
		decoded, err := base64.StdEncoding.DecodeString(psk)
		if err != nil {
			t.Fatalf("IPSec PSK bukan base64 valid: %v", err)
		}

		// Harus 32 bytes
		if len(decoded) != 32 {
			t.Fatalf("IPSec PSK harus 32 bytes, dapat %d", len(decoded))
		}
	})
}
