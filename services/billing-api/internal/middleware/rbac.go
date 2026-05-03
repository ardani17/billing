package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// RBAC membuat Fiber middleware untuk kontrol akses berbasis role.
// Middleware ini memeriksa role pengguna dari JWT claims (disimpan di c.Locals oleh Auth middleware)
// dan menentukan apakah pengguna diizinkan mengakses endpoint berdasarkan konfigurasi RBACConfig.
//
// Alur pengecekan:
//  1. Ekstrak role dari c.Locals("role")
//  2. Super_admin → bypass semua pengecekan
//  3. Cek apakah role ada di AllowedRoles
//  4. Cek MethodRestrictions per role (jika ada)
//  5. Reseller → hanya boleh akses /v1/reseller/* dan /v1/auth/*
//  6. Jika semua lolos → lanjut ke handler berikutnya
func RBAC(config domain.RBACConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Ekstrak role dari locals (di-set oleh Auth middleware)
		roleStr, ok := c.Locals("role").(string)
		if !ok || roleStr == "" {
			return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "role tidak ditemukan")
		}
		role := domain.UserRole(roleStr)

		// Super admin bypass semua pengecekan
		if role == domain.RoleSuperAdmin {
			return c.Next()
		}

		// Cek apakah role termasuk dalam daftar role yang diizinkan
		allowed := false
		for _, r := range config.AllowedRoles {
			if r == role {
				allowed = true
				break
			}
		}
		if !allowed {
			return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses")
		}

		// Cek pembatasan HTTP method per role
		if methods, hasRestriction := config.MethodRestrictions[role]; hasRestriction {
			methodAllowed := false
			for _, m := range methods {
				if m == c.Method() {
					methodAllowed = true
					break
				}
			}
			if !methodAllowed {
				return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses untuk operasi ini")
			}
		}

		// Reseller hanya boleh akses /v1/reseller/* dan /v1/auth/*
		if role == domain.RoleReseller {
			path := c.Path()
			if !strings.HasPrefix(path, "/v1/reseller/") && !strings.HasPrefix(path, "/v1/auth/") {
				return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses")
			}
		}

		return c.Next()
	}
}
