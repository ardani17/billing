package auth

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 8.6**
func TestProperty_JWTTokenRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat tenant_id dan user_id acak dalam format UUID
		tenantID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`).Draw(t, "tenant_id")
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`).Draw(t, "user_id")

		roles := []string{"super_admin", "tenant_admin", "operator", "teknisi", "kasir", "reseller"}
		role := rapid.SampledFrom(roles).Draw(t, "role")

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

		// Buat token
		token, err := GenerateToken(cfg, claims)
		if err != nil {
			t.Fatalf("GenerateToken failed: %v", err)
		}
		if token == "" {
			t.Fatal("GenerateToken returned empty token")
		}

		// Validasi token
		validatedClaims, err := ValidateToken(cfg.Secret, token)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}

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
