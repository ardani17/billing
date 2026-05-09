package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginRateLimiter mengelola rate limiting untuk login attempts menggunakan Redis.
// Menggunakan email sebagai key agar attacker tidak bisa bypass dengan ganti IP.
// Key format: rate:login:{email}, TTL otomatis expire setelah lockDuration.
type LoginRateLimiter struct {
	redis        *redis.Client
	maxAttempts  int
	lockDuration time.Duration
}

// NewLoginRateLimiter membuat instance baru LoginRateLimiter.
// maxAttempts adalah jumlah maksimal percobaan login gagal sebelum akun dikunci.
// lockDuration adalah durasi penguncian akun setelah melebihi batas percobaan.
func NewLoginRateLimiter(redisClient *redis.Client, maxAttempts int, lockDuration time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		redis:        redisClient,
		maxAttempts:  maxAttempts,
		lockDuration: lockDuration,
	}
}

// rateLimitKey mengembalikan Redis key untuk rate limiting berdasarkan email.
func rateLimitKey(email string) string {
	return fmt.Sprintf("rate:login:%s", email)
}

// Periksa memeriksa apakah email masih boleh melakukan login.
// Mengembalikan (allowed, remainingSeconds, error):
//   - allowed: true jika counter belum mencapai maxAttempts
//   - remainingSeconds: sisa waktu lock dalam detik (0 jika belum terkunci)
func (r *LoginRateLimiter) Check(ctx context.Context, email string) (bool, int, error) {
	key := rateLimitKey(email)

	// Ambil nilai counter dari Redis
	val, err := r.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		// Key tidak ada, berarti belum ada percobaan gagal
		return true, 0, nil
	}
	if err != nil {
		return false, 0, fmt.Errorf("gagal membaca rate limit counter: %w", err)
	}

	// Jika counter sudah mencapai atau melebihi batas, tolak login
	if val >= r.maxAttempts {
		// Ambil sisa TTL untuk informasi ke user
		ttl, err := r.redis.TTL(ctx, key).Result()
		if err != nil {
			return false, 0, fmt.Errorf("gagal membaca TTL rate limit: %w", err)
		}

		remainingSeconds := int(ttl.Seconds())
		if remainingSeconds < 0 {
			remainingSeconds = 0
		}

		return false, remainingSeconds, nil
	}

	return true, 0, nil
}

// Increment menambah counter gagal login untuk email.
// Jika ini adalah percobaan pertama (counter == 1), atur TTL ke lockDuration.
// TTL hanya di-atur sekali agar durasi lock konsisten dari percobaan pertama.
func (r *LoginRateLimiter) Increment(ctx context.Context, email string) error {
	key := rateLimitKey(email)

	// INCR atomik: menambah counter, membuat key jika belum ada
	count, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("gagal increment rate limit counter: %w", err)
	}

	// Set TTL hanya pada percobaan pertama agar durasi lock konsisten
	if count == 1 {
		if err := r.redis.Expire(ctx, key, r.lockDuration).Err(); err != nil {
			return fmt.Errorf("gagal set TTL rate limit: %w", err)
		}
	}

	return nil
}

// Reset menghapus counter gagal login untuk email.
// Dipanggil setelah login berhasil agar user bisa login kembali tanpa batasan.
func (r *LoginRateLimiter) Reset(ctx context.Context, email string) error {
	key := rateLimitKey(email)

	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("gagal reset rate limit counter: %w", err)
	}

	return nil
}
