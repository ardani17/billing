package usecase

import (
	"encoding/hex"
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 12.5, 12.2, 12.3**
//

var hexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestProperty_TokenHashRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat a secure token (plaintext + hash)
		plaintext, hash, err := GenerateSecureToken()
		if err != nil {
			t.Fatalf("GenerateSecureToken failed: %v", err)
		}

		if !hexPattern.MatchString(plaintext) {
			t.Errorf("plaintext is not 64 hex characters: got %q (len=%d)", plaintext, len(plaintext))
		}

		decoded, err := hex.DecodeString(plaintext)
		if err != nil {
			t.Fatalf("plaintext is not valid hex: %v", err)
		}
		if len(decoded) != 32 {
			t.Errorf("plaintext decoded to %d bytes, want 32", len(decoded))
		}

		if !hexPattern.MatchString(hash) {
			t.Errorf("hash is not 64 hex characters: got %q (len=%d)", hash, len(hash))
		}

		recomputedHash := HashToken(plaintext)
		if recomputedHash != hash {
			t.Errorf("hash round-trip failed: HashToken(plaintext)=%q, stored hash=%q", recomputedHash, hash)
		}
	})
}

func TestProperty_HashTokenDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat a random string of arbitrary length
		input := rapid.StringMatching(`[a-zA-Z0-9]{1,200}`).Draw(t, "input")

		hash1 := HashToken(input)
		hash2 := HashToken(input)

		if hash1 != hash2 {
			t.Errorf("HashToken is not deterministic: hash1=%q, hash2=%q for input=%q", hash1, hash2, input)
		}

		if !hexPattern.MatchString(hash1) {
			t.Errorf("HashToken output is not 64 hex characters: got %q (len=%d)", hash1, len(hash1))
		}
	})
}
