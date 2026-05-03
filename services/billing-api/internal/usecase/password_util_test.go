package usecase

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// Feature: auth-rbac, Property 2: Bcrypt Password Round-Trip
// **Validates: Requirements 11.5, 11.1, 2.3**
//
// For any valid password string (at least 8 characters), hashing the password
// with bcrypt then comparing the original password against the hash using
// bcrypt.CompareHashAndPassword SHALL return nil (match).
// Also verifies that a different password does NOT match the hash.
func TestProperty_BcryptPasswordRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random password: 8 to 72 chars (bcrypt max is 72 bytes)
		password := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "password")

		// Hash the password using bcrypt.MinCost for faster test execution
		hash, err := HashPassword(password, bcrypt.MinCost)
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		// Property: VerifyPassword with the original password must return nil (match)
		if err := VerifyPassword(hash, password); err != nil {
			t.Errorf("VerifyPassword failed for original password: %v", err)
		}

		// Generate a different password to verify non-match
		otherPassword := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "other_password")

		// Only test non-match if the passwords are actually different
		if otherPassword != password {
			if err := VerifyPassword(hash, otherPassword); err == nil {
				t.Errorf("VerifyPassword should have failed for different password %q against hash of %q", otherPassword, password)
			}
		}
	})
}

// Feature: auth-rbac, Property 3: Bcrypt Collision Resistance
// **Validates: Requirements 11.6**
//
// For any two distinct password strings, hashing each with bcrypt SHALL produce
// different hash values.
func TestProperty_BcryptCollisionResistance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate two random passwords (>= 8 chars, max 72 chars for bcrypt)
		password1 := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "password1")
		password2 := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{8,72}`).Draw(t, "password2")

		// Only test when passwords are distinct
		if password1 == password2 {
			t.Skip("generated identical passwords, skipping")
		}

		// Hash both passwords using bcrypt.MinCost for faster test execution
		hash1, err := HashPassword(password1, bcrypt.MinCost)
		if err != nil {
			t.Fatalf("HashPassword failed for password1: %v", err)
		}

		hash2, err := HashPassword(password2, bcrypt.MinCost)
		if err != nil {
			t.Fatalf("HashPassword failed for password2: %v", err)
		}

		// Property: hashes of distinct passwords must be different
		if hash1 == hash2 {
			t.Errorf("collision detected: hash(%q) == hash(%q) == %q", password1, password2, hash1)
		}
	})
}
