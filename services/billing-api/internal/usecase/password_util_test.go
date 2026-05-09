package usecase

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 11.5, 11.1, 2.3**
func TestProperty_BcryptPasswordRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat a random password: 8 to 72 chars (bcrypt max is 72 bytes)
		password := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "password")

		hash, err := HashPassword(password, bcrypt.MinCost)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		if err := VerifyPassword(hash, password); err != nil {
			t.Errorf("VerifyPassword failed for original password: %v", err)
		}

		otherPassword := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "other_password")

		if otherPassword != password {
			if err := VerifyPassword(hash, otherPassword); err == nil {
				t.Errorf("VerifyPassword should have failed for different password %q against hash of %q", otherPassword, password)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 11.6**
//
// different hash values.
func TestProperty_BcryptCollisionResistance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat two random passwords (>= 8 chars, max 72 chars untuk bcrypt)
		password1 := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "password1")
		password2 := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "password2")

		if password1 == password2 {
			t.Skip("generated identical passwords, skipping")
		}

		// Hash both passwords using bcrypt.MinCost untuk faster test execution
		hash1, err := HashPassword(password1, bcrypt.MinCost)
		if err != nil {
			t.Fatalf("HashPassword failed for password1: %v", err)
		}

		hash2, err := HashPassword(password2, bcrypt.MinCost)
		if err != nil {
			t.Fatalf("HashPassword failed for password2: %v", err)
		}

		if hash1 == hash2 {
			t.Errorf("collision detected: hash(%q) == hash(%q) == %q", password1, password2, hash1)
		}
	})
}
