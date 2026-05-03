// validator.go mendaftarkan custom validator untuk field pelanggan.
// Custom validator: phone_id (format telepon Indonesia) dan mac_addr (format MAC address).
package handler

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// phoneIDRegex memvalidasi format telepon Indonesia: +62 diikuti 9-13 digit.
// Total panjang: 12-16 karakter termasuk prefix +62.
var phoneIDRegex = regexp.MustCompile(`^\+62\d{9,13}$`)

// macAddrRegex memvalidasi format MAC address: enam grup dua digit hex dipisahkan titik dua.
// Contoh: AA:BB:CC:DD:EE:FF (case-insensitive).
var macAddrRegex = regexp.MustCompile(`^[0-9A-Fa-f]{2}(:[0-9A-Fa-f]{2}){5}$`)

// RegisterCustomValidators mendaftarkan custom validator pada instance validator.
// - phone_id: validasi format telepon Indonesia (+62, 9-13 digit setelah prefix)
// - mac_addr: validasi format MAC address (AA:BB:CC:DD:EE:FF)
func RegisterCustomValidators(v *validator.Validate) {
	_ = v.RegisterValidation("phone_id", validatePhoneID)
	_ = v.RegisterValidation("mac_addr", validateMACAddress)
}

// validatePhoneID memvalidasi bahwa field dimulai dengan +62 diikuti 9-13 digit.
func validatePhoneID(fl validator.FieldLevel) bool {
	return phoneIDRegex.MatchString(fl.Field().String())
}

// validateMACAddress memvalidasi bahwa field sesuai format MAC address XX:XX:XX:XX:XX:XX.
func validateMACAddress(fl validator.FieldLevel) bool {
	return macAddrRegex.MatchString(fl.Field().String())
}
