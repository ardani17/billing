package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property 5: Receipt Number Format Round-Trip
// =============================================================================

// Feature: payment-manual, Property 5: Receipt Number Format Round-Trip
// **Validates: Requirements 7.4, 14.1, 14.2, 14.3**
//
// For any valid year (2000-2099), month (1-12), and sequence (1-99999),
// FormatReceiptNumber(year, month, seq) produces a string that when parsed
// with ParseReceiptNumber yields the original year, month, and sequence.
// SEQ is zero-padded to minimum 4 digits.
func TestProperty_ReceiptNumberFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(2000, 2099).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		seq := rapid.IntRange(1, 99999).Draw(t, "seq")

		// Format the receipt number
		formatted := FormatReceiptNumber(year, month, seq)

		// Parse it back
		parsedYear, parsedMonth, parsedSeq, err := ParseReceiptNumber(formatted)
		if err != nil {
			t.Fatalf(
				"ParseReceiptNumber(%q) returned error: %v (year=%d, month=%d, seq=%d)",
				formatted, err, year, month, seq,
			)
		}

		// Property 5a: parsed year matches original
		if parsedYear != year {
			t.Fatalf(
				"Round-trip year mismatch: original=%d, parsed=%d from %q",
				year, parsedYear, formatted,
			)
		}

		// Property 5b: parsed month matches original
		if parsedMonth != month {
			t.Fatalf(
				"Round-trip month mismatch: original=%d, parsed=%d from %q",
				month, parsedMonth, formatted,
			)
		}

		// Property 5c: parsed sequence matches original
		if parsedSeq != seq {
			t.Fatalf(
				"Round-trip seq mismatch: original=%d, parsed=%d from %q",
				seq, parsedSeq, formatted,
			)
		}

		// Property 5d: formatted string starts with "PAY-"
		if len(formatted) < 4 || formatted[:4] != "PAY-" {
			t.Fatalf(
				"Formatted receipt number %q does not start with 'PAY-'",
				formatted,
			)
		}

		// Property 5e: SEQ part is zero-padded to minimum 4 digits
		// Extract the SEQ part (after the third '-')
		dashCount := 0
		seqStart := 0
		for i, c := range formatted {
			if c == '-' {
				dashCount++
				if dashCount == 3 {
					seqStart = i + 1
					break
				}
			}
		}
		seqPart := formatted[seqStart:]
		if len(seqPart) < 4 {
			t.Fatalf(
				"SEQ part %q has length %d, expected at least 4 digits in %q",
				seqPart, len(seqPart), formatted,
			)
		}
	})
}
