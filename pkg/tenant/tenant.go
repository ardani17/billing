// Paket tenant menyediakan middleware Fiber dan helper untuk mengelola
// konteks tenant dalam sistem multi-tenant ISPBoss.
// Mengekstrak tenant_id dari JWT token dan menyimpannya di context.
package tenant

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/pkg/auth"
)

// contextKey adalah tipe kustom untuk key context agar tidak bentrok
// dengan key dari package lain.
type contextKey string

// tenantKey adalah key yang digunakan untuk menyimpan tenant_id di context.
const tenantKey contextKey = "tenant_id"

// localsKeyTenantID adalah key untuk menyimpan tenant_id di Fiber locals.
const localsKeyTenantID = "tenant_id"

// Middleware membuat Fiber middleware yang mengekstrak tenant_id dari JWT claims
// dan menyimpannya ke Fiber locals serta Go context.
// Mengembalikan HTTP 401 jika Authorization header tidak ada, token tidak valid,
// atau tenant_id kosong di dalam claims.
func Middleware(jwtSecret string) fiber.Handler {
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

		// Pastikan tenant_id ada di claims
		if claims.TenantID == "" {
			return unauthorizedResponse(c, "tenant_id tidak ditemukan dalam token")
		}

		// Simpan tenant_id di Fiber locals agar bisa diakses handler
		c.Locals(localsKeyTenantID, claims.TenantID)

		// Simpan tenant_id di Go context untuk digunakan layer di bawahnya
		ctx := context.WithValue(c.UserContext(), tenantKey, claims.TenantID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// FromContext mengambil tenant_id dari Go context.
// Mengembalikan string kosong jika tenant_id tidak ditemukan.
func FromContext(ctx context.Context) string {
	val, ok := ctx.Value(tenantKey).(string)
	if !ok {
		return ""
	}
	return val
}

// SetForTest menyimpan tenant_id ke Go context untuk keperluan testing.
// Hanya digunakan di unit test agar tidak perlu JWT token.
func SetForTest(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// MustFromContext mengambil tenant_id dari Go context.
// Panic jika tenant_id tidak ditemukan - hanya untuk penggunaan internal
// di mana tenant_id dijamin sudah ada (setelah melewati middleware).
func MustFromContext(ctx context.Context) string {
	val := FromContext(ctx)
	if val == "" {
		panic("tenant: tenant_id tidak ditemukan di context")
	}
	return val
}

// unauthorizedResponse mengembalikan respons JSON 401 dengan format standar API.
func unauthorizedResponse(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}
