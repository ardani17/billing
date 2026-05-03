package handler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"pgregory.net/rapid"
)

// Feature: customer-crud, Property 6: Field Validation Rules
// **Validates: Requirements 22.1, 22.3, 22.4, 22.5, 22.6, 22.7**
//
// For any string input to the phone validator, it SHALL be accepted iff it starts
// with +62 followed by 9-13 digits. For any float input to coordinate validators,
// latitude SHALL be accepted iff in [-90,90] and longitude in [-180,180].
// For any string input to mac_address, it SHALL be accepted iff it matches
// XX:XX:XX:XX:XX:XX where X is hex. For any integer due_date, it SHALL be
// accepted iff in [1,28]. For any string name, it SHALL be accepted iff length
// in [3,255]. For any string address, it SHALL be accepted iff non-empty and
// length <= 1000.

// testStruct is a helper struct used to validate individual fields via struct tags.
type testPhoneStruct struct {
	Phone string `validate:"phone_id"`
}

type testMACStruct struct {
	MAC string `validate:"mac_addr"`
}

type testLatStruct struct {
	Latitude float64 `validate:"min=-90,max=90"`
}

type testLngStruct struct {
	Longitude float64 `validate:"min=-180,max=180"`
}

type testDueDateStruct struct {
	DueDate int `validate:"min=1,max=28"`
}

type testNameStruct struct {
	Name string `validate:"min=3,max=255"`
}

type testAddressStruct struct {
	Address string `validate:"required,max=1000"`
}

func newTestValidator() *validator.Validate {
	v := validator.New()
	RegisterCustomValidators(v)
	return v
}

func TestProperty_PhoneValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		// Decide whether to generate a valid or invalid phone
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var phone string
		var expectValid bool

		if generateValid {
			// Valid: +62 followed by 9-13 digits
			digitCount := rapid.IntRange(9, 13).Draw(t, "digitCount")
			digits := make([]byte, digitCount)
			for i := range digits {
				digits[i] = byte('0' + rapid.IntRange(0, 9).Draw(t, fmt.Sprintf("digit_%d", i)))
			}
			phone = "+62" + string(digits)
			expectValid = true
		} else {
			// Generate various invalid phones
			invalidType := rapid.IntRange(0, 5).Draw(t, "invalidType")
			switch invalidType {
			case 0:
				// Wrong prefix
				phone = "+61" + strings.Repeat("1", 10)
			case 1:
				// Too few digits after +62 (0-8 digits)
				digitCount := rapid.IntRange(0, 8).Draw(t, "fewDigits")
				phone = "+62" + strings.Repeat("5", digitCount)
			case 2:
				// Too many digits after +62 (14+ digits)
				digitCount := rapid.IntRange(14, 20).Draw(t, "manyDigits")
				phone = "+62" + strings.Repeat("5", digitCount)
			case 3:
				// No + prefix
				phone = "62" + strings.Repeat("5", 10)
			case 4:
				// Contains non-digit characters after +62
				phone = "+62abc123456"
			case 5:
				// Empty string
				phone = ""
			}
			expectValid = false
		}

		err := v.Struct(testPhoneStruct{Phone: phone})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("phone=%q: expected valid=%v, got valid=%v", phone, expectValid, isValid)
		}
	})
}

func TestProperty_MACAddressValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var mac string
		var expectValid bool

		if generateValid {
			// Valid: six groups of two hex digits separated by colons
			hexChars := "0123456789ABCDEFabcdef"
			groups := make([]string, 6)
			for i := 0; i < 6; i++ {
				c1 := hexChars[rapid.IntRange(0, len(hexChars)-1).Draw(t, fmt.Sprintf("h1_%d", i))]
				c2 := hexChars[rapid.IntRange(0, len(hexChars)-1).Draw(t, fmt.Sprintf("h2_%d", i))]
				groups[i] = string([]byte{c1, c2})
			}
			mac = strings.Join(groups, ":")
			expectValid = true
		} else {
			invalidType := rapid.IntRange(0, 4).Draw(t, "invalidType")
			switch invalidType {
			case 0:
				// Too few groups
				mac = "AA:BB:CC:DD:EE"
			case 1:
				// Too many groups
				mac = "AA:BB:CC:DD:EE:FF:11"
			case 2:
				// Non-hex characters
				mac = "GG:HH:II:JJ:KK:LL"
			case 3:
				// Wrong separator
				mac = "AA-BB-CC-DD-EE-FF"
			case 4:
				// Empty
				mac = ""
			}
			expectValid = false
		}

		err := v.Struct(testMACStruct{MAC: mac})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("mac=%q: expected valid=%v, got valid=%v", mac, expectValid, isValid)
		}
	})
}

