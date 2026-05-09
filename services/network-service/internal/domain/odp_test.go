package domain

import (
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

var validSplitterTypes = map[string]int{
	SplitterType1x4:  4,
	SplitterType1x8:  8,
	SplitterType1x16: 16,
	SplitterType1x32: 32,
}

var validSplitterTypeSlice = []string{
	SplitterType1x4,
	SplitterType1x8,
	SplitterType1x16,
	SplitterType1x32,
}

// =============================================================================
// =============================================================================

// TestProperty_SplitterCapacityValidTypes memverifikasi bahwa untuk sembarang
// sesuai rasio splitter.
//
// **Memvalidasi: Kebutuhan 8.5**
func TestProperty_SplitterCapacityValidTypes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		splitterType := rapid.SampledFrom(validSplitterTypeSlice).Draw(t, "splitterType")

		result := SplitterCapacity(splitterType)
		expected := validSplitterTypes[splitterType]

		if result != expected {
			t.Errorf(
				"SplitterCapacity(%q) = %d, ingin %d",
				splitterType, result, expected,
			)
		}
	})
}

// TestProperty_SplitterCapacityInvalidTypes memverifikasi bahwa untuk sembarang
//
// **Memvalidasi: Kebutuhan 8.5**
func TestProperty_SplitterCapacityInvalidTypes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat string acak
		randomStr := rapid.String().Draw(t, "randomStr")

		for _, valid := range validSplitterTypeSlice {
			if randomStr == valid {
				return // skip iterasi ini, kebetulan valid
			}
		}

		result := SplitterCapacity(randomStr)
		if result != 0 {
			t.Errorf(
				"SplitterCapacity(%q) = %d, ingin 0 untuk tipe splitter tidak valid",
				randomStr, result,
			)
		}
	})
}

// TestProperty_SplitterCapacityMatchesColonNumber memverifikasi bahwa untuk
// setelah tanda titik dua dalam string tipe splitter (misal "1:32" -> 32).
//
// **Memvalidasi: Kebutuhan 8.5**
func TestProperty_SplitterCapacityMatchesColonNumber(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		splitterType := rapid.SampledFrom(validSplitterTypeSlice).Draw(t, "splitterType")

		capacity := SplitterCapacity(splitterType)

		parts := strings.SplitN(splitterType, ":", 2)
		if len(parts) != 2 {
			t.Fatalf("format splitter type %q tidak mengandung titik dua", splitterType)
		}

		expectedNum, err := strconv.Atoi(parts[1])
		if err != nil {
			t.Fatalf("gagal parse angka dari %q: %v", parts[1], err)
		}

		if capacity != expectedNum {
			t.Errorf(
				"SplitterCapacity(%q) = %d, ingin %d (angka setelah titik dua)",
				splitterType, capacity, expectedNum,
			)
		}
	})
}

// =============================================================================
// Example-based tests untuk splitter capacity mapping
// =============================================================================

// TestSplitterCapacity_ValidTypes memverifikasi kapasitas untuk semua 4 tipe
//
// **Memvalidasi: Kebutuhan 8.5**
func TestSplitterCapacity_ValidTypes(t *testing.T) {
	cases := []struct {
		splitterType string
		expected     int
	}{
		{"1:4", 4},
		{"1:8", 8},
		{"1:16", 16},
		{"1:32", 32},
	}

	for _, tc := range cases {
		t.Run(tc.splitterType, func(t *testing.T) {
			result := SplitterCapacity(tc.splitterType)
			if result != tc.expected {
				t.Errorf(
					"SplitterCapacity(%q) = %d, ingin %d",
					tc.splitterType, result, tc.expected,
				)
			}
		})
	}
}

// TestSplitterCapacity_InvalidTypes memverifikasi bahwa tipe splitter tidak
//
// **Memvalidasi: Kebutuhan 8.5**
func TestSplitterCapacity_InvalidTypes(t *testing.T) {
	invalidTypes := []struct {
		name         string
		splitterType string
	}{
		{"string kosong", ""},
		{"tanpa prefix", "32"},
		{"format salah", "1-32"},
		{"rasio tidak didukung", "1:64"},
		{"rasio tidak didukung kecil", "1:2"},
		{"huruf besar", "1:4 "},
		{"spasi di depan", " 1:4"},
		{"teks acak", "abc"},
		{"angka saja", "123"},
		{"prefix salah", "2:4"},
	}

	for _, tc := range invalidTypes {
		t.Run(tc.name, func(t *testing.T) {
			result := SplitterCapacity(tc.splitterType)
			if result != 0 {
				t.Errorf(
					"SplitterCapacity(%q) = %d, ingin 0 untuk tipe tidak valid",
					tc.splitterType, result,
				)
			}
		})
	}
}
