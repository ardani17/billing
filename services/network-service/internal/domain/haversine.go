package domain

import (
	"fmt"
	"math"
)

// =============================================================================
// Konstanta Haversine — parameter untuk kalkulasi jarak geospasial
// =============================================================================

const (
	// EarthRadiusMeters adalah radius rata-rata bumi dalam meter.
	// Digunakan sebagai parameter dalam formula Haversine.
	EarthRadiusMeters = 6371000.0
)

// =============================================================================
// Haversine — menghitung jarak antara dua koordinat GPS dalam meter
// =============================================================================

// Haversine menghitung jarak antara dua koordinat GPS dalam meter
// menggunakan formula Haversine dengan radius bumi 6371000 meter.
//
// Parameter lat1, lng1 adalah koordinat titik pertama (derajat).
// Parameter lat2, lng2 adalah koordinat titik kedua (derajat).
// Mengembalikan jarak dalam meter.
//
// Fungsi pure — tidak ada side effect, cocok untuk property-based testing.
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	// Konversi derajat ke radian
	lat1Rad := degreesToRadians(lat1)
	lng1Rad := degreesToRadians(lng1)
	lat2Rad := degreesToRadians(lat2)
	lng2Rad := degreesToRadians(lng2)

	// Selisih latitude dan longitude
	dLat := lat2Rad - lat1Rad
	dLng := lng2Rad - lng1Rad

	// Formula Haversine
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusMeters * c
}

// =============================================================================
// CalculateRouteDistance — menghitung total jarak polyline dari array koordinat
// =============================================================================

// CalculateRouteDistance menghitung total jarak polyline dari array koordinat.
// Menjumlahkan Haversine distance antara setiap pasangan koordinat berurutan.
// Mengembalikan 0 jika kurang dari 2 koordinat.
//
// Format koordinat: [][2]float64 dimana [0] = latitude, [1] = longitude.
//
// Fungsi pure — tidak ada side effect, cocok untuk property-based testing.
func CalculateRouteDistance(coordinates [][2]float64) float64 {
	if len(coordinates) < 2 {
		return 0
	}

	var totalDistance float64
	for i := 1; i < len(coordinates); i++ {
		totalDistance += Haversine(
			coordinates[i-1][0], coordinates[i-1][1],
			coordinates[i][0], coordinates[i][1],
		)
	}

	return totalDistance
}

// =============================================================================
// ValidateCoordinate — validasi koordinat GPS dalam range yang valid
// =============================================================================

// ValidateCoordinate memvalidasi apakah koordinat GPS berada dalam range yang valid.
// Latitude harus dalam range [-90, 90] dan longitude dalam range [-180, 180].
// Mengembalikan ErrInvalidCoordinates jika koordinat di luar range.
func ValidateCoordinate(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("%w: latitude %.6f di luar range [-90, 90]", ErrInvalidCoordinates, lat)
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("%w: longitude %.6f di luar range [-180, 180]", ErrInvalidCoordinates, lng)
	}
	return nil
}

// =============================================================================
// Helper internal — konversi derajat ke radian
// =============================================================================

// degreesToRadians mengkonversi sudut dari derajat ke radian.
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
