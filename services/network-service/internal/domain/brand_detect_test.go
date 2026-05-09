package domain

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// brandKeywords memetakan setiap OLTBrand ke daftar keyword yang dikenali.
// Sesuai dengan logika DetectBrand di olt.go dan Kebutuhan 6.6.
var brandKeywords = map[OLTBrand][]string{
	BrandZTE:       {"ZTE", "ZXA10"},
	BrandHuawei:    {"Huawei", "MA56"},
	BrandFiberHome: {"FiberHome", "AN5516"},
	BrandVSOL:      {"VSOL", "V1600"},
	BrandHSGQ:      {"HSGQ"},
}

// allBrandKeywordsLower berisi semua keyword brand dalam lowercase untuk pengecekan.
var allBrandKeywordsLower []string

func init() {
	for _, keywords := range brandKeywords {
		for _, kw := range keywords {
			allBrandKeywordsLower = append(allBrandKeywordsLower, strings.ToLower(kw))
		}
	}
}

// containsAnyBrandKeyword memeriksa apakah string mengandung keyword brand apapun.
func containsAnyBrandKeyword(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range allBrandKeywordsLower {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// =============================================================================
// =============================================================================

// TestProperty_BrandDetectionFromSysDescr memverifikasi bahwa untuk sembarang
// brand keyword yang disisipkan dalam teks acak, DetectBrand mengembalikan
// OLTBrand constant yang sesuai, terlepas dari teks sekitarnya.
//
// **Memvalidasi: Kebutuhan 6.2, 6.6**
func TestProperty_BrandDetectionFromSysDescr(t *testing.T) {
	// Generator untuk semua brand
	allBrands := make([]OLTBrand, 0, len(brandKeywords))
	for brand := range brandKeywords {
		allBrands = append(allBrands, brand)
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pilih brand acak
		brand := rapid.SampledFrom(allBrands).Draw(t, "brand")
		keywords := brandKeywords[brand]

		// Pilih keyword acak dari brand tersebut
		keyword := rapid.SampledFrom(keywords).Draw(t, "keyword")

		// Buat teks acak sebelum dan sesudah keyword
		prefix := rapid.String().Draw(t, "prefix")
		suffix := rapid.String().Draw(t, "suffix")

		// Pastikan prefix dan suffix tidak mengandung keyword brand lain
		// yang bisa mengubah hasil deteksi (karena DetectBrand pakai switch urutan)
		if containsAnyBrandKeyword(prefix) || containsAnyBrandKeyword(suffix) {
			return // skip iterasi ini
		}

		sysDescr := prefix + keyword + suffix
		result := DetectBrand(sysDescr)

		if result != brand {
			t.Errorf(
				"DetectBrand(%q) = %q, ingin %q (keyword: %q)",
				sysDescr, result, brand, keyword,
			)
		}
	})
}

// TestProperty_BrandDetectionNoBrandKeyword memverifikasi bahwa untuk sembarang
// string yang TIDAK mengandung keyword brand apapun, DetectBrand mengembalikan
// string kosong.
//
// **Memvalidasi: Kebutuhan 6.2, 6.6**
func TestProperty_BrandDetectionNoBrandKeyword(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat string acak
		s := rapid.String().Draw(t, "sysDescr")

		// Skip jika kebetulan mengandung keyword brand
		if containsAnyBrandKeyword(s) {
			return
		}

		result := DetectBrand(s)
		if result != "" {
			t.Errorf(
				"DetectBrand(%q) = %q, seharusnya string kosong untuk input tanpa keyword brand",
				s, result,
			)
		}
	})
}

// TestProperty_BrandDetectionCaseInsensitive memverifikasi bahwa DetectBrand
// bersifat case-insensitive - keyword dalam huruf besar, kecil, atau campuran
// tetap terdeteksi dengan benar.
//
// **Memvalidasi: Kebutuhan 6.2, 6.6**
func TestProperty_BrandDetectionCaseInsensitive(t *testing.T) {
	allBrands := make([]OLTBrand, 0, len(brandKeywords))
	for brand := range brandKeywords {
		allBrands = append(allBrands, brand)
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pilih brand dan keyword acak
		brand := rapid.SampledFrom(allBrands).Draw(t, "brand")
		keywords := brandKeywords[brand]
		keyword := rapid.SampledFrom(keywords).Draw(t, "keyword")

		// Ubah case setiap karakter secara acak
		var builder strings.Builder
		for _, ch := range keyword {
			if rapid.Bool().Draw(t, "upper") {
				builder.WriteString(strings.ToUpper(string(ch)))
			} else {
				builder.WriteString(strings.ToLower(string(ch)))
			}
		}
		randomCaseKeyword := builder.String()

		result := DetectBrand(randomCaseKeyword)
		if result != brand {
			t.Errorf(
				"DetectBrand(%q) = %q, ingin %q (keyword asli: %q)",
				randomCaseKeyword, result, brand, keyword,
			)
		}
	})
}

// =============================================================================
// Example-based tests untuk sysDescr string dari perangkat OLT nyata
// =============================================================================

// TestBrandDetection_KnownSysDescr memverifikasi deteksi brand dari sysDescr
// string yang diketahui dari perangkat OLT nyata.
//
// **Memvalidasi: Kebutuhan 6.2, 6.6**
func TestBrandDetection_KnownSysDescr(t *testing.T) {
	cases := []struct {
		name     string
		sysDescr string
		expected OLTBrand
	}{
		// ZTE
		{"ZTE C320", "ZTE ZXA10 C320 Version V2.1.0", BrandZTE},
		{"ZTE C300", "ZXA10 C300 Optical Access System", BrandZTE},
		{"ZTE C600", "ZTE C600 GPON OLT", BrandZTE},

		// Huawei
		{"Huawei MA5680T", "Huawei Integrated Access Software MA5680T", BrandHuawei},
		{"Huawei MA5608T", "MA5608T Huawei SmartAX", BrandHuawei},
		{"Huawei MA5683T", "Huawei Technologies MA5683T V800R017", BrandHuawei},

		// FiberHome
		{"FiberHome AN5516-01", "FiberHome AN5516-01 GPON OLT", BrandFiberHome},
		{"FiberHome AN5516-04", "AN5516-04 FiberHome Communication", BrandFiberHome},

		// VSOL
		{"VSOL V1600G", "VSOL V1600G GPON OLT", BrandVSOL},
		{"VSOL V1600D", "V1600D-8 VSOL Technology", BrandVSOL},

		// HSGQ
		{"HSGQ OLT", "HSGQ GPON OLT System", BrandHSGQ},

		// Tidak dikenali
		{"Unknown vendor", "Cisco IOS Software C7200", ""},
		{"Empty string", "", ""},
		{"Random text", "some random device description", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := DetectBrand(tc.sysDescr)
			if result != tc.expected {
				t.Errorf(
					"DetectBrand(%q) = %q, ingin %q",
					tc.sysDescr, result, tc.expected,
				)
			}
		})
	}
}
