package middleware

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 9.2, 9.3, 9.4, 9.6, 14.2**
//

func TestProperty_RBACEnforcesEndpointRoleMethodAccessRules(t *testing.T) {
	// Define a known RBAC config untuk testing:
	// AllowedRoles: tenant_admin, operator, kasir
	rbacConfig := domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin,
			domain.RoleOperator,
			domain.RoleKasir,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleKasir: {"GET"},
		},
	}

	allRoles := []domain.UserRole{
		domain.RoleSuperAdmin,
		domain.RoleTenantAdmin,
		domain.RoleOperator,
		domain.RoleTeknisi,
		domain.RoleKasir,
		domain.RoleReseller,
	}

	// HTTP methods to test
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	// Paths to test - includes reseller-allowed dan reseller-disallowed paths
	testPaths := []string{
		"/v1/customers/123",
		"/v1/invoices/456",
		"/v1/reseller/vouchers",
		"/v1/auth/me",
		"/v1/settings/users",
		"/v1/reports/monthly",
	}

	isAllowedRole := func(role domain.UserRole) bool {
		for _, r := range rbacConfig.AllowedRoles {
			if r == role {
				return true
			}
		}
		return false
	}

	isMethodAllowed := func(role domain.UserRole, method string) bool {
		methods, hasRestriction := rbacConfig.MethodRestrictions[role]
		if !hasRestriction {
			return true
		}
		for _, m := range methods {
			if m == method {
				return true
			}
		}
		return false
	}

	isResellerPathAllowed := func(path string) bool {
		return len(path) >= len("/v1/reseller/") && path[:len("/v1/reseller/")] == "/v1/reseller/" ||
			len(path) >= len("/v1/auth/") && path[:len("/v1/auth/")] == "/v1/auth/"
	}

	rapid.Check(t, func(t *rapid.T) {
		// Draw random role, method, dan path
		role := rapid.SampledFrom(allRoles).Draw(t, "role")
		method := rapid.SampledFrom(httpMethods).Draw(t, "method")
		path := rapid.SampledFrom(testPaths).Draw(t, "path")

		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", string(role))
			return c.Next()
		})

		app.Use(RBAC(rbacConfig))

		app.All("/*", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(method, path, nil)

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		// Drain body to avoid resource leaks
		_, _ = io.ReadAll(resp.Body)

		statusCode := resp.StatusCode

		var expectedAllowed bool

		switch {
		case role == domain.RoleSuperAdmin:
			expectedAllowed = true

		case !isAllowedRole(role):
			expectedAllowed = false

		case !isMethodAllowed(role, method):
			expectedAllowed = false

		case role == domain.RoleReseller && !isResellerPathAllowed(path):
			expectedAllowed = false

		default:
			expectedAllowed = true
		}

		if expectedAllowed {
			if statusCode != http.StatusOK {
				t.Errorf("role=%s method=%s path=%s: expected 200 (allowed), got %d",
					role, method, path, statusCode)
			}
		} else {
			if statusCode != http.StatusForbidden {
				t.Errorf("role=%s method=%s path=%s: expected 403 (forbidden), got %d",
					role, method, path, statusCode)
			}
		}
	})
}

func TestProperty_RBACResellerPathRestriction(t *testing.T) {
	rbacConfig := domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin,
			domain.RoleOperator,
			domain.RoleReseller,
		},
		MethodRestrictions: map[domain.UserRole][]string{},
	}

	// Path yang boleh diakses reseller
	allowedPaths := []string{
		"/v1/reseller/vouchers",
		"/v1/reseller/dashboard",
		"/v1/reseller/deposits",
		"/v1/auth/me",
		"/v1/auth/logout",
		"/v1/auth/sessions",
	}

	// Path yang tidak boleh diakses reseller
	forbiddenPaths := []string{
		"/v1/customers/123",
		"/v1/invoices/456",
		"/v1/payments/789",
		"/v1/settings/users",
		"/v1/mikrotik/devices",
		"/v1/reports/monthly",
	}

	rapid.Check(t, func(t *rapid.T) {
		isAllowedPath := rapid.Bool().Draw(t, "isAllowedPath")
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")

		var path string
		if isAllowedPath {
			path = rapid.SampledFrom(allowedPaths).Draw(t, "allowedPath")
		} else {
			path = rapid.SampledFrom(forbiddenPaths).Draw(t, "forbiddenPath")
		}

		app := fiber.New()

		// Atur role menjadi reseller
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", string(domain.RoleReseller))
			return c.Next()
		})

		app.Use(RBAC(rbacConfig))

		app.All("/*", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(method, path, nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		_, _ = io.ReadAll(resp.Body)

		statusCode := resp.StatusCode

		if isAllowedPath {
			if statusCode != http.StatusOK {
				t.Errorf("reseller method=%s path=%s: expected 200 (allowed path), got %d",
					method, path, statusCode)
			}
		} else {
			if statusCode != http.StatusForbidden {
				t.Errorf("reseller method=%s path=%s: expected 403 (forbidden path), got %d",
					method, path, statusCode)
			}
		}
	})
}

func TestProperty_RBACSuperAdminAlwaysAllowed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numAllowed := rapid.IntRange(0, 4).Draw(t, "numAllowedRoles")
		nonSuperRoles := []domain.UserRole{
			domain.RoleTenantAdmin,
			domain.RoleOperator,
			domain.RoleTeknisi,
			domain.RoleKasir,
		}

		var allowedRoles []domain.UserRole
		for i := 0; i < numAllowed && i < len(nonSuperRoles); i++ {
			allowedRoles = append(allowedRoles, nonSuperRoles[i])
		}

		rbacConfig := domain.RBACConfig{
			AllowedRoles:       allowedRoles,
			MethodRestrictions: map[domain.UserRole][]string{},
		}

		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}).Draw(t, "method")
		path := fmt.Sprintf("/v1/%s/%s",
			rapid.SampledFrom([]string{"customers", "invoices", "settings", "reports", "mikrotik"}).Draw(t, "resource"),
			rapid.StringMatching(`[a-z0-9]{3,10}`).Draw(t, "id"),
		)

		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", string(domain.RoleSuperAdmin))
			return c.Next()
		})

		app.Use(RBAC(rbacConfig))

		app.All("/*", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(method, path, nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		_, _ = io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("super_admin method=%s path=%s config.AllowedRoles=%v: expected 200, got %d",
				method, path, allowedRoles, resp.StatusCode)
		}
	})
}