func TestProperty_LatitudeValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var lat float64
		var expectValid bool

		if generateValid {
			lat = rapid.Float64Range(-90, 90).Draw(t, "lat")
			expectValid = true
		} else {
			// Generate out-of-range latitude
			if rapid.Bool().Draw(t, "above") {
				lat = rapid.Float64Range(90.001, 1000).Draw(t, "latAbove")
			} else {
				lat = rapid.Float64Range(-1000, -90.001).Draw(t, "latBelow")
			}
			expectValid = false
		}

		err := v.Struct(testLatStruct{Latitude: lat})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("latitude=%v: expected valid=%v, got valid=%v", lat, expectValid, isValid)
		}
	})
}

func TestProperty_LongitudeValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var lng float64
		var expectValid bool

		if generateValid {
			lng = rapid.Float64Range(-180, 180).Draw(t, "lng")
			expectValid = true
		} else {
			if rapid.Bool().Draw(t, "above") {
				lng = rapid.Float64Range(180.001, 1000).Draw(t, "lngAbove")
			} else {
				lng = rapid.Float64Range(-1000, -180.001).Draw(t, "lngBelow")
			}
			expectValid = false
		}

		err := v.Struct(testLngStruct{Longitude: lng})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("longitude=%v: expected valid=%v, got valid=%v", lng, expectValid, isValid)
		}
	})
}

func TestProperty_DueDateValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var dueDate int
		var expectValid bool

		if generateValid {
			dueDate = rapid.IntRange(1, 28).Draw(t, "dueDate")
			expectValid = true
		} else {
			if rapid.Bool().Draw(t, "above") {
				dueDate = rapid.IntRange(29, 100).Draw(t, "dueDateAbove")
			} else {
				dueDate = rapid.IntRange(-100, 0).Draw(t, "dueDateBelow")
			}
			expectValid = false
		}

		err := v.Struct(testDueDateStruct{DueDate: dueDate})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("due_date=%d: expected valid=%v, got valid=%v", dueDate, expectValid, isValid)
		}
	})
}

func TestProperty_NameValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var name string
		var expectValid bool

		if generateValid {
			// Valid: length 3-255
			length := rapid.IntRange(3, 255).Draw(t, "nameLen")
			name = strings.Repeat("a", length)
			expectValid = true
		} else {
			invalidType := rapid.IntRange(0, 1).Draw(t, "invalidType")
			switch invalidType {
			case 0:
				// Too short (1-2 chars)
				length := rapid.IntRange(1, 2).Draw(t, "shortLen")
				name = strings.Repeat("a", length)
			case 1:
				// Too long (256+ chars)
				length := rapid.IntRange(256, 300).Draw(t, "longLen")
				name = strings.Repeat("a", length)
			}
			expectValid = false
		}

		err := v.Struct(testNameStruct{Name: name})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("name (len=%d): expected valid=%v, got valid=%v", len(name), expectValid, isValid)
		}
	})
}

func TestProperty_AddressValidation(t *testing.T) {
	v := newTestValidator()

	rapid.Check(t, func(t *rapid.T) {
		generateValid := rapid.Bool().Draw(t, "generateValid")

		var address string
		var expectValid bool

		if generateValid {
			// Valid: non-empty, max 1000
			length := rapid.IntRange(1, 1000).Draw(t, "addrLen")
			address = strings.Repeat("x", length)
			expectValid = true
		} else {
			invalidType := rapid.IntRange(0, 1).Draw(t, "invalidType")
			switch invalidType {
			case 0:
				// Empty string
				address = ""
			case 1:
				// Too long (1001+ chars)
				length := rapid.IntRange(1001, 1100).Draw(t, "longLen")
				address = strings.Repeat("x", length)
			}
			expectValid = false
		}

		err := v.Struct(testAddressStruct{Address: address})
		isValid := err == nil

		if isValid != expectValid {
			t.Fatalf("address (len=%d): expected valid=%v, got valid=%v", len(address), expectValid, isValid)
		}
	})
}
