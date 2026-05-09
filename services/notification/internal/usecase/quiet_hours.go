package usecase

import (
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// QuietHoursChecker - pengecekan jam tenang notifikasi
// =============================================================================

// QuietHoursChecker bertanggung jawab untuk mengecek apakah waktu saat ini
// berada di luar jam aktif (quiet hours), sehingga notifikasi otomatis
// tidak dikirim di malam hari atau di luar jam operasional tenant.
// Struct ini stateless dan tidak memiliki dependency eksternal.
type QuietHoursChecker struct{}

// NewQuietHoursChecker membuat instance baru QuietHoursChecker.
func NewQuietHoursChecker() *QuietHoursChecker {
	return &QuietHoursChecker{}
}

// IsQuietHours mengecek apakah waktu saat ini berada di luar jam aktif.
// Jam aktif didefinisikan sebagai rentang [start, end) dalam timezone tenant.
// Mengembalikan true jika sekarang BUKAN jam aktif (quiet hours / jangan kirim),
// mengembalikan false jika sekarang DALAM jam aktif (boleh kirim).
//
// Parameter:
//   - now: waktu saat ini (UTC)
//   - tz: nama timezone tenant (contoh: "Asia/Jakarta")
//   - start: jam mulai aktif dalam format HH:MM (contoh: "07:00")
//   - end: jam selesai aktif dalam format HH:MM (contoh: "21:00")
func (q *QuietHoursChecker) IsQuietHours(now time.Time, tz, start, end string) bool {
	// Parsing timezone tenant
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Jika timezone tidak valid, anggap bukan quiet hours (kirim saja)
		return false
	}

	// Konversi waktu ke timezone lokal tenant
	localNow := now.In(loc)

	// Parsing jam mulai dan selesai aktif
	startTime, err := parseHHMM(start, localNow, loc)
	if err != nil {
		return false
	}
	endTime, err := parseHHMM(end, localNow, loc)
	if err != nil {
		return false
	}

	// Waktu lokal sebelum jam mulai aktif -> quiet hours
	// Waktu lokal pada atau setelah jam selesai aktif -> quiet hours
	// Waktu lokal dalam rentang [start, end) -> bukan quiet hours
	return localNow.Before(startTime) || !localNow.Before(endTime)
}

// IsBypassEvent mengecek apakah event_type termasuk dalam daftar bypass
// yang dikecualikan dari pembatasan quiet hours.
// Event bypass dikirim langsung tanpa menunggu jam aktif.
func (q *QuietHoursChecker) IsBypassEvent(eventType string) bool {
	return domain.IsBypassEvent(eventType)
}

// CalculateScheduledAt menghitung waktu pengiriman terjadwal berikutnya,
// yaitu awal jam aktif berikutnya di timezone tenant.
// Jika jam mulai aktif hari ini sudah lewat, dijadwalkan untuk besok.
//
// Parameter:
//   - tz: nama timezone tenant (contoh: "Asia/Jakarta")
//   - start: jam mulai aktif dalam format HH:MM (contoh: "07:00")
func (q *QuietHoursChecker) CalculateScheduledAt(tz, start string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Cadangan ke Asia/Jakarta jika timezone tidak valid
		loc, _ = time.LoadLocation("Asia/Jakarta")
	}

	now := time.Now().In(loc)

	scheduled, err := parseHHMM(start, now, loc)
	if err != nil {
		// Cadangan ke jam 07:00 jika format tidak valid
		scheduled = time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, loc)
	}

	// Jika jam mulai aktif hari ini sudah lewat, jadwalkan untuk besok
	if !now.Before(scheduled) {
		scheduled = scheduled.AddDate(0, 0, 1)
	}

	return scheduled
}

// parseHHMM mem-parsing string format "HH:MM" menjadi time.Time pada tanggal
// yang sama dengan referensi, menggunakan lokasi yang diberikan.
func parseHHMM(hhmm string, ref time.Time, loc *time.Location) (time.Time, error) {
	var hour, minute int
	_, err := fmt.Sscanf(hhmm, "%d:%d", &hour, &minute)
	if err != nil {
		return time.Time{}, fmt.Errorf("format waktu tidak valid: %s", hhmm)
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day(), hour, minute, 0, 0, loc), nil
}
