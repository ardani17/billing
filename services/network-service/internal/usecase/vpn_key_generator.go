package usecase

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// vpnKeyGenerator — implementasi VPNKeyGenerator menggunakan curve25519
// =============================================================================

// vpnKeyGenerator menghasilkan key pair dan credential untuk VPN tunnel.
// Menggunakan crypto/rand untuk semua operasi random.
type vpnKeyGenerator struct{}

// NewVPNKeyGenerator membuat instance VPNKeyGenerator baru.
func NewVPNKeyGenerator() domain.VPNKeyGenerator {
	return &vpnKeyGenerator{}
}

// GenerateWireGuardKeyPair menghasilkan pasangan public key dan private key WireGuard.
// Private key = 32 byte acak (di-clamp sesuai spesifikasi WireGuard).
// Public key = curve25519.ScalarBaseMult dari private key.
// Mengembalikan kedua key dalam format base64.
func (g *vpnKeyGenerator) GenerateWireGuardKeyPair() (publicKey, privateKey string, err error) {
	// Generate 32 byte acak untuk private key
	var privKey [32]byte
	if _, err := io.ReadFull(rand.Reader, privKey[:]); err != nil {
		return "", "", fmt.Errorf("%w: gagal generate random bytes", domain.ErrKeyGenerationFailed)
	}

	// Clamp private key sesuai spesifikasi WireGuard (RFC 7748)
	privKey[0] &= 248
	privKey[31] &= 127
	privKey[31] |= 64

	// Hitung public key menggunakan curve25519 scalar base multiplication
	pubKey, err := curve25519.X25519(privKey[:], curve25519.Basepoint)
	if err != nil {
		return "", "", fmt.Errorf("%w: gagal hitung public key", domain.ErrKeyGenerationFailed)
	}

	// Encode ke base64 untuk penyimpanan
	publicKey = base64.StdEncoding.EncodeToString(pubKey)
	privateKey = base64.StdEncoding.EncodeToString(privKey[:])

	return publicKey, privateKey, nil
}

// GeneratePreSharedKey menghasilkan pre-shared key 256-bit (32 bytes) untuk WireGuard.
// Mengembalikan key dalam format base64.
func (g *vpnKeyGenerator) GeneratePreSharedKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("%w: gagal generate pre-shared key", domain.ErrKeyGenerationFailed)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// GenerateCredentials menghasilkan username dan password random untuk L2TP/PPTP/SSTP.
// Username format: "vpn-{tunnelName}-{random6hex}"
// Password: 32 byte acak dalam format hex-encoded.
func (g *vpnKeyGenerator) GenerateCredentials(tunnelName string) (username, password string, err error) {
	// Generate 3 byte acak untuk suffix hex (6 karakter hex)
	suffix := make([]byte, 3)
	if _, err := io.ReadFull(rand.Reader, suffix); err != nil {
		return "", "", fmt.Errorf("%w: gagal generate username suffix", domain.ErrKeyGenerationFailed)
	}
	username = fmt.Sprintf("vpn-%s-%s", tunnelName, hex.EncodeToString(suffix))

	// Generate 32 byte acak untuk password (hex-encoded = 64 karakter)
	passBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, passBytes); err != nil {
		return "", "", fmt.Errorf("%w: gagal generate password", domain.ErrKeyGenerationFailed)
	}
	password = hex.EncodeToString(passBytes)

	return username, password, nil
}

// GenerateIPSecPSK menghasilkan IPSec pre-shared key (32 bytes acak, base64-encoded).
// Digunakan untuk L2TP/IPSec tunnel.
func (g *vpnKeyGenerator) GenerateIPSecPSK() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("%w: gagal generate ipsec psk", domain.ErrKeyGenerationFailed)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}
