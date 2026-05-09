package domain

import (
	"errors"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

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

func isValidMethod(provider GatewayProvider, method string) bool {
	if provider == GatewayMidtrans {
		return ValidMidtransMethods[method]
	}
	return ValidXenditMethods[method]
}

// TestProperty_ValidateEnabledMethods memverifikasi bahwa untuk sembarang
// gateway provider (Xendit atau Midtrans) dan sembarang method string:
// - ValidateEnabledMethods(provider, []string{method}) mengembalikan nil
//
// **Memvalidasi: Kebutuhan 1.8**
func TestProperty_ValidateEnabledMethods(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih provider acak: Xendit atau Midtrans
		provider := rapid.SampledFrom([]GatewayProvider{GatewayXendit, GatewayMidtrans}).Draw(t, "provider")

		// Buat string method secara acak
		method := rapid.String().Draw(t, "method")

		// Panggil ValidateEnabledMethods dengan satu method
		err := ValidateEnabledMethods(provider, []string{method})

		if isValidMethod(provider, method) {
			if err != nil {
				t.Errorf("method %q valid untuk %s, tapi error: %v", method, provider, err)
			}
		} else {
			if err == nil {
				t.Errorf("method %q tidak valid untuk %s, tapi tidak error", method, provider)
			}
			if !errors.Is(err, ErrInvalidEnabledMethods) {
				t.Errorf("error = %v, ingin ErrInvalidEnabledMethods", err)
			}
		}
	})

	t.Run("valid_methods_always_pass", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			provider := rapid.SampledFrom([]GatewayProvider{GatewayXendit, GatewayMidtrans}).Draw(t, "provider")

			validMethods := allValidMethods(provider)
			method := rapid.SampledFrom(validMethods).Draw(t, "validMethod")

			err := ValidateEnabledMethods(provider, []string{method})
			if err != nil {
				t.Errorf("method valid %q untuk %s seharusnya nil, tapi error: %v", method, provider, err)
			}
		})
	})
}
