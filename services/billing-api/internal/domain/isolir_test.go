package domain

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: isolir-system, Property 1: Backoff delay calculation is deterministic and monotonically increasing
// **Validates: Requirements 5.4**
//
// For any retryCount in [0, 4] and any reference time now,
// CalculateNextRetryAt(retryCount, now) SHALL return now + backoffDelays[retryCount],
// and the delay sequence SHALL be monotonically non-decreasing.
func TestProperty_BackoffDelayDeterministicAndMonotonic(t *testing.T) {
	// Jadwal backoff yang diharapkan: 0, 5m, 30m, 2h, 6h
	expectedDelays := []time.Duration{
		0,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
		6 * time.Hour,
	}

	rapid.Check(t, func(t *rapid.T) {
		retryCount := rapid.IntRange(0, 4).Draw(t, "retryCount")
		// Gunakan timestamp acak dalam rentang wajar (2020-2030)
		unixSec := rapid.Int64Range(
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
			time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
		).Draw(t, "unixSec")
		now := time.Unix(unixSec, 0).UTC()

		result := CalculateNextRetryAt(retryCount, now)
		expected := now.Add(expectedDelays[retryCount])

		// Property 1a: Hasil harus sama persis dengan now + backoffDelays[retryCount]
		if !result.Equal(expected) {
			t.Fatalf(
				"CalculateNextRetryAt(%d, %v) = %v, expected %v (delay=%v)",
				retryCount, now, result, expected, expectedDelays[retryCount],
			)
		}

		// Property 1b: Urutan delay harus monotonically non-decreasing
		for i := 1; i < len(expectedDelays); i++ {
			if expectedDelays[i] < expectedDelays[i-1] {
				t.Fatalf(
					"backoff delay sequence is not monotonically non-decreasing: delay[%d]=%v < delay[%d]=%v",
					i, expectedDelays[i], i-1, expectedDelays[i-1],
				)
			}
		}

		// Property 1c: Jika retryCount > 0, delay saat ini >= delay sebelumnya
		if retryCount > 0 {
			prevResult := CalculateNextRetryAt(retryCount-1, now)
			if result.Before(prevResult) {
				t.Fatalf(
					"CalculateNextRetryAt(%d, now) = %v is before CalculateNextRetryAt(%d, now) = %v — not monotonically non-decreasing",
					retryCount, result, retryCount-1, prevResult,
				)
			}
		}
	})
}

// Feature: isolir-system, Property 5: Overdue eligibility detection with timezone awareness
// **Validates: Requirements 2.3, 4.2, 12.1, 12.2**
//
// For any invoice with a due_date, any threshold_days (grace_period_days or suspend_days),
// and any valid tenant timezone, a customer is eligible for status transition if and only if
// daysOverdue(due_date, currentDateInTimezone(timezone)) > threshold_days.
// The calculation SHALL use the tenant's configured timezone to determine the current date.
func TestProperty_OverdueEligibilityWithTimezoneAwareness(t *testing.T) {
	// Timezone yang valid sesuai Requirement 12.3
	validTimezones := []string{"Asia/Jakarta", "Asia/Makassar", "Asia/Jayapura"}

	rapid.Check(t, func(t *rapid.T) {
		// Generate dueDate acak dalam rentang wajar (2020-2030)
		dueSec := rapid.Int64Range(
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
			time.Date(2030, 6, 30, 23, 59, 59, 0, time.UTC).Unix(),
		).Draw(t, "dueSec")
		dueDate := time.Unix(dueSec, 0).UTC()

		// Generate currentDate acak, bisa sebelum atau sesudah dueDate
		// Rentang: dueDate - 60 hari sampai dueDate + 120 hari
		offsetDays := rapid.IntRange(-60, 120).Draw(t, "offsetDays")
		currentDate := dueDate.Add(time.Duration(offsetDays) * 24 * time.Hour)

		// Generate threshold acak (grace_period_days atau suspend_days)
		threshold := rapid.IntRange(0, 90).Draw(t, "threshold")

		// Pilih timezone acak dari daftar valid
		tzIdx := rapid.IntRange(0, len(validTimezones)-1).Draw(t, "tzIdx")
		tz := validTimezones[tzIdx]

		// --- Property 5a: daysOverdue selalu mengembalikan non-negative integer ---
		days := daysOverdue(dueDate, currentDate)
		if days < 0 {
			t.Fatalf("daysOverdue(%v, %v) = %d, expected non-negative", dueDate, currentDate, days)
		}

		// --- Property 5b: daysOverdue konsisten dengan perhitungan manual ---
		// Hitung expected days secara manual
		diff := currentDate.Sub(dueDate)
		expectedDays := int(diff.Hours() / 24)
		if expectedDays < 0 {
			expectedDays = 0
		}
		if days != expectedDays {
			t.Fatalf(
				"daysOverdue(%v, %v) = %d, expected %d (diff=%v)",
				dueDate, currentDate, days, expectedDays, diff,
			)
		}

		// --- Property 5c: Eligibility threshold check ---
		// Customer eligible for transition iff daysOverdue > threshold
		eligible := days > threshold

		// Verifikasi: eligible iff currentDate lebih dari threshold hari setelah dueDate
		thresholdDuration := time.Duration(threshold) * 24 * time.Hour
		pastThreshold := currentDate.Sub(dueDate) > thresholdDuration

		if eligible != pastThreshold {
			t.Fatalf(
				"eligibility mismatch: daysOverdue=%d, threshold=%d, eligible=%v, pastThreshold=%v (dueDate=%v, currentDate=%v, diff=%v)",
				days, threshold, eligible, pastThreshold, dueDate, currentDate, diff,
			)
		}

		// --- Property 5d: Timezone awareness — currentDateInTimezone menggunakan timezone tenant ---
		loc, err := time.LoadLocation(tz)
		if err != nil {
			t.Fatalf("failed to load timezone %s: %v", tz, err)
		}

		// Konversi currentDate ke timezone tenant
		currentInTZ := currentDate.In(loc)

		// daysOverdue harus konsisten terlepas dari representasi timezone
		// karena time.Sub bekerja pada waktu absolut (UTC underneath)
		daysInTZ := daysOverdue(dueDate, currentInTZ)
		if daysInTZ != days {
			t.Fatalf(
				"timezone inconsistency: daysOverdue with UTC=%d, with %s=%d (dueDate=%v, currentDate=%v)",
				days, tz, daysInTZ, dueDate, currentDate,
			)
		}

		// --- Property 5e: currentDateInTimezone fallback ke Asia/Jakarta untuk timezone invalid ---
		fallbackDate := currentDateInTimezone("Invalid/Timezone")
		jakartaLoc, _ := time.LoadLocation("Asia/Jakarta")
		if fallbackDate.Location().String() != jakartaLoc.String() {
			t.Fatalf(
				"currentDateInTimezone with invalid tz should fallback to Asia/Jakarta, got %s",
				fallbackDate.Location().String(),
			)
		}

		// Verifikasi currentDateInTimezone mengembalikan waktu di timezone yang benar
		for _, validTZ := range validTimezones {
			tzDate := currentDateInTimezone(validTZ)
			expectedLoc, _ := time.LoadLocation(validTZ)
			if tzDate.Location().String() != expectedLoc.String() {
				t.Fatalf(
					"currentDateInTimezone(%s) returned location %s, expected %s",
					validTZ, tzDate.Location().String(), expectedLoc.String(),
				)
			}
		}
	})
}
