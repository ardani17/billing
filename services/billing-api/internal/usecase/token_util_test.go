package usecase

import (
	"encoding/hex"
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

// Feature: auth-rbac, Property 4: Token Hash Round-Trip
// **Validates: Requirements 12.5, 12.2, 12.3**
//
// For any generated secure token (32+ bytes), computing SHA-256 of the plaintext
// token and storing the hash, then later computing SHA-256 of the same plaintext
// and looking up the hash SHALL find the correct record.

var hexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestProperty_TokenHashRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a secure token (plaintext + hash)
		plaintext, hash, err := GenerateSecureToken()
		if err != nil {
			t.Fatalf("GenerateSecureToken failed: %v", err)
		}

		// Verify plaintext is 64 hex characters (32 bytes encoded as hex)
		if !hexPattern.MatchString(plaintext) {
			t.Errorf("plaintext is not 64 hex characters: got %q (len=%d)", plaintext, len(plaintext))
		}

		// Verify the plaintext decodes to exactly 32 bytes
		decoded, err := hex.DecodeString(plaintext)
		if err != nil {
			t.Fatalf("plaintext is not valid hex: %v", err)
		}
		if len(decoded) != 32 {
			t.Errorf("plaintext decoded to %d bytes, want 32", len(decoded))
		}

		// Verify hash is 64 hex characters (SHA-256 produces 32 bytes = 64 hex chars)
		if !hexPattern.MatchString(hash) {
			t.Errorf("hash is not 64 hex characters: got %q (len=%d)", hash, len(hash))
		}

		// Property: SHA-256 of plaintext must match the stored hash (round-trip)
		recomputedHash := HashToken(plaintext)
		if recomputedHash != hash {
			t.Errorf("hash round-trip failed: HashToken(plaintext)=%q, stored hash=%q", recomputedHash, hash)
		}
	})
}

func TestProperty_HashTokenDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random string of arbitrary length
		input := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "input")

		// Call HashToken twice on the same input
		hash1 := HashToken(input)
		hash2 := HashToken(input)

		// Property: HashToken must be deterministic — same input always produces same output
		if hash1 != hash2 {
			t.Errorf("HashToken is not deterministic: hash1=%q, hash2=%q for input=%q", hash1, hash2, input)
		}

		// Verify hash output is always 64 hex characters (SHA-256)
		if !hexPattern.MatchString(hash1) {
			t.Errorf("HashToken output is not 64 hex characters: got %q (len=%d)", hash1, len(hash1))
		}
	})
}
