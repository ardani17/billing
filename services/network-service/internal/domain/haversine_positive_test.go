package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

// TestHaversinePositiveDistance memverifikasi bahwa untuk array koordinat
// dengan ≥2 titik distinct (minimal dua titik berurutan berbeda),
// CalculateRouteDistance menghasilkan nilai > 0.
//
// **Memvalidasi: Kebutuhan 25.3, 3.6**
func TestHaversinePositiveDistance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat 2 koordinat distinct - pastikan berbeda minimal 0.001 derajat
		lat1 := rapid.Float64Range(-89.0, 89.0).Draw(t, "lat1")
		lng1 := rapid.Float64Range(-179.0, 179.0).Draw(t, "lng1")

		// Buat offset minimal 0.001 derajat untuk memastikan titik berbeda
		latOffset := rapid.Float64Range(0.001, 1.0).Draw(t, "latOffset")
		lngOffset := rapid.Float64Range(0.001, 1.0).Draw(t, "lngOffset")

		// Pilih arah offset secara acak (positif atau negatif)
		latSign := rapid.SampledFrom([]float64{-1.0, 1.0}).Draw(t, "latSign")
		lngSign := rapid.SampledFrom([]float64{-1.0, 1.0}).Draw(t, "lngSign")

		lat2 := lat1 + latOffset*latSign
		lng2 := lng1 + lngOffset*lngSign

		if lat2 > 90.0 {
			lat2 = 90.0
		}
		if lat2 < -90.0 {
			lat2 = -90.0
		}
		if lng2 > 180.0 {
			lng2 = 180.0
		}
		if lng2 < -180.0 {
			lng2 = -180.0
		}

		// Verifikasi Haversine antara dua titik distinct menghasilkan > 0
		dist := Haversine(lat1, lng1, lat2, lng2)
		if dist <= 0 {
			t.Fatalf(
				"Haversine harus > 0 untuk titik distinct: (%f, %f) → (%f, %f), got %f",
				lat1, lng1, lat2, lng2, dist,
			)
		}

		// Verifikasi CalculateRouteDistance dengan array ≥2 titik distinct
		coords := [][2]float64{{lat1, lng1}, {lat2, lng2}}
		routeDist := CalculateRouteDistance(coords)
		if routeDist <= 0 {
			t.Fatalf(
				"CalculateRouteDistance harus > 0 untuk ≥2 titik distinct, got %f",
				routeDist,
			)
		}
	})
}

// TestHaversinePositiveDistanceMultiplePoints memverifikasi bahwa array
// dengan ≥2 titik distinct menghasilkan CalculateRouteDistance > 0,
// bahkan dengan jumlah titik yang lebih banyak.
//
// **Memvalidasi: Kebutuhan 25.3**
func TestHaversinePositiveDistanceMultiplePoints(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat jumlah titik antara 2-10
		n := rapid.IntRange(2, 10).Draw(t, "numPoints")

		// Buat titik pertama
		coords := make([][2]float64, n)
		coords[0] = [2]float64{
			rapid.Float64Range(-89.0, 89.0).Draw(t, "lat0"),
			rapid.Float64Range(-179.0, 179.0).Draw(t, "lng0"),
		}

		// Buat titik-titik berikutnya, pastikan minimal satu pasangan berurutan distinct
		for i := 1; i < n; i++ {
			lat := rapid.Float64Range(-89.0, 89.0).Draw(t, "lat")
			lng := rapid.Float64Range(-179.0, 179.0).Draw(t, "lng")
			coords[i] = [2]float64{lat, lng}
		}

		// Pastikan titik pertama dan kedua distinct (minimal 0.001 derajat)
		offset := rapid.Float64Range(0.001, 1.0).Draw(t, "offset")
		sign := rapid.SampledFrom([]float64{-1.0, 1.0}).Draw(t, "sign")
		coords[1][0] = coords[0][0] + offset*sign
		if coords[1][0] > 90.0 {
			coords[1][0] = 90.0
		}
		if coords[1][0] < -90.0 {
			coords[1][0] = -90.0
		}

		routeDist := CalculateRouteDistance(coords)
		if routeDist <= 0 {
			t.Fatalf(
				"CalculateRouteDistance harus > 0 untuk array dengan titik distinct, got %f (n=%d)",
				routeDist, n,
			)
		}
	})
}

// =============================================================================
// =============================================================================

// TestDistanceCalculationDeterminism memverifikasi bahwa dua kali kalkulasi
//
// **Memvalidasi: Kebutuhan 25.3, 3.6**
func TestDistanceCalculationDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat jumlah titik antara 0-20
		n := rapid.IntRange(0, 20).Draw(t, "numPoints")

		// Buat array koordinat acak
		coords := make([][2]float64, n)
		for i := 0; i < n; i++ {
			lat := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat")
			lng := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng")
			coords[i] = [2]float64{lat, lng}
		}

		result1 := CalculateRouteDistance(coords)
		result2 := CalculateRouteDistance(coords)

		// Kedua hasil harus identik (bit-untuk-bit)
		if result1 != result2 {
			t.Fatalf(
				"CalculateRouteDistance tidak deterministik: panggilan pertama=%f, panggilan kedua=%f",
				result1, result2,
			)
		}
	})
}
