package domain

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property Test: Haversine Segment Additivity
// =============================================================================

// TestHaversineSegmentAdditivity memverifikasi bahwa CalculateRouteDistance
// untuk array koordinat [A,B,C,...,N] sama dengan penjumlahan Haversine
// per segment: Haversine(A,B) + Haversine(B,C) + ... + Haversine(M,N).
//
// **Validates: Requirements 25.1, 25.4**
func TestHaversineSegmentAdditivity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate jumlah titik antara 2-20
		n := rapid.IntRange(2, 20).Draw(t, "numPoints")

		// Generate array koordinat dengan lat dalam [-90,90] dan lng dalam [-180,180]
		coords := make([][2]float64, n)
		for i := 0; i < n; i++ {
			lat := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat")
			lng := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng")
			coords[i] = [2]float64{lat, lng}
		}

		// Hitung total jarak menggunakan CalculateRouteDistance
		totalDistance := CalculateRouteDistance(coords)

		// Hitung penjumlahan Haversine per segment secara manual
		var sumSegments float64
		for i := 1; i < len(coords); i++ {
			sumSegments += Haversine(
				coords[i-1][0], coords[i-1][1],
				coords[i][0], coords[i][1],
			)
		}

		// Bandingkan dengan toleransi floating point kecil
		if math.Abs(totalDistance-sumSegments) > 1e-9 {
			t.Fatalf(
				"segment additivity gagal: CalculateRouteDistance=%.15f, sum segments=%.15f, selisih=%.15e",
				totalDistance, sumSegments, math.Abs(totalDistance-sumSegments),
			)
		}
	})
}

// =============================================================================
// Unit Test: Jarak Jakarta-Bandung ≈ 120km
// =============================================================================

// TestJakartaBandungDistance memverifikasi bahwa jarak Jakarta-Bandung
// yang dihitung menggunakan CalculateRouteDistance mendekati 120km (±5km).
func TestJakartaBandungDistance(t *testing.T) {
	// Koordinat Jakarta dan Bandung
	jakarta := [2]float64{-6.2088, 106.8456}
	bandung := [2]float64{-6.9175, 107.6191}

	coords := [][2]float64{jakarta, bandung}
	distance := CalculateRouteDistance(coords)

	// Toleransi ±5km (5000 meter)
	expectedMeters := 120000.0
	toleranceMeters := 5000.0

	if math.Abs(distance-expectedMeters) > toleranceMeters {
		t.Errorf(
			"jarak Jakarta-Bandung di luar toleransi: got %.2f meter, expected %.2f ± %.2f meter",
			distance, expectedMeters, toleranceMeters,
		)
	}
}

// =============================================================================
// Unit Test: Edge cases — 0 dan 1 koordinat mengembalikan 0
// =============================================================================

// TestCalculateRouteDistanceEmptyCoordinates memverifikasi bahwa
// CalculateRouteDistance mengembalikan 0 untuk array kosong.
func TestCalculateRouteDistanceEmptyCoordinates(t *testing.T) {
	coords := [][2]float64{}
	distance := CalculateRouteDistance(coords)

	if distance != 0 {
		t.Errorf("expected 0 untuk array kosong, got %f", distance)
	}
}

// TestCalculateRouteDistanceSingleCoordinate memverifikasi bahwa
// CalculateRouteDistance mengembalikan 0 untuk array dengan 1 koordinat.
func TestCalculateRouteDistanceSingleCoordinate(t *testing.T) {
	coords := [][2]float64{{-6.2088, 106.8456}}
	distance := CalculateRouteDistance(coords)

	if distance != 0 {
		t.Errorf("expected 0 untuk 1 koordinat, got %f", distance)
	}
}
