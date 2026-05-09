package middleware

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 5.1, 5.2, 5.3**
//
// previous failures.

func TestProperty_RateLimiterEnforcesLockout(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Siapkan miniredis sebagai Redis in-memory
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		const maxAttempts = 5
		lockDuration := 15 * time.Minute

		limiter := NewLoginRateLimiter(redisClient, maxAttempts, lockDuration)
		ctx := context.Background()

		// Buat a random email
		email := rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,8}\.[a-z]{2,4}`).Draw(t, "email")

		// Buat a random number of failed attempts (0 to 10)
		n := rapid.IntRange(0, 10).Draw(t, "failedAttempts")

		// Perform N failed login attempts: Increment then Periksa
		for i := 0; i < n; i++ {
			if err := limiter.Increment(ctx, email); err != nil {
				t.Fatalf("Increment failed at attempt %d: %v", i+1, err)
			}

			allowed, remainingSeconds, err := limiter.Check(ctx, email)
			if err != nil {
				t.Fatalf("Check failed at attempt %d: %v", i+1, err)
			}

			attemptCount := i + 1 // 1-indexed count of failures so far

			if attemptCount < maxAttempts {
				if !allowed {
					t.Errorf("after %d failures (< %d), Check returned allowed=false, want true",
						attemptCount, maxAttempts)
				}
				if remainingSeconds != 0 {
					t.Errorf("after %d failures (< %d), remainingSeconds=%d, want 0",
						attemptCount, maxAttempts, remainingSeconds)
				}
			} else {
				if allowed {
					t.Errorf("after %d failures (>= %d), Check returned allowed=true, want false",
						attemptCount, maxAttempts)
				}
				if remainingSeconds <= 0 {
					t.Errorf("after %d failures (>= %d), remainingSeconds=%d, want > 0",
						attemptCount, maxAttempts, remainingSeconds)
				}
			}
		}

		if err := limiter.Reset(ctx, email); err != nil {
			t.Fatalf("Reset failed: %v", err)
		}

		allowed, remainingSeconds, err := limiter.Check(ctx, email)
		if err != nil {
			t.Fatalf("Check after Reset failed: %v", err)
		}
		if !allowed {
			t.Errorf("after Reset, Check returned allowed=false, want true")
		}
		if remainingSeconds != 0 {
			t.Errorf("after Reset, remainingSeconds=%d, want 0", remainingSeconds)
		}

		key := fmt.Sprintf("rate:login:%s", email)
		exists, err := redisClient.Exists(ctx, key).Result()
		if err != nil {
			t.Fatalf("Redis Exists check failed: %v", err)
		}
		if exists != 0 {
			t.Errorf("after Reset, Redis key %q still exists, want deleted", key)
		}
	})
}
