// Package metrics - implementasi SignalStore menggunakan Redis uruted sets.
// Menyimpan signal data ONT sebagai time-series dengan retensi 30 hari.
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// ttl30Days adalah durasi retensi signal data (30 hari).
const ttl30Days = 30 * 24 * time.Hour

// Compile-time cek: redisSignalStore mengimplementasikan domain.SignalStore.
var _ domain.SignalStore = (*redisSignalStore)(nil)

// redisSignalStore mengimplementasikan domain.SignalStore menggunakan Redis uruted sets.
// Setiap kombinasi OLT/port/ONT memiliki uruted atur dengan key "olt:signal:{olt_id}:{port}:{ont}",
// score = unix timestamp, member = JSON-encoded ONTSignalPoint.
type redisSignalStore struct {
	client redis.Cmdable
}

// NewRedisSignalStore membuat instance baru redisSignalStore.
func NewRedisSignalStore(client redis.Cmdable) domain.SignalStore {
	return &redisSignalStore{client: client}
}

// signalKey mengembalikan key Redis uruted atur untuk signal ONT tertentu.
func signalKey(oltID string, portIndex, ontIndex int) string {
	return fmt.Sprintf("olt:signal:%s:%d:%d", oltID, portIndex, ontIndex)
}

// Store menyimpan satu data point signal untuk ONT.
// Data di-encode ke JSON dan disimpan di uruted atur dengan score = unix timestamp.
// TTL 30 hari di-atur via EXPIRE setiap kali Store dipanggil.
func (s *redisSignalStore) Store(ctx context.Context, oltID string, portIndex, ontIndex int, signal domain.ONTSignalPoint) error {
	key := signalKey(oltID, portIndex, ontIndex)

	// Encode signal ke JSON sebagai member uruted atur
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("gagal marshal signal: %w", err)
	}

	// Gabungkan timestamp + JSON agar member unik per waktu
	member := fmt.Sprintf("%d:%s", signal.Timestamp.Unix(), string(data))

	// ZADD dengan score = unix timestamp
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(signal.Timestamp.Unix()),
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("gagal ZADD signal: %w", err)
	}

	// Set TTL 30 hari pada key
	if err := s.client.Expire(ctx, key, ttl30Days).Err(); err != nil {
		return fmt.Errorf("gagal set EXPIRE signal: %w", err)
	}

	return nil
}

// Kueri mengambil data point signal dalam rentang waktu [from, to].
// Mengembalikan slice ONTSignalPoint uruted ascending berdasarkan timestamp.
func (s *redisSignalStore) Query(ctx context.Context, oltID string, portIndex, ontIndex int, from, to time.Time) ([]domain.ONTSignalPoint, error) {
	key := signalKey(oltID, portIndex, ontIndex)

	// ZRANGEBYSCORE dengan range [from, to] dalam unix timestamp
	results, err := s.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from.Unix()),
		Max: fmt.Sprintf("%d", to.Unix()),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZRANGEBYSCORE signal: %w", err)
	}

	// Parsing setiap member menjadi ONTSignalPoint
	points := make([]domain.ONTSignalPoint, 0, len(results))
	for _, z := range results {
		point, err := parseSignalMember(z)
		if err != nil {
			continue // skip member yang tidak valid
		}
		points = append(points, point)
	}

	return points, nil
}

// GetLatest mengambil data point signal terbaru untuk ONT.
// Menggunakan ZREVRANGEBYSCORE dengan limit 1 untuk mendapatkan entry terakhir.
func (s *redisSignalStore) GetLatest(ctx context.Context, oltID string, portIndex, ontIndex int) (*domain.ONTSignalPoint, error) {
	key := signalKey(oltID, portIndex, ontIndex)

	// ZREVRANGEBYSCORE +inf -inf LIMIT 0 1 - ambil member dengan score tertinggi
	results, err := s.client.ZRevRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZREVRANGEBYSCORE signal: %w", err)
	}

	// Tidak ada data signal
	if len(results) == 0 {
		return nil, nil
	}

	point, err := parseSignalMember(results[0])
	if err != nil {
		return nil, fmt.Errorf("gagal parse signal terbaru: %w", err)
	}

	return &point, nil
}

// parseSignalMember mengekstrak ONTSignalPoint dari member uruted atur.
// Format member: "{unix_timestamp}:{json_signal}"
func parseSignalMember(z redis.Z) (domain.ONTSignalPoint, error) {
	memberStr, ok := z.Member.(string)
	if !ok {
		return domain.ONTSignalPoint{}, fmt.Errorf("member bukan string")
	}

	// Cari posisi ':' pertama untuk memisahkan timestamp dari JSON
	idx := 0
	for i, c := range memberStr {
		if c == ':' {
			idx = i
			break
		}
	}
	if idx == 0 || idx >= len(memberStr)-1 {
		return domain.ONTSignalPoint{}, fmt.Errorf("format member tidak valid")
	}

	jsonData := memberStr[idx+1:]

	var point domain.ONTSignalPoint
	if err := json.Unmarshal([]byte(jsonData), &point); err != nil {
		return domain.ONTSignalPoint{}, fmt.Errorf("gagal unmarshal signal: %w", err)
	}

	// Gunakan score sebagai timestamp (lebih akurat daripada prefix member)
	point.Timestamp = time.Unix(int64(z.Score), 0)

	return point, nil
}
