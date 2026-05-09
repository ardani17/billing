package crypto

import (
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

// printableStringGen menghasilkan string non-kosong yang hanya berisi
// karakter printable (ASCII 32-126). Ini mensimulasikan credential OLT
// yang realistis: password, community string, username.
func printableStringGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Panjang minimal 1, maksimal 128 karakter (batas wajar credential)
		length := rapid.IntRange(1, 128).Draw(t, "length")
		runes := make([]rune, length)
		for i := range runes {
			// Karakter printable ASCII: spasi (32) sampai tilde (126)
			runes[i] = rune(rapid.IntRange(32, 126).Draw(t, "char"))
		}
		return string(runes)
	})
}

// TestProperty_OLT_CredentialEncryptionRoundTrip memverifikasi bahwa untuk
// sembarang credential string non-kosong (printable), encrypt lalu decrypt
// menghasilkan credential asli. Properti ini memastikan tidak ada data loss
// saat penyimpanan dan pengambilan credential OLT.
//
// **Memvalidasi: Kebutuhan 18.1, 18.2, 18.5**
func TestProperty_OLT_CredentialEncryptionRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat key acak 32 bytes (AES-256)
		key := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "key")
		// Buat credential non-kosong dengan karakter printable
		credential := printableStringGen().Draw(t, "credential")

		enc, err := NewAESEncryptor(key)
		if err != nil {
			t.Fatalf("gagal membuat encryptor: %v", err)
		}

		// Encrypt credential
		ciphertext, err := enc.Encrypt(credential)
		if err != nil {
			t.Fatalf("encrypt gagal: %v", err)
		}

		// Decrypt harus mengembalikan credential asli
		decrypted, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("decrypt gagal: %v", err)
		}

		if decrypted != credential {
			t.Errorf(
				"round-trip gagal: credential=%q, decrypted=%q",
				credential, decrypted,
			)
		}

		// Ciphertext harus berbeda dari plaintext (enkripsi nyata)
		if ciphertext == credential {
			t.Errorf(
				"ciphertext sama dengan credential: %q",
				credential,
			)
		}

		// Verifikasi credential hanya berisi karakter printable
		for _, r := range credential {
			if !unicode.IsPrint(r) {
				t.Errorf("credential mengandung karakter non-printable: %U", r)
			}
		}
	})
}

// TestProperty_OLT_EncryptionProducesDifferentCiphertexts memverifikasi bahwa
// dua kali enkripsi credential yang sama menghasilkan ciphertext berbeda
// karena random nonce unik per operasi. Ini penting untuk keamanan:
// penyerang tidak bisa mendeteksi credential identik dari ciphertext.
//
// **Memvalidasi: Kebutuhan 18.1, 18.2, 18.5**
func TestProperty_OLT_EncryptionProducesDifferentCiphertexts(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat key acak 32 bytes
		key := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "key")
		// Buat credential non-kosong
		credential := printableStringGen().Draw(t, "credential")

		enc, err := NewAESEncryptor(key)
		if err != nil {
			t.Fatalf("gagal membuat encryptor: %v", err)
		}

		// Encrypt dua kali dengan credential yang sama
		ct1, err := enc.Encrypt(credential)
		if err != nil {
			t.Fatalf("encrypt pertama gagal: %v", err)
		}

		ct2, err := enc.Encrypt(credential)
		if err != nil {
			t.Fatalf("encrypt kedua gagal: %v", err)
		}

		// Dua ciphertext harus berbeda karena nonce acak
		if ct1 == ct2 {
			t.Errorf(
				"dua kali encrypt credential %q menghasilkan ciphertext identik: %q",
				credential, ct1,
			)
		}

		// Keduanya harus bisa didekripsi ke credential asli
		dec1, err := enc.Decrypt(ct1)
		if err != nil {
			t.Fatalf("decrypt ct1 gagal: %v", err)
		}
		dec2, err := enc.Decrypt(ct2)
		if err != nil {
			t.Fatalf("decrypt ct2 gagal: %v", err)
		}

		if dec1 != credential || dec2 != credential {
			t.Errorf(
				"decrypt gagal: credential=%q, dec1=%q, dec2=%q",
				credential, dec1, dec2,
			)
		}
	})
}

// =============================================================================
// Example-based tests - credential OLT tipikal
// =============================================================================

// TestExample_OLT_SNMPCommunityEncryption menguji enkripsi/dekripsi
// SNMP community string yang umum dipakai di OLT.
func TestExample_OLT_SNMPCommunityEncryption(t *testing.T) {
	// Key tetap untuk test deterministik
	key := []byte("01234567890123456789012345678901") // 32 bytes

	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("gagal membuat encryptor: %v", err)
	}

	// Community string tipikal OLT
	communities := []string{
		"public",
		"private",
		"isp-monitoring-2024",
		"zte@read-only",
		"huawei#snmpv2c!secure",
	}

	for _, community := range communities {
		t.Run(community, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(community)
			if err != nil {
				t.Fatalf("encrypt gagal untuk %q: %v", community, err)
			}

			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decrypt gagal untuk %q: %v", community, err)
			}

			if decrypted != community {
				t.Errorf("round-trip gagal: want=%q, got=%q", community, decrypted)
			}
		})
	}
}

// TestExample_OLT_CLIPasswordEncryption menguji enkripsi/dekripsi
// password CLI (SSH/Telnet) yang umum dipakai di OLT.
func TestExample_OLT_CLIPasswordEncryption(t *testing.T) {
	key := []byte("01234567890123456789012345678901")

	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("gagal membuat encryptor: %v", err)
	}

	// Password CLI tipikal OLT
	passwords := []string{
		"admin123",
		"Zte@C320!2024",
		"huawei-ma5608t-enable",
		"P@ssw0rd#FiberHome",
		"vsol-v1600g-cli!",
		"hsgq_admin_2024",
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(password)
			if err != nil {
				t.Fatalf("encrypt gagal untuk %q: %v", password, err)
			}

			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decrypt gagal untuk %q: %v", password, err)
			}

			if decrypted != password {
				t.Errorf("round-trip gagal: want=%q, got=%q", password, decrypted)
			}
		})
	}
}
