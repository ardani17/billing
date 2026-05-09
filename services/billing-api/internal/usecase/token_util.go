package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateSecureToken membuat token random 32 bytes dan mengembalikan hex string
// beserta SHA-256 hash-nya. Plaintext token dikirim ke user (via email atau API respons),
// sedangkan hash disimpan di database untuk validasi.
func GenerateSecureToken() (plaintext string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	plaintext = hex.EncodeToString(bytes)
	h := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(h[:])
	return plaintext, hash, nil
}

// HashToken menghitung SHA-256 hash dari token plaintext.
// Digunakan saat validasi token: hash input token lalu bandingkan dengan hash di database.
func HashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}
