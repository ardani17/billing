package domain

import (
	"errors"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property-based test untuk validasi enabled methods per provider
// =============================================================================

// allValidMethods mengembalikan slice berisi semua metode valid untuk provider.
func allValidMethods(provider GatewayProvider) []string {
	validMap := ValidXenditMethods
	if provider == GatewayMidtrans {
		validMap = ValidMidtransMethods
	}
	methods := make([]string, 0, len(validMap))
	for m := range validMap {
		methods = append(methods, m)
	}
	return methods
}

// isValidMethod mengecek apakah method ada di set valid untuk provider.
func isValidMethod(provider GatewayProvider, method string) bool {
	if provider == GatewayMidtrans {
		return ValidMidtransMethods[method]
	}
	return ValidXenditMethods[method]
}

// TestProperty_ValidateEnabledMethods memverifikasi bahwa untuk sembarang
// gateway provider (Xendit atau Midtrans) dan sembarang method string:
// - ValidateEnabledMethods(provider, []string{method}) mengembalikan nil
//   jika dan hanya jika method ada di set valid untuk provider tersebut.
// - Mengembalikan error yang membungkus ErrInvalidEnabledMethods untuk
//   method yang tidak ada di set valid.
//
// **Validates: Requirements 1.8**
func TestProperty_ValidateEnabledMethods(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih provider acak: Xendit atau Midtrans
		provider := rapid.SampledFrom([]GatewayProvider{GatewayXendit, GatewayMidtrans}).Draw(t, "provider")

		// Generate method string acak
		method := rapid.String().Draw(t, "method")

		// Panggil ValidateEnabledMethods dengan satu method
		err := ValidateEnabledMethods(provider, []string{method})

		if isValidMethod(provider, method) {
			// Method valid: harus mengembalikan nil
			if err != nil {
				t.Errorf("method %q valid untuk %s, tapi error: %v", method, provider, err)
			}
		} else {
			// Method tidak valid: harus mengembalikan error ErrInvalidEnabledMethods
			if err == nil {
				t.Errorf("method %q tidak valid untuk %s, tapi tidak error", method, provider)
			}
			if !errors.Is(err, ErrInvalidEnabledMethods) {
				t.Errorf("error = %v, ingin ErrInvalidEnabledMethods", err)
			}
		}
	})

	// Sub-test: metode valid dari set asli selalu lolos validasi
	t.Run("valid_methods_always_pass", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			provider := rapid.SampledFrom([]GatewayProvider{GatewayXendit, GatewayMidtrans}).Draw(t, "provider")

			// Ambil metode valid dari set asli
			validMethods := allValidMethods(provider)
			method := rapid.SampledFrom(validMethods).Draw(t, "validMethod")

			err := ValidateEnabledMethods(provider, []string{method})
			if err != nil {
				t.Errorf("method valid %q untuk %s seharusnya nil, tapi error: %v", method, provider, err)
			}
		})
	})
}
