package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 10.1**
//
// Untuk setiap waktu t di timezone tenant dan konfigurasi quiet hours [start, end):
//   - Jika t berada DI LUAR rentang [start, end) (sebelum start atau >= end),
//     IsQuietHours HARUS mengembalikan true (quiet / jangan kirim).
//   - Jika t berada DI DALAM rentang [start, end),
//     IsQuietHours HARUS mengembalikan false (aktif / boleh kirim).
func TestProperty_QuietHoursBlocking(t *testing.T) {
	checker := NewQuietHoursChecker()
	tz := "Asia/Jakarta"
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("gagal load timezone %s: %v", tz, err)
	}

	rapid.Check(t, func(t *rapid.T) {
		// Buat jam start dan end di mana start < end
		startHour := rapid.IntRange(0, 20).Draw(t, "startHour")
		endHour := rapid.IntRange(startHour+1, 23).Draw(t, "endHour")

		startStr := fmt.Sprintf("%02d:00", startHour)
		endStr := fmt.Sprintf("%02d:00", endHour)

		// Buat jam acak untuk pengujian (0-23)
		testHour := rapid.IntRange(0, 23).Draw(t, "testHour")
		testMinute := rapid.IntRange(0, 59).Draw(t, "testMinute")

		// Buat waktu di timezone Jakarta
		now := time.Date(2025, 6, 15, testHour, testMinute, 0, 0, loc)

		result := checker.IsQuietHours(now, tz, startStr, endStr)

		// Tentukan apakah waktu berada di dalam rentang [start, end)
		insideActiveHours := testHour >= startHour && testHour < endHour

		if insideActiveHours && result {
			t.Fatalf(
				"Waktu %02d:%02d berada di dalam jam aktif [%s, %s) tapi IsQuietHours=true (seharusnya false)",
				testHour, testMinute, startStr, endStr,
			)
		}
		if !insideActiveHours && !result {
			t.Fatalf(
				"Waktu %02d:%02d berada di luar jam aktif [%s, %s) tapi IsQuietHours=false (seharusnya true)",
				testHour, testMinute, startStr, endStr,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 10.4**
//
// Untuk setiap waktu (termasuk di luar quiet hours) dan setiap event_type
// dalam daftar bypass (payment.online.received, payment.recorded,
// notification.un_isolir, notification.reactivated), IsBypassEvent HARUS
// mengembalikan true. Untuk event_type yang TIDAK ada dalam daftar bypass,
// IsBypassEvent HARUS mengembalikan false.
func TestProperty_QuietHoursBypassForExemptEvents(t *testing.T) {
	checker := NewQuietHoursChecker()

	t.Run("bypass events selalu return true", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Pilih salah satu event dari daftar bypass
			idx := rapid.IntRange(0, len(domain.BypassEventTypes)-1).Draw(t, "bypassIdx")
			eventType := domain.BypassEventTypes[idx]

			result := checker.IsBypassEvent(eventType)
			if !result {
				t.Fatalf(
					"Event bypass %q seharusnya return true, tapi return false",
					eventType,
				)
			}
		})
	})

	t.Run("non-bypass events selalu return false", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Buat event_type acak yang BUKAN bypass event
			eventType := rapid.StringMatching(`[a-z][a-z0-9_\.]{3,30}`).Draw(t, "eventType")

			// Pastikan bukan bypass event
			for _, bypass := range domain.BypassEventTypes {
				if eventType == bypass {
					t.Skip("generated event matches bypass list, skipping")
					return
				}
			}

			result := checker.IsBypassEvent(eventType)
			if result {
				t.Fatalf(
					"Event non-bypass %q seharusnya return false, tapi return true",
					eventType,
				)
			}
		})
	})
}
