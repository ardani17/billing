package domain

import (
	"errors"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property Test: Coordinate Validation
// Memverifikasi bahwa ValidateCoordinate menerima koordinat valid dan
// menolak koordinat di luar range yang diizinkan.
//
// **Property 7: Coordinate Validation**
// **Validates: Requirements 7.7**
// =============================================================================

// TestCoordinateValidation_ValidCoordinates memverifikasi bahwa koordinat
// dengan latitude dalam [-90, 90] dan longitude dalam [-180, 180]
// selalu diterima (ValidateCoordinate mengembalikan nil).
func TestCoordinateValidation_ValidCoordinates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate latitude dalam range valid [-90, 90]
		lat := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat")
		// Generate longitude dalam range valid [-180, 180]
		lng := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng")

		err := ValidateCoordinate(lat, lng)
		if err != nil {
			t.Fatalf(
				"koordinat valid ditolak: lat=%.6f, lng=%.6f, error=%v",
				lat, lng, err,
			)
		}
	})
}

// TestCoordinateValidation_InvalidLatitudeTooHigh memverifikasi bahwa
// latitude di atas 90 selalu ditolak dengan error ErrInvalidCoordinates.
func TestCoordinateValidation_InvalidLatitudeTooHigh(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate latitude di atas 90 (exclusive), range (90, 1000]
		lat := rapid.Float64Range(90.0+1e-9, 1000.0).Draw(t, "lat")
		// Generate longitude valid agar hanya latitude yang menyebabkan error
		lng := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng")

		err := ValidateCoordinate(lat, lng)
		if err == nil {
			t.Fatalf(
				"latitude terlalu tinggi tidak ditolak: lat=%.6f, lng=%.6f",
				lat, lng,
			)
		}
		if !errors.Is(err, ErrInvalidCoordinates) {
			t.Fatalf(
				"error bukan ErrInvalidCoordinates: lat=%.6f, lng=%.6f, error=%v",
				lat, lng, err,
			)
		}
	})
}

// TestCoordinateValidation_InvalidLatitudeTooLow memverifikasi bahwa
// latitude di bawah -90 selalu ditolak dengan error ErrInvalidCoordinates.
func TestCoordinateValidation_InvalidLatitudeTooLow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate latitude di bawah -90 (exclusive), range [-1000, -90)
		lat := rapid.Float64Range(-1000.0, -90.0-1e-9).Draw(t, "lat")
		// Generate longitude valid agar hanya latitude yang menyebabkan error
		lng := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng")

		err := ValidateCoordinate(lat, lng)
		if err == nil {
			t.Fatalf(
				"latitude terlalu rendah tidak ditolak: lat=%.6f, lng=%.6f",
				lat, lng,
			)
		}
		if !errors.Is(err, ErrInvalidCoordinates) {
			t.Fatalf(
				"error bukan ErrInvalidCoordinates: lat=%.6f, lng=%.6f, error=%v",
				lat, lng, err,
			)
		}
	})
}

// TestCoordinateValidation_InvalidLongitudeTooHigh memverifikasi bahwa
// longitude di atas 180 selalu ditolak dengan error ErrInvalidCoordinates.
func TestCoordinateValidation_InvalidLongitudeTooHigh(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate latitude valid agar hanya longitude yang menyebabkan error
		lat := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat")
		// Generate longitude di atas 180 (exclusive), range (180, 1000]
		lng := rapid.Float64Range(180.0+1e-9, 1000.0).Draw(t, "lng")

		err := ValidateCoordinate(lat, lng)
		if err == nil {
			t.Fatalf(
				"longitude terlalu tinggi tidak ditolak: lat=%.6f, lng=%.6f",
				lat, lng,
			)
		}
		if !errors.Is(err, ErrInvalidCoordinates) {
			t.Fatalf(
				"error bukan ErrInvalidCoordinates: lat=%.6f, lng=%.6f, error=%v",
				lat, lng, err,
			)
		}
	})
}

// TestCoordinateValidation_InvalidLongitudeTooLow memverifikasi bahwa
// longitude di bawah -180 selalu ditolak dengan error ErrInvalidCoordinates.
func TestCoordinateValidation_InvalidLongitudeTooLow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate latitude valid agar hanya longitude yang menyebabkan error
		lat := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat")
		// Generate longitude di bawah -180 (exclusive), range [-1000, -180)
		lng := rapid.Float64Range(-1000.0, -180.0-1e-9).Draw(t, "lng")

		err := ValidateCoordinate(lat, lng)
		if err == nil {
			t.Fatalf(
				"longitude terlalu rendah tidak ditolak: lat=%.6f, lng=%.6f",
				lat, lng,
			)
		}
		if !errors.Is(err, ErrInvalidCoordinates) {
			t.Fatalf(
				"error bukan ErrInvalidCoordinates: lat=%.6f, lng=%.6f, error=%v",
				lat, lng, err,
			)
		}
	})
}
