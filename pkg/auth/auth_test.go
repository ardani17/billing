package auth

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: auth-rbac, Property 1: JWT Token Round-Trip
// **Validates: Requirements 8.6**
//
// For any valid set of claims (tenant_id, user_id, role), generating a JWT token
// with GenerateToken then validating it with ValidateToken SHALL return claims
// where tenant_id, user_id, and role are identical to the original input.
func TestProperty_JWTTokenRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random UUID-format tenant_id and user_id
		tenantID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`).Draw(t, "tenant_id")
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`).Draw(t, "user_id")

		// Pick a random role from the 6 valid roles
		roles := []string{"super_admin", "tenant_admin", "operator", "teknisi", "kasir", "reseller"}
		role := rapid.SampledFrom(roles).Draw(t, "role")

		// Configure token generation with a test secret and 1h expiry
		cfg := TokenConfig{
			Secret: "test-secret-key-at-least-32-chars!!",
			Expiry: time.Hour,
			Issuer: "ispboss",
		}

		claims := Claims{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
		}

		// Generate token
		token, err := GenerateToken(cfg, claims)
		if err != nil {
			t.Fatalf("GenerateToken failed: %v", err)
		}
		if token == "" {
			t.Fatal("GenerateToken returned empty token")
		}

		// Validate token
		validatedClaims, err := ValidateToken(cfg.Secret, token)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}

		// Assert round-trip: validated claims must match original input
		if validatedClaims.TenantID != tenantID {
			t.Errorf("TenantID mismatch: got %q, want %q", validatedClaims.TenantID, tenantID)
		}
		if validatedClaims.UserID != userID {
			t.Errorf("UserID mismatch: got %q, want %q", validatedClaims.UserID, userID)
		}
		if validatedClaims.Role != role {
			t.Errorf("Role mismatch: got %q, want %q", validatedClaims.Role, role)
		}
	})
}
