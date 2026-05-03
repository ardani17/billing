package domain

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// validIndonesianMonths berisi 12 nama bulan dalam bahasa Indonesia.
var validIndonesianMonths = []string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// Feature: notification-service, Property 3: FormatMoney round-trip
// **Validates: Requirements 5.5**
//
// Untuk setiap int64 amount >= 0, FormatMoney(amount) menghasilkan string yang
// dimulai dengan "Rp " diikuti digit dengan pemisah ribuan (titik).
// Parsing bagian numerik (hapus "Rp " dan titik) menghasilkan amount asli.
func TestProperty_FormatMoney(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate amount non-negatif
		amount := rapid.Int64Range(0, 999_999_999_999).Draw(t, "amount")

		result := FormatMoney(amount)

		// Verifikasi: harus dimulai dengan "Rp "
		if !strings.HasPrefix(result, "Rp ") {
			t.Fatalf("FormatMoney(%d) = %q, tidak dimulai dengan 'Rp '", amount, result)
		}

		// Ambil bagian numerik setelah "Rp "
		numericPart := strings.TrimPrefix(result, "Rp ")

		// Verifikasi: bagian numerik hanya mengandung digit dan titik
		for _, ch := range numericPart {
			if ch != '.' && (ch < '0' || ch > '9') {
				t.Fatalf(
					"FormatMoney(%d) = %q, bagian numerik %q mengandung karakter tidak valid: %c",
					amount, result, numericPart, ch,
				)
			}
		}

		// Hapus titik pemisah ribuan dan parse kembali
		cleaned := strings.ReplaceAll(numericPart, ".", "")
		parsed, err := strconv.ParseInt(cleaned, 10, 64)
		if err != nil {
			t.Fatalf(
				"FormatMoney(%d) = %q, gagal parse bagian numerik %q: %v",
				amount, result, cleaned, err,
			)
		}

		// Verifikasi round-trip: nilai yang di-parse harus sama dengan amount asli
		if parsed != amount {
			t.Fatalf(
				"FormatMoney round-trip gagal: amount=%d, formatted=%q, parsed=%d",
				amount, result, parsed,
			)
		}
	})
}

// Feature: notification-service, Property 4: FormatDateID contains valid Indonesian month
// **Validates: Requirements 5.6**
//
// Untuk setiap time.Time yang valid, FormatDateID(t) menghasilkan string yang
// mengandung nomor hari dan salah satu dari 12 nama bulan Indonesia diikuti tahun.
func TestProperty_FormatDateID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate timestamp acak dalam rentang wajar (tahun 1970-2100)
		unixSec := rapid.Int64Range(0, 4102444800).Draw(t, "unixSec")
		ts := time.Unix(unixSec, 0).UTC()

		result := FormatDateID(ts)

		// Verifikasi format: "D NamaBulan YYYY"
		expectedDay := ts.Day()
		expectedYear := ts.Year()
		expectedMonth := validIndonesianMonths[ts.Month()-1]

		expectedStr := fmt.Sprintf("%d %s %d", expectedDay, expectedMonth, expectedYear)
		if result != expectedStr {
			t.Fatalf(
				"FormatDateID(%v) = %q, expected %q",
				ts, result, expectedStr,
			)
		}

		// Verifikasi: output mengandung salah satu nama bulan Indonesia
		foundMonth := false
		for _, month := range validIndonesianMonths {
			if strings.Contains(result, month) {
				foundMonth = true
				break
			}
		}
		if !foundMonth {
			t.Fatalf(
				"FormatDateID(%v) = %q, tidak mengandung nama bulan Indonesia yang valid",
				ts, result,
			)
		}
	})
}
