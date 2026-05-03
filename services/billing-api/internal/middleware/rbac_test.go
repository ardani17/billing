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

// Feature: auth-rbac, Property 10: RBAC Enforces Endpoint-Role-Method Access Rules
// **Validates: Requirements 9.2, 9.3, 9.4, 9.6, 14.2**
//
// For any combination of (role, endpoint path, HTTP method), the RBAC middleware
// SHALL allow access if and only if the role is in the endpoint's allowed roles
// AND (if method restrictions exist for that role) the HTTP method is in the
// allowed methods. Super_admin SHALL always be allowed. Reseller SHALL only
// access paths starting with /v1/reseller/ or /v1/auth/*.

func TestProperty_RBACEnforcesEndpointRoleMethodAccessRules(t *testing.T) {
	// Define a known RBAC config for testing:
	// AllowedRoles: tenant_admin, operator, kasir
	// MethodRestrictions: kasir can only GET
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

	// All roles in the system
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

	// Paths to test — includes reseller-allowed and reseller-disallowed paths
	testPaths := []string{
		"/v1/customers/123",
		"/v1/invoices/456",
		"/v1/reseller/vouchers",
		"/v1/auth/me",
		"/v1/settings/users",
		"/v1/reports/monthly",
	}

	// Helper: check if a role is in the allowed roles list
	isAllowedRole := func(role domain.UserRole) bool {
		for _, r := range rbacConfig.AllowedRoles {
			if r == role {
				return true
			}
		}
		return false
	}

	// Helper: check if a method is allowed for a role given method restrictions
	isMethodAllowed := func(role domain.UserRole, method string) bool {
		methods, hasRestriction := rbacConfig.MethodRestrictions[role]
		if !hasRestriction {
			// No restriction means all methods allowed
			return true
		}
		for _, m := range methods {
			if m == method {
				return true
			}
		}
		return false
	}

	// Helper: check if a path is allowed for reseller
	isResellerPathAllowed := func(path string) bool {
		return len(path) >= len("/v1/reseller/") && path[:len("/v1/reseller/")] == "/v1/reseller/" ||
			len(path) >= len("/v1/auth/") && path[:len("/v1/auth/")] == "/v1/auth/"
	}

	rapid.Check(t, func(t *rapid.T) {
		// Draw random role, method, and path
		role := rapid.SampledFrom(allRoles).Draw(t, "role")
		method := rapid.SampledFrom(httpMethods).Draw(t, "method")
		path := rapid.SampledFrom(testPaths).Draw(t, "path")

		// Set up Fiber app with a helper middleware that sets role in locals,
		// then the RBAC middleware, then a test handler that returns 200
		app := fiber.New()

		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", string(role))
			return c.Next()
		})

		app.Use(RBAC(rbacConfig))

		// Register a catch-all handler for all methods
		app.All("/*", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		// Build the request
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

		// Compute expected result based on RBAC rules
		var expectedAllowed bool

		switch {
		case role == domain.RoleSuperAdmin:
			// Property: super_admin always allowed
			expectedAllowed = true

		case !isAllowedRole(role):
			// Property: roles NOT in AllowedRoles get 403
			expectedAllowed = false

		case !isMethodAllowed(role, method):
			// Property: roles with MethodRestrictions only get 200 for allowed methods
			expectedAllowed = false

		case role == domain.RoleReseller && !isResellerPathAllowed(path):
			// Property: reseller gets 403 for paths outside /v1/reseller/* and /v1/auth/*
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
	// Specific property test for reseller path restriction:
	// Reseller is in AllowedRoles but should only access /v1/reseller/* and /v1/auth/*
	rbacConfig := domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin,
			domain.RoleOperator,
			domain.RoleReseller,
		},
		MethodRestrictions: map[domain.UserRole][]string{},
	}

	// Paths that reseller IS allowed to access
	allowedPaths := []string{
		"/v1/reseller/vouchers",
		"/v1/reseller/dashboard",
		"/v1/reseller/deposits",
		"/v1/auth/me",
		"/v1/auth/logout",
		"/v1/auth/sessions",
	}

	// Paths that reseller is NOT allowed to access
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

		// Set role to reseller
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
	// Property: super_admin always gets 200 regardless of config
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random RBAC config — even with empty AllowedRoles,
		// super_admin should still pass
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
