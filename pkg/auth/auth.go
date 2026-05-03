// Package auth menyediakan fungsi untuk membuat dan memvalidasi JWT token.
// Digunakan oleh semua Go service untuk autentikasi dan otorisasi.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Daftar error yang mungkin dikembalikan oleh fungsi auth.
var (
	ErrEmptySecret   = errors.New("secret tidak boleh kosong")
	ErrEmptyToken    = errors.New("token tidak boleh kosong")
	ErrInvalidToken  = errors.New("token tidak valid")
	ErrExpiredToken  = errors.New("token sudah kedaluwarsa")
	ErrInvalidClaims = errors.New("claims tidak valid")
)

// Claims berisi data yang di-embed dalam JWT token.
// Menyimpan informasi tenant, user, dan role untuk otorisasi.
type Claims struct {
	jwt.RegisteredClaims
	TenantID       string `json:"tenant_id"`
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	ImpersonatorID string `json:"impersonator_id,omitempty"` // untuk super admin impersonation
}

// TokenConfig berisi konfigurasi untuk pembuatan JWT token.
type TokenConfig struct {
	// Secret adalah kunci rahasia untuk menandatangani token.
	Secret string

	// Expiry adalah durasi masa berlaku token.
	Expiry time.Duration

	// Issuer adalah identitas penerbit token.
	Issuer string
}

// GenerateToken membuat JWT token baru dengan claims yang diberikan.
// Mengatur ExpiresAt, IssuedAt, dan Issuer dari konfigurasi.
// Mengembalikan token string yang sudah ditandatangani atau error.
func GenerateToken(cfg TokenConfig, claims Claims) (string, error) {
	if cfg.Secret == "" {
		return "", ErrEmptySecret
	}

	now := time.Now()

	// Atur waktu dan issuer dari konfigurasi
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(cfg.Expiry))
	claims.Issuer = cfg.Issuer

	// Buat token dengan metode signing HMAC SHA-256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Tandatangani token dengan secret
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("gagal menandatangani token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken memvalidasi JWT token dan mengembalikan claims.
// Mengembalikan error jika token tidak valid, kedaluwarsa, atau signature tidak cocok.
func ValidateToken(secret string, tokenString string) (*Claims, error) {
	if secret == "" {
		return nil, ErrEmptySecret
	}
	if tokenString == "" {
		return nil, ErrEmptyToken
	}

	// Parse token dengan validasi signature menggunakan HMAC
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Pastikan metode signing sesuai dengan yang diharapkan (HMAC)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode signing tidak diharapkan: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		// Periksa apakah error karena token kedaluwarsa
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err.Error())
	}

	// Pastikan token valid dan claims bisa di-cast
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
