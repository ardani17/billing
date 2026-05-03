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

// Feature: auth-rbac, Property 9: Rate Limiter Enforces Lockout Correctly
// **Validates: Requirements 5.1, 5.2, 5.3**
//
// For any email address, after N consecutive failed login attempts (where N < 5),
// the rate limiter SHALL allow the next attempt. After exactly 5 consecutive failed
// attempts, the rate limiter SHALL block further attempts for 15 minutes. After a
// successful login, the rate limiter SHALL reset the counter to zero regardless of
// previous failures.

func TestProperty_RateLimiterEnforcesLockout(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Set up miniredis as in-memory Redis
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

		// Generate a random email
		email := rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,8}\.[a-z]{2,4}`).Draw(t, "email")

		// Generate a random number of failed attempts (0 to 10)
		n := rapid.IntRange(0, 10).Draw(t, "failedAttempts")

		// Perform N failed login attempts: Increment then Check
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
				// Property: after N < 5 failures, next attempt should be allowed
				if !allowed {
					t.Errorf("after %d failures (< %d), Check returned allowed=false, want true",
						attemptCount, maxAttempts)
				}
				if remainingSeconds != 0 {
					t.Errorf("after %d failures (< %d), remainingSeconds=%d, want 0",
						attemptCount, maxAttempts, remainingSeconds)
				}
			} else {
				// Property: after >= 5 failures, should be blocked
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

		// Property: after Reset, Check should return allowed=true again
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

		// Verify the key is actually gone from Redis (counter reset to zero)
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
