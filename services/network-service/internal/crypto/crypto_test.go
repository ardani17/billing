package crypto

import (
	"bytes"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Feature: mikrotik-router, Property 9: Encryption nonce uniqueness
// =============================================================================

// TestProperty_EncryptionNonceUniqueness memverifikasi bahwa untuk sembarang
// plaintext dan sembarang key 32-byte yang valid, dua kali pemanggilan
// Encrypt(plaintext) menghasilkan ciphertext yang berbeda karena nonce acak
// yang unik per operasi.
//
// **Validates: Requirements 8.5**
func TestProperty_EncryptionNonceUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate key acak 32 bytes
		key := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "key")
		// Generate plaintext acak
		plaintext := rapid.String().Draw(t, "plaintext")

		enc, err := NewAESEncryptor(key)
		if err != nil {
			t.Fatalf("gagal membuat encryptor: %v", err)
		}

		// Encrypt dua kali dengan plaintext yang sama
		ct1, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt pertama gagal: %v", err)
		}

		ct2, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt kedua gagal: %v", err)
		}

		// Dua ciphertext harus berbeda karena nonce acak
		if ct1 == ct2 {
			t.Errorf(
				"dua kali encrypt plaintext %q menghasilkan ciphertext identik: %q",
				plaintext, ct1,
			)
		}
	})
}

// =============================================================================
// Feature: mikrotik-router, Property 10: Encryption round-trip
// =============================================================================

// TestProperty_EncryptionRoundTrip memverifikasi bahwa untuk sembarang string
// valid dan sembarang key 32-byte yang valid, Decrypt(Encrypt(plaintext))
// mengembalikan plaintext asli. Untuk input non-kosong, ciphertext harus
// berbeda dari plaintext.
//
// **Validates: Requirements 8.6**
func TestProperty_EncryptionRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate key acak 32 bytes
		key := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "key")
		// Generate plaintext acak
		plaintext := rapid.String().Draw(t, "plaintext")

		enc, err := NewAESEncryptor(key)
		if err != nil {
			t.Fatalf("gagal membuat encryptor: %v", err)
		}

		// Encrypt plaintext
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt gagal: %v", err)
		}

		// Decrypt harus mengembalikan plaintext asli
		decrypted, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("decrypt gagal: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf(
				"round-trip gagal: plaintext=%q, decrypt=%q",
				plaintext, decrypted,
			)
		}

		// Untuk input non-kosong, ciphertext harus berbeda dari plaintext
		if len(plaintext) > 0 && ciphertext == plaintext {
			t.Errorf(
				"ciphertext sama dengan plaintext untuk input non-kosong: %q",
				plaintext,
			)
		}
	})
}

// =============================================================================
// Feature: mikrotik-router, Property 11: Wrong key decryption error safety
// =============================================================================

// TestProperty_WrongKeyDecryptionErrorSafety memverifikasi bahwa untuk
// sembarang ciphertext yang dienkripsi dengan key K1 dan sembarang key K2
// yang berbeda (K2 ≠ K1, keduanya 32 bytes), Decrypt(ciphertext, K2)
// mengembalikan error. Pesan error tidak boleh mengandung bytes key atau
// plaintext asli.
//
// **Validates: Requirements 8.7**
func TestProperty_WrongKeyDecryptionErrorSafety(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate dua key berbeda masing-masing 32 bytes
		key1 := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "key1")
		key2 := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "key2")

		// Pastikan key1 != key2; jika sama, skip iterasi ini
		if bytes.Equal(key1, key2) {
			return
		}

		// Generate plaintext acak
		plaintext := rapid.String().Draw(t, "plaintext")

		// Encrypt dengan key1
		enc1, err := NewAESEncryptor(key1)
		if err != nil {
			t.Fatalf("gagal membuat encryptor key1: %v", err)
		}

		ciphertext, err := enc1.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt gagal: %v", err)
		}

		// Decrypt dengan key2 yang berbeda — harus error
		enc2, err := NewAESEncryptor(key2)
		if err != nil {
			t.Fatalf("gagal membuat encryptor key2: %v", err)
		}

		_, decErr := enc2.Decrypt(ciphertext)
		if decErr == nil {
			t.Error("decrypt dengan key berbeda seharusnya mengembalikan error")
			return
		}

		errMsg := decErr.Error()

		// Pesan error tidak boleh mengandung bytes key1
		if len(key1) > 0 && strings.Contains(errMsg, string(key1)) {
			t.Errorf("pesan error mengandung bytes key1: %q", errMsg)
		}

		// Pesan error tidak boleh mengandung bytes key2
		if len(key2) > 0 && strings.Contains(errMsg, string(key2)) {
			t.Errorf("pesan error mengandung bytes key2: %q", errMsg)
		}

		// Pesan error tidak boleh mengandung plaintext asli.
		// Hanya cek untuk plaintext >= 3 karakter agar menghindari false positive
		// dari karakter tunggal yang kebetulan muncul di pesan error bahasa Indonesia.
		if len(plaintext) >= 3 && strings.Contains(errMsg, plaintext) {
			t.Errorf(
				"pesan error mengandung plaintext asli %q: %q",
				plaintext, errMsg,
			)
		}
	})
}
