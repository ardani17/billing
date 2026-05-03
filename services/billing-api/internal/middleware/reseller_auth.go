// Package middleware berisi Fiber middleware untuk billing-api.
// File ini berisi middleware autentikasi khusus reseller dan rate limiter login reseller.
package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/pkg/auth"
)

// ResellerAuth membuat Fiber middleware untuk validasi JWT token reseller.
// Mengekstrak token dari header Authorization (format: Bearer <token>),
// memvalidasi signature dan expiry, lalu menyimpan claims di Fiber locals.
// Middleware ini menolak token yang bukan milik reseller (role != "reseller").
// Mengembalikan 401 jika token tidak ada, tidak valid, atau bukan token reseller.
func ResellerAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Ambil header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return unauthorizedResponse(c, "header Authorization tidak ditemukan")
		}

		// Pastikan format Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return unauthorizedResponse(c, "format Authorization harus Bearer <token>")
		}

		tokenString := parts[1]

		// Validasi token menggunakan pkg/auth
		claims, err := auth.ValidateToken(jwtSecret, tokenString)
		if err != nil {
			return unauthorizedResponse(c, "token tidak valid: "+err.Error())
		}

		// Tolak token yang bukan milik reseller (cegah admin mengakses endpoint reseller)
		if claims.Role != "reseller" {
			return unauthorizedResponse(c, "token bukan milik reseller")
		}

		// Simpan claims di Fiber locals untuk digunakan handler
		// user_id berisi reseller_id (UUID reseller)
		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// ResellerLoginRateLimiterMiddleware membuat Fiber middleware wrapper untuk LoginRateLimiter
// yang digunakan pada endpoint login reseller.
// Middleware ini memeriksa rate limit berdasarkan nomor telepon dari request body.
// Jika nomor telepon terkunci, mengembalikan HTTP 429 dengan sisa waktu lock.
func ResellerLoginRateLimiterMiddleware(rateLimiter *LoginRateLimiter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse phone dari request body tanpa mengkonsumsi body
		var body struct {
			Phone string `json:"phone"`
		}
		if err := c.BodyParser(&body); err != nil || body.Phone == "" {
			// Jika tidak bisa parse phone, lanjutkan ke handler (handler akan validasi)
			return c.Next()
		}

		// Cek rate limit menggunakan phone sebagai key
		allowed, remainingSec, err := rateLimiter.Check(context.Background(), body.Phone)
		if err != nil {
			// Jika error, lanjutkan ke handler (jangan block user karena error internal)
			return c.Next()
		}

		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "ACCOUNT_LOCKED",
					"message": fmt.Sprintf("akun terkunci sementara, coba lagi dalam %d detik", remainingSec),
				},
			})
		}

		return c.Next()
	}
}
