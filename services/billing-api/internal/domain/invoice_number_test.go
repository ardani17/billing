package domain

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: invoice-generation, Property 2: Invoice Number Format Round-Trip
// **Validates: Requirements 7.4**
//
// Untuk setiap prefix valid (non-empty alphanumeric), year (2000-2099),
// month (1-12), dan sequence (positive integer), FormatInvoiceNumber(prefix, year, month, seq)
// menghasilkan string dengan format {prefix}-{YYYY}-{MM}-{SEQ} dimana SEQ zero-padded
// minimal 3 digit, dan parsing komponen kembali menghasilkan nilai asli.
func TestProperty_InvoiceNumberFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generator: prefix non-empty alphanumeric (1-10 karakter)
		prefix := rapid.StringMatching(`[A-Za-z0-9]{1,10}`).Draw(t, "prefix")
		year := rapid.IntRange(2000, 2099).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		seq := rapid.IntRange(1, 99999).Draw(t, "seq")

		// Format nomor invoice
		result := FormatInvoiceNumber(prefix, year, month, seq)

		// Property 2a: Hasil harus sesuai format {prefix}-{YYYY}-{MM}-{SEQ}
		parts := strings.SplitN(result, "-", 4)
		if len(parts) != 4 {
			t.Fatalf(
				"FormatInvoiceNumber(%q, %d, %d, %d) = %q, expected 4 parts separated by '-', got %d parts",
				prefix, year, month, seq, result, len(parts),
			)
		}

		// Property 2b: Parsing prefix kembali menghasilkan nilai asli
		parsedPrefix := parts[0]
		if parsedPrefix != prefix {
			t.Fatalf(
				"Parsed prefix %q != original prefix %q from result %q",
				parsedPrefix, prefix, result,
			)
		}

		// Property 2c: Parsing year kembali menghasilkan nilai asli
		parsedYear, err := strconv.Atoi(parts[1])
		if err != nil {
			t.Fatalf("Failed to parse year from %q: %v", result, err)
		}
		if parsedYear != year {
			t.Fatalf(
				"Parsed year %d != original year %d from result %q",
				parsedYear, year, result,
			)
		}

		// Property 2d: Parsing month kembali menghasilkan nilai asli
		parsedMonth, err := strconv.Atoi(parts[2])
		if err != nil {
			t.Fatalf("Failed to parse month from %q: %v", result, err)
		}
		if parsedMonth != month {
			t.Fatalf(
				"Parsed month %d != original month %d from result %q",
				parsedMonth, month, result,
			)
		}

		// Property 2e: Parsing sequence kembali menghasilkan nilai asli
		parsedSeq, err := strconv.Atoi(parts[3])
		if err != nil {
			t.Fatalf("Failed to parse seq from %q: %v", result, err)
		}
		if parsedSeq != seq {
			t.Fatalf(
				"Parsed seq %d != original seq %d from result %q",
				parsedSeq, seq, result,
			)
		}

		// Property 2f: SEQ harus zero-padded minimal 3 digit
		seqStr := parts[3]
		if len(seqStr) < 3 {
			t.Fatalf(
				"SEQ part %q has length %d, expected at least 3 digits in result %q",
				seqStr, len(seqStr), result,
			)
		}

		// Property 2g: Year harus 4 digit
		if len(parts[1]) != 4 {
			t.Fatalf(
				"Year part %q has length %d, expected 4 digits in result %q",
				parts[1], len(parts[1]), result,
			)
		}

		// Property 2h: Month harus 2 digit
		if len(parts[2]) != 2 {
			t.Fatalf(
				"Month part %q has length %d, expected 2 digits in result %q",
				parts[2], len(parts[2]), result,
			)
		}

		// Property 2i: Keseluruhan format harus cocok dengan rekonstruksi manual
		expectedSeqStr := fmt.Sprintf("%03d", seq)
		if seq >= 1000 {
			expectedSeqStr = fmt.Sprintf("%d", seq)
		}
		expectedResult := fmt.Sprintf("%s-%04d-%02d-%s", prefix, year, month, expectedSeqStr)
		if result != expectedResult {
			t.Fatalf(
				"FormatInvoiceNumber(%q, %d, %d, %d) = %q, expected %q",
				prefix, year, month, seq, result, expectedResult,
			)
		}
	})
}
