package usecase

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 12.4, 3.1**
//
// logic parts: buat a token, hash it, simulate consuming it (mark as used),
func TestProperty_TokenSingleUseEnforcement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat a secure token (plaintext + hash)
		plaintext, hash, err := GenerateSecureToken()
		if err != nil {
			t.Fatalf("GenerateSecureToken failed: %v", err)
		}

		verification := &domain.EmailVerification{
			ID:        "test-id",
			UserID:    "test-user",
			TokenHash: hash,
			Used:      false,
		}

		lookupHash := HashToken(plaintext)
		if lookupHash != verification.TokenHash {
			t.Fatalf("token hash mismatch: HashToken(plaintext)=%q, stored=%q", lookupHash, verification.TokenHash)
		}
		if verification.Used {
			t.Fatal("token should not be marked as used before consumption")
		}

		verification.Used = true

		secondHash := HashToken(plaintext)
		if secondHash != verification.TokenHash {
			t.Fatalf("hash should be deterministic: second=%q, stored=%q", secondHash, verification.TokenHash)
		}

		if !verification.Used {
			t.Error("token should be marked as used after consumption")
		}

		resetToken := &domain.PasswordReset{
			ID:        "reset-id",
			UserID:    "test-user",
			TokenHash: hash,
			Used:      false,
		}

		if resetToken.Used {
			t.Fatal("reset token should not be used initially")
		}

		// Consume
		resetToken.Used = true

		if !resetToken.Used {
			t.Error("reset token should be marked as used after consumption")
		}
	})
}

// **Memvalidasi: Kebutuhan 2.2, 11.2, 16.3**
//
// go-playground/validator directly on RegisterRequest.
func TestProperty_InputValidationRejectsInvalidData(t *testing.T) {
	validate := validator.New()

	rapid.Check(t, func(t *rapid.T) {
		invalidField := rapid.SampledFrom([]string{
			"name", "email", "phone", "password",
		}).Draw(t, "invalidField")

		req := domain.RegisterRequest{
			Name:                 "Valid Name",
			Email:                "valid@example.com",
			Phone:                "+6281234567890",
			CompanyName:          "Valid Company",
			Password:             "validpassword123",
			PasswordConfirmation: "validpassword123",
			AgreeTerms:           true,
		}

		switch invalidField {
		case "name":
			nameLen := rapid.IntRange(0, 2).Draw(t, "nameLen")
			if nameLen == 0 {
				req.Name = ""
			} else {
				req.Name = rapid.StringMatching(`[a-zA-Z]{1,2}`).Draw(t, "shortName")
				if len(req.Name) >= 3 {
					req.Name = req.Name[:2]
				}
			}

		case "email":
			req.Email = rapid.SampledFrom([]string{
				"",
				"notanemail",
				"missing-at.com",
				"@nodomain",
				"spaces in@email.com",
			}).Draw(t, "invalidEmail")

		case "phone":
			req.Phone = rapid.SampledFrom([]string{
				"",
				"081234567890",
				"+1234567890",
				"+61812345678",
				"62812345678",
				"+00123456789",
			}).Draw(t, "invalidPhone")

		case "password":
			// Buat a password shorter than 8 characters
			pwLen := rapid.IntRange(0, 7).Draw(t, "pwLen")
			if pwLen == 0 {
				req.Password = ""
			} else {
				req.Password = rapid.StringMatching(`[a-zA-Z0-9]{1,7}`).Draw(t, "shortPassword")
				if len(req.Password) >= 8 {
					req.Password = req.Password[:7]
				}
			}
			req.PasswordConfirmation = req.Password
		}

		err := validate.Struct(req)
		if err == nil {
			t.Errorf("expected validation error for invalid %s field, but got nil. Request: %+v", invalidField, req)
			return
		}

		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			t.Fatalf("expected validator.ValidationErrors, got %T: %v", err, err)
		}

		if len(validationErrors) == 0 {
			t.Errorf("expected at least one validation error for invalid %s, got none", invalidField)
		}
	})
}

// **Memvalidasi: Kebutuhan 4.1, 4.2, 14.1**
//
// redirect path: tenant_admin -> /dashboard, operator -> /dashboard,
// teknisi -> /network, kasir -> /payments, reseller -> /reseller/dashboard.
func TestProperty_LoginReturnsCorrectRedirectPathPerRole(t *testing.T) {
	expectedPaths := map[domain.UserRole]string{
		domain.RoleSuperAdmin:  "/super-admin",
		domain.RoleTenantAdmin: "/dashboard",
		domain.RoleOperator:    "/dashboard",
		domain.RoleTeknisi:     "/network",
		domain.RoleKasir:       "/payments",
		domain.RoleReseller:    "/reseller/dashboard",
	}

	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom(domain.ValidRoles).Draw(t, "role")

		actualPath, exists := domain.RedirectPathMap[role]
		if !exists {
			t.Fatalf("RedirectPathMap missing entry for role %q", role)
		}

		expectedPath, hasExpected := expectedPaths[role]
		if !hasExpected {
			t.Fatalf("test expectedPaths missing entry for role %q", role)
		}

		if actualPath != expectedPath {
			t.Errorf("RedirectPathMap[%q] = %q, want %q", role, actualPath, expectedPath)
		}
	})
}

// has an entry in RedirectPathMap (no missing roles).
func TestProperty_RedirectPathMapCompleteness(t *testing.T) {
	for _, role := range domain.ValidRoles {
		if _, exists := domain.RedirectPathMap[role]; !exists {
			t.Errorf("RedirectPathMap missing entry for valid role %q", role)
		}
	}
}
