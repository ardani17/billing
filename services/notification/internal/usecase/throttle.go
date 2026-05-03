package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// ThrottleChecker — pengecekan anti-spam throttle notifikasi
// =============================================================================

// ThrottleChecker bertanggung jawab untuk membatasi jumlah pesan yang dikirim
// ke pelanggan dalam satu hari (daily limit) dan memastikan jeda minimum
// antar pesan (cooldown), mencegah spam ke pelanggan.
type ThrottleChecker struct {
	logRepo domain.LogRepository
}

// NewThrottleChecker membuat instance baru ThrottleChecker dengan dependency LogRepository.
func NewThrottleChecker(logRepo domain.LogRepository) *ThrottleChecker {
	return &ThrottleChecker{logRepo: logRepo}
}

// CheckDailyLimit mengecek apakah pelanggan sudah melebihi batas harian notifikasi.
// Menghitung jumlah notifikasi yang sudah dikirim hari ini (berdasarkan timezone tenant)
// dan membandingkan dengan limit yang dikonfigurasi.
//
// Mengembalikan:
//   - true jika limit tercapai (harus skip pengiriman)
//   - false jika masih di bawah limit (boleh kirim)
func (t *ThrottleChecker) CheckDailyLimit(ctx context.Context, tenantID, customerID, tz string, limit int) (bool, error) {
	count, err := t.logRepo.CountTodayByCustomer(ctx, tenantID, customerID, tz)
	if err != nil {
		return false, fmt.Errorf("gagal menghitung notifikasi harian: %w", err)
	}

	// Jika jumlah notifikasi hari ini >= limit, harus skip
	return count >= limit, nil
}

// CheckCooldown mengecek apakah pelanggan masih dalam periode cooldown
// (jeda minimum antar pesan). Memeriksa waktu pengiriman terakhir dan
// membandingkan dengan durasi cooldown yang dikonfigurasi.
//
// Mengembalikan:
//   - shouldDelay: true jika harus menunda pengiriman
//   - delayUntil: waktu kapan boleh kirim lagi (nil jika tidak perlu delay)
//   - error: jika terjadi kesalahan saat query database
func (t *ThrottleChecker) CheckCooldown(ctx context.Context, tenantID, customerID string, cooldownMinutes int) (bool, *time.Time, error) {
	lastSent, err := t.logRepo.LastSentToCustomer(ctx, tenantID, customerID)
	if err != nil {
		return false, nil, fmt.Errorf("gagal mengambil waktu pengiriman terakhir: %w", err)
	}

	// Jika belum pernah kirim, tidak perlu cooldown
	if lastSent == nil {
		return false, nil, nil
	}

	// Hitung waktu cooldown berakhir
	cooldownDuration := time.Duration(cooldownMinutes) * time.Minute
	delayUntil := lastSent.Add(cooldownDuration)

	// Jika waktu sekarang masih dalam periode cooldown, harus delay
	if time.Since(*lastSent) < cooldownDuration {
		return true, &delayUntil, nil
	}

	// Cooldown sudah lewat, boleh kirim
	return false, nil, nil
}

// IsBypassEvent mengecek apakah event_type termasuk dalam daftar bypass
// yang dikecualikan dari pembatasan throttle.
// Event bypass dikirim langsung tanpa memeriksa batas harian atau cooldown.
func (t *ThrottleChecker) IsBypassEvent(eventType string) bool {
	return domain.IsBypassEvent(eventType)
}
