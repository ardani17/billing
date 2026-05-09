// Package middleware berisi Fiber middleware untuk notification service.
// Termasuk autentikasi JWT, konteks tenant, dan logging permintaan.
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/pkg/auth"
)

// Auth membuat Fiber middleware untuk validasi JWT token.
// Mengekstrak token dari header Authorization (format: Bearer <token>),
// memvalidasi signature dan expiry, lalu menyimpan claims di Fiber locals.
// Mengembalikan 401 jika token tidak ada atau tidak valid.
func Auth(jwtSecret string) fiber.Handler {
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

		// Simpan claims di Fiber locals untuk digunakan handler
		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// unauthorizedResponse mengembalikan respons JSON 401 dengan format standar.
func unauthorizedResponse(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}
