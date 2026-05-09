package domain

import (
	"errors"
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 3.4, 14.8, 15.1, 15.3**
func TestProperty_ResellerMarginIntegrity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sellPrice := rapid.Int64Range(1, 1_000_000).Draw(t, "sellPrice")
		resellerPrice := rapid.Int64Range(0, 1_000_000).Draw(t, "resellerPrice")

		err := ValidateResellerMargin(sellPrice, resellerPrice)
		margin := sellPrice - resellerPrice
		shouldBeValid := resellerPrice < sellPrice && margin >= 500

		if shouldBeValid {
			if err != nil {
				t.Fatalf("expected nil for sell=%d reseller=%d (margin=%d), got %v",
					sellPrice, resellerPrice, margin, err)
			}
		} else {
			if err == nil {
				t.Fatalf("expected error for sell=%d reseller=%d (margin=%d), got nil",
					sellPrice, resellerPrice, margin)
			}
			if !errors.Is(err, ErrInsufficientMargin) {
				t.Fatalf("expected ErrInsufficientMargin, got %v", err)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 2.8, 3.7**
//
// spaces replaced by hyphens, containing no uppercase letters atau spaces.
func TestProperty_ProfileNameAutoGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, "name")

		result := GenerateProfileName(name)

		// Properti 5a: Hasil harus lowercase (tidak ada huruf besar)
		for _, r := range result {
			if unicode.IsUpper(r) {
				t.Fatalf("result %q contains uppercase character %q for input %q", result, string(r), name)
			}
		}

		// Properti 5b: Tidak boleh ada spasi
		if strings.Contains(result, " ") {
			t.Fatalf("result %q contains space for input %q", result, name)
		}

		// Properti 5c: Hasil harus sama dengan lowercase(name) dengan spasi diganti hyphen
		expected := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		if result != expected {
			t.Fatalf("expected %q, got %q for input %q", expected, result, name)
		}
	})
}

// **Memvalidasi: Kebutuhan 2.5, 14.10**
func TestProperty_BurstFieldsAllOrNothing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 4 boolean untuk menentukan apakah field diisi atau nil
		hasDown := rapid.Bool().Draw(t, "hasDown")
		hasUp := rapid.Bool().Draw(t, "hasUp")
		hasThreshold := rapid.Bool().Draw(t, "hasThreshold")
		hasTime := rapid.Bool().Draw(t, "hasTime")

		var burstDown, burstUp, burstThreshold, burstTime *int

		if hasDown {
			v := rapid.IntRange(1, 1000).Draw(t, "burstDown")
			burstDown = &v
		}
		if hasUp {
			v := rapid.IntRange(1, 1000).Draw(t, "burstUp")
			burstUp = &v
		}
		if hasThreshold {
			v := rapid.IntRange(1, 1000).Draw(t, "burstThreshold")
			burstThreshold = &v
		}
		if hasTime {
			v := rapid.IntRange(1, 1000).Draw(t, "burstTime")
			burstTime = &v
		}

		err := ValidateBurstFields(burstDown, burstUp, burstThreshold, burstTime)

		count := 0
		if hasDown {
			count++
		}
		if hasUp {
			count++
		}
		if hasThreshold {
			count++
		}
		if hasTime {
			count++
		}

		allOrNone := count == 0 || count == 4

		if allOrNone {
			if err != nil {
				t.Fatalf("expected nil for count=%d, got %v", count, err)
			}
		} else {
			if err == nil {
				t.Fatalf("expected ErrBurstFieldsIncomplete for count=%d, got nil", count)
			}
			if !errors.Is(err, ErrBurstFieldsIncomplete) {
				t.Fatalf("expected ErrBurstFieldsIncomplete, got %v", err)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 9.1, 9.2**
func TestProperty_DuplicateNameGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		originalName := rapid.StringMatching(`[a-zA-Z0-9 ]{1,30}`).Draw(t, "originalName")

		// Buat daftar nama yang sudah ada (0-10 nama copy)
		numExisting := rapid.IntRange(0, 10).Draw(t, "numExisting")
		existingNames := make([]string, 0, numExisting)
		for i := 0; i < numExisting; i++ {
			if i == 0 {
				existingNames = append(existingNames, originalName+" (Copy)")
			} else {
				existingNames = append(existingNames, rapid.Just(originalName+" (Copy "+
					strings.TrimSpace(rapid.SampledFrom([]string{
						"2", "3", "4", "5", "6", "7", "8", "9", "10",
					}).Draw(t, "suffix"))+")").Draw(t, "existingName"))
			}
		}

		result := GenerateDuplicateName(originalName, existingNames)

		// Properti 11a: Hasil tidak boleh ada di existingNames
		for _, n := range existingNames {
			if result == n {
				t.Fatalf("result %q is in existingNames %v", result, existingNames)
			}
		}

		// Properti 11b: Hasil harus dimulai dengan originalName
		if !strings.HasPrefix(result, originalName+" (Copy") {
			t.Fatalf("result %q does not start with %q", result, originalName+" (Copy")
		}

		// Properti 11c: Jika tidak ada collision, hasilnya harus "{name} (Copy)"
		existingSet := make(map[string]struct{}, len(existingNames))
		for _, n := range existingNames {
			existingSet[n] = struct{}{}
		}
		copyName := originalName + " (Copy)"
		if _, found := existingSet[copyName]; !found {
			if result != copyName {
				t.Fatalf("expected %q when no collision, got %q", copyName, result)
			}
		}
	})
}
