package domain

import (
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

func nonEmptyStringWithoutColons() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		chars := rapid.SliceOfN(
			rapid.RuneFrom(nil, unicode.Letter, unicode.Digit, &unicode.RangeTable{
				R16: []unicode.Range16{
					{Lo: '-', Hi: '-', Stride: 1},
					{Lo: '_', Hi: '_', Stride: 1},
					{Lo: '.', Hi: '.', Stride: 1},
					{Lo: ' ', Hi: ' ', Stride: 1},
				},
			}),
			1, 50,
		).Draw(t, "chars")
		return string(chars)
	})
}

// TestProperty_CommentRoundTrip memverifikasi bahwa untuk sembarang customer_id
// dan tenant_id (non-empty, tanpa colon), BuildComment menghasilkan string yang
// dan IsISPBossComment mengembalikan true.
//
// **Memvalidasi: Kebutuhan 1.4, 8.9**
func TestProperty_CommentRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customerID := nonEmptyStringWithoutColons().Draw(t, "customerID")
		tenantID := nonEmptyStringWithoutColons().Draw(t, "tenantID")

		comment := BuildComment(customerID, tenantID)

		parsedCustomerID, parsedTenantID, err := ParseComment(comment)
		if err != nil {
			t.Fatalf("ParseComment(%q) returned error: %v", comment, err)
		}
		if parsedCustomerID != customerID {
			t.Errorf("customer_id mismatch: got %q, want %q", parsedCustomerID, customerID)
		}
		if parsedTenantID != tenantID {
			t.Errorf("tenant_id mismatch: got %q, want %q", parsedTenantID, tenantID)
		}

		if !IsISPBossComment(comment) {
			t.Errorf("IsISPBossComment(%q) = false, want true", comment)
		}
	})
}

// TestProperty_NonISPBossCommentDetection memverifikasi bahwa untuk sembarang
// string yang TIDAK dimulai dengan "ISPBoss:", IsISPBossComment mengembalikan false.
//
// **Memvalidasi: Kebutuhan 1.4, 8.9**
func TestProperty_NonISPBossCommentDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		comment := rapid.String().Draw(t, "comment")

		if strings.HasPrefix(comment, "ISPBoss:") {
			return
		}

		if IsISPBossComment(comment) {
			t.Errorf("IsISPBossComment(%q) = true, want false for non-ISPBoss comment", comment)
		}
	})
}

// =============================================================================
// =============================================================================

// TestProperty_GenerateProfileNameSafe memverifikasi bahwa untuk sembarang
// package name string, GenerateProfileName menghasilkan string yang:
// - Tidak mengandung spasi
// - Hanya mengandung karakter alfanumerik dan hyphen
// - Idempotent: GenerateProfileName(GenerateProfileName(name)) == GenerateProfileName(name)
//
// **Memvalidasi: Kebutuhan 2.4**
func TestProperty_GenerateProfileNameSafe(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		packageName := rapid.String().Draw(t, "packageName")

		result := GenerateProfileName(packageName)

		if strings.Contains(result, " ") {
			t.Errorf("GenerateProfileName(%q) = %q contains spaces", packageName, result)
		}

		for _, r := range result {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				t.Errorf("GenerateProfileName(%q) = %q contains invalid character %q", packageName, result, string(r))
			}
		}

		doubleResult := GenerateProfileName(result)
		if doubleResult != result {
			t.Errorf("GenerateProfileName is not idempotent: GenerateProfileName(%q) = %q, but GenerateProfileName(%q) = %q",
				packageName, result, result, doubleResult)
		}

		// GenerateProfileName only preserves a-z0-9 (ASCII), so we cek untuk ASCII alphanumeric
		hasAlphanumeric := false
		for _, r := range packageName {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				hasAlphanumeric = true
				break
			}
		}
		if hasAlphanumeric && result == "" {
			t.Errorf("GenerateProfileName(%q) = empty string, but input has alphanumeric characters", packageName)
		}
	})
}
