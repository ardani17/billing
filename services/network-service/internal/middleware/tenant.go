package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/pkg/tenant"
)

// TenantContext membuat Fiber middleware yang mengekstrak tenant_id dari JWT
// dan menyimpannya ke Go context menggunakan pkg/tenant.
// Middleware ini adalah wrapper tipis di atas pkg/tenant.Middleware.
// Harus dipasang setelah Auth middleware agar JWT sudah tervalidasi.
func TenantContext(jwtSecret string) fiber.Handler {
	return tenant.Middleware(jwtSecret)
}
