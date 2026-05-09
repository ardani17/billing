package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// AESEncryptor - implementasi CredentialEncryptor menggunakan AES-256-GCM
// =============================================================================

// AESEncryptor mengenkripsi dan mendekripsi credential router
// menggunakan AES-256-GCM dengan random nonce per operasi.
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor membuat instance AESEncryptor baru.
// Parameter key harus tepat 32 bytes (256 bit) untuk AES-256.
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 32 {
		return nil, domain.ErrInvalidEncryptionKey
	}
	// Salin key agar caller tidak bisa mengubah setelah konstruksi
	keyCopy := make([]byte, 32)
	copy(keyCopy, key)
	return &AESEncryptor{key: keyCopy}, nil
}

// Encrypt mengenkripsi plaintext menggunakan AES-256-GCM.
// Mengembalikan base64-encoded string berisi (nonce + ciphertext + tag).
// Setiap pemanggilan menghasilkan nonce acak 12 bytes yang unik.
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	// Buat cipher block dari master key
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat cipher", domain.ErrEncryptionFailed)
	}

	// Buat GCM wrapper dari cipher block
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat GCM", domain.ErrEncryptionFailed)
	}

	// Buat nonce acak (12 bytes untuk GCM standar)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: gagal generate nonce", domain.ErrEncryptionFailed)
	}

	// Enkripsi plaintext; Seal menambahkan ciphertext + tag setelah nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode ke base64 untuk penyimpanan di database
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt mendekripsi ciphertext yang dienkripsi dengan Encrypt.
// Input berupa base64-encoded string berisi (nonce + ciphertext + tag).
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	// Decode base64 menjadi bytes
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: gagal decode base64", domain.ErrDecryptionFailed)
	}

	// Buat cipher block dari master key
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat cipher", domain.ErrDecryptionFailed)
	}

	// Buat GCM wrapper dari cipher block
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: gagal membuat GCM", domain.ErrDecryptionFailed)
	}

	// Validasi panjang data minimal harus lebih dari nonce size (12 bytes)
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext terlalu pendek", domain.ErrDecryptionFailed)
	}

	// Pisahkan nonce (12 bytes pertama) dan ciphertext+tag (sisanya)
	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Dekripsi dan verifikasi tag
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("%w: gagal dekripsi", domain.ErrDecryptionFailed)
	}

	return string(plaintext), nil
}
