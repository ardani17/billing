package gateway

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Utilitas enkripsi AES-256-GCM untuk API key dan webhook secret
// =============================================================================

// EncryptAESGCM mengenkripsi plaintext menggunakan AES-256-GCM.
// masterKey harus 32 bytes (256 bit).
// Mengembalikan base64-encoded string berisi (nonce + ciphertext + tag).
func EncryptAESGCM(plaintext string, masterKey []byte) (string, error) {
	// Validasi panjang master key harus 32 bytes
	if len(masterKey) != 32 {
		return "", fmt.Errorf("%w: master key harus 32 bytes", domain.ErrEncryptionFailed)
	}

	// Buat cipher block dari master key
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat cipher: %v", domain.ErrEncryptionFailed, err)
	}

	// Buat GCM wrapper dari cipher block
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat GCM: %v", domain.ErrEncryptionFailed, err)
	}

	// Buat nonce acak (12 bytes untuk GCM standar)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: gagal generate nonce: %v", domain.ErrEncryptionFailed, err)
	}

	// Enkripsi plaintext; Seal menambahkan ciphertext + tag setelah nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode ke base64 untuk penyimpanan di database
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAESGCM mendekripsi ciphertext yang dienkripsi dengan EncryptAESGCM.
// Input berupa base64-encoded string berisi (nonce + ciphertext + tag).
// masterKey harus sama dengan yang digunakan saat enkripsi.
func DecryptAESGCM(ciphertext string, masterKey []byte) (string, error) {
	// Validasi panjang master key harus 32 bytes
	if len(masterKey) != 32 {
		return "", fmt.Errorf("%w: master key harus 32 bytes", domain.ErrDecryptionFailed)
	}

	// Decode base64 menjadi bytes
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: gagal decode base64: %v", domain.ErrDecryptionFailed, err)
	}

	// Buat cipher block dari master key
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat cipher: %v", domain.ErrDecryptionFailed, err)
	}

	// Buat GCM wrapper dari cipher block
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat GCM: %v", domain.ErrDecryptionFailed, err)
	}

	// Validasi panjang data minimal harus lebih dari nonce size
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext terlalu pendek", domain.ErrDecryptionFailed)
	}

	// Pisahkan nonce (12 bytes pertama) dan ciphertext+tag (sisanya)
	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Dekripsi dan verifikasi tag
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("%w: gagal dekripsi: %v", domain.ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// MaskAPIKey mengembalikan API key yang di-mask, hanya menampilkan 4 karakter terakhir.
// Karakter sebelumnya diganti dengan asterisk (*).
// Contoh: "xnd_production_abc123xyz" -> "********************3xyz"
// Key dengan panjang < 4 karakter dikembalikan apa adanya.
func MaskAPIKey(apiKey string) string {
	if len(apiKey) < 4 {
		return apiKey
	}

	// Ambil 4 karakter terakhir
	lastFour := apiKey[len(apiKey)-4:]

	// Ganti karakter sebelumnya dengan asterisk
	masked := strings.Repeat("*", len(apiKey)-4) + lastFour

	return masked
}
