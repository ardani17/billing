package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 7.4, 14.1, 14.2, 14.3**
//
// SEQ is zero-padded to minimum 4 digits.
func TestProperty_ReceiptNumberFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(2000, 2099).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		seq := rapid.IntRange(1, 99999).Draw(t, "seq")

		formatted := FormatReceiptNumber(year, month, seq)

		parsedYear, parsedMonth, parsedSeq, err := ParseReceiptNumber(formatted)
		if err != nil {
			t.Fatalf(
				"ParseReceiptNumber(%q) returned error: %v (year=%d, month=%d, seq=%d)",
				formatted, err, year, month, seq,
			)
		}

		if parsedYear != year {
			t.Fatalf(
				"Round-trip year mismatch: original=%d, parsed=%d from %q",
				year, parsedYear, formatted,
			)
		}

		if parsedMonth != month {
			t.Fatalf(
				"Round-trip month mismatch: original=%d, parsed=%d from %q",
				month, parsedMonth, formatted,
			)
		}

		if parsedSeq != seq {
			t.Fatalf(
				"Round-trip seq mismatch: original=%d, parsed=%d from %q",
				seq, parsedSeq, formatted,
			)
		}

		if len(formatted) < 4 || formatted[:4] != "PAY-" {
			t.Fatalf(
				"Formatted receipt number %q does not start with 'PAY-'",
				formatted,
			)
		}

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
