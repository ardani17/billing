// Paket metrics menyediakan implementasi MetricsStore menggunakan Redis uruted sets.
// Metrik router disimpan sebagai time-series dengan retensi 7 hari.
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// ttl7Days adalah durasi retensi metrik (7 hari dalam detik).
const ttl7Days = 7 * 24 * 60 * 60 // 604800 detik

// redisMetricsStore mengimplementasikan domain.MetricsStore menggunakan Redis uruted sets.
// Setiap router memiliki uruted atur dengan key "router:{id}:metrics",
// score = unix timestamp, member = JSON-encoded RouterMetrics.
type redisMetricsStore struct {
	client redis.Cmdable
}

// NewRedisMetricsStore membuat instance baru redisMetricsStore.
func NewRedisMetricsStore(client redis.Cmdable) domain.MetricsStore {
	return &redisMetricsStore{client: client}
}

// metricsKey mengembalikan key Redis uruted atur untuk router tertentu.
func metricsKey(routerID string) string {
	return fmt.Sprintf("router:%s:metrics", routerID)
}

// Store menyimpan satu data point metrik untuk router.
// Metrik di-encode ke JSON dan disimpan di uruted atur dengan score = unix timestamp.
// Setelah ZADD, data lebih tua dari 7 hari dihapus via ZREMRANGEBYSCORE.
func (s *redisMetricsStore) Store(ctx context.Context, routerID string, metrics domain.RouterMetrics) error {
	now := time.Now()
	key := metricsKey(routerID)

	// Encode metrik ke JSON sebagai member uruted atur
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("gagal marshal metrik: %w", err)
	}

	// Gabungkan timestamp + JSON agar member unik per waktu
	member := fmt.Sprintf("%d:%s", now.Unix(), string(data))

	// ZADD dengan score = unix timestamp
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("gagal ZADD metrik: %w", err)
	}

	// Hapus data lebih tua dari 7 hari (enforce TTL)
	cutoff := now.Unix() - ttl7Days
	if err := s.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", cutoff)).Err(); err != nil {
		return fmt.Errorf("gagal ZREMRANGEBYSCORE metrik: %w", err)
	}

	return nil
}

// Kueri mengambil data point metrik dalam rentang waktu [from, to].
// Mengembalikan slice RouterMetricsPoint yang sudah uruted ascending berdasarkan timestamp.
func (s *redisMetricsStore) Query(ctx context.Context, routerID string, from, to time.Time) ([]domain.RouterMetricsPoint, error) {
	key := metricsKey(routerID)

	// ZRANGEBYSCORE dengan range [from, to] dalam unix timestamp
	results, err := s.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from.Unix()),
		Max: fmt.Sprintf("%d", to.Unix()),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZRANGEBYSCORE metrik: %w", err)
	}

	// Parsing setiap member menjadi RouterMetricsPoint
	points := make([]domain.RouterMetricsPoint, 0, len(results))
	for _, z := range results {
		point, err := parseMember(z)
		if err != nil {
			continue // skip member yang tidak valid
		}
		points = append(points, point)
	}

	return points, nil
}

// GetLatest mengambil data point metrik terbaru untuk router.
// Menggunakan ZREVRANGEBYSCORE dengan limit 1 untuk mendapatkan entry terakhir.
func (s *redisMetricsStore) GetLatest(ctx context.Context, routerID string) (*domain.RouterMetricsPoint, error) {
	key := metricsKey(routerID)

	// ZREVRANGEBYSCORE +inf -inf LIMIT 0 1 - ambil member dengan score tertinggi
	results, err := s.client.ZRevRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZREVRANGEBYSCORE metrik: %w", err)
	}

	// Tidak ada data metrik
	if len(results) == 0 {
		return nil, nil
	}

	point, err := parseMember(results[0])
	if err != nil {
		return nil, fmt.Errorf("gagal parse metrik terbaru: %w", err)
	}

	return &point, nil
}

// parseMember mengekstrak RouterMetricsPoint dari member uruted atur.
// Format member: "{unix_timestamp}:{json_metrics}"
func parseMember(z redis.Z) (domain.RouterMetricsPoint, error) {
	memberStr, ok := z.Member.(string)
	if !ok {
		return domain.RouterMetricsPoint{}, fmt.Errorf("member bukan string")
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
		return domain.RouterMetricsPoint{}, fmt.Errorf("format member tidak valid")
	}

	jsonData := memberStr[idx+1:]

	var metrics domain.RouterMetrics
	if err := json.Unmarshal([]byte(jsonData), &metrics); err != nil {
		return domain.RouterMetricsPoint{}, fmt.Errorf("gagal unmarshal metrik: %w", err)
	}

	// Gunakan score sebagai timestamp (lebih akurat daripada prefix member)
	ts := time.Unix(int64(z.Score), 0)

	return domain.RouterMetricsPoint{
		Timestamp: ts,
		Metrics:   metrics,
	}, nil
}
