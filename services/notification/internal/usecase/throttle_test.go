package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Mock LogRepository — implementasi sederhana untuk testing throttle
// =============================================================================

// mockLogRepo adalah mock sederhana yang mengimplementasikan domain.LogRepository
// dengan nilai yang bisa dikonfigurasi untuk CountTodayByCustomer dan LastSentToCustomer.
type mockLogRepo struct {
	countToday int
	lastSent   *time.Time
}

func (m *mockLogRepo) Create(_ context.Context, _ *domain.NotificationLog) (*domain.NotificationLog, error) {
	return nil, nil
}

func (m *mockLogRepo) GetByID(_ context.Context, _ string) (*domain.NotificationLog, error) {
	return nil, nil
}

func (m *mockLogRepo) Update(_ context.Context, _ *domain.NotificationLog) error {
	return nil
}

func (m *mockLogRepo) List(_ context.Context, _ domain.LogListParams) (*domain.LogListResult, error) {
	return nil, nil
}

func (m *mockLogRepo) FindByDedupKey(_ context.Context, _ string, _ int) (*domain.NotificationLog, error) {
	return nil, nil
}

func (m *mockLogRepo) CountTodayByCustomer(_ context.Context, _, _, _ string) (int, error) {
	return m.countToday, nil
}

func (m *mockLogRepo) LastSentToCustomer(_ context.Context, _, _ string) (*time.Time, error) {
	return m.lastSent, nil
}

// Feature: notification-service, Property 9: Throttle daily limit enforcement
// **Validates: Requirements 11.2**
//
// Untuk setiap count notifikasi yang sudah dikirim hari ini dan limit harian:
// - Jika count >= limit, CheckDailyLimit HARUS mengembalikan true (skip pengiriman).
// - Jika count < limit, CheckDailyLimit HARUS mengembalikan false (boleh kirim).
func TestProperty_ThrottleDailyLimitEnforcement(t *testing.T) {
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Generate count dan limit acak
		count := rapid.IntRange(0, 100).Draw(t, "count")
		limit := rapid.IntRange(1, 20).Draw(t, "limit")

		// Buat mock repo dengan count yang dikonfigurasi
		repo := &mockLogRepo{countToday: count}
		checker := NewThrottleChecker(repo)

		shouldSkip, err := checker.CheckDailyLimit(ctx, "tenant-1", "customer-1", "Asia/Jakarta", limit)
		if err != nil {
			t.Fatalf("CheckDailyLimit error: %v", err)
		}

		if count >= limit && !shouldSkip {
			t.Fatalf(
				"count=%d >= limit=%d, seharusnya skip (true), tapi return false",
				count, limit,
			)
		}
		if count < limit && shouldSkip {
			t.Fatalf(
				"count=%d < limit=%d, seharusnya proceed (false), tapi return true",
				count, limit,
			)
		}
	})
}

// Feature: notification-service, Property 10: Throttle cooldown delay
// **Validates: Requirements 11.4**
//
// Untuk setiap waktu pengiriman terakhir (lastSent) dan cooldown_minutes:
// - Jika now - lastSent < cooldown, CheckCooldown HARUS mengembalikan shouldDelay=true
//   dengan delayUntil = lastSent + cooldown.
// - Jika now - lastSent >= cooldown, CheckCooldown HARUS mengembalikan shouldDelay=false.
// - Jika belum pernah kirim (lastSent=nil), HARUS mengembalikan shouldDelay=false.
func TestProperty_ThrottleCooldownDelay(t *testing.T) {
	ctx := context.Background()

	t.Run("dengan lastSent", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate cooldown antara 5-120 menit (sesuai validasi settings)
			cooldownMinutes := rapid.IntRange(5, 120).Draw(t, "cooldownMinutes")

			// Generate berapa menit yang sudah berlalu sejak pengiriman terakhir
			elapsedMinutes := rapid.IntRange(0, 240).Draw(t, "elapsedMinutes")

			// Hitung lastSent berdasarkan elapsed
			now := time.Now()
			lastSent := now.Add(-time.Duration(elapsedMinutes) * time.Minute)

			repo := &mockLogRepo{lastSent: &lastSent}
			checker := NewThrottleChecker(repo)

			shouldDelay, delayUntil, err := checker.CheckCooldown(ctx, "tenant-1", "customer-1", cooldownMinutes)
			if err != nil {
				t.Fatalf("CheckCooldown error: %v", err)
			}

			if elapsedMinutes < cooldownMinutes {
				// Masih dalam cooldown, harus delay
				if !shouldDelay {
					t.Fatalf(
						"elapsed=%d menit < cooldown=%d menit, seharusnya delay (true), tapi return false",
						elapsedMinutes, cooldownMinutes,
					)
				}
				if delayUntil == nil {
					t.Fatalf("shouldDelay=true tapi delayUntil=nil")
				}
				// delayUntil harus = lastSent + cooldown
				expectedDelay := lastSent.Add(time.Duration(cooldownMinutes) * time.Minute)
				if delayUntil.Sub(expectedDelay).Abs() > time.Second {
					t.Fatalf(
						"delayUntil=%v, expected=%v (selisih > 1 detik)",
						delayUntil, expectedDelay,
					)
				}
			} else {
				// Cooldown sudah lewat, boleh kirim
				if shouldDelay {
					t.Fatalf(
						"elapsed=%d menit >= cooldown=%d menit, seharusnya proceed (false), tapi return true",
						elapsedMinutes, cooldownMinutes,
					)
				}
			}
		})
	})

	t.Run("tanpa lastSent (belum pernah kirim)", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cooldownMinutes := rapid.IntRange(5, 120).Draw(t, "cooldownMinutes")

			repo := &mockLogRepo{lastSent: nil}
			checker := NewThrottleChecker(repo)

			shouldDelay, delayUntil, err := checker.CheckCooldown(ctx, "tenant-1", "customer-1", cooldownMinutes)
			if err != nil {
				t.Fatalf("CheckCooldown error: %v", err)
			}

			if shouldDelay {
				t.Fatalf("Belum pernah kirim, seharusnya tidak delay, tapi shouldDelay=true")
			}
			if delayUntil != nil {
				t.Fatalf("Belum pernah kirim, seharusnya delayUntil=nil, tapi %v", delayUntil)
			}
		})
	})
}

// Feature: notification-service, Property 11: Throttle bypass for exempt events
// **Validates: Requirements 11.5**
//
// Untuk setiap event_type dalam daftar bypass (payment.online.received,
// payment.recorded, notification.un_isolir, notification.reactivated),
// IsBypassEvent HARUS mengembalikan true. Untuk event_type yang TIDAK ada
// dalam daftar bypass, IsBypassEvent HARUS mengembalikan false.
func TestProperty_ThrottleBypassForExemptEvents(t *testing.T) {
	checker := NewThrottleChecker(&mockLogRepo{})

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
			// Generate event_type acak yang BUKAN bypass event
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
